package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type Environment string

const (
	EnvironmentLocal      Environment = "local"
	EnvironmentTest       Environment = "test"
	EnvironmentStaging    Environment = "staging"
	EnvironmentProduction Environment = "production"
)

type Config struct {
	App      AppConfig
	HTTP     HTTPConfig
	Postgres PostgresConfig
	Redis    RedisConfig
	Worker   WorkerConfig
	Auth     AuthConfig
}

type AuthConfig struct {
	JWTSecret       []byte
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
}

type AppConfig struct {
	Environment Environment
	LogLevel    string
}

type HTTPConfig struct {
	Addr              string
	FrontendOrigins   []string
	ReadHeaderTimeout time.Duration
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	RequestTimeout    time.Duration
	ShutdownTimeout   time.Duration
	MaxBodyBytes      int64
}

type PostgresConfig struct {
	URL             string
	PoolMax         int32
	PoolMin         int32
	ConnectTimeout  time.Duration
	HealthTimeout   time.Duration
	MaxConnLifetime time.Duration
	MaxConnIdleTime time.Duration
}

type RedisConfig struct {
	URL           string
	DialTimeout   time.Duration
	ReadTimeout   time.Duration
	WriteTimeout  time.Duration
	HealthTimeout time.Duration
}

type WorkerConfig struct {
	HealthInterval time.Duration
}

type LookupFunc func(string) (string, bool)

func Load() (Config, error) {
	return LoadFrom(os.LookupEnv)
}

func LoadFrom(lookup LookupFunc) (Config, error) {
	var validationErrors []error

	environmentValue, ok := lookup("APP_ENV")
	if !ok || strings.TrimSpace(environmentValue) == "" {
		return Config{}, fmt.Errorf("APP_ENV is required")
	}
	environment, err := parseEnvironment(environmentValue)
	if err != nil {
		return Config{}, err
	}

	databaseURL := requiredOutsideLocal(
		lookup,
		"DATABASE_URL",
		"postgres://aibos:aibos_local@localhost:5432/aibos?sslmode=disable",
		environment,
		&validationErrors,
	)
	redisURL := requiredOutsideLocal(
		lookup,
		"REDIS_URL",
		"redis://localhost:6379/0",
		environment,
		&validationErrors,
	)
	originsValue := requiredOutsideLocal(
		lookup,
		"FRONTEND_ORIGINS",
		"http://localhost:3000",
		environment,
		&validationErrors,
	)
	jwtSecretValue := requiredOutsideLocal(
		lookup,
		"JWT_SECRET",
		"aibos-local-development-secret-key-32b-min",
		environment,
		&validationErrors,
	)
	if environment == EnvironmentProduction && (len(jwtSecretValue) < 32 || jwtSecretValue == "aibos-local-development-secret-key-32b-min") {
		validationErrors = append(validationErrors, fmt.Errorf("JWT_SECRET must be at least 32 characters in production"))
	}

	cfg := Config{
		App: AppConfig{
			Environment: environment,
			LogLevel:    valueOrDefault(lookup, "LOG_LEVEL", "info"),
		},
		HTTP: HTTPConfig{
			Addr:              valueOrDefault(lookup, "HTTP_ADDR", ":8080"),
			FrontendOrigins:   parseOrigins(originsValue, environment, &validationErrors),
			ReadHeaderTimeout: parseDuration(lookup, "HTTP_READ_HEADER_TIMEOUT", "5s", &validationErrors),
			ReadTimeout:       parseDuration(lookup, "HTTP_READ_TIMEOUT", "15s", &validationErrors),
			WriteTimeout:      parseDuration(lookup, "HTTP_WRITE_TIMEOUT", "20s", &validationErrors),
			IdleTimeout:       parseDuration(lookup, "HTTP_IDLE_TIMEOUT", "60s", &validationErrors),
			RequestTimeout:    parseDuration(lookup, "HTTP_REQUEST_TIMEOUT", "15s", &validationErrors),
			ShutdownTimeout:   parseDuration(lookup, "HTTP_SHUTDOWN_TIMEOUT", "15s", &validationErrors),
			MaxBodyBytes:      parseInt64(lookup, "HTTP_MAX_BODY_BYTES", 1<<20, &validationErrors),
		},
		Postgres: PostgresConfig{
			URL:             databaseURL,
			PoolMax:         parseInt32(lookup, "DB_POOL_MAX", 20, &validationErrors),
			PoolMin:         parseInt32(lookup, "DB_POOL_MIN", 2, &validationErrors),
			ConnectTimeout:  parseDuration(lookup, "DB_CONNECT_TIMEOUT", "5s", &validationErrors),
			HealthTimeout:   parseDuration(lookup, "DB_HEALTH_TIMEOUT", "2s", &validationErrors),
			MaxConnLifetime: parseDuration(lookup, "DB_MAX_CONN_LIFETIME", "30m", &validationErrors),
			MaxConnIdleTime: parseDuration(lookup, "DB_MAX_CONN_IDLE_TIME", "5m", &validationErrors),
		},
		Redis: RedisConfig{
			URL:           redisURL,
			DialTimeout:   parseDuration(lookup, "REDIS_DIAL_TIMEOUT", "3s", &validationErrors),
			ReadTimeout:   parseDuration(lookup, "REDIS_READ_TIMEOUT", "2s", &validationErrors),
			WriteTimeout:  parseDuration(lookup, "REDIS_WRITE_TIMEOUT", "2s", &validationErrors),
			HealthTimeout: parseDuration(lookup, "REDIS_HEALTH_TIMEOUT", "2s", &validationErrors),
		},
		Worker: WorkerConfig{
			HealthInterval: parseDuration(lookup, "WORKER_HEALTH_INTERVAL", "30s", &validationErrors),
		},
		Auth: AuthConfig{
			JWTSecret:       []byte(jwtSecretValue),
			AccessTokenTTL:  parseDuration(lookup, "ACCESS_TOKEN_TTL", "15m", &validationErrors),
			RefreshTokenTTL: parseDuration(lookup, "REFRESH_TOKEN_TTL", "168h", &validationErrors),
		},
	}

	validate(&cfg, &validationErrors)
	if len(validationErrors) > 0 {
		return Config{}, errors.Join(validationErrors...)
	}

	return cfg, nil
}

