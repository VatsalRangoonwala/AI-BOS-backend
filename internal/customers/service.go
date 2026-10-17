package customers

import (
	"context"
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

func (s *Service) CreateCustomer(ctx context.Context, businessID, userID string, input CreateCustomerInput, meta RequestMeta) (*Customer, error) {
	name := strings.TrimSpace(input.Name)
	if name == "" {
		name = strings.TrimSpace(input.FullName)
	}
	if name == "" {
		return nil, ErrInvalidCustomerName
	}

	phone := input.Phone
	if phone == nil && input.Mobile != nil {
		phone = input.Mobile
	}

	customer := &Customer{
		BusinessID: businessID,
		Name:       name,
		Phone:      phone,
		Email:      input.Email,
		Address:    input.Address,
		Notes:      input.Notes,
	}

	if err := s.repo.Create(ctx, customer); err != nil {
		return nil, err
	}

	_ = s.auditLogger.Log(ctx, audit.Entry{
		BusinessID:    &businessID,
		UserID:        &userID,
		Action:        "customer.created",
		AggregateType: "customer",
		AggregateID:   &customer.ID,
		Metadata: map[string]any{
			"name":  customer.Name,
			"phone": customer.Phone,
			"email": customer.Email,
		},
		RequestID: meta.RequestID,
		IPAddress: meta.IPAddress,
		UserAgent: meta.UserAgent,
	})

	return customer, nil
}

func (s *Service) GetCustomer(ctx context.Context, businessID, customerID string) (*Customer, error) {
	return s.repo.GetByID(ctx, businessID, customerID)
}

func (s *Service) UpdateCustomer(ctx context.Context, businessID, customerID, userID string, input UpdateCustomerInput, meta RequestMeta) (*Customer, error) {
	customer, err := s.repo.Update(ctx, businessID, customerID, input)
	if err != nil {
		return nil, err
	}

	_ = s.auditLogger.Log(ctx, audit.Entry{
		BusinessID:    &businessID,
		UserID:        &userID,
		Action:        "customer.updated",
		AggregateType: "customer",
		AggregateID:   &customerID,
		Metadata: map[string]any{
			"updatedFields": input,
		},
		RequestID: meta.RequestID,
		IPAddress: meta.IPAddress,
		UserAgent: meta.UserAgent,
	})

	return customer, nil
}

func (s *Service) ArchiveCustomer(ctx context.Context, businessID, customerID, userID string, meta RequestMeta) error {
	if err := s.repo.Archive(ctx, businessID, customerID); err != nil {
		return err
	}

	_ = s.auditLogger.Log(ctx, audit.Entry{
		BusinessID:    &businessID,
		UserID:        &userID,
		Action:        "customer.archived",
		AggregateType: "customer",
		AggregateID:   &customerID,
		Metadata:      map[string]any{},
		RequestID:     meta.RequestID,
		IPAddress:     meta.IPAddress,
		UserAgent:     meta.UserAgent,
	})

	return nil
}

func (s *Service) ListCustomers(ctx context.Context, businessID string, query ListCustomersQuery) ([]Customer, string, error) {
	return s.repo.List(ctx, businessID, query)
}

func (s *Service) GetCustomerLedger(ctx context.Context, businessID, customerID string) (*CustomerLedgerSummary, error) {
	// Verify customer exists in business
	customer, err := s.repo.GetByID(ctx, businessID, customerID)
	if err != nil {
		return nil, err
	}

	// For Phase 2 baseline (invoices/payments are Phase 3), return initial ledger summary:
	return &CustomerLedgerSummary{
		CustomerID:       customer.ID,
		TotalBilledMinor: 0,
		TotalPaidMinor:   0,
		OutstandingMinor: 0,
		Currency:         "INR",
	}, nil
}
