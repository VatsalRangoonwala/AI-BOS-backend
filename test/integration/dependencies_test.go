//go:build integration

package integration_test

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/VatsalRangoonwala/AI-BOS-backend/internal/platform/httpserver"
	"github.com/VatsalRangoonwala/AI-BOS-backend/internal/platform/observability"
	"github.com/VatsalRangoonwala/AI-BOS-backend/internal/platform/postgres"
	redisstore "github.com/VatsalRangoonwala/AI-BOS-backend/internal/platform/redis"
)

func TestPostgresAndRedisAreReachable(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	database, err := postgres.Open(ctx, postgres.Config{
		URL:             environmentOrDefault("TEST_DATABASE_URL", "postgres://aibos:aibos_local@localhost:5433/aibos?sslmode=disable"),
		PoolMax:         4,
		PoolMin:         1,
		ConnectTimeout:  3 * time.Second,
		HealthTimeout:   3 * time.Second,
		MaxConnLifetime: 5 * time.Minute,
		MaxConnIdleTime: time.Minute,
	})
	if err != nil {
		t.Fatalf("open PostgreSQL pool: %v", err)
	}
	defer database.Close()
	if err := database.Check(ctx); err != nil {
		t.Fatalf("PostgreSQL health check: %v", err)
	}

	cache, err := redisstore.Open(redisstore.Config{
		URL:           environmentOrDefault("TEST_REDIS_URL", "redis://localhost:6380/0"),
		DialTimeout:   3 * time.Second,
		ReadTimeout:   3 * time.Second,
		WriteTimeout:  3 * time.Second,
		HealthTimeout: 3 * time.Second,
	})
	if err != nil {
		t.Fatalf("open Redis client: %v", err)
	}
	defer func() {
		if closeErr := cache.Close(); closeErr != nil {
			t.Errorf("close Redis client: %v", closeErr)
		}
	}()
	if err := cache.Check(ctx); err != nil {
		t.Fatalf("Redis health check: %v", err)
	}

	server := httpserver.New(httpserver.Config{
		Addr:              "127.0.0.1:0",
		FrontendOrigins:   []string{"http://localhost:3000"},
		ReadHeaderTimeout: time.Second,
		ReadTimeout:       time.Second,
		WriteTimeout:      2 * time.Second,
		IdleTimeout:       time.Second,
		RequestTimeout:    time.Second,
		ShutdownTimeout:   time.Second,
		MaxBodyBytes:      1024,
	}, slog.New(slog.NewJSONHandler(io.Discard, nil)), httpserver.Dependencies{
		Postgres: database,
		Redis:    cache,
	}, observability.NoopHooks())
	request := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	response := httptest.NewRecorder()
	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("readiness status = %d, want 200; body = %s", response.Code, response.Body.String())
	}
}

func environmentOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
