package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Config struct {
	URL             string
	PoolMax         int32
	PoolMin         int32
	ConnectTimeout  time.Duration
	HealthTimeout   time.Duration
	MaxConnLifetime time.Duration
	MaxConnIdleTime time.Duration
}

type Pool interface {
	Ping(context.Context) error
	Close()
}

type Client struct {
	pool          Pool
	healthTimeout time.Duration
}

func Open(ctx context.Context, cfg Config) (*Client, error) {
	poolConfig, err := pgxpool.ParseConfig(cfg.URL)
	if err != nil {
		return nil, errors.New("parse DATABASE_URL")
	}
	poolConfig.MaxConns = cfg.PoolMax
	poolConfig.MinConns = cfg.PoolMin
	poolConfig.MaxConnLifetime = cfg.MaxConnLifetime
	poolConfig.MaxConnIdleTime = cfg.MaxConnIdleTime
	poolConfig.ConnConfig.ConnectTimeout = cfg.ConnectTimeout

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, errors.New("initialize PostgreSQL pool")
	}
	return New(pool, cfg.HealthTimeout), nil
}

func New(pool Pool, healthTimeout time.Duration) *Client {
	return &Client{pool: pool, healthTimeout: healthTimeout}
}

func (client *Client) Check(ctx context.Context) error {
	healthCtx, cancel := context.WithTimeout(ctx, client.healthTimeout)
	defer cancel()
	return client.pool.Ping(healthCtx)
}

func (client *Client) Close() {
	client.pool.Close()
}
