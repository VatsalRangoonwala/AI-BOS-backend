package ratelimit

import (
	"context"
	"testing"
	"time"
)

func TestNilLimiterFailsOpen(t *testing.T) {
	l := New(nil)
	allowed, err := l.Allow(context.Background(), "test_key", 10, time.Minute)
	if err != nil {
		t.Fatalf("unexpected error from nil limiter: %v", err)
	}
	if !allowed {
		t.Fatal("expected nil limiter to fail open and allow request")
	}
}
