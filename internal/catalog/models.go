package catalog

import (
	"errors"
	"time"
)

var (
	ErrProductNotFound         = errors.New("product not found or inaccessible")
	ErrSKUAlreadyExists        = errors.New("a product with this SKU already exists in this business")
	ErrInvalidProductData      = errors.New("product name and SKU are required")
	ErrNegativeStockNotAllowed = errors.New("stock deduction would result in negative inventory")
	ErrInsufficientStock       = errors.New("insufficient stock for requested operation")
)

type Product struct {
	ID                 string     `json:"id"`
	BusinessID         string     `json:"businessId"`
	Name               string     `json:"name"`
	SKU                string     `json:"sku"`
	Barcode            *string    `json:"barcode,omitempty"`
	Category           string     `json:"category"`
	Description        *string    `json:"description,omitempty"`
	Unit               string     `json:"unit"`
	SalePriceMinor     int64      `json:"salePriceMinor"`
	CostPriceMinor     int64      `json:"costPriceMinor"`
	Currency           string     `json:"currency"`
	TaxRateBasisPoints int        `json:"taxRateBasisPoints"`
	ReorderLevel       int        `json:"reorderLevel"`
	OnHand             int        `json:"onHand"`
	Reserved           int        `json:"reserved"`
	ArchivedAt         *time.Time `json:"archivedAt,omitempty"`
	CreatedAt          time.Time  `json:"createdAt"`
	UpdatedAt          time.Time  `json:"updatedAt"`
}

type InventoryBalance struct {
	BusinessID string    `json:"businessId"`
	ProductID  string    `json:"productId"`
	LocationID *string   `json:"locationId,omitempty"`
	OnHand     int       `json:"onHand"`
	Reserved   int       `json:"reserved"`
	Version    int       `json:"version"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

type ProductInventoryItem struct {
	ProductID    string `json:"productId"`
	ProductName  string `json:"productName"`
	SKU          string `json:"sku"`
	Unit         string `json:"unit"`
	OnHand       int    `json:"onHand"`
	Reserved     int    `json:"reserved"`
	Available    int    `json:"available"`
	ReorderLevel int    `json:"reorderLevel"`
	IsLowStock   bool   `json:"isLowStock"`
}

type StockMovement struct {
	ID             string    `json:"id"`
	BusinessID     string    `json:"businessId"`
	ProductID      string    `json:"productId"`
	Quantity       int       `json:"quantity"`
	MovementType   string    `json:"movementType"`
	SourceType     string    `json:"sourceType"`
	SourceID       *string   `json:"sourceId,omitempty"`
	Notes          *string   `json:"notes,omitempty"`
	CreatedBy      *string   `json:"createdBy,omitempty"`
	IdempotencyKey *string   `json:"idempotencyKey,omitempty"`
	OccurredAt     time.Time `json:"occurredAt"`
	CreatedAt      time.Time `json:"createdAt"`
}

type CreateProductInput struct {
	Name               string  `json:"name"`
	SKU                string  `json:"sku"`
	Barcode            *string `json:"barcode,omitempty"`
	Category           string  `json:"category,omitempty"`
	Description        *string `json:"description,omitempty"`
	Unit               string  `json:"unit,omitempty"`
	SalePriceMinor     int64   `json:"salePriceMinor"`
	CostPriceMinor     int64   `json:"costPriceMinor"`
	Currency           string  `json:"currency,omitempty"`
	TaxRateBasisPoints int     `json:"taxRateBasisPoints,omitempty"`
	ReorderLevel       int     `json:"reorderLevel,omitempty"`
	InitialStock       *int    `json:"initialStock,omitempty"`
}

type UpdateProductInput struct {
	Name               *string `json:"name,omitempty"`
	Barcode            *string `json:"barcode,omitempty"`
	Category           *string `json:"category,omitempty"`
	Description        *string `json:"description,omitempty"`
	Unit               *string `json:"unit,omitempty"`
	SalePriceMinor     *int64  `json:"salePriceMinor,omitempty"`
	CostPriceMinor     *int64  `json:"costPriceMinor,omitempty"`
	TaxRateBasisPoints *int    `json:"taxRateBasisPoints,omitempty"`
	ReorderLevel       *int    `json:"reorderLevel,omitempty"`
}

type StockAdjustmentInput struct {
	ProductID      string  `json:"productId"`
	Quantity       int     `json:"quantity"` // signed delta: positive to add, negative to deduct
	MovementType   string  `json:"movementType"`
	Notes          *string `json:"notes,omitempty"`
	IdempotencyKey *string `json:"idempotencyKey,omitempty"`
}

type ListProductsQuery struct {
	Query  string
	Limit  int
	Cursor string
}

type ListInventoryQuery struct {
	Query        string
	LowStockOnly bool
	Limit        int
	Cursor       string
}

type ListMovementsQuery struct {
	Limit  int
	Cursor string
}
