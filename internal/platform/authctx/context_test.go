package authctx

import (
	"context"
	"testing"
)

func TestAuthContext(t *testing.T) {
	ctx := context.Background()

	// Empty context
	_, ok := UserFromContext(ctx)
	if ok {
		t.Fatal("expected false for empty context")
	}

	// Injected user
	user := AuthUser{ID: "usr_123", Email: "alice@example.com"}
	ctxWithUser := WithAuth(ctx, user)
	extracted, ok := UserFromContext(ctxWithUser)
	if !ok || extracted.ID != user.ID || extracted.Email != user.Email {
		t.Fatalf("expected %+v, got %+v", user, extracted)
	}

	// Injected tenant
	tenant := TenantContext{BusinessID: "biz_123", Role: "owner"}
	ctxWithTenant := WithTenant(ctx, tenant)
	extractedTenant, ok := TenantFromContext(ctxWithTenant)
	if !ok || extractedTenant.BusinessID != tenant.BusinessID || extractedTenant.Role != tenant.Role {
		t.Fatalf("expected %+v, got %+v", tenant, extractedTenant)
	}
}
