package redis

import (
	"context"
	"errors"
	"time"

	redisclient "github.com/redis/go-redis/v9"
)

type Config struct {
	URL           string
	DialTimeout   time.Duration
	ReadTimeout   time.Duration
	WriteTimeout  time.Duration
	HealthTimeout time.Duration
}

type Backend interface {
	Ping(context.Context) error
	Close() error
}

type Client struct {
	backend       Backend
	healthTimeout time.Duration
}

type backend struct {
	client *redisclient.Client
}

func Open(cfg Config) (*Client, error) {
	options, err := redisclient.ParseURL(cfg.URL)
	if err != nil {
		return nil, errors.New("parse REDIS_URL")
	}
	options.DialTimeout = cfg.DialTimeout
	options.ReadTimeout = cfg.ReadTimeout
	options.WriteTimeout = cfg.WriteTimeout

	return New(&backend{client: redisclient.NewClient(options)}, cfg.HealthTimeout), nil
}

func New(backend Backend, healthTimeout time.Duration) *Client {
	return &Client{backend: backend, healthTimeout: healthTimeout}
}

func (client *Client) Check(ctx context.Context) error {
	healthCtx, cancel := context.WithTimeout(ctx, client.healthTimeout)
	defer cancel()
	return client.backend.Ping(healthCtx)
}

func (client *Client) Close() error {
	return client.backend.Close()
}

func (backend *backend) Ping(ctx context.Context) error {
	return backend.client.Ping(ctx).Err()
}

func (backend *backend) Close() error {
	return backend.client.Close()
}
