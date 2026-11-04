package integration

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	flightv1 "flightbooking/api/flight/v1"
	"flightbooking/internal/booking"
	"flightbooking/internal/flight"
	"flightbooking/internal/rpcmiddleware"

	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type fixture struct {
	db, bookings *sql.DB
	client       flightv1.FlightServiceClient
	cache        *redis.Client
	ctx          context.Context
	id           string
}

func setup(t *testing.T, seats int) *fixture {
	t.Helper()
	if os.Getenv("RUN_INTEGRATION") != "1" {
		t.Skip("run make integration")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	open := func(key string) *sql.DB {
		db, err := sql.Open("pgx", os.Getenv(key))
		if err != nil {
			t.Fatal(err)
		}

		t.Cleanup(func() { db.Close() })

		return db
	}

	f := &fixture{
		db: open("FLIGHT_DATABASE_URL"), bookings: open("BOOKING_DATABASE_URL"), ctx: ctx,
		id: uuid.NewString(),
	}
	_, err := f.db.ExecContext(
		ctx,
		`INSERT INTO flights (id,flight_number,airline_code,origin,destination,departure_at,arrival_at,total_seats,available_seats,price_minor,status) VALUES ($1,$2,'SU','VKO','KZN','2026-12-10 09:00:00Z','2026-12-10 10:30:00Z',$3,$3,12345,'SCHEDULED')`,
		f.id, "TEST-"+f.id, seats,
	)
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(
		func() {
			f.bookings.Exec(
				`DELETE FROM recovery_jobs WHERE booking_id IN (SELECT id FROM bookings WHERE flight_id=$1)`,
				f.id,
			)
			f.bookings.Exec(`DELETE FROM bookings WHERE flight_id=$1`, f.id)
			f.db.Exec(`DELETE FROM seat_reservations WHERE flight_id=$1`, f.id)
			f.db.Exec(`DELETE FROM flights WHERE id=$1`, f.id)
		},
	)

	conn, err := grpc.NewClient(
		os.Getenv("FLIGHT_GRPC_ADDR"), grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDisableRetry(), grpc.WithUnaryInterceptor(
			rpcmiddleware.Client(
				rpcmiddleware.Config{
					Key: os.Getenv("GRPC_API_KEY"), Timeout: time.Second,
					Backoff: 10 * time.Millisecond, Attempts: 3, Threshold: 5, Window: time.Second,
					OpenTimeout: time.Second,
				},
			),
		),
	)
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { conn.Close() })

	f.client = flightv1.NewFlightServiceClient(conn)
	f.cache = redis.NewFailoverClient(
		&redis.FailoverOptions{
			MasterName:    "flights",
			SentinelAddrs: strings.Split(os.Getenv("REDIS_SENTINEL_ADDRS"), ","),
		},
	)
	t.Cleanup(func() { f.cache.Close() })

	return f
}

func (f *fixture) seats(t *testing.T) int {
	t.Helper()
	var n int
	if err := f.db.QueryRowContext(
		f.ctx, `SELECT available_seats FROM flights WHERE id=$1`, f.id,
	).Scan(&n); err != nil {
		t.Fatal(err)
	}

	return n
}

func TestConcurrentLastSeat(t *testing.T) {
	f := setup(t, 1)
	var won atomic.Int32
	var winner string
	var mu sync.Mutex
	var wg sync.WaitGroup

	for range 24 {
		wg.Go(
			func() {
				id := uuid.NewString()
				_, err := f.client.ReserveSeats(
					f.ctx,
					&flightv1.ReserveSeatsRequest{FlightId: f.id, BookingId: id, SeatCount: 1},
				)
				if err == nil {
					won.Add(1)
					mu.Lock()
					winner = id
					mu.Unlock()
				} else if status.Code(err) != codes.ResourceExhausted {
					t.Errorf("unexpected: %v", err)
				}
			},
		)
	}

	wg.Wait()
	if won.Load() != 1 || f.seats(t) != 0 {
		t.Fatalf("won=%d seats=%d", won.Load(), f.seats(t))
	}

	for range 3 {
		if _, err := f.client.ReleaseReservation(
			f.ctx, &flightv1.ReleaseReservationRequest{BookingId: winner},
		); err != nil {
			t.Fatal(err)
		}
	}

	if f.seats(t) != 1 {
		t.Fatal("release counted twice")
	}
}

