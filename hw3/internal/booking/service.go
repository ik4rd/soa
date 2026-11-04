package booking

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"log"
	"math"
	"net/mail"
	"strings"
	"time"

	flightv1 "flightbooking/api/flight/v1"
	"flightbooking/internal/platform"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type CreateRequest struct {
	UserID         string `json:"user_id"`
	FlightID       string `json:"flight_id"`
	PassengerName  string `json:"passenger_name"`
	PassengerEmail string `json:"passenger_email"`
	SeatCount      int32  `json:"seat_count"`
}

type Booking struct {
	ID string `json:"id"`
	CreateRequest
	TotalPriceMinor int64  `json:"total_price_minor"`
	Status          string `json:"status"`
}

type Service struct {
	Flights flightv1.FlightServiceClient
	DB      *sql.DB
}

const columns = `id, user_id, flight_id, passenger_name, passenger_email, seat_count, total_price_minor, status`

type scanner interface{ Scan(...any) error }

func readBooking(row scanner) (Booking, error) {
	var b Booking
	err := row.Scan(
		&b.ID, &b.UserID, &b.FlightID, &b.PassengerName, &b.PassengerEmail, &b.SeatCount,
		&b.TotalPriceMinor, &b.Status,
	)

	return b, platform.DBError(err)
}

func (s *Service) Get(ctx context.Context, id string) (Booking, error) {
	if !platform.UUID(id) {
		return Booking{}, status.Error(codes.InvalidArgument, "booking id must be a UUID")
	}

	return readBooking(
		s.DB.QueryRowContext(
			ctx, `SELECT `+columns+` FROM bookings WHERE id=$1`, id,
		),
	)
}

func (s *Service) List(ctx context.Context, user string) ([]Booking, error) {
	if strings.TrimSpace(user) == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	rows, err := s.DB.QueryContext(
		ctx, `SELECT `+columns+` FROM bookings WHERE user_id=$1 ORDER BY id`, user,
	)
	if err != nil {
		return nil, platform.DBError(err)
	}
	defer rows.Close()

	var result []Booking

	for rows.Next() {
		b, err := readBooking(rows)
		if err != nil {
			return nil, err
		}

		result = append(result, b)
	}

	return result, platform.DBError(rows.Err())
}

func (s *Service) lock(ctx context.Context, id string, try bool) (*sql.Conn, func(), error) {
	conn, err := s.DB.Conn(ctx)
	if err != nil {
		return nil, nil, platform.DBError(err)
	}

	locked := true
	if try {
		err = conn.QueryRowContext(
			ctx, `SELECT pg_try_advisory_lock(hashtextextended($1,0))`, id,
		).Scan(&locked)
	} else {
		_, err = conn.ExecContext(ctx, `SELECT pg_advisory_lock(hashtextextended($1,0))`, id)
	}

	if err != nil || !locked {
		if err != nil {
			conn.Raw(func(any) error { return driver.ErrBadConn })
		}

		conn.Close()

		return nil, nil, platform.DBError(err)
	}

	release := func() {
		cleanup, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		if _, err := conn.ExecContext(
			cleanup, `SELECT pg_advisory_unlock(hashtextextended($1,0))`, id,
		); err != nil {
			conn.Raw(func(any) error { return driver.ErrBadConn })
		}

		conn.Close()
	}

	return conn, release, nil
}

func (s *Service) Create(ctx context.Context, req CreateRequest) (Booking, error) {
	address, err := mail.ParseAddress(req.PassengerEmail)
	if strings.TrimSpace(req.UserID) == "" || strings.TrimSpace(req.PassengerName) == "" || err != nil || address.Address != req.PassengerEmail || !platform.UUID(req.FlightID) || req.SeatCount <= 0 {
		return Booking{}, status.Error(
			codes.InvalidArgument,
			"valid user_id, flight_id, passenger_name, passenger_email and positive seat_count are required",
		)
	}

	f, err := s.Flights.GetFlight(ctx, &flightv1.GetFlightRequest{FlightId: req.FlightID})
	if err != nil {
		return Booking{}, err
	}

	if f.PriceMinor <= 0 || f.PriceMinor > math.MaxInt64/int64(req.SeatCount) {
		return Booking{}, status.Error(codes.InvalidArgument, "total price is out of range")
	}

	b := Booking{
		ID: uuid.NewString(), CreateRequest: req,
		TotalPriceMinor: f.PriceMinor * int64(req.SeatCount), Status: "CONFIRMED",
	}
	conn, unlock, err := s.lock(ctx, b.ID, false)
	if err != nil {
		return Booking{}, err
	}
	defer unlock()

	if _, err = conn.ExecContext(
		ctx, `INSERT INTO recovery_jobs VALUES ($1,'RELEASE')`, b.ID,
	); err != nil {
		return Booking{}, platform.DBError(err)
	}

	reservation, err := s.Flights.ReserveSeats(
		ctx, &flightv1.ReserveSeatsRequest{
			BookingId: b.ID, FlightId: req.FlightID, SeatCount: req.SeatCount,
		},
	)
	if err != nil {
		return Booking{}, err
	}

	if reservation.Status != flightv1.ReservationStatus_RESERVATION_STATUS_ACTIVE {
		return Booking{}, status.Error(codes.FailedPrecondition, "reservation is not active")
	}

	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return Booking{}, platform.DBError(err)
	}
	defer tx.Rollback()

	if _, err = tx.ExecContext(
		ctx, `INSERT INTO bookings (`+columns+`) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`, b.ID, b.UserID,
		b.FlightID, b.PassengerName, b.PassengerEmail, b.SeatCount, b.TotalPriceMinor, b.Status,
	); err != nil {
		return Booking{}, platform.DBError(err)
	}

	if _, err = tx.ExecContext(
		ctx, `DELETE FROM recovery_jobs WHERE booking_id=$1`, b.ID,
	); err != nil {
		return Booking{}, platform.DBError(err)
	}

	if err = tx.Commit(); err != nil {
		return Booking{}, platform.DBError(err)
	}

	return b, nil
}

