package platform

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func Env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}

	return fallback
}

func Duration(key, fallback string) time.Duration {
	value, err := time.ParseDuration(Env(key, fallback))
	if err != nil || value <= 0 {
		log.Fatalf("invalid %s", key)
	}

	return value
}

func Integer(key string, fallback int) int {
	value, err := strconv.Atoi(Env(key, strconv.Itoa(fallback)))
	if err != nil || value <= 0 {
		log.Fatalf("invalid %s", key)
	}

	return value
}

func Database() *sql.DB {
	db, err := sql.Open("pgx", os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal(err)
	}

	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(5)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		log.Fatal(err)
	}

	return db
}

func UUID(value string) bool { id, err := uuid.Parse(value); return err == nil && id.String() == value }

func DBError(err error) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, sql.ErrNoRows) {
		return status.Error(codes.NotFound, "not found")
	}

	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return status.FromContextError(err).Err()
	}

	log.Printf("database: %v", err)

	return status.Error(codes.Unavailable, "database unavailable")
}
