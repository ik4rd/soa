package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	flightv1 "flightbooking/api/flight/v1"
	"flightbooking/internal/flight"
	"flightbooking/internal/platform"
	"flightbooking/internal/rpcmiddleware"

	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
)

func main() {
	key := os.Getenv("GRPC_API_KEY")
	if key == "" {
		log.Fatal("GRPC_API_KEY is required")
	}

	ttl := platform.Duration("CACHE_TTL", "5m")
	if ttl < 5*time.Minute || ttl > 10*time.Minute {
		log.Fatal("CACHE_TTL must be between 5m and 10m")
	}

	db := platform.Database()
	defer db.Close()

	cache := redis.NewFailoverClient(
		&redis.FailoverOptions{
			MasterName: platform.Env("REDIS_MASTER_NAME", "flights"),
			SentinelAddrs: strings.Split(
				platform.Env("REDIS_SENTINEL_ADDRS", "localhost:26379"), ",",
			),
			DialTimeout: 200 * time.Millisecond, ReadTimeout: 200 * time.Millisecond,
			WriteTimeout: 200 * time.Millisecond,
			MaxRetries:   -1, DialerRetries: 1, ContextTimeoutEnabled: true,
		},
	)
	defer cache.Close()

	addr := platform.Env("GRPC_ADDR", ":50051")
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatal(err)
	}

	server := grpc.NewServer(grpc.UnaryInterceptor(rpcmiddleware.Auth(key)))
	flightv1.RegisterFlightServiceServer(server, &flight.Service{DB: db, Cache: cache, TTL: ttl})

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	done := make(chan struct{})
	go func() {
		defer close(done)

		<-ctx.Done()
		timer := time.AfterFunc(10*time.Second, server.Stop)
		defer timer.Stop()

		server.GracefulStop()
	}()

	log.Printf("Flight listening on %s", addr)
	if err := server.Serve(listener); err != nil {
		log.Fatal(err)
	}

	<-done
}
