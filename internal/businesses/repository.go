package businesses

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
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

// CreateBusinessWithMember atomically creates a business tenant and assigns the caller as owner.
func (r *Repository) CreateBusinessWithMember(ctx context.Context, biz *Business, userID string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	now := time.Now().UTC()
	biz.CreatedAt = now
	biz.UpdatedAt = now
	if biz.ID == "" {
		biz.ID = uuid.NewString()
	}
	if biz.Status == "" {
		biz.Status = "active"
	}
	if biz.Timezone == "" {
		biz.Timezone = "Asia/Kolkata"
	}
	if biz.Currency == "" {
		biz.Currency = "INR"
	}
	if biz.InvoicePrefix == "" {
		biz.InvoicePrefix = "INV-"
	}
	if biz.InvoiceSequence == 0 {
		biz.InvoiceSequence = 1001
	}

	insertBizQuery := `
		INSERT INTO businesses (
			id, name, legal_name, phone, address, timezone,
			currency, invoice_prefix, invoice_sequence, status, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`
	_, err = tx.Exec(ctx, insertBizQuery,
		biz.ID,
		biz.Name,
		biz.LegalName,
		biz.Phone,
		biz.Address,
		biz.Timezone,
		biz.Currency,
		biz.InvoicePrefix,
		biz.InvoiceSequence,
		biz.Status,
		biz.CreatedAt,
		biz.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert business: %w", err)
	}

	insertMemQuery := `
		INSERT INTO memberships (
			id, business_id, user_id, role, status, accepted_at, created_at, updated_at
		) VALUES ($1, $2, $3, 'owner', 'active', $4, $4, $4)
	`
	_, err = tx.Exec(ctx, insertMemQuery, uuid.NewString(), biz.ID, userID, now)
	if err != nil {
		return fmt.Errorf("insert owner membership: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit business transaction: %w", err)
	}
	return nil
}

// GetBusinessByID scopes the fetch to verify caller's active membership in the business.
func (r *Repository) GetBusinessByID(ctx context.Context, businessID string, callerUserID string) (*Business, string, error) {
	query := `
		SELECT b.id, b.name, b.legal_name, b.phone, b.address, b.timezone,
		       b.currency, b.invoice_prefix, b.invoice_sequence, b.status, b.created_at, b.updated_at,
		       m.role
		FROM businesses b
		INNER JOIN memberships m ON m.business_id = b.id
		WHERE b.id = $1 AND m.user_id = $2 AND m.status = 'active'
	`
	var b Business
	var role string
	err := r.pool.QueryRow(ctx, query, businessID, callerUserID).Scan(
		&b.ID,
		&b.Name,
		&b.LegalName,
		&b.Phone,
		&b.Address,
		&b.Timezone,
		&b.Currency,
		&b.InvoicePrefix,
		&b.InvoiceSequence,
		&b.Status,
		&b.CreatedAt,
		&b.UpdatedAt,
		&role,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, "", ErrBusinessNotFound
		}
		return nil, "", fmt.Errorf("get business by id: %w", err)
	}
	return &b, role, nil
}

type UpdateBusinessInput struct {
	Name          *string
	LegalName     *string
	Phone         *string
	Address       *string
	Timezone      *string
	InvoicePrefix *string
}

