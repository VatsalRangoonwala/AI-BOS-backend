package redis

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakeBackend struct {
	ping   func(context.Context) error
	closed bool
}

func (backend *fakeBackend) Ping(ctx context.Context) error {
	return backend.ping(ctx)
}

func (backend *fakeBackend) Close() error {
	backend.closed = true
	return nil
}

func TestClientCheckReturnsRedisFailure(t *testing.T) {
	t.Parallel()

	expected := errors.New("redis unavailable")
	backend := &fakeBackend{ping: func(context.Context) error { return expected }}
	client := New(backend, time.Second)

	if err := client.Check(context.Background()); !errors.Is(err, expected) {
		t.Fatalf("Check() error = %v, want %v", err, expected)
	}
}

func TestClientCheckAppliesHealthTimeout(t *testing.T) {
	t.Parallel()

	backend := &fakeBackend{ping: func(ctx context.Context) error {
		<-ctx.Done()
		return ctx.Err()
	}}
	client := New(backend, 20*time.Millisecond)

	startedAt := time.Now()
	err := client.Check(context.Background())
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Check() error = %v, want deadline exceeded", err)
	}
	if elapsed := time.Since(startedAt); elapsed > time.Second {
		t.Fatalf("Check() took %s, timeout was not applied", elapsed)
	}
}

func TestClientCloseClosesBackend(t *testing.T) {
	t.Parallel()

	backend := &fakeBackend{ping: func(context.Context) error { return nil }}
	if err := New(backend, time.Second).Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if !backend.closed {
		t.Fatal("Close() did not close the backend")
	}
}
