package authctx

import (
	"context"
	"errors"
)

type contextKey string

const (
	userContextKey   contextKey = "aibos_auth_user"
	tenantContextKey contextKey = "aibos_auth_tenant"
)

var (
	ErrUnauthenticated = errors.New("unauthenticated")
	ErrNoTenantContext = errors.New("no active tenant context")
	ErrUnauthorized    = errors.New("unauthorized: insufficient role permissions")
)

type AuthUser struct {
	ID    string
	Email string
}

type TenantContext struct {
	BusinessID string
	Role       string // "owner", "staff", "admin"
}

// WithAuth stores the authenticated user in the context.
func WithAuth(ctx context.Context, user AuthUser) context.Context {
	return context.WithValue(ctx, userContextKey, user)
}

// UserFromContext retrieves the authenticated user from the context.
func UserFromContext(ctx context.Context) (AuthUser, bool) {
	u, ok := ctx.Value(userContextKey).(AuthUser)
	return u, ok
}

// WithTenant stores the active business/tenant context and membership role.
func WithTenant(ctx context.Context, tenant TenantContext) context.Context {
	return context.WithValue(ctx, tenantContextKey, tenant)
}

// TenantFromContext retrieves the active tenant context.
func TenantFromContext(ctx context.Context) (TenantContext, bool) {
	t, ok := ctx.Value(tenantContextKey).(TenantContext)
	return t, ok
}
