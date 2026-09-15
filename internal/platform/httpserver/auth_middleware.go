package httpserver

import (
	"net/http"
	"strings"

	"github.com/VatsalRangoonwala/AI-BOS-backend/internal/platform/authctx"
	"github.com/VatsalRangoonwala/AI-BOS-backend/internal/platform/security"
)

// Authenticate extracts and verifies JWT access tokens from Authorization header or cookie.
func Authenticate(jwtSecret []byte) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenString := extractToken(r)
			if tokenString != "" {
				claims, err := security.ValidateAccessToken(tokenString, jwtSecret)
				if err == nil && claims != nil {
					ctx := authctx.WithAuth(r.Context(), authctx.AuthUser{
						ID:    claims.UserID,
						Email: claims.Email,
					})
					r = r.WithContext(ctx)
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RequireAuth enforces that the request has an authenticated user.
func RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, ok := authctx.UserFromContext(r.Context())
		if !ok {
			writeError(w, r, http.StatusUnauthorized, "unauthorized", "Authentication is required", nil)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// RequireTenantRole enforces that the active membership has one of the allowed roles.
func RequireTenantRole(allowedRoles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tenant, ok := authctx.TenantFromContext(r.Context())
			if !ok {
				writeError(w, r, http.StatusForbidden, "forbidden", "No tenant context", nil)
				return
			}

			allowed := false
			for _, role := range allowedRoles {
				if tenant.Role == role {
					allowed = true
					break
				}
			}

			if !allowed {
				writeError(w, r, http.StatusForbidden, "forbidden", "Insufficient role permissions for this operation", nil)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func extractToken(r *http.Request) string {
	authHeader := r.Header.Get("Authorization")
	if strings.HasPrefix(authHeader, "Bearer ") {
		return strings.TrimPrefix(authHeader, "Bearer ")
	}

	if cookie, err := r.Cookie("aibos_access_token"); err == nil {
		return cookie.Value
	}

	return ""
}