func TestReservationIdempotencyAndValidation(t *testing.T) {
	f := setup(t, 10)
	id := uuid.NewString()
	req := &flightv1.ReserveSeatsRequest{FlightId: f.id, BookingId: id, SeatCount: 2}
	var wg sync.WaitGroup

	for range 12 {
		wg.Go(
			func() {
				if _, err := f.client.ReserveSeats(f.ctx, req); err != nil {
					t.Errorf("reserve: %v", err)
				}
			},
		)
	}

	wg.Wait()
	if f.seats(t) != 8 {
		t.Fatal("duplicate reservation")
	}

	req.SeatCount = 3
	if _, err := f.client.ReserveSeats(f.ctx, req); status.Code(err) != codes.AlreadyExists {
		t.Fatalf("mismatch: %v", err)
	}

	_, err := f.client.ReleaseReservation(f.ctx, &flightv1.ReleaseReservationRequest{BookingId: id})
	if err != nil {
		t.Fatal(err)
	}

	req.SeatCount = 2
	if _, err := f.client.ReserveSeats(f.ctx, req); status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("resurrected reservation: %v", err)
	}

	unknown := uuid.NewString()
	t.Cleanup(
		func() {
			f.db.Exec(
				`DELETE FROM reservation_cancellations WHERE booking_id=$1`, unknown,
			)
		},
	)

	if _, err := f.client.ReleaseReservation(
		f.ctx, &flightv1.ReleaseReservationRequest{BookingId: unknown},
	); status.Code(err) != codes.NotFound {
		t.Fatal(err)
	}

	if _, err := f.client.ReserveSeats(
		f.ctx, &flightv1.ReserveSeatsRequest{FlightId: f.id, BookingId: unknown, SeatCount: 1},
	); status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("late reserve: %v", err)
	}

	for _, r := range []*flightv1.ReserveSeatsRequest{
		{
			FlightId: f.id, BookingId: uuid.NewString(), SeatCount: 0,
		}, {FlightId: "bad", BookingId: id, SeatCount: 1},
	} {
		if _, err := f.client.ReserveSeats(f.ctx, r); status.Code(err) != codes.InvalidArgument {
			t.Fatal(err)
		}
	}

	if _, err := f.client.GetFlight(
		f.ctx, &flightv1.GetFlightRequest{FlightId: uuid.NewString()},
	); status.Code(err) != codes.NotFound {
		t.Fatal(err)
	}
}

