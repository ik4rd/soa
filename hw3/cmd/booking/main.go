package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	flightv1 "flightbooking/api/flight/v1"
	"flightbooking/internal/booking"
	"flightbooking/internal/platform"
	"flightbooking/internal/rpcmiddleware"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	config := rpcmiddleware.Config{
		Key: os.Getenv("GRPC_API_KEY"), Timeout: platform.Duration("RPC_TIMEOUT", "2s"),
		Attempts:    platform.Integer("RETRY_MAX_ATTEMPTS", 3),
		Backoff:     platform.Duration("RETRY_INITIAL_BACKOFF", "100ms"),
		Threshold:   platform.Integer("CB_FAILURE_THRESHOLD", 5),
		Window:      platform.Duration("CB_WINDOW", "30s"),
		OpenTimeout: platform.Duration("CB_OPEN_TIMEOUT", "15s"),
	}

	db := platform.Database()
	defer db.Close()

	conn, err := grpc.NewClient(
		platform.Env("FLIGHT_GRPC_ADDR", "localhost:50051"),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(rpcmiddleware.Client(config)), grpc.WithDisableRetry(),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	service := &booking.Service{DB: db, Flights: flightv1.NewFlightServiceClient(conn)}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	done := make(chan struct{})
	go func() { defer close(done); service.RunRecovery(ctx) }()

	server := &http.Server{
		Addr: platform.Env("HTTP_ADDR", ":8080"), Handler: service.Handler(),
		ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 20 * time.Second,
		WriteTimeout: 20 * time.Second, IdleTimeout: 60 * time.Second,
	}
	shutdownDone := make(chan struct{})
	go func() {
		defer close(shutdownDone)

		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()

		if err := server.Shutdown(shutdown); err != nil {
			log.Printf("shutdown: %v", err)
		}
	}()

	log.Printf("Booking listening on %s", server.Addr)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}

	<-shutdownDone
	<-done
}
