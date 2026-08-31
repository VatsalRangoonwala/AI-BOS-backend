package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/VatsalRangoonwala/AI-BOS-backend/internal/platform/config"
	"github.com/VatsalRangoonwala/AI-BOS-backend/internal/platform/logging"
	"github.com/VatsalRangoonwala/AI-BOS-backend/internal/platform/postgres"
	redisstore "github.com/VatsalRangoonwala/AI-BOS-backend/internal/platform/redis"
)

func main() {
	logger := logging.New(os.Stdout, "info")
	if err := run(logger); err != nil {
		logger.Error("worker stopped", "error", err)
		os.Exit(1)
	}
}

func run(bootstrapLogger *slog.Logger) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	logger := logging.New(os.Stdout, cfg.App.LogLevel).With("service", "worker", "environment", cfg.App.Environment)

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

	if err := database.Check(ctx); err != nil {
		return errors.New("postgres startup health check failed")
	}
	if err := cache.Check(ctx); err != nil {
		return errors.New("redis startup health check failed")
	}

	logger.Info("worker started")
	healthTicker := time.NewTicker(cfg.Worker.HealthInterval)
	defer healthTicker.Stop()
	for {
		select {
		case <-ctx.Done():
			bootstrapLogger.Debug("worker shutdown complete")
			return nil
		case <-healthTicker.C:
			if err := database.Check(ctx); err != nil {
				logger.Warn("dependency health check failed", "dependency", "postgres")
			}
			if err := cache.Check(ctx); err != nil {
				logger.Warn("dependency health check failed", "dependency", "redis")
			}
		}
	}
}