// UpdateBusiness updates business tenant metadata if caller has owner role.
func (r *Repository) UpdateBusiness(ctx context.Context, businessID string, callerUserID string, input UpdateBusinessInput) (*Business, error) {
	// First check membership & role
	biz, role, err := r.GetBusinessByID(ctx, businessID, callerUserID)
	if err != nil {
		return nil, err
	}
	if role != "owner" {
		return nil, ErrUnauthorizedAction
	}

	if input.Name != nil {
		biz.Name = *input.Name
	}
	if input.LegalName != nil {
		biz.LegalName = input.LegalName
	}
	if input.Phone != nil {
		biz.Phone = input.Phone
	}
	if input.Address != nil {
		biz.Address = input.Address
	}
	if input.Timezone != nil {
		biz.Timezone = *input.Timezone
	}
	if input.InvoicePrefix != nil {
		biz.InvoicePrefix = *input.InvoicePrefix
	}
	biz.UpdatedAt = time.Now().UTC()

	query := `
		UPDATE businesses
		SET name = $2, legal_name = $3, phone = $4, address = $5,
		    timezone = $6, invoice_prefix = $7, updated_at = $8
		WHERE id = $1
	`
	_, err = r.pool.Exec(ctx, query,
		biz.ID,
		biz.Name,
		biz.LegalName,
		biz.Phone,
		biz.Address,
		biz.Timezone,
		biz.InvoicePrefix,
		biz.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("update business: %w", err)
	}

	return biz, nil
}

// ListBusinessesForUser returns all businesses where the user has active membership.
func (r *Repository) ListBusinessesForUser(ctx context.Context, userID string) ([]UserBusinessSummary, error) {
	query := `
		SELECT b.id, b.name, m.role, m.status
		FROM businesses b
		INNER JOIN memberships m ON m.business_id = b.id
		WHERE m.user_id = $1 AND m.status = 'active'
		ORDER BY b.created_at ASC
	`
	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("list user businesses: %w", err)
	}
	defer rows.Close()

	var summaries []UserBusinessSummary
	for rows.Next() {
		var s UserBusinessSummary
		if err := rows.Scan(&s.BusinessID, &s.BusinessName, &s.Role, &s.Status); err != nil {
			return nil, fmt.Errorf("scan user business: %w", err)
		}
		summaries = append(summaries, s)
	}
	return summaries, nil
}

// ListMembers returns all members for a business if caller is an active member.
func (r *Repository) ListMembers(ctx context.Context, businessID string, callerUserID string) ([]MemberDetail, error) {
	// Verify caller belongs to business
	_, _, err := r.GetBusinessByID(ctx, businessID, callerUserID)
	if err != nil {
		return nil, err
	}

	query := `
		SELECT m.id, m.business_id, m.user_id, m.role, m.status, m.created_at, m.updated_at,
		       u.id, u.email, u.full_name, u.status
		FROM memberships m
		INNER JOIN users u ON u.id = m.user_id
		WHERE m.business_id = $1
		ORDER BY m.created_at ASC
	`
	rows, err := r.pool.Query(ctx, query, businessID)
	if err != nil {
		return nil, fmt.Errorf("list members: %w", err)
	}
	defer rows.Close()

	var members []MemberDetail
	for rows.Next() {
		var m MemberDetail
		err := rows.Scan(
			&m.ID,
			&m.BusinessID,
			&m.UserID,
			&m.Role,
			&m.Status,
			&m.CreatedAt,
			&m.UpdatedAt,
			&m.User.ID,
			&m.User.Email,
			&m.User.FullName,
			&m.User.Status,
		)
		if err != nil {
			return nil, fmt.Errorf("scan member: %w", err)
		}
		members = append(members, m)
	}
	return members, nil
}

// InviteMember invites a user by email to the business. Caller must be owner.
func (r *Repository) InviteMember(ctx context.Context, businessID string, callerUserID string, inviteeEmail string, role string) (*MemberDetail, error) {
	_, callerRole, err := r.GetBusinessByID(ctx, businessID, callerUserID)
	if err != nil {
		return nil, err
	}
	if callerRole != "owner" {
		return nil, ErrUnauthorizedAction
	}

	// Find or identify user by email
	var targetUser UserSummary
	findUserQuery := `SELECT id, email, full_name, status FROM users WHERE LOWER(email) = LOWER($1)`
	err = r.pool.QueryRow(ctx, findUserQuery, inviteeEmail).Scan(&targetUser.ID, &targetUser.Email, &targetUser.FullName, &targetUser.Status)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("user with this email does not exist yet; user must register before invitation")
		}
		return nil, fmt.Errorf("find user for invite: %w", err)
	}

	now := time.Now().UTC()
	memberID := uuid.NewString()
	insertQuery := `
		INSERT INTO memberships (id, business_id, user_id, role, status, invited_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, 'invited', $5, $5, $5)
	`
	_, err = r.pool.Exec(ctx, insertQuery, memberID, businessID, targetUser.ID, role, now)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, ErrMemberAlreadyExists
		}
		return nil, fmt.Errorf("insert invitation: %w", err)
	}

	return &MemberDetail{
		ID:         memberID,
		BusinessID: businessID,
		UserID:     targetUser.ID,
		Role:       role,
		Status:     "invited",
		User:       targetUser,
		CreatedAt:  now,
		UpdatedAt:  now,
	}, nil
}

