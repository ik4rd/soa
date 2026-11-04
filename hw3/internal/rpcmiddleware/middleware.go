package rpcmiddleware

import (
	"context"
	"crypto/subtle"
	"log"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func Auth(key string) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo,
		next grpc.UnaryHandler) (any, error) {
		values := metadata.ValueFromIncomingContext(ctx, "x-api-key")
		if key == "" || len(values) != 1 || subtle.ConstantTimeCompare(
			[]byte(values[0]), []byte(key),
		) != 1 {
			return nil, status.Error(codes.Unauthenticated, "invalid service credentials")
		}

		return next(ctx, req)
	}
}

type Config struct {
	Key                                   string
	Timeout, Backoff, Window, OpenTimeout time.Duration
	Attempts, Threshold                   int
}

type breaker struct {
	mu         sync.Mutex
	state      string
	failures   int
	since      time.Time
	generation uint64
	config     Config
}

func transient(err error) bool {
	c := status.Code(err)
	return c == codes.Unavailable || c == codes.DeadlineExceeded
}

func (b *breaker) transition(state string) {
	log.Printf("circuit breaker %s -> %s", b.state, state)
	b.state, b.since, b.failures = state, time.Now(), 0
	b.generation++
}

func (b *breaker) enter() (uint64, bool, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.state == "OPEN" && time.Since(b.since) >= b.config.OpenTimeout {
		b.transition("HALF_OPEN")
		return b.generation, true, nil
	}

	if b.state != "CLOSED" {
		return 0, false, status.Error(codes.Unavailable, "flight service circuit is open")
	}

	if time.Since(b.since) >= b.config.Window {
		b.failures = 0
		b.since = time.Now()
	}

	return b.generation, false, nil
}

func (b *breaker) finish(generation uint64, err error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if generation != b.generation {
		return
	}

	if b.state == "HALF_OPEN" {
		if transient(err) {
			b.transition("OPEN")
		} else {
			b.transition("CLOSED")
		}

		return
	}

	if !transient(err) {
		b.failures = 0
		return
	}

	b.failures++
	if b.failures >= b.config.Threshold {
		b.transition("OPEN")
	}
}

func Client(c Config) grpc.UnaryClientInterceptor {
	if c.Attempts < 1 || c.Attempts > 3 || c.Threshold < 1 || c.Timeout <= 0 || c.Backoff <= 0 || c.Window <= 0 || c.OpenTimeout <= 0 || c.Key == "" {
		panic("invalid RPC configuration")
	}

	b := &breaker{state: "CLOSED", since: time.Now(), config: c}

	return func(ctx context.Context, method string, req, reply any, conn *grpc.ClientConn,
		invoke grpc.UnaryInvoker, options ...grpc.CallOption) error {
		if ctx.Err() != nil {
			return status.FromContextError(ctx.Err()).Err()
		}

		generation, probe, err := b.enter()
		if err != nil {
			return err
		}

		attempts := c.Attempts
		if probe {
			attempts = 1
		}

		md, _ := metadata.FromOutgoingContext(ctx)
		md = md.Copy()
		md.Set("x-api-key", c.Key)
		ctx = metadata.NewOutgoingContext(ctx, md)

		for attempt := 0; attempt < attempts; attempt++ {
			callCtx, cancel := context.WithTimeout(ctx, c.Timeout)
			err = invoke(callCtx, method, req, reply, conn, options...)
			cancel()
			if !transient(err) || ctx.Err() != nil || attempt == attempts-1 {
				break
			}

			delay := c.Backoff * time.Duration(1<<attempt)
			log.Printf(
				"grpc retry method=%s attempt=%d backoff=%s code=%s", method, attempt+2, delay,
				status.Code(err),
			)
			timer := time.NewTimer(delay)

			select {
			case <-ctx.Done():
				timer.Stop()
				err = status.FromContextError(ctx.Err()).Err()
			case <-timer.C:
			}

			if ctx.Err() != nil {
				break
			}
		}

		if ctx.Err() == nil {
			b.finish(generation, err)
		} else {
			b.finish(generation, status.Error(codes.Canceled, "caller canceled"))
		}

		return err
	}
}
