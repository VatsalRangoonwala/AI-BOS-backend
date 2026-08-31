package postgres

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakePool struct {
	ping   func(context.Context) error
	closed bool
}

func (pool *fakePool) Ping(ctx context.Context) error {
	return pool.ping(ctx)
}

func (pool *fakePool) Close() {
	pool.closed = true
}

func TestClientCheckReturnsPostgresFailure(t *testing.T) {
	t.Parallel()

	expected := errors.New("postgres unavailable")
	pool := &fakePool{ping: func(context.Context) error { return expected }}
	client := New(pool, time.Second)

	if err := client.Check(context.Background()); !errors.Is(err, expected) {
		t.Fatalf("Check() error = %v, want %v", err, expected)
	}
}

func TestClientCheckAppliesHealthTimeout(t *testing.T) {
	t.Parallel()

	pool := &fakePool{ping: func(ctx context.Context) error {
		<-ctx.Done()
		return ctx.Err()
	}}
	client := New(pool, 20*time.Millisecond)

	startedAt := time.Now()
	err := client.Check(context.Background())
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Check() error = %v, want deadline exceeded", err)
	}
	if elapsed := time.Since(startedAt); elapsed > time.Second {
		t.Fatalf("Check() took %s, timeout was not applied", elapsed)
	}
}

func TestClientCloseClosesPool(t *testing.T) {
	t.Parallel()

	pool := &fakePool{ping: func(context.Context) error { return nil }}
	New(pool, time.Second).Close()
	if !pool.closed {
		t.Fatal("Close() did not close the pool")
	}
}
