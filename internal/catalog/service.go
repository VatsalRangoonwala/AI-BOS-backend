package catalog

import (
	"context"
	"errors"
	"strings"

	"github.com/VatsalRangoonwala/AI-BOS-backend/internal/audit"
)

type RequestMeta struct {
	IPAddress string
	UserAgent string
	RequestID string
}

type Service struct {
	repo        *Repository
	auditLogger audit.Logger
}

func NewService(repo *Repository, auditLogger audit.Logger) *Service {
	return &Service{
		repo:        repo,
		auditLogger: auditLogger,
	}
}

func (s *Service) CreateProduct(ctx context.Context, businessID, userID string, input CreateProductInput, meta RequestMeta) (*Product, error) {
	name := strings.TrimSpace(input.Name)
	sku := strings.TrimSpace(input.SKU)
	if name == "" || sku == "" {
		return nil, ErrInvalidProductData
	}
	if input.SalePriceMinor < 0 || input.CostPriceMinor < 0 {
		return nil, errors.New("prices cannot be negative")
	}
	if input.TaxRateBasisPoints < 0 {
		return nil, errors.New("tax rate cannot be negative")
	}
	if input.ReorderLevel < 0 {
		return nil, errors.New("reorder level cannot be negative")
	}

	initialStock := 0
	if input.InitialStock != nil {
		if *input.InitialStock < 0 {
			return nil, errors.New("initial stock cannot be negative")
		}
		initialStock = *input.InitialStock
	}

	unit := strings.TrimSpace(input.Unit)
	if unit == "" {
		unit = "unit"
	}
	currency := strings.TrimSpace(input.Currency)
	if currency == "" {
		currency = "INR"
	}

	category := strings.TrimSpace(input.Category)
	if category == "" {
		category = "General"
	}
	var barcodePtr *string
	if input.Barcode != nil {
		trimmed := strings.TrimSpace(*input.Barcode)
		if trimmed != "" {
			barcodePtr = &trimmed
		}
	}

	product := &Product{
		BusinessID:         businessID,
		Name:               name,
		SKU:                sku,
		Barcode:            barcodePtr,
		Category:           category,
		Description:        input.Description,
		Unit:               unit,
		SalePriceMinor:     input.SalePriceMinor,
		CostPriceMinor:     input.CostPriceMinor,
		Currency:           currency,
		TaxRateBasisPoints: input.TaxRateBasisPoints,
		ReorderLevel:       input.ReorderLevel,
	}

	if err := s.repo.CreateProduct(ctx, product, initialStock, userID); err != nil {
		return nil, err
	}

	_ = s.auditLogger.Log(ctx, audit.Entry{
		BusinessID:    &businessID,
		UserID:        &userID,
		Action:        "product.created",
		AggregateType: "product",
		AggregateID:   &product.ID,
		Metadata: map[string]any{
			"name":         product.Name,
			"sku":          product.SKU,
			"initialStock": initialStock,
		},
		RequestID: meta.RequestID,
		IPAddress: meta.IPAddress,
		UserAgent: meta.UserAgent,
	})

	return product, nil
}

func (s *Service) GetProduct(ctx context.Context, businessID, productID string) (*Product, error) {
	return s.repo.GetProductByID(ctx, businessID, productID)
}

func (s *Service) UpdateProduct(ctx context.Context, businessID, productID, userID string, input UpdateProductInput, meta RequestMeta) (*Product, error) {
	if input.SalePriceMinor != nil && *input.SalePriceMinor < 0 {
		return nil, errors.New("sale price cannot be negative")
	}
	if input.CostPriceMinor != nil && *input.CostPriceMinor < 0 {
		return nil, errors.New("cost price cannot be negative")
	}
	if input.TaxRateBasisPoints != nil && *input.TaxRateBasisPoints < 0 {
		return nil, errors.New("tax rate cannot be negative")
	}
	if input.ReorderLevel != nil && *input.ReorderLevel < 0 {
		return nil, errors.New("reorder level cannot be negative")
	}

	product, err := s.repo.UpdateProduct(ctx, businessID, productID, input)
	if err != nil {
		return nil, err
	}

	_ = s.auditLogger.Log(ctx, audit.Entry{
		BusinessID:    &businessID,
		UserID:        &userID,
		Action:        "product.updated",
		AggregateType: "product",
		AggregateID:   &productID,
		Metadata: map[string]any{
			"updatedFields": input,
		},
		RequestID: meta.RequestID,
		IPAddress: meta.IPAddress,
		UserAgent: meta.UserAgent,
	})

	return product, nil
}

func (s *Service) ArchiveProduct(ctx context.Context, businessID, productID, userID string, meta RequestMeta) error {
	if err := s.repo.ArchiveProduct(ctx, businessID, productID); err != nil {
		return err
	}

	_ = s.auditLogger.Log(ctx, audit.Entry{
		BusinessID:    &businessID,
		UserID:        &userID,
		Action:        "product.archived",
		AggregateType: "product",
		AggregateID:   &productID,
		Metadata:      map[string]any{},
		RequestID:     meta.RequestID,
		IPAddress:     meta.IPAddress,
		UserAgent:     meta.UserAgent,
	})

	return nil
}

func (s *Service) ListProducts(ctx context.Context, businessID string, query ListProductsQuery) ([]Product, string, error) {
	return s.repo.ListProducts(ctx, businessID, query)
}

func (s *Service) ListInventory(ctx context.Context, businessID string, query ListInventoryQuery) ([]ProductInventoryItem, string, error) {
	return s.repo.ListInventory(ctx, businessID, query)
}

func (s *Service) RecordStockAdjustment(ctx context.Context, businessID, userID string, input StockAdjustmentInput, meta RequestMeta) (*StockMovement, error) {
	if input.ProductID == "" {
		return nil, errors.New("productId is required")
	}
	if input.Quantity == 0 {
		return nil, errors.New("adjustment quantity cannot be zero")
	}

	movement, err := s.repo.RecordStockAdjustment(ctx, businessID, input, userID)
	if err != nil {
		return nil, err
	}

	_ = s.auditLogger.Log(ctx, audit.Entry{
		BusinessID:    &businessID,
		UserID:        &userID,
		Action:        "inventory.adjusted",
		AggregateType: "stock_movement",
		AggregateID:   &movement.ID,
		Metadata: map[string]any{
			"productId":      movement.ProductID,
			"quantity":       movement.Quantity,
			"movementType":   movement.MovementType,
			"idempotencyKey": movement.IdempotencyKey,
		},
		RequestID: meta.RequestID,
		IPAddress: meta.IPAddress,
		UserAgent: meta.UserAgent,
	})

	return movement, nil
}

func (s *Service) ListMovements(ctx context.Context, businessID, productID string, query ListMovementsQuery) ([]StockMovement, string, error) {
	return s.repo.ListMovements(ctx, businessID, productID, query)
}