func (s *Service) Cancel(ctx context.Context, id string) (Booking, error) {
	if !platform.UUID(id) {
		return Booking{}, status.Error(codes.InvalidArgument, "booking id must be a UUID")
	}

	conn, unlock, err := s.lock(ctx, id, false)
	if err != nil {
		return Booking{}, err
	}
	defer unlock()

	b, err := readBooking(
		conn.QueryRowContext(
			ctx, `SELECT `+columns+` FROM bookings WHERE id=$1`, id,
		),
	)
	if err != nil || b.Status == "CANCELLED" {
		return b, err
	}

	if _, err = conn.ExecContext(
		ctx, `INSERT INTO recovery_jobs VALUES ($1,'CANCEL') ON CONFLICT DO NOTHING`, id,
	); err != nil {
		return Booking{}, platform.DBError(err)
	}

	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return Booking{}, platform.DBError(err)
	}
	defer tx.Rollback()

	if err = s.finish(ctx, tx, id, "CANCEL"); err != nil {
		return Booking{}, err
	}

	b.Status = "CANCELLED"

	return b, nil
}

func (s *Service) finish(ctx context.Context, tx *sql.Tx, id, action string) error {
	_, err := s.Flights.ReleaseReservation(ctx, &flightv1.ReleaseReservationRequest{BookingId: id})
	if err != nil && status.Code(err) != codes.NotFound {
		return err
	}

	if action == "CANCEL" {
		if _, err = tx.ExecContext(
			ctx, `UPDATE bookings SET status='CANCELLED' WHERE id=$1`, id,
		); err != nil {
			return platform.DBError(err)
		}
	}

	if _, err = tx.ExecContext(
		ctx, `DELETE FROM recovery_jobs WHERE booking_id=$1`, id,
	); err != nil {
		return platform.DBError(err)
	}

	return platform.DBError(tx.Commit())
}

func (s *Service) Recover(ctx context.Context) error {
	rows, err := s.DB.QueryContext(ctx, `SELECT booking_id FROM recovery_jobs ORDER BY booking_id`)
	if err != nil {
		return platform.DBError(err)
	}

	var ids []string

	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			rows.Close()
			return platform.DBError(err)
		}

		ids = append(ids, id)
	}

	err = rows.Err()
	rows.Close()
	if err != nil {
		return platform.DBError(err)
	}

	for _, id := range ids {
		conn, unlock, err := s.lock(ctx, id, true)
		if err != nil {
			return err
		}

		if conn == nil {
			continue
		}

		err = func() error {
			defer unlock()

			var action string
			err := conn.QueryRowContext(
				ctx, `SELECT action FROM recovery_jobs WHERE booking_id=$1`, id,
			).Scan(&action)
			if errors.Is(err, sql.ErrNoRows) {
				return nil
			}

			if err != nil {
				return platform.DBError(err)
			}

			tx, err := conn.BeginTx(ctx, nil)
			if err != nil {
				return platform.DBError(err)
			}
			defer tx.Rollback()

			if err = s.finish(ctx, tx, id, action); err != nil {
				return err
			}

			log.Printf("recovered booking=%s action=%s", id, action)

			return nil
		}()

		if err != nil {
			return err
		}
	}

	return nil
}

func (s *Service) RunRecovery(ctx context.Context) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			jobCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
			err := s.Recover(jobCtx)
			cancel()
			if err != nil {
				log.Printf("recovery: %v", err)
			}
		}
	}
}
