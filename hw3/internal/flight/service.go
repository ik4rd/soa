package flight

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"regexp"
	"time"

	flightv1 "flightbooking/api/flight/v1"
	"flightbooking/internal/platform"

	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Service struct {
	flightv1.UnimplementedFlightServiceServer
	DB    *sql.DB
	Cache *redis.Client
	TTL   time.Duration
}

var iata = regexp.MustCompile(`^[A-Z]{3}$`)

const flightColumns = `id, flight_number, airline_code, origin, destination, departure_at, arrival_at, total_seats, available_seats, price_minor, status`

type scanner interface{ Scan(...any) error }

func readFlight(row scanner) (*flightv1.Flight, error) {
	f := &flightv1.Flight{}
	var departure, arrival time.Time
	var state string
	err := row.Scan(
		&f.Id, &f.FlightNumber, &f.AirlineCode, &f.Origin, &f.Destination, &departure, &arrival,
		&f.TotalSeats, &f.AvailableSeats, &f.PriceMinor, &state,
	)
	if err != nil {
		return nil, platform.DBError(err)
	}

	f.DepartureAt, f.ArrivalAt = timestamppb.New(departure), timestamppb.New(arrival)
	f.Status = flightv1.FlightStatus(flightv1.FlightStatus_value["FLIGHT_STATUS_"+state])

	return f, nil
}

func (s *Service) cached(ctx context.Context, key string, result proto.Message) bool {
	if s.Cache != nil {
		data, err := s.Cache.Get(ctx, key).Bytes()
		if err == nil && proto.Unmarshal(data, result) == nil {
			log.Printf("cache hit key=%s", key)
			return true
		}

		if err != nil && !errors.Is(err, redis.Nil) {
			log.Printf("cache read: %v", err)
		}
	}

	log.Printf("cache miss key=%s", key)

	return false
}

func (s *Service) cache(ctx context.Context, key string, result proto.Message) {
	if s.Cache == nil {
		return
	}

	data, err := proto.Marshal(result)
	if err == nil {
		err = s.Cache.Set(ctx, key, data, s.TTL).Err()
	}

	if err != nil {
		log.Printf("cache write: %v", err)
	}
}

func (s *Service) GetFlight(ctx context.Context, req *flightv1.GetFlightRequest) (*flightv1.Flight,
	error) {
	if !platform.UUID(req.FlightId) {
		return nil, status.Error(codes.InvalidArgument, "flight_id must be a UUID")
	}

	var version string
	err := s.DB.QueryRowContext(
		ctx,
		`SELECT v.version FROM flights f JOIN route_versions v USING (origin, destination) WHERE f.id=$1`,
		req.FlightId,
	).Scan(&version)
	if err != nil {
		return nil, platform.DBError(err)
	}

	key := "flight:" + req.FlightId + ":" + version
	f := &flightv1.Flight{}
	if s.cached(ctx, key, f) {
		return f, nil
	}

	f, err = readFlight(
		s.DB.QueryRowContext(
			ctx, `SELECT `+flightColumns+` FROM flights WHERE id=$1`, req.FlightId,
		),
	)
	if err == nil {
		s.cache(ctx, key, f)
	}

	return f, err
}

func (s *Service) SearchFlights(ctx context.Context,
	req *flightv1.SearchFlightsRequest) (*flightv1.SearchFlightsResponse, error) {
	if !iata.MatchString(req.Origin) || !iata.MatchString(req.Destination) || req.Origin == req.Destination {
		return nil, status.Error(
			codes.InvalidArgument, "origin and destination must be distinct IATA codes",
		)
	}

	var day any
	date := "all"
	if req.Date != nil {
		if req.Date.CheckValid() != nil || !req.Date.AsTime().Equal(req.Date.AsTime().Truncate(24*time.Hour)) {
			return nil, status.Error(codes.InvalidArgument, "date must be midnight UTC")
		}

		day = req.Date.AsTime()
		date = req.Date.AsTime().Format(time.DateOnly)
	}

	var version string
	err := s.DB.QueryRowContext(
		ctx, `SELECT version FROM route_versions WHERE origin=$1 AND destination=$2`, req.Origin,
		req.Destination,
	).Scan(&version)
	if errors.Is(err, sql.ErrNoRows) {
		version, err = "empty", nil
	}

	if err != nil {
		return nil, platform.DBError(err)
	}

	key := fmt.Sprintf("search:%s:%s:%s:%s", req.Origin, req.Destination, date, version)
	result := &flightv1.SearchFlightsResponse{Flights: []*flightv1.Flight{}}
	if s.cached(ctx, key, result) {
		return result, nil
	}

	rows, err := s.DB.QueryContext(
		ctx,
		`SELECT `+flightColumns+` FROM flights WHERE origin=$1 AND destination=$2 AND status='SCHEDULED' AND ($3::timestamptz IS NULL OR (departure_at >= $3 AND departure_at < $3::timestamptz + interval '1 day')) ORDER BY departure_at, id`,
		req.Origin, req.Destination, day,
	)
	if err != nil {
		return nil, platform.DBError(err)
	}
	defer rows.Close()

	for rows.Next() {
		f, err := readFlight(rows)
		if err != nil {
			return nil, err
		}

		result.Flights = append(result.Flights, f)
	}

	if err := rows.Err(); err != nil {
		return nil, platform.DBError(err)
	}

	s.cache(ctx, key, result)

	return result, nil
}

func readReservation(row scanner) (*flightv1.SeatReservation, error) {
	r := &flightv1.SeatReservation{}
	var state string
	err := row.Scan(&r.BookingId, &r.FlightId, &r.SeatCount, &state)
	r.Status = flightv1.ReservationStatus(flightv1.ReservationStatus_value["RESERVATION_STATUS_"+state])

	return r, err
}