func TestAuthenticationAllMethods(t *testing.T) {
	f := setup(t, 1)
	conn, err := grpc.NewClient(
		os.Getenv("FLIGHT_GRPC_ADDR"), grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	client := flightv1.NewFlightServiceClient(conn)

	for _, key := range []string{"", "wrong"} {
		ctx := metadata.AppendToOutgoingContext(f.ctx, "x-api-key", key)
		_, e1 := client.GetFlight(ctx, &flightv1.GetFlightRequest{FlightId: f.id})
		_, e2 := client.SearchFlights(
			ctx, &flightv1.SearchFlightsRequest{Origin: "VKO", Destination: "KZN"},
		)
		_, e3 := client.ReserveSeats(
			ctx,
			&flightv1.ReserveSeatsRequest{FlightId: f.id, BookingId: uuid.NewString(), SeatCount: 1},
		)
		_, e4 := client.ReleaseReservation(
			ctx, &flightv1.ReleaseReservationRequest{BookingId: uuid.NewString()},
		)

		for _, err := range []error{e1, e2, e3, e4} {
			if status.Code(err) != codes.Unauthenticated {
				t.Fatalf("auth: %v", err)
			}
		}
	}
}

func TestCacheInvalidationAndFallback(t *testing.T) {
	f := setup(t, 5)
	get := func() *flightv1.Flight {
		v, err := f.client.GetFlight(f.ctx, &flightv1.GetFlightRequest{FlightId: f.id})
		if err != nil {
			t.Fatal(err)
		}

		return v
	}

	search := func(date *timestamppb.Timestamp) int32 {
		v, err := f.client.SearchFlights(
			f.ctx, &flightv1.SearchFlightsRequest{Origin: "VKO", Destination: "KZN", Date: date},
		)
		if err != nil {
			t.Fatal(err)
		}

		for _, flight := range v.Flights {
			if flight.Id == f.id {
				return flight.AvailableSeats
			}
		}

		t.Fatal("missing flight")

		return 0
	}

	day := timestamppb.New(time.Date(2026, 12, 10, 0, 0, 0, 0, time.UTC))
	get()
	get()
	search(nil)
	search(day)
	keys, err := f.cache.Keys(f.ctx, "flight:"+f.id+":*").Result()
	if err != nil || len(keys) == 0 {
		t.Fatalf("no cached flight: %v", err)
	}

	for _, key := range keys {
		ttl, err := f.cache.TTL(f.ctx, key).Result()
		if err != nil || ttl <= 0 || ttl > 10*time.Minute {
			t.Fatalf("TTL=%s err=%v", ttl, err)
		}
	}

	id := uuid.NewString()
	if _, err := f.client.ReserveSeats(
		f.ctx, &flightv1.ReserveSeatsRequest{FlightId: f.id, BookingId: id, SeatCount: 2},
	); err != nil {
		t.Fatal(err)
	}

	if get().AvailableSeats != 3 || search(nil) != 3 || search(day) != 3 {
		t.Fatal("stale cache after reserve")
	}

	if _, err := f.client.ReleaseReservation(
		f.ctx, &flightv1.ReleaseReservationRequest{BookingId: id},
	); err != nil {
		t.Fatal(err)
	}

	if get().AvailableSeats != 5 || search(nil) != 5 || search(day) != 5 {
		t.Fatal("stale cache after release")
	}

	if _, err := f.db.ExecContext(
		f.ctx, `UPDATE flights SET price_minor=999 WHERE id=$1`, f.id,
	); err != nil {
		t.Fatal(err)
	}

	if get().PriceMinor != 999 {
		t.Fatal("stale cache after flight update")
	}

	unavailable := redis.NewClient(
		&redis.Options{
			Addr: "127.0.0.1:1", MaxRetries: -1, DialTimeout: 10 * time.Millisecond,
		},
	)
	defer unavailable.Close()

	service := flight.Service{DB: f.db, Cache: unavailable, TTL: 5 * time.Minute}
	got, err := service.GetFlight(f.ctx, &flightv1.GetFlightRequest{FlightId: f.id})
	if err != nil || got.PriceMinor != 999 {
		t.Fatalf("Redis fallback: %v", err)
	}
}

func request(t *testing.T, method, path string, body any, want int) []byte {
	t.Helper()
	data, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}

	req, err := http.NewRequest(method, os.Getenv("BOOKING_URL")+path, bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}

	req.Header.Set("Content-Type", "application/json")
	response, err := (&http.Client{Timeout: 20 * time.Second}).Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()

	data, err = io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}

	if response.StatusCode != want {
		t.Fatalf("%s %s: status=%d want=%d body=%s", method, path, response.StatusCode, want, data)
	}

	return data
}

func TestRESTWorkflow(t *testing.T) {
	f := setup(t, 5)
	request(t, "GET", "/flights?origin=VKO&destination=KZN&date=2026-12-10", nil, 200)
	request(t, "GET", "/flights?origin=VKO", nil, 400)
	request(t, "GET", "/flights?origin=VKO&destination=KZN&date=bad", nil, 400)
	request(t, "GET", "/flights/"+uuid.NewString(), nil, 404)
	request(t, "GET", "/bookings/"+uuid.NewString(), nil, 404)
	request(t, "GET", "/bookings", nil, 400)
	input := booking.CreateRequest{
		UserID: f.id, FlightID: f.id, PassengerName: "Иван Иванов",
		PassengerEmail: "ivan@example.org", SeatCount: 2,
	}
	var b booking.Booking
	if err := json.Unmarshal(request(t, "POST", "/bookings", input, 201), &b); err != nil {
		t.Fatal(err)
	}

	if b.Status != "CONFIRMED" || b.TotalPriceMinor != 24690 || f.seats(t) != 3 {
		t.Fatalf("bad booking: %+v", b)
	}

	request(t, "GET", "/bookings/"+b.ID, nil, 200)
	var list []booking.Booking
	json.Unmarshal(request(t, "GET", "/bookings?user_id="+f.id, nil, 200), &list)
	if len(list) != 1 {
		t.Fatalf("bookings=%d", len(list))
	}

	f.db.ExecContext(f.ctx, `UPDATE flights SET price_minor=1 WHERE id=$1`, f.id)
	json.Unmarshal(request(t, "GET", "/bookings/"+b.ID, nil, 200), &b)
	if b.TotalPriceMinor != 24690 {
		t.Fatal("price snapshot changed")
	}

	input.SeatCount = 9
	request(t, "POST", "/bookings", input, 409)

	for range 2 {
		json.Unmarshal(request(t, "POST", "/bookings/"+b.ID+"/cancel", nil, 200), &b)
	}

	if b.Status != "CANCELLED" || f.seats(t) != 5 {
		t.Fatal("bad cancellation")
	}

	input.SeatCount = 0
	request(t, "POST", "/bookings", input, 400)
}

