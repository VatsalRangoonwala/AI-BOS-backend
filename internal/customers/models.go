package customers

import (
	"errors"
	"time"
)

var (
	ErrCustomerNotFound    = errors.New("customer not found or inaccessible")
	ErrInvalidCustomerName = errors.New("customer name is required and cannot be empty")
)

type Customer struct {
	ID         string     `json:"id"`
	BusinessID string     `json:"businessId"`
	Name       string     `json:"name"`
	Phone      *string    `json:"phone,omitempty"`
	Email      *string    `json:"email,omitempty"`
	Address    *string    `json:"address,omitempty"`
	Notes      *string    `json:"notes,omitempty"`
	ArchivedAt *time.Time `json:"archivedAt,omitempty"`
	CreatedAt  time.Time  `json:"createdAt"`
	UpdatedAt  time.Time  `json:"updatedAt"`
}

type CustomerLedgerSummary struct {
	CustomerID       string `json:"customerId"`
	TotalBilledMinor int64  `json:"totalBilledMinor"`
	TotalPaidMinor   int64  `json:"totalPaidMinor"`
	OutstandingMinor int64  `json:"outstandingMinor"`
	Currency         string `json:"currency"`
}

type CreateCustomerInput struct {
	Name    string  `json:"name"`
	Phone   *string `json:"phone,omitempty"`
	Email   *string `json:"email,omitempty"`
	Address *string `json:"address,omitempty"`
	Notes   *string `json:"notes,omitempty"`
}

type UpdateCustomerInput struct {
	Name    *string `json:"name,omitempty"`
	Phone   *string `json:"phone,omitempty"`
	Email   *string `json:"email,omitempty"`
	Address *string `json:"address,omitempty"`
	Notes   *string `json:"notes,omitempty"`
}

type ListCustomersQuery struct {
	Query  string
	Limit  int
	Cursor string
}