// UpdateMember updates member role or status. Caller must be owner.
func (r *Repository) UpdateMember(ctx context.Context, businessID string, callerUserID string, memberID string, newRole, newStatus *string) (*MemberDetail, error) {
	_, callerRole, err := r.GetBusinessByID(ctx, businessID, callerUserID)
	if err != nil {
		return nil, err
	}
	if callerRole != "owner" {
		return nil, ErrUnauthorizedAction
	}

	// Fetch current target member
	var current MemberDetail
	fetchQuery := `
		SELECT m.id, m.business_id, m.user_id, m.role, m.status, m.created_at, m.updated_at,
		       u.id, u.email, u.full_name, u.status
		FROM memberships m
		INNER JOIN users u ON u.id = m.user_id
		WHERE m.id = $1 AND m.business_id = $2
	`
	err = r.pool.QueryRow(ctx, fetchQuery, memberID, businessID).Scan(
		&current.ID, &current.BusinessID, &current.UserID, &current.Role, &current.Status, &current.CreatedAt, &current.UpdatedAt,
		&current.User.ID, &current.User.Email, &current.User.FullName, &current.User.Status,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrMemberNotFound
		}
		return nil, fmt.Errorf("fetch member: %w", err)
	}

	if current.Role == "owner" && newRole != nil && *newRole != "owner" {
		return nil, errors.New("cannot change owner role; ownership transfer required")
	}

	roleToSet := current.Role
	if newRole != nil {
		roleToSet = *newRole
	}
	statusToSet := current.Status
	if newStatus != nil {
		statusToSet = *newStatus
	}

	now := time.Now().UTC()
	updateQuery := `
		UPDATE memberships
		SET role = $3, status = $4, updated_at = $5
		WHERE id = $1 AND business_id = $2
	`
	_, err = r.pool.Exec(ctx, updateQuery, memberID, businessID, roleToSet, statusToSet, now)
	if err != nil {
		return nil, fmt.Errorf("update member: %w", err)
	}

	current.Role = roleToSet
	current.Status = statusToSet
	current.UpdatedAt = now
	return &current, nil
}

// RemoveMember removes a membership from the business. Caller must be owner.
func (r *Repository) RemoveMember(ctx context.Context, businessID string, callerUserID string, memberID string) error {
	_, callerRole, err := r.GetBusinessByID(ctx, businessID, callerUserID)
	if err != nil {
		return err
	}
	if callerRole != "owner" {
		return ErrUnauthorizedAction
	}

	var targetRole string
	checkQuery := `SELECT role FROM memberships WHERE id = $1 AND business_id = $2`
	err = r.pool.QueryRow(ctx, checkQuery, memberID, businessID).Scan(&targetRole)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrMemberNotFound
		}
		return fmt.Errorf("find member to remove: %w", err)
	}
	if targetRole == "owner" {
		return ErrCannotRemoveOwner
	}

	deleteQuery := `DELETE FROM memberships WHERE id = $1 AND business_id = $2`
	_, err = r.pool.Exec(ctx, deleteQuery, memberID, businessID)
	if err != nil {
		return fmt.Errorf("delete membership: %w", err)
	}
	return nil
}
