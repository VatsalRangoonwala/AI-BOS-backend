package customers

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) Create(ctx context.Context, customer *Customer) error {
	now := time.Now().UTC()
	customer.CreatedAt = now
	customer.UpdatedAt = now
	if customer.ID == "" {
		customer.ID = uuid.NewString()
	}

	query := `
		INSERT INTO customers (
			id, business_id, name, phone, email, address, notes, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	_, err := r.pool.Exec(ctx, query,
		customer.ID,
		customer.BusinessID,
		customer.Name,
		customer.Phone,
		customer.Email,
		customer.Address,
		customer.Notes,
		customer.CreatedAt,
		customer.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert customer: %w", err)
	}

	return nil
}

func (r *Repository) GetByID(ctx context.Context, businessID, customerID string) (*Customer, error) {
	query := `
		SELECT id, business_id, name, phone, email, address, notes, archived_at, created_at, updated_at
		FROM customers
		WHERE id = $1 AND business_id = $2 AND archived_at IS NULL
	`
	var c Customer
	err := r.pool.QueryRow(ctx, query, customerID, businessID).Scan(
		&c.ID,
		&c.BusinessID,
		&c.Name,
		&c.Phone,
		&c.Email,
		&c.Address,
		&c.Notes,
		&c.ArchivedAt,
		&c.CreatedAt,
		&c.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrCustomerNotFound
		}
		return nil, fmt.Errorf("get customer by id: %w", err)
	}

	return &c, nil
}

func (r *Repository) Update(ctx context.Context, businessID, customerID string, input UpdateCustomerInput) (*Customer, error) {
	existing, err := r.GetByID(ctx, businessID, customerID)
	if err != nil {
		return nil, err
	}

	if input.Name == nil && input.FullName != nil {
		input.Name = input.FullName
	}
	if input.Phone == nil && input.Mobile != nil {
		input.Phone = input.Mobile
	}

	if input.Name != nil {
		trimmed := strings.TrimSpace(*input.Name)
		if trimmed == "" {
			return nil, ErrInvalidCustomerName
		}
		existing.Name = trimmed
	}
	if input.Phone != nil {
		existing.Phone = input.Phone
	}
	if input.Email != nil {
		existing.Email = input.Email
	}
	if input.Address != nil {
		existing.Address = input.Address
	}
	if input.Notes != nil {
		existing.Notes = input.Notes
	}
	existing.UpdatedAt = time.Now().UTC()

	query := `
		UPDATE customers
		SET name = $1, phone = $2, email = $3, address = $4, notes = $5, updated_at = $6
		WHERE id = $7 AND business_id = $8 AND archived_at IS NULL
	`
	tag, err := r.pool.Exec(ctx, query,
		existing.Name,
		existing.Phone,
		existing.Email,
		existing.Address,
		existing.Notes,
		existing.UpdatedAt,
		customerID,
		businessID,
	)
	if err != nil {
		return nil, fmt.Errorf("update customer: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return nil, ErrCustomerNotFound
	}

	return existing, nil
}

func (r *Repository) Archive(ctx context.Context, businessID, customerID string) error {
	now := time.Now().UTC()
	query := `
		UPDATE customers
		SET archived_at = $1, updated_at = $1
		WHERE id = $2 AND business_id = $3 AND archived_at IS NULL
	`
	tag, err := r.pool.Exec(ctx, query, now, customerID, businessID)
	if err != nil {
		return fmt.Errorf("archive customer: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrCustomerNotFound
	}

	return nil
}

func (r *Repository) List(ctx context.Context, businessID string, q ListCustomersQuery) ([]Customer, string, error) {
	limit := q.Limit
	if limit <= 0 || limit > 100 {
		limit = 25
	}

	var cursorCreatedAt time.Time
	var cursorID string
	if q.Cursor != "" {
		decoded, err := base64.StdEncoding.DecodeString(q.Cursor)
		if err == nil {
			parts := strings.Split(string(decoded), "#")
			if len(parts) == 2 {
				if parsedTime, err := time.Parse(time.RFC3339Nano, parts[0]); err == nil {
					cursorCreatedAt = parsedTime
					cursorID = parts[1]
				}
			}
		}
	}

	args := []any{businessID}
	queryBuilder := strings.Builder{}
	queryBuilder.WriteString(`
		SELECT id, business_id, name, phone, email, address, notes, archived_at, created_at, updated_at
		FROM customers
		WHERE business_id = $1 AND archived_at IS NULL
	`)

	if strings.TrimSpace(q.Query) != "" {
		searchTerm := "%" + strings.TrimSpace(q.Query) + "%"
		args = append(args, searchTerm)
		paramIndex := len(args)
		queryBuilder.WriteString(fmt.Sprintf(" AND (name ILIKE $%d OR phone ILIKE $%d OR email ILIKE $%d)", paramIndex, paramIndex, paramIndex))
	}

	if !cursorCreatedAt.IsZero() && cursorID != "" {
		args = append(args, cursorCreatedAt, cursorID)
		paramIndex := len(args)
		queryBuilder.WriteString(fmt.Sprintf(" AND (created_at, id) < ($%d, $%d)", paramIndex-1, paramIndex))
	}

	args = append(args, limit+1)
	queryBuilder.WriteString(fmt.Sprintf(" ORDER BY created_at DESC, id DESC LIMIT $%d", len(args)))

	rows, err := r.pool.Query(ctx, queryBuilder.String(), args...)
	if err != nil {
		return nil, "", fmt.Errorf("list customers: %w", err)
	}
	defer rows.Close()

	items := make([]Customer, 0, limit)
	for rows.Next() {
		var c Customer
		if err := rows.Scan(
			&c.ID,
			&c.BusinessID,
			&c.Name,
			&c.Phone,
			&c.Email,
			&c.Address,
			&c.Notes,
			&c.ArchivedAt,
			&c.CreatedAt,
			&c.UpdatedAt,
		); err != nil {
			return nil, "", fmt.Errorf("scan customer: %w", err)
		}
		items = append(items, c)
	}

	if err := rows.Err(); err != nil {
		return nil, "", fmt.Errorf("iterate customers: %w", err)
	}

	nextCursor := ""
	if len(items) > limit {
		lastItem := items[limit-1]
		items = items[:limit]
		cursorRaw := fmt.Sprintf("%s#%s", lastItem.CreatedAt.Format(time.RFC3339Nano), lastItem.ID)
		nextCursor = base64.StdEncoding.EncodeToString([]byte(cursorRaw))
	}

	return items, nextCursor, nil
}