type faultyClient struct {
	flightv1.FlightServiceClient
	reserve func(context.Context, *flightv1.ReserveSeatsRequest) (*flightv1.SeatReservation, error)
	release func(context.Context, *flightv1.ReleaseReservationRequest) (*flightv1.SeatReservation,
		error)
}

func (c faultyClient) ReserveSeats(ctx context.Context, r *flightv1.ReserveSeatsRequest,
	_ ...grpc.CallOption) (*flightv1.SeatReservation, error) {
	return c.reserve(ctx, r)
}
func (c faultyClient) ReleaseReservation(ctx context.Context, r *flightv1.ReleaseReservationRequest,
	opts ...grpc.CallOption) (*flightv1.SeatReservation, error) {
	if c.release != nil {
		return c.release(ctx, r)
	}

	return c.FlightServiceClient.ReleaseReservation(ctx, r, opts...)
}

func TestRecoveryAfterLostReserveResponse(t *testing.T) {
	f := setup(t, 5)
	var id string
	c := faultyClient{
		FlightServiceClient: f.client, reserve: func(ctx context.Context,
			r *flightv1.ReserveSeatsRequest) (*flightv1.SeatReservation, error) {
			id = r.BookingId
			if _, err := f.client.ReserveSeats(ctx, r); err != nil {
				return nil, err
			}

			return nil, status.Error(codes.DeadlineExceeded, "lost response")
		},
	}

	service := booking.Service{DB: f.bookings, Flights: c}
	_, err := service.Create(
		f.ctx, booking.CreateRequest{
			UserID: f.id, FlightID: f.id, PassengerName: "Test", PassengerEmail: "test@example.org",
			SeatCount: 2,
		},
	)
	if status.Code(err) != codes.DeadlineExceeded {
		t.Fatal(err)
	}

	var count int
	f.bookings.QueryRowContext(f.ctx, `SELECT count(*) FROM bookings WHERE id=$1`, id).Scan(&count)
	if count != 0 {
		t.Fatal("booking created after failed reserve")
	}

	service.Flights = f.client
	if err := service.Recover(f.ctx); err != nil {
		t.Fatal(err)
	}

	if f.seats(t) != 5 {
		t.Fatal("leaked reservation")
	}
}

func TestRecoveryAfterBookingWriteFailure(t *testing.T) {
	f := setup(t, 5)
	var id string
	c := faultyClient{
		FlightServiceClient: f.client, reserve: func(ctx context.Context,
			r *flightv1.ReserveSeatsRequest) (*flightv1.SeatReservation, error) {
			id = r.BookingId
			if _, err := f.bookings.ExecContext(
				ctx,
				`INSERT INTO bookings VALUES ($1,$2,$3,'Test','test@example.org',1,1,'CANCELLED')`,
				id, f.id, f.id,
			); err != nil {
				return nil, err
			}

			return f.client.ReserveSeats(ctx, r)
		},
	}

	service := booking.Service{DB: f.bookings, Flights: c}
	_, err := service.Create(
		f.ctx, booking.CreateRequest{
			UserID: f.id, FlightID: f.id, PassengerName: "Test", PassengerEmail: "test@example.org",
			SeatCount: 2,
		},
	)
	if status.Code(err) != codes.Unavailable {
		t.Fatalf("expected failed booking insert: %v", err)
	}

	var reservations int
	if err := f.db.QueryRowContext(
		f.ctx, `SELECT count(*) FROM seat_reservations WHERE booking_id=$1`, id,
	).Scan(&reservations); err != nil || reservations != 1 {
		t.Fatalf(
			"reserve did not commit before booking insert failed: count=%d err=%v", reservations,
			err,
		)
	}

	service.Flights = f.client
	if err := service.Recover(f.ctx); err != nil {
		t.Fatal(err)
	}

	if f.seats(t) != 5 {
		t.Fatal("DB failure leaked seats")
	}
}

