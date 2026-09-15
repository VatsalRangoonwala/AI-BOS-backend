package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/VatsalRangoonwala/AI-BOS-backend/internal/audit"
	"github.com/VatsalRangoonwala/AI-BOS-backend/internal/businesses"
	"github.com/VatsalRangoonwala/AI-BOS-backend/internal/identity"
	"github.com/VatsalRangoonwala/AI-BOS-backend/internal/platform/config"
	"github.com/VatsalRangoonwala/AI-BOS-backend/internal/platform/httpserver"
	"github.com/VatsalRangoonwala/AI-BOS-backend/internal/platform/logging"
	"github.com/VatsalRangoonwala/AI-BOS-backend/internal/platform/observability"
	"github.com/VatsalRangoonwala/AI-BOS-backend/internal/platform/postgres"
	"github.com/VatsalRangoonwala/AI-BOS-backend/internal/platform/ratelimit"
	redisstore "github.com/VatsalRangoonwala/AI-BOS-backend/internal/platform/redis"
)

func main() {
	logger := logging.New(os.Stdout, "info")
	if err := run(logger); err != nil {
		logger.Error("api stopped", "error", err)
		os.Exit(1)
	}
}

func run(bootstrapLogger *slog.Logger) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	logger := logging.New(os.Stdout, cfg.App.LogLevel).With("service", "api", "environment", cfg.App.Environment)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	database, err := postgres.Open(ctx, postgres.Config{
		URL:             cfg.Postgres.URL,
		PoolMax:         cfg.Postgres.PoolMax,
		PoolMin:         cfg.Postgres.PoolMin,
		ConnectTimeout:  cfg.Postgres.ConnectTimeout,
		HealthTimeout:   cfg.Postgres.HealthTimeout,
		MaxConnLifetime: cfg.Postgres.MaxConnLifetime,
		MaxConnIdleTime: cfg.Postgres.MaxConnIdleTime,
	})
	if err != nil {
		return err
	}
	defer database.Close()

	cache, err := redisstore.Open(redisstore.Config{
		URL:           cfg.Redis.URL,
		DialTimeout:   cfg.Redis.DialTimeout,
		ReadTimeout:   cfg.Redis.ReadTimeout,
		WriteTimeout:  cfg.Redis.WriteTimeout,
		HealthTimeout: cfg.Redis.HealthTimeout,
	})
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := cache.Close(); closeErr != nil {
			logger.Error("close Redis client", "error", closeErr)
		}
	}()

	auditRepo := audit.NewRepository(database.RawPool())
	rateLimiter := ratelimit.New(cache.RawClient())

	bizRepo := businesses.NewRepository(database.RawPool())
	bizService := businesses.NewService(bizRepo, auditRepo)
	bizHandler := businesses.NewHandler(bizService)

	userRepo := identity.NewRepository(database.RawPool())
	identityService := identity.NewService(
		userRepo,
		bizRepo,
		auditRepo,
		cfg.Auth.JWTSecret,
		cfg.Auth.AccessTokenTTL,
		cfg.Auth.RefreshTokenTTL,
	)
	identityHandler := identity.NewHandler(
		identityService,
		bizService,
		rateLimiter,
		cfg.App.Environment == config.EnvironmentProduction,
	)

	server := httpserver.New(httpserver.Config{
		Addr:              cfg.HTTP.Addr,
		FrontendOrigins:   cfg.HTTP.FrontendOrigins,
		ReadHeaderTimeout: cfg.HTTP.ReadHeaderTimeout,
		ReadTimeout:       cfg.HTTP.ReadTimeout,
		WriteTimeout:      cfg.HTTP.WriteTimeout,
		IdleTimeout:       cfg.HTTP.IdleTimeout,
		RequestTimeout:    cfg.HTTP.RequestTimeout,
		ShutdownTimeout:   cfg.HTTP.ShutdownTimeout,
		MaxBodyBytes:      cfg.HTTP.MaxBodyBytes,
		Production:        cfg.App.Environment == config.EnvironmentProduction,
		JWTSecret:         cfg.Auth.JWTSecret,
	}, logger, httpserver.Dependencies{
		Postgres:   database,
		Redis:      cache,
		Identity:   identityHandler.Routes(cfg.Auth.JWTSecret),
		Businesses: bizHandler.Routes(cfg.Auth.JWTSecret),
		Me:         http.HandlerFunc(identityHandler.GetMe),
	}, observability.NoopHooks())

	logger.Info("api starting", "addr", cfg.HTTP.Addr)
	if err := server.Run(ctx); err != nil {
		return err
	}
	bootstrapLogger.Debug("api shutdown complete")
	return nil
}