func (s *Service) transaction(ctx context.Context, bookingID string) (*sql.Tx, error) {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, platform.DBError(err)
	}

	if _, err = tx.ExecContext(
		ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, bookingID,
	); err != nil {
		tx.Rollback()
		return nil, platform.DBError(err)
	}

	return tx, nil
}

func (s *Service) ReserveSeats(ctx context.Context,
	req *flightv1.ReserveSeatsRequest) (*flightv1.SeatReservation, error) {
	if !platform.UUID(req.BookingId) || !platform.UUID(req.FlightId) || req.SeatCount <= 0 {
		return nil, status.Error(
			codes.InvalidArgument,
			"UUID booking_id, UUID flight_id and positive seat_count are required",
		)
	}

	tx, err := s.transaction(ctx, req.BookingId)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	existing, err := readReservation(
		tx.QueryRowContext(
			ctx,
			`SELECT booking_id, flight_id, seat_count, status FROM seat_reservations WHERE booking_id=$1`,
			req.BookingId,
		),
	)
	if err == nil {
		if existing.FlightId != req.FlightId || existing.SeatCount != req.SeatCount {
			return nil, status.Error(
				codes.AlreadyExists, "booking_id already has different reservation parameters",
			)
		}

		if existing.Status != flightv1.ReservationStatus_RESERVATION_STATUS_ACTIVE {
			return nil, status.Error(codes.FailedPrecondition, "reservation already released")
		}

		return existing, nil
	}

	if !errors.Is(err, sql.ErrNoRows) {
		return nil, platform.DBError(err)
	}

	var cancelled bool
	if err = tx.QueryRowContext(
		ctx, `SELECT EXISTS (SELECT 1 FROM reservation_cancellations WHERE booking_id=$1)`,
		req.BookingId,
	).Scan(&cancelled); err != nil {
		return nil, platform.DBError(err)
	}

	if cancelled {
		return nil, status.Error(codes.FailedPrecondition, "reservation already canceled")
	}

	f, err := readFlight(
		tx.QueryRowContext(
			ctx, `SELECT `+flightColumns+` FROM flights WHERE id=$1 FOR UPDATE`, req.FlightId,
		),
	)
	if err != nil {
		return nil, err
	}

	if f.Status != flightv1.FlightStatus_FLIGHT_STATUS_SCHEDULED {
		return nil, status.Error(codes.FailedPrecondition, "flight is not scheduled")
	}

	if f.AvailableSeats < req.SeatCount {
		return nil, status.Error(codes.ResourceExhausted, "not enough seats")
	}

	if _, err = tx.ExecContext(
		ctx, `UPDATE flights SET available_seats=available_seats-$2 WHERE id=$1`, req.FlightId,
		req.SeatCount,
	); err != nil {
		return nil, platform.DBError(err)
	}

	if _, err = tx.ExecContext(
		ctx, `INSERT INTO seat_reservations VALUES ($1,$2,$3,'ACTIVE')`, req.BookingId,
		req.FlightId, req.SeatCount,
	); err != nil {
		return nil, platform.DBError(err)
	}

	if err = tx.Commit(); err != nil {
		return nil, platform.DBError(err)
	}

	return &flightv1.SeatReservation{
		BookingId: req.BookingId, FlightId: req.FlightId, SeatCount: req.SeatCount,
		Status: flightv1.ReservationStatus_RESERVATION_STATUS_ACTIVE,
	}, nil
}

func (s *Service) ReleaseReservation(ctx context.Context,
	req *flightv1.ReleaseReservationRequest) (*flightv1.SeatReservation, error) {
	if !platform.UUID(req.BookingId) {
		return nil, status.Error(codes.InvalidArgument, "booking_id must be a UUID")
	}

	tx, err := s.transaction(ctx, req.BookingId)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	r, err := readReservation(
		tx.QueryRowContext(
			ctx,
			`SELECT booking_id, flight_id, seat_count, status FROM seat_reservations WHERE booking_id=$1 FOR UPDATE`,
			req.BookingId,
		),
	)
	if errors.Is(err, sql.ErrNoRows) {
		if _, err = tx.ExecContext(
			ctx, `INSERT INTO reservation_cancellations VALUES ($1) ON CONFLICT DO NOTHING`,
			req.BookingId,
		); err != nil {
			return nil, platform.DBError(err)
		}

		if err = tx.Commit(); err != nil {
			return nil, platform.DBError(err)
		}

		return nil, status.Error(codes.NotFound, "reservation not found; late reserve prevented")
	}

	if err != nil {
		return nil, platform.DBError(err)
	}

	if r.Status != flightv1.ReservationStatus_RESERVATION_STATUS_ACTIVE {
		return r, nil
	}

	if _, err = tx.ExecContext(
		ctx, `SELECT id FROM flights WHERE id=$1 FOR UPDATE`, r.FlightId,
	); err != nil {
		return nil, platform.DBError(err)
	}

	if _, err = tx.ExecContext(
		ctx, `UPDATE flights SET available_seats=available_seats+$2 WHERE id=$1`, r.FlightId,
		r.SeatCount,
	); err != nil {
		return nil, platform.DBError(err)
	}

	if _, err = tx.ExecContext(
		ctx, `UPDATE seat_reservations SET status='RELEASED' WHERE booking_id=$1`, req.BookingId,
	); err != nil {
		return nil, platform.DBError(err)
	}

	if err = tx.Commit(); err != nil {
		return nil, platform.DBError(err)
	}

	r.Status = flightv1.ReservationStatus_RESERVATION_STATUS_RELEASED

	return r, nil
}
