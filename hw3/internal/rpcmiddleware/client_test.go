package rpcmiddleware

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func configuration() Config {
	return Config{
		Key: "secret", Timeout: 20 * time.Millisecond, Backoff: 10 * time.Millisecond, Attempts: 3,
		Threshold: 2, Window: time.Second, OpenTimeout: 30 * time.Millisecond,
	}
}

func TestRetryAndMetadata(t *testing.T) {
	for _, code := range []codes.Code{
		codes.Unavailable, codes.DeadlineExceeded, codes.InvalidArgument, codes.NotFound,
		codes.ResourceExhausted, codes.Unauthenticated,
	} {
		t.Run(
			code.String(), func(t *testing.T) {
				interceptor := Client(configuration())
				calls := 0
				started := time.Now()
				err := interceptor(
					context.Background(), "test", nil, nil, nil,
					func(ctx context.Context, _ string, _, _ any, _ *grpc.ClientConn,
						_ ...grpc.CallOption) error {
						calls++
						md, _ := metadata.FromOutgoingContext(ctx)
						if md.Get("x-api-key")[0] != "secret" {
							t.Fatal("missing credentials")
						}

						if _, ok := ctx.Deadline(); !ok {
							t.Fatal("missing deadline")
						}

						return status.Error(code, "test")
					},
				)

				expected := 1
				if transient(err) {
					expected = 3
					if time.Since(started) < 30*time.Millisecond {
						t.Fatal("missing exponential backoff")
					}
				}

				if calls != expected {
					t.Fatalf("calls=%d expected=%d", calls, expected)
				}
			},
		)
	}
}

func TestPerAttemptTimeout(t *testing.T) {
	c := configuration()
	c.Timeout = 5 * time.Millisecond
	c.Backoff = time.Millisecond
	calls := 0
	err := Client(c)(
		context.Background(), "test", nil, nil, nil,
		func(ctx context.Context, _ string, _, _ any, _ *grpc.ClientConn,
			_ ...grpc.CallOption) error {
			calls++
			<-ctx.Done()

			return status.FromContextError(ctx.Err()).Err()
		},
	)

	if status.Code(err) != codes.DeadlineExceeded || calls != 3 {
		t.Fatalf("calls=%d err=%v", calls, err)
	}
}

func TestCancellationStopsBackoff(t *testing.T) {
	c := configuration()
	c.Backoff = time.Second
	ctx, cancel := context.WithCancel(context.Background())
	calls := 0
	err := Client(c)(
		ctx, "test", nil, nil, nil,
		func(context.Context, string, any, any, *grpc.ClientConn, ...grpc.CallOption) error {
			calls++
			cancel()

			return status.Error(codes.Unavailable, "down")
		},
	)

	if calls != 1 || err == nil {
		t.Fatalf("calls=%d err=%v", calls, err)
	}
}

func TestCircuitBreakerSingleProbe(t *testing.T) {
	c := configuration()
	c.Attempts = 1
	interceptor := Client(c)
	var calls atomic.Int32
	invoke := func(context.Context, string, any, any, *grpc.ClientConn, ...grpc.CallOption) error {
		calls.Add(1)
		return status.Error(codes.Unavailable, "down")
	}

	call := func(invoke grpc.UnaryInvoker) error {
		return interceptor(context.Background(), "test", nil, nil, nil, invoke)
	}

	for range 3 {
		call(invoke)
	}

	if calls.Load() != 2 {
		t.Fatalf("open breaker made RPC: %d", calls.Load())
	}

	time.Sleep(c.OpenTimeout + 5*time.Millisecond)
	entered, release := make(chan struct{}), make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()

		call(
			func(context.Context, string, any, any, *grpc.ClientConn, ...grpc.CallOption) error {
				calls.Add(1)
				close(entered)
				<-release

				return nil
			},
		)
	}()

	<-entered

	for range 20 {
		if status.Code(call(invoke)) != codes.Unavailable {
			t.Fatal("concurrent probe admitted")
		}
	}

	close(release)
	wg.Wait()
	if calls.Load() != 3 {
		t.Fatalf("probe count=%d", calls.Load())
	}

	if err := call(
		func(context.Context, string, any, any, *grpc.ClientConn,
			...grpc.CallOption) error {
			return nil
		},
	); err != nil {
		t.Fatalf("not closed: %v", err)
	}
}

func TestFailedProbeAndWindow(t *testing.T) {
	c := configuration()
	c.Attempts = 3
	c.Threshold = 1
	interceptor := Client(c)
	calls := 0
	invoke := func(context.Context, string, any, any, *grpc.ClientConn, ...grpc.CallOption) error {
		calls++
		return status.Error(codes.Unavailable, "down")
	}

	interceptor(context.Background(), "test", nil, nil, nil, invoke)
	time.Sleep(c.OpenTimeout + 5*time.Millisecond)
	interceptor(context.Background(), "test", nil, nil, nil, invoke)
	interceptor(context.Background(), "test", nil, nil, nil, invoke)
	if calls != 4 {
		t.Fatalf("failed probe retried or did not open: %d", calls)
	}

	b := &breaker{
		state: "CLOSED", since: time.Now().Add(-time.Second), failures: 1, config: configuration(),
	}
	b.config.Window = time.Millisecond
	generation, _, _ := b.enter()
	b.finish(generation, status.Error(codes.Unavailable, "down"))
	if b.state != "CLOSED" || b.failures != 1 {
		t.Fatal("window did not expire")
	}
}

func TestAuth(t *testing.T) {
	for _, key := range []string{"", "wrong", "secret"} {
		ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-api-key", key))
		called := false
		_, err := Auth("secret")(
			ctx, nil, nil,
			func(context.Context, any) (any, error) { called = true; return nil, nil },
		)

		if called != (key == "secret") || (key != "secret" && status.Code(err) != codes.Unauthenticated) {
			t.Fatalf("key=%q called=%v err=%v", key, called, err)
		}
	}
}
