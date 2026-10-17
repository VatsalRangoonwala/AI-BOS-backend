package catalog_test

import (
	"context"
	"testing"

	"github.com/VatsalRangoonwala/AI-BOS-backend/internal/audit"
	"github.com/VatsalRangoonwala/AI-BOS-backend/internal/catalog"
)

type mockAuditLogger struct{}

func (m *mockAuditLogger) Log(ctx context.Context, entry audit.Entry) error {
	return nil
}

func TestCatalogValidation(t *testing.T) {
	svc := catalog.NewService(nil, &mockAuditLogger{})

	t.Run("Reject empty product name or SKU", func(t *testing.T) {
		_, err := svc.CreateProduct(context.Background(), "biz-1", "user-1", catalog.CreateProductInput{
			Name: "   ",
			SKU:  "SKU-1",
		}, catalog.RequestMeta{})
		if err != catalog.ErrInvalidProductData {
			t.Fatalf("expected ErrInvalidProductData, got %v", err)
		}

		_, err = svc.CreateProduct(context.Background(), "biz-1", "user-1", catalog.CreateProductInput{
			Name: "Widget",
			SKU:  "   ",
		}, catalog.RequestMeta{})
		if err != catalog.ErrInvalidProductData {
			t.Fatalf("expected ErrInvalidProductData, got %v", err)
		}
	})

	t.Run("Reject negative prices or negative stock", func(t *testing.T) {
		negPrice := int64(-500)
		_, err := svc.CreateProduct(context.Background(), "biz-1", "user-1", catalog.CreateProductInput{
			Name:           "Widget",
			SKU:            "SKU-1",
			SalePriceMinor: negPrice,
		}, catalog.RequestMeta{})
		if err == nil {
			t.Fatal("expected error for negative sale price")
		}

		negStock := -10
		_, err = svc.CreateProduct(context.Background(), "biz-1", "user-1", catalog.CreateProductInput{
			Name:         "Widget",
			SKU:          "SKU-1",
			InitialStock: &negStock,
		}, catalog.RequestMeta{})
		if err == nil {
			t.Fatal("expected error for negative initial stock")
		}
	})

	t.Run("Reject zero quantity stock adjustment", func(t *testing.T) {
		_, err := svc.RecordStockAdjustment(context.Background(), "biz-1", "user-1", catalog.StockAdjustmentInput{
			ProductID: "prod-1",
			Quantity:  0,
		}, catalog.RequestMeta{})
		if err == nil {
			t.Fatal("expected error for zero adjustment quantity")
		}
	})
}