func parseEnvironment(value string) (Environment, error) {
	environment := Environment(strings.ToLower(strings.TrimSpace(value)))
	switch environment {
	case EnvironmentLocal, EnvironmentTest, EnvironmentStaging, EnvironmentProduction:
		return environment, nil
	default:
		return "", fmt.Errorf("APP_ENV must be one of local, test, staging, production")
	}
}

func requiredOutsideLocal(
	lookup LookupFunc,
	key string,
	localDefault string,
	environment Environment,
	validationErrors *[]error,
) string {
	if value, ok := lookup(key); ok && strings.TrimSpace(value) != "" {
		return strings.TrimSpace(value)
	}
	if environment == EnvironmentLocal {
		return localDefault
	}
	*validationErrors = append(*validationErrors, fmt.Errorf("%s is required when APP_ENV is %s", key, environment))
	return ""
}

func valueOrDefault(lookup LookupFunc, key, fallback string) string {
	if value, ok := lookup(key); ok && strings.TrimSpace(value) != "" {
		return strings.TrimSpace(value)
	}
	return fallback
}

func parseDuration(lookup LookupFunc, key, fallback string, validationErrors *[]error) time.Duration {
	value := valueOrDefault(lookup, key, fallback)
	duration, err := time.ParseDuration(value)
	if err != nil {
		*validationErrors = append(*validationErrors, fmt.Errorf("%s must be a valid duration", key))
		return 0
	}
	return duration
}

func parseInt32(lookup LookupFunc, key string, fallback int32, validationErrors *[]error) int32 {
	value, ok := lookup(key)
	if !ok || strings.TrimSpace(value) == "" {
		return fallback
	}
	parsed, err := strconv.ParseInt(strings.TrimSpace(value), 10, 32)
	if err != nil {
		*validationErrors = append(*validationErrors, fmt.Errorf("%s must be a valid 32-bit integer", key))
		return 0
	}
	return int32(parsed)
}

func parseInt64(lookup LookupFunc, key string, fallback int64, validationErrors *[]error) int64 {
	value, ok := lookup(key)
	if !ok || strings.TrimSpace(value) == "" {
		return fallback
	}
	parsed, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
	if err != nil {
		*validationErrors = append(*validationErrors, fmt.Errorf("%s must be a valid integer", key))
		return 0
	}
	return parsed
}

func parseOrigins(value string, environment Environment, validationErrors *[]error) []string {
	if value == "" {
		return nil
	}

	seen := make(map[string]struct{})
	origins := make([]string, 0)
	for rawOrigin := range strings.SplitSeq(value, ",") {
		origin := strings.TrimSpace(rawOrigin)
		if origin == "" {
			continue
		}
		if origin == "*" {
			*validationErrors = append(*validationErrors, fmt.Errorf("FRONTEND_ORIGINS must not contain a wildcard"))
			continue
		}
		parsed, err := url.Parse(origin)
		if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Path != "" || parsed.RawQuery != "" || parsed.Fragment != "" {
			*validationErrors = append(*validationErrors, fmt.Errorf("FRONTEND_ORIGINS entries must be absolute HTTP origins without paths"))
			continue
		}
		if environment == EnvironmentProduction && parsed.Scheme != "https" {
			*validationErrors = append(*validationErrors, fmt.Errorf("FRONTEND_ORIGINS entries must use HTTPS in production"))
			continue
		}
		if _, ok := seen[origin]; ok {
			continue
		}
		seen[origin] = struct{}{}
		origins = append(origins, origin)
	}
	return origins
}

