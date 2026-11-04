package booking

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"

	flightv1 "flightbooking/api/flight/v1"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func respond(w http.ResponseWriter, value any, err error, success int) {
	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		code := http.StatusInternalServerError

		switch status.Code(err) {
		case codes.InvalidArgument:
			code = 400
		case codes.NotFound:
			code = 404
		case codes.AlreadyExists, codes.ResourceExhausted, codes.FailedPrecondition:
			code = 409
		case codes.Unauthenticated, codes.PermissionDenied:
			code = 502
		case codes.Unavailable:
			code = 503
		case codes.DeadlineExceeded:
			code = 504
		case codes.Canceled:
			code = 408
		}

		message := status.Convert(err).Message()
		if code == 500 || code == 502 {
			log.Printf("request: %v", err)
			message = http.StatusText(code)
		}

		w.WriteHeader(code)
		json.NewEncoder(w).Encode(map[string]string{"error": message})

		return
	}

	var data []byte
	if message, ok := value.(proto.Message); ok {
		data, err = (protojson.MarshalOptions{
			UseProtoNames: true, EmitUnpopulated: true,
		}).Marshal(message)
	} else {
		data, err = json.Marshal(value)
	}

	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte(`{"error":"response encoding failed"}`))

		return
	}

	w.WriteHeader(success)
	w.Write(data)
}

func (s *Service) Handler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc(
		"GET /flights", func(w http.ResponseWriter, r *http.Request) {
			q := r.URL.Query()
			req := &flightv1.SearchFlightsRequest{
				Origin: q.Get("origin"), Destination: q.Get("destination"),
			}
			if q.Has("date") {
				day, err := time.Parse(time.DateOnly, q.Get("date"))
				if err != nil {
					respond(
						w, nil, status.Error(codes.InvalidArgument, "date must be YYYY-MM-DD"), 0,
					)
					return
				}

				req.Date = timestamppb.New(day)
			}

			result, err := s.Flights.SearchFlights(r.Context(), req)
			respond(w, result, err, 200)
		},
	)

	mux.HandleFunc(
		"GET /flights/{id}", func(w http.ResponseWriter, r *http.Request) {
			result, err := s.Flights.GetFlight(
				r.Context(), &flightv1.GetFlightRequest{FlightId: r.PathValue("id")},
			)
			respond(w, result, err, 200)
		},
	)

	mux.HandleFunc(
		"POST /bookings", func(w http.ResponseWriter, r *http.Request) {
			var req CreateRequest
			decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10))
			decoder.DisallowUnknownFields()
			if err := decoder.Decode(&req); err != nil {
				respond(w, nil, status.Error(codes.InvalidArgument, "invalid JSON booking"), 0)
				return
			}

			if decoder.Decode(new(any)) != io.EOF {
				respond(
					w, nil, status.Error(codes.InvalidArgument, "expected a single JSON object"), 0,
				)
				return
			}

			result, err := s.Create(r.Context(), req)
			if err == nil {
				w.Header().Set("Location", "/bookings/"+result.ID)
			}

			respond(w, result, err, 201)
		},
	)

	mux.HandleFunc(
		"GET /bookings/{id}", func(w http.ResponseWriter, r *http.Request) {
			result, err := s.Get(r.Context(), r.PathValue("id"))
			respond(w, result, err, 200)
		},
	)

	mux.HandleFunc(
		"POST /bookings/{id}/cancel", func(w http.ResponseWriter, r *http.Request) {
			result, err := s.Cancel(r.Context(), r.PathValue("id"))
			respond(w, result, err, 200)
		},
	)

	mux.HandleFunc(
		"GET /bookings", func(w http.ResponseWriter, r *http.Request) {
			result, err := s.List(r.Context(), r.URL.Query().Get("user_id"))
			respond(w, result, err, 200)
		},
	)

	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
			defer cancel()

			mux.ServeHTTP(w, r.WithContext(ctx))
		},
	)
}
