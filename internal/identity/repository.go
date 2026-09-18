package identity

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) CreateUser(ctx context.Context, user *User) error {
	query := `
		INSERT INTO users (id, email, password_hash, full_name, mobile, status, verified_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	now := time.Now().UTC()
	user.CreatedAt = now
	user.UpdatedAt = now
	user.Email = strings.ToLower(strings.TrimSpace(user.Email))

	_, err := r.pool.Exec(ctx, query,
		user.ID,
		user.Email,
		user.PasswordHash,
		user.FullName,
		user.Mobile,
		user.Status,
		user.VerifiedAt,
		user.CreatedAt,
		user.UpdatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" { // unique_violation
			return ErrUserAlreadyExists
		}
		return fmt.Errorf("insert user: %w", err)
	}
	return nil
}

func (r *Repository) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	query := `
		SELECT id, email, password_hash, full_name, mobile, status, verified_at, created_at, updated_at
		FROM users
		WHERE LOWER(email) = LOWER($1)
	`
	var u User
	err := r.pool.QueryRow(ctx, query, strings.TrimSpace(email)).Scan(
		&u.ID,
		&u.Email,
		&u.PasswordHash,
		&u.FullName,
		&u.Mobile,
		&u.Status,
		&u.VerifiedAt,
		&u.CreatedAt,
		&u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("get user by email: %w", err)
	}
	u.EmailVerifiedAt = u.VerifiedAt
	return &u, nil
}

func (r *Repository) GetUserByID(ctx context.Context, id string) (*User, error) {
	query := `
		SELECT id, email, password_hash, full_name, mobile, status, verified_at, created_at, updated_at
		FROM users
		WHERE id = $1
	`
	var u User
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&u.ID,
		&u.Email,
		&u.PasswordHash,
		&u.FullName,
		&u.Mobile,
		&u.Status,
		&u.VerifiedAt,
		&u.CreatedAt,
		&u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("get user by id: %w", err)
	}
	u.EmailVerifiedAt = u.VerifiedAt
	return &u, nil
}

func (r *Repository) UpdatePassword(ctx context.Context, userID string, passwordHash string) error {
	query := `
		UPDATE users
		SET password_hash = $2, updated_at = NOW()
		WHERE id = $1
	`
	tag, err := r.pool.Exec(ctx, query, userID, passwordHash)
	if err != nil {
		return fmt.Errorf("update user password: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrUserNotFound
	}
	return nil
}

func (r *Repository) SetEmailVerified(ctx context.Context, userID string) error {
	query := `
		UPDATE users
		SET status = 'active', verified_at = NOW(), updated_at = NOW()
		WHERE id = $1
	`
	tag, err := r.pool.Exec(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("set email verified: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrUserNotFound
	}
	return nil
}

func (r *Repository) CreateRefreshSession(ctx context.Context, session *RefreshSession) error {
	query := `
		INSERT INTO refresh_sessions (id, user_id, token_hash, device_info, ip_address, expires_at, last_seen_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	now := time.Now().UTC()
	session.CreatedAt = now
	session.LastSeenAt = now

	_, err := r.pool.Exec(ctx, query,
		session.ID,
		session.UserID,
		session.TokenHash,
		session.DeviceInfo,
		session.IPAddress,
		session.ExpiresAt,
		session.LastSeenAt,
		session.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert refresh session: %w", err)
	}
	return nil
}

func (r *Repository) GetRefreshSession(ctx context.Context, tokenHash string) (*RefreshSession, error) {
	query := `
		SELECT id, user_id, token_hash, device_info, ip_address, expires_at, revoked_at, last_seen_at, created_at
		FROM refresh_sessions
		WHERE token_hash = $1 AND revoked_at IS NULL AND expires_at > NOW()
	`
	var s RefreshSession
	err := r.pool.QueryRow(ctx, query, tokenHash).Scan(
		&s.ID,
		&s.UserID,
		&s.TokenHash,
		&s.DeviceInfo,
		&s.IPAddress,
		&s.ExpiresAt,
		&s.RevokedAt,
		&s.LastSeenAt,
		&s.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrSessionNotFound
		}
		return nil, fmt.Errorf("get refresh session: %w", err)
	}
	return &s, nil
}

func (r *Repository) RevokeRefreshSession(ctx context.Context, tokenHash string) error {
	query := `
		UPDATE refresh_sessions
		SET revoked_at = NOW()
		WHERE token_hash = $1 AND revoked_at IS NULL
	`
	_, err := r.pool.Exec(ctx, query, tokenHash)
	if err != nil {
		return fmt.Errorf("revoke refresh session: %w", err)
	}
	return nil
}

func (r *Repository) RevokeAllUserSessions(ctx context.Context, userID string) error {
	query := `
		UPDATE refresh_sessions
		SET revoked_at = NOW()
		WHERE user_id = $1 AND revoked_at IS NULL
	`
	_, err := r.pool.Exec(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("revoke all user sessions: %w", err)
	}
	return nil
}

func (r *Repository) CreateAuthToken(ctx context.Context, token *AuthToken) error {
	query := `
		INSERT INTO auth_tokens (id, user_id, token_hash, token_type, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	token.CreatedAt = time.Now().UTC()
	_, err := r.pool.Exec(ctx, query,
		token.ID,
		token.UserID,
		token.TokenHash,
		token.TokenType,
		token.ExpiresAt,
		token.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert auth token: %w", err)
	}
	return nil
}

func (r *Repository) ConsumeAuthToken(ctx context.Context, tokenHash, tokenType string) (*AuthToken, error) {
	query := `
		UPDATE auth_tokens
		SET consumed_at = NOW()
		WHERE token_hash = $1 AND token_type = $2 AND consumed_at IS NULL AND expires_at > NOW()
		RETURNING id, user_id, token_hash, token_type, expires_at, consumed_at, created_at
	`
	var t AuthToken
	err := r.pool.QueryRow(ctx, query, tokenHash, tokenType).Scan(
		&t.ID,
		&t.UserID,
		&t.TokenHash,
		&t.TokenType,
		&t.ExpiresAt,
		&t.ConsumedAt,
		&t.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrTokenInvalid
		}
		return nil, fmt.Errorf("consume auth token: %w", err)
	}
	return &t, nil
}
