package customers_test

import (
	"context"
	"testing"

	"github.com/VatsalRangoonwala/AI-BOS-backend/internal/audit"
	"github.com/VatsalRangoonwala/AI-BOS-backend/internal/customers"
)

type mockAuditLogger struct{}

func (m *mockAuditLogger) Log(ctx context.Context, entry audit.Entry) error {
	return nil
}

func TestCustomerValidation(t *testing.T) {
	t.Run("Reject empty customer name", func(t *testing.T) {
		svc := customers.NewService(nil, &mockAuditLogger{})
		_, err := svc.CreateCustomer(context.Background(), "biz-1", "user-1", customers.CreateCustomerInput{
			Name: "   ",
		}, customers.RequestMeta{})
		if err != customers.ErrInvalidCustomerName {
			t.Fatalf("expected ErrInvalidCustomerName, got %v", err)
		}
	})
}