func validate(cfg *Config, validationErrors *[]error) {
	if cfg.App.LogLevel != "debug" && cfg.App.LogLevel != "info" && cfg.App.LogLevel != "warn" && cfg.App.LogLevel != "error" {
		*validationErrors = append(*validationErrors, fmt.Errorf("LOG_LEVEL must be one of debug, info, warn, error"))
	}
	if cfg.HTTP.Addr == "" {
		*validationErrors = append(*validationErrors, fmt.Errorf("HTTP_ADDR must not be empty"))
	}
	if len(cfg.HTTP.FrontendOrigins) == 0 {
		*validationErrors = append(*validationErrors, fmt.Errorf("FRONTEND_ORIGINS must contain at least one origin"))
	}
	positiveDuration("HTTP_READ_HEADER_TIMEOUT", cfg.HTTP.ReadHeaderTimeout, validationErrors)
	positiveDuration("HTTP_READ_TIMEOUT", cfg.HTTP.ReadTimeout, validationErrors)
	positiveDuration("HTTP_WRITE_TIMEOUT", cfg.HTTP.WriteTimeout, validationErrors)
	positiveDuration("HTTP_IDLE_TIMEOUT", cfg.HTTP.IdleTimeout, validationErrors)
	positiveDuration("HTTP_REQUEST_TIMEOUT", cfg.HTTP.RequestTimeout, validationErrors)
	positiveDuration("HTTP_SHUTDOWN_TIMEOUT", cfg.HTTP.ShutdownTimeout, validationErrors)
	positiveDuration("DB_CONNECT_TIMEOUT", cfg.Postgres.ConnectTimeout, validationErrors)
	positiveDuration("DB_HEALTH_TIMEOUT", cfg.Postgres.HealthTimeout, validationErrors)
	positiveDuration("DB_MAX_CONN_LIFETIME", cfg.Postgres.MaxConnLifetime, validationErrors)
	positiveDuration("DB_MAX_CONN_IDLE_TIME", cfg.Postgres.MaxConnIdleTime, validationErrors)
	positiveDuration("REDIS_DIAL_TIMEOUT", cfg.Redis.DialTimeout, validationErrors)
	positiveDuration("REDIS_READ_TIMEOUT", cfg.Redis.ReadTimeout, validationErrors)
	positiveDuration("REDIS_WRITE_TIMEOUT", cfg.Redis.WriteTimeout, validationErrors)
	positiveDuration("REDIS_HEALTH_TIMEOUT", cfg.Redis.HealthTimeout, validationErrors)
	positiveDuration("WORKER_HEALTH_INTERVAL", cfg.Worker.HealthInterval, validationErrors)

	if cfg.HTTP.MaxBodyBytes <= 0 {
		*validationErrors = append(*validationErrors, fmt.Errorf("HTTP_MAX_BODY_BYTES must be greater than zero"))
	}
	if cfg.Postgres.PoolMax <= 0 {
		*validationErrors = append(*validationErrors, fmt.Errorf("DB_POOL_MAX must be greater than zero"))
	}
	if cfg.Postgres.PoolMin < 0 {
		*validationErrors = append(*validationErrors, fmt.Errorf("DB_POOL_MIN must not be negative"))
	}
	if cfg.Postgres.PoolMin > cfg.Postgres.PoolMax {
		*validationErrors = append(*validationErrors, fmt.Errorf("DB_POOL_MIN must not exceed DB_POOL_MAX"))
	}
	if cfg.HTTP.RequestTimeout > cfg.HTTP.WriteTimeout {
		*validationErrors = append(*validationErrors, fmt.Errorf("HTTP_REQUEST_TIMEOUT must not exceed HTTP_WRITE_TIMEOUT"))
	}

	validateDependencyURL("DATABASE_URL", cfg.Postgres.URL, "postgres", cfg.App.Environment, validationErrors)
	validateDependencyURL("REDIS_URL", cfg.Redis.URL, "redis", cfg.App.Environment, validationErrors)
}

func positiveDuration(key string, value time.Duration, validationErrors *[]error) {
	if value <= 0 {
		*validationErrors = append(*validationErrors, fmt.Errorf("%s must be greater than zero", key))
	}
}

func validateDependencyURL(key, value, dependency string, environment Environment, validationErrors *[]error) {
	if value == "" {
		return
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.Host == "" {
		*validationErrors = append(*validationErrors, fmt.Errorf("%s must be a valid connection URL", key))
		return
	}

	switch dependency {
	case "postgres":
		if parsed.Scheme != "postgres" && parsed.Scheme != "postgresql" {
			*validationErrors = append(*validationErrors, fmt.Errorf("DATABASE_URL must use postgres or postgresql scheme"))
			return
		}
		if environment == EnvironmentProduction {
			sslMode := parsed.Query().Get("sslmode")
			if sslMode != "require" && sslMode != "verify-ca" && sslMode != "verify-full" {
				*validationErrors = append(*validationErrors, fmt.Errorf("DATABASE_URL must require TLS in production"))
			}
		}
	case "redis":
		if parsed.Scheme != "redis" && parsed.Scheme != "rediss" {
			*validationErrors = append(*validationErrors, fmt.Errorf("REDIS_URL must use redis or rediss scheme"))
			return
		}
		if environment == EnvironmentProduction && parsed.Scheme != "rediss" {
			*validationErrors = append(*validationErrors, fmt.Errorf("REDIS_URL must use TLS in production"))
		}
	}
}
