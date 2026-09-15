package config

import (
	"strings"
	"testing"
	"time"
)

func TestLoadFromUsesSafeLocalDefaults(t *testing.T) {
	t.Parallel()

	cfg, err := LoadFrom(mapLookup(map[string]string{"APP_ENV": "local"}))
	if err != nil {
		t.Fatalf("LoadFrom() error = %v", err)
	}

	if cfg.App.Environment != EnvironmentLocal {
		t.Fatalf("Environment = %q, want local", cfg.App.Environment)
	}
	if cfg.HTTP.Addr != ":8080" {
		t.Fatalf("HTTP address = %q, want :8080", cfg.HTTP.Addr)
	}
	if cfg.Postgres.URL != "postgres://aibos:aibos_local@localhost:5432/aibos?sslmode=disable" {
		t.Fatalf("unexpected local PostgreSQL URL")
	}
	if cfg.Redis.URL != "redis://localhost:6379/0" {
		t.Fatalf("unexpected local Redis URL")
	}
	if cfg.HTTP.MaxBodyBytes != 1<<20 {
		t.Fatalf("MaxBodyBytes = %d, want %d", cfg.HTTP.MaxBodyBytes, 1<<20)
	}
}

func TestLoadFromRequiresEnvironment(t *testing.T) {
	t.Parallel()

	_, err := LoadFrom(mapLookup(nil))
	if err == nil || !strings.Contains(err.Error(), "APP_ENV is required") {
		t.Fatalf("LoadFrom() error = %v, want APP_ENV required error", err)
	}
}

func TestLoadFromRejectsMissingProductionConfiguration(t *testing.T) {
	t.Parallel()

	_, err := LoadFrom(mapLookup(map[string]string{"APP_ENV": "production"}))
	if err == nil {
		t.Fatal("LoadFrom() error = nil, want validation error")
	}
	for _, key := range []string{"DATABASE_URL", "REDIS_URL", "FRONTEND_ORIGINS", "JWT_SECRET"} {
		if !strings.Contains(err.Error(), key) {
			t.Errorf("error %q does not mention %s", err, key)
		}
	}
}

func TestLoadFromAcceptsSecureProductionConfiguration(t *testing.T) {
	t.Parallel()

	cfg, err := LoadFrom(mapLookup(map[string]string{
		"APP_ENV":          "production",
		"DATABASE_URL":     "postgres://aibos:secret@db.example.com:5432/aibos?sslmode=verify-full",
		"REDIS_URL":        "rediss://:secret@redis.example.com:6379/0",
		"FRONTEND_ORIGINS": "https://app.example.com,https://admin.example.com",
		"JWT_SECRET":       "a-secure-production-jwt-secret-key-32-chars-long",
		"DB_POOL_MIN":      "4",
		"DB_POOL_MAX":      "40",
	}))
	if err != nil {
		t.Fatalf("LoadFrom() error = %v", err)
	}
	if cfg.Postgres.PoolMin != 4 || cfg.Postgres.PoolMax != 40 {
		t.Fatalf("pool bounds = %d/%d, want 4/40", cfg.Postgres.PoolMin, cfg.Postgres.PoolMax)
	}
	if len(cfg.HTTP.FrontendOrigins) != 2 {
		t.Fatalf("origins = %v, want two entries", cfg.HTTP.FrontendOrigins)
	}
}

func TestLoadFromRejectsUnsafeAndInconsistentValues(t *testing.T) {
	t.Parallel()

	_, err := LoadFrom(mapLookup(map[string]string{
		"APP_ENV":              "production",
		"DATABASE_URL":         "postgres://aibos:do-not-print@db.example.com:5432/aibos?sslmode=disable",
		"REDIS_URL":            "redis://:do-not-print@redis.example.com:6379/0",
		"FRONTEND_ORIGINS":     "http://app.example.com,*",
		"DB_POOL_MIN":          "10",
		"DB_POOL_MAX":          "5",
		"HTTP_REQUEST_TIMEOUT": "30s",
		"HTTP_WRITE_TIMEOUT":   "10s",
	}))
	if err == nil {
		t.Fatal("LoadFrom() error = nil, want validation error")
	}
	if strings.Contains(err.Error(), "do-not-print") {
		t.Fatalf("validation error leaked a secret: %v", err)
	}
	for _, expected := range []string{
		"DATABASE_URL must require TLS",
		"REDIS_URL must use TLS",
		"FRONTEND_ORIGINS",
		"DB_POOL_MIN must not exceed DB_POOL_MAX",
		"HTTP_REQUEST_TIMEOUT must not exceed HTTP_WRITE_TIMEOUT",
	} {
		if !strings.Contains(err.Error(), expected) {
			t.Errorf("error %q does not contain %q", err, expected)
		}
	}
}

func TestLoadFromParsesDurations(t *testing.T) {
	t.Parallel()

	cfg, err := LoadFrom(mapLookup(map[string]string{"APP_ENV": "local", "DB_HEALTH_TIMEOUT": "750ms"}))
	if err != nil {
		t.Fatalf("LoadFrom() error = %v", err)
	}
	if cfg.Postgres.HealthTimeout != 750*time.Millisecond {
		t.Fatalf("Postgres health timeout = %s, want 750ms", cfg.Postgres.HealthTimeout)
	}
}

func mapLookup(values map[string]string) LookupFunc {
	return func(key string) (string, bool) {
		value, ok := values[key]
		return value, ok
	}
}
