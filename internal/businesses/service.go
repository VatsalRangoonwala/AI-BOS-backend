package businesses

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

type CreateBusinessInput struct {
	Name          string
	LegalName     *string
	Phone         *string
	Address       *string
	Timezone      string
	Currency      string
	InvoicePrefix string
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

func (s *Service) CreateBusiness(ctx context.Context, userID string, input CreateBusinessInput, meta RequestMeta) (*Business, error) {
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, errors.New("business name is required")
	}

	biz := &Business{
		Name:            name,
		LegalName:       input.LegalName,
		Phone:           input.Phone,
		Address:         input.Address,
		Timezone:        input.Timezone,
		Currency:        input.Currency,
		InvoicePrefix:   input.InvoicePrefix,
		InvoiceSequence: 1001,
		Status:          "active",
	}

	if err := s.repo.CreateBusinessWithMember(ctx, biz, userID); err != nil {
		return nil, err
	}

	if s.auditLogger != nil {
		_ = s.auditLogger.Log(ctx, audit.Entry{
			BusinessID:    &biz.ID,
			UserID:        &userID,
			Action:        "business.created",
			AggregateType: "business",
			AggregateID:   &biz.ID,
			Metadata:      map[string]any{"name": biz.Name},
			RequestID:     meta.RequestID,
			IPAddress:     meta.IPAddress,
			UserAgent:     meta.UserAgent,
		})
	}

	return biz, nil
}

func (s *Service) GetBusiness(ctx context.Context, businessID string, callerUserID string) (*Business, string, error) {
	return s.repo.GetBusinessByID(ctx, businessID, callerUserID)
}

func (s *Service) UpdateBusiness(ctx context.Context, businessID string, callerUserID string, input UpdateBusinessInput, meta RequestMeta) (*Business, error) {
	biz, err := s.repo.UpdateBusiness(ctx, businessID, callerUserID, input)
	if err != nil {
		return nil, err
	}

	if s.auditLogger != nil {
		_ = s.auditLogger.Log(ctx, audit.Entry{
			BusinessID:    &businessID,
			UserID:        &callerUserID,
			Action:        "business.updated",
			AggregateType: "business",
			AggregateID:   &businessID,
			Metadata:      map[string]any{"name": biz.Name},
			RequestID:     meta.RequestID,
			IPAddress:     meta.IPAddress,
			UserAgent:     meta.UserAgent,
		})
	}

	return biz, nil
}

func (s *Service) ListUserBusinesses(ctx context.Context, userID string) ([]UserBusinessSummary, error) {
	return s.repo.ListBusinessesForUser(ctx, userID)
}

func (s *Service) ListMembers(ctx context.Context, businessID string, callerUserID string) ([]MemberDetail, error) {
	return s.repo.ListMembers(ctx, businessID, callerUserID)
}

func (s *Service) InviteMember(ctx context.Context, businessID string, callerUserID string, email, role string, meta RequestMeta) (*MemberDetail, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" {
		return nil, errors.New("email is required")
	}
	if role != "owner" && role != "staff" && role != "admin" {
		return nil, errors.New("invalid role; must be owner, staff, or admin")
	}

	member, err := s.repo.InviteMember(ctx, businessID, callerUserID, email, role)
	if err != nil {
		return nil, err
	}

	if s.auditLogger != nil {
		_ = s.auditLogger.Log(ctx, audit.Entry{
			BusinessID:    &businessID,
			UserID:        &callerUserID,
			Action:        "member.invited",
			AggregateType: "member",
			AggregateID:   &member.ID,
			Metadata:      map[string]any{"email": email, "role": role},
			RequestID:     meta.RequestID,
			IPAddress:     meta.IPAddress,
			UserAgent:     meta.UserAgent,
		})
	}

	return member, nil
}

func (s *Service) UpdateMember(ctx context.Context, businessID string, callerUserID string, memberID string, role, status *string, meta RequestMeta) (*MemberDetail, error) {
	member, err := s.repo.UpdateMember(ctx, businessID, callerUserID, memberID, role, status)
	if err != nil {
		return nil, err
	}

	if s.auditLogger != nil {
		_ = s.auditLogger.Log(ctx, audit.Entry{
			BusinessID:    &businessID,
			UserID:        &callerUserID,
			Action:        "member.updated",
			AggregateType: "member",
			AggregateID:   &memberID,
			Metadata:      map[string]any{"role": member.Role, "status": member.Status},
			RequestID:     meta.RequestID,
			IPAddress:     meta.IPAddress,
			UserAgent:     meta.UserAgent,
		})
	}

	return member, nil
}

func (s *Service) RemoveMember(ctx context.Context, businessID string, callerUserID string, memberID string, meta RequestMeta) error {
	if err := s.repo.RemoveMember(ctx, businessID, callerUserID, memberID); err != nil {
		return err
	}

	if s.auditLogger != nil {
		_ = s.auditLogger.Log(ctx, audit.Entry{
			BusinessID:    &businessID,
			UserID:        &callerUserID,
			Action:        "member.removed",
			AggregateType: "member",
			AggregateID:   &memberID,
			Metadata:      map[string]any{"memberId": memberID},
			RequestID:     meta.RequestID,
			IPAddress:     meta.IPAddress,
			UserAgent:     meta.UserAgent,
		})
	}

	return nil
}