func TestRecoveryAfterLostReleaseResponse(t *testing.T) {
	f := setup(t, 5)
	service := booking.Service{DB: f.bookings, Flights: f.client}
	b, err := service.Create(
		f.ctx, booking.CreateRequest{
			UserID: f.id, FlightID: f.id, PassengerName: "Test", PassengerEmail: "test@example.org",
			SeatCount: 2,
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	service.Flights = faultyClient{
		FlightServiceClient: f.client, release: func(ctx context.Context,
			r *flightv1.ReleaseReservationRequest) (*flightv1.SeatReservation, error) {
			if _, err := f.client.ReleaseReservation(ctx, r); err != nil {
				return nil, err
			}

			return nil, status.Error(codes.Unavailable, "lost response")
		},
	}

	if _, err := service.Cancel(f.ctx, b.ID); status.Code(err) != codes.Unavailable {
		t.Fatal(err)
	}

	service.Flights = f.client
	if err := service.Recover(f.ctx); err != nil {
		t.Fatal(err)
	}

	b, err = service.Get(f.ctx, b.ID)
	if err != nil || b.Status != "CANCELLED" || f.seats(t) != 5 {
		t.Fatalf("recovery: %+v %v", b, err)
	}
}

func TestSearchFiltersAndConstraints(t *testing.T) {
	f := setup(t, 5)

	for _, state := range []string{"DEPARTED", "CANCELLED", "COMPLETED"} {
		if _, err := f.db.ExecContext(
			f.ctx, `UPDATE flights SET status=$2 WHERE id=$1`, f.id, state,
		); err != nil {
			t.Fatal(err)
		}

		list, err := f.client.SearchFlights(
			f.ctx, &flightv1.SearchFlightsRequest{Origin: "VKO", Destination: "KZN"},
		)
		if err != nil {
			t.Fatal(err)
		}

		for _, item := range list.Flights {
			if item.Id == f.id {
				t.Fatalf("search returned %s", state)
			}
		}

		if _, err := f.client.ReserveSeats(
			f.ctx,
			&flightv1.ReserveSeatsRequest{FlightId: f.id, BookingId: uuid.NewString(), SeatCount: 1},
		); status.Code(err) != codes.FailedPrecondition {
			t.Fatal(err)
		}
	}

	for _, assignment := range []string{
		"available_seats=-1", "total_seats=0", "price_minor=0", "arrival_at=departure_at",
		"origin='bad'",
	} {
		if _, err := f.db.ExecContext(
			f.ctx, `UPDATE flights SET `+assignment+` WHERE id=$1`, f.id,
		); err == nil {
			t.Fatalf("constraint allowed %s", assignment)
		}
	}
}

func TestCacheOldFillIsUnreachable(t *testing.T) {
	f := setup(t, 5)
	old, err := f.client.GetFlight(f.ctx, &flightv1.GetFlightRequest{FlightId: f.id})
	if err != nil {
		t.Fatal(err)
	}

	var version string
	if err := f.db.QueryRowContext(
		f.ctx, `SELECT version FROM route_versions WHERE origin='VKO' AND destination='KZN'`,
	).Scan(&version); err != nil {
		t.Fatal(err)
	}

	if _, err := f.client.ReserveSeats(
		f.ctx,
		&flightv1.ReserveSeatsRequest{FlightId: f.id, BookingId: uuid.NewString(), SeatCount: 1},
	); err != nil {
		t.Fatal(err)
	}

	data, err := proto.Marshal(old)
	if err != nil {
		t.Fatal(err)
	}

	if err := f.cache.Set(
		f.ctx, fmt.Sprintf("flight:%s:%s", f.id, version), data, 5*time.Minute,
	).Err(); err != nil {
		t.Fatal(err)
	}

	got, err := f.client.GetFlight(f.ctx, &flightv1.GetFlightRequest{FlightId: f.id})
	if err != nil || got.AvailableSeats != 4 {
		t.Fatalf("old fill reached reader: %v %v", got, err)
	}
}
