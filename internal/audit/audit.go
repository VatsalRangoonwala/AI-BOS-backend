package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Entry struct {
	BusinessID    *string
	UserID        *string
	Action        string
	AggregateType string
	AggregateID   *string
	Metadata      map[string]any
	RequestID     string
	IPAddress     string
	UserAgent     string
}

type Logger interface {
	Log(ctx context.Context, entry Entry) error
}

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) Log(ctx context.Context, entry Entry) error {
	metaJSON, err := json.Marshal(entry.Metadata)
	if err != nil {
		metaJSON = []byte("{}")
	}

	query := `
		INSERT INTO audit_logs (
			business_id, user_id, action, aggregate_type, aggregate_id,
			metadata, request_id, ip_address, user_agent, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`
	now := time.Now().UTC()
	_, err = r.pool.Exec(ctx, query,
		entry.BusinessID,
		entry.UserID,
		entry.Action,
		entry.AggregateType,
		entry.AggregateID,
		metaJSON,
		entry.RequestID,
		entry.IPAddress,
		entry.UserAgent,
		now,
	)
	if err != nil {
		return fmt.Errorf("insert audit log: %w", err)
	}
	return nil
}
