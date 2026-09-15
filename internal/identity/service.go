package identity

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/VatsalRangoonwala/AI-BOS-backend/internal/audit"
	"github.com/VatsalRangoonwala/AI-BOS-backend/internal/businesses"
	"github.com/VatsalRangoonwala/AI-BOS-backend/internal/platform/security"
	"github.com/google/uuid"
)

type RegisterInput struct {
	Email        string
	Password     string
	Name         string
	BusinessName string
}

type LoginInput struct {
	Email    string
	Password string
}

type RequestMeta struct {
	IPAddress string
	UserAgent string
	RequestID string
}

type Service struct {
	userRepo     *Repository
	businessRepo *businesses.Repository
	auditLogger  audit.Logger
	jwtSecret    []byte
	tokenTTL     time.Duration
	sessionTTL   time.Duration
}

func NewService(
	userRepo *Repository,
	businessRepo *businesses.Repository,
	auditLogger audit.Logger,
	jwtSecret []byte,
	tokenTTL time.Duration,
	sessionTTL time.Duration,
) *Service {
	if tokenTTL <= 0 {
		tokenTTL = 15 * time.Minute
	}
	if sessionTTL <= 0 {
		sessionTTL = 7 * 24 * time.Hour
	}
	return &Service{
		userRepo:     userRepo,
		businessRepo: businessRepo,
		auditLogger:  auditLogger,
		jwtSecret:    jwtSecret,
		tokenTTL:     tokenTTL,
		sessionTTL:   sessionTTL,
	}
}

func (s *Service) Register(ctx context.Context, input RegisterInput, meta RequestMeta) (*AuthTokens, error) {
	input.Email = strings.ToLower(strings.TrimSpace(input.Email))
	input.Name = strings.TrimSpace(input.Name)
	if len(input.Password) < 8 {
		return nil, ErrPasswordTooShort
	}
	if input.Email == "" || input.Name == "" {
		return nil, errors.New("email and name are required")
	}

	passwordHash, err := security.HashPassword(input.Password)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	userID := uuid.NewString()
	user := &User{
		ID:           userID,
		Email:        input.Email,
		PasswordHash: passwordHash,
		Name:         input.Name,
		Status:       "active",
	}

	if err := s.userRepo.CreateUser(ctx, user); err != nil {
		return nil, err
	}

	// Create initial business tenant if specified, or default to a business with user's name
	bizName := strings.TrimSpace(input.BusinessName)
	if bizName == "" {
		bizName = fmt.Sprintf("%s's Store", user.Name)
	}
	if s.businessRepo != nil {
		biz := &businesses.Business{
			ID:              uuid.NewString(),
			Name:            bizName,
			Timezone:        "Asia/Kolkata",
			Currency:        "INR",
			InvoicePrefix:   "INV-",
			InvoiceSequence: 1001,
			Status:          "active",
		}
		if err := s.businessRepo.CreateBusinessWithMember(ctx, biz, user.ID); err != nil {
			return nil, fmt.Errorf("create initial business: %w", err)
		}
	}

	rawRefreshToken, err := security.GenerateRandomToken(32)
	if err != nil {
		return nil, fmt.Errorf("generate refresh token: %w", err)
	}

	now := time.Now().UTC()
	session := &RefreshSession{
		ID:         uuid.NewString(),
		UserID:     user.ID,
		TokenHash:  security.HashToken(rawRefreshToken),
		DeviceInfo: meta.UserAgent,
		IPAddress:  meta.IPAddress,
		ExpiresAt:  now.Add(s.sessionTTL),
		LastSeenAt: now,
		CreatedAt:  now,
	}
	if err := s.userRepo.CreateRefreshSession(ctx, session); err != nil {
		return nil, fmt.Errorf("create refresh session: %w", err)
	}

	accessToken, err := security.GenerateAccessToken(user.ID, user.Email, s.jwtSecret, s.tokenTTL)
	if err != nil {
		return nil, fmt.Errorf("generate access token: %w", err)
	}

	if s.auditLogger != nil {
		_ = s.auditLogger.Log(ctx, audit.Entry{
			UserID:        &user.ID,
			Action:        "user.registered",
			AggregateType: "user",
			AggregateID:   &user.ID,
			Metadata:      map[string]any{"email": user.Email},
			RequestID:     meta.RequestID,
			IPAddress:     meta.IPAddress,
			UserAgent:     meta.UserAgent,
		})
	}

	return &AuthTokens{
		AccessToken:  accessToken,
		RefreshToken: rawRefreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    int(s.tokenTTL.Seconds()),
		User:         user,
	}, nil
}

func (s *Service) Login(ctx context.Context, input LoginInput, meta RequestMeta) (*AuthTokens, error) {
	input.Email = strings.ToLower(strings.TrimSpace(input.Email))
	user, err := s.userRepo.GetUserByEmail(ctx, input.Email)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}

	valid, err := security.VerifyPassword(input.Password, user.PasswordHash)
	if err != nil || !valid {
		return nil, ErrInvalidCredentials
	}

	if user.Status == "suspended" {
		return nil, ErrAccountSuspended
	}

	rawRefreshToken, err := security.GenerateRandomToken(32)
	if err != nil {
		return nil, fmt.Errorf("generate refresh token: %w", err)
	}

	now := time.Now().UTC()
	session := &RefreshSession{
		ID:         uuid.NewString(),
		UserID:     user.ID,
		TokenHash:  security.HashToken(rawRefreshToken),
		DeviceInfo: meta.UserAgent,
		IPAddress:  meta.IPAddress,
		ExpiresAt:  now.Add(s.sessionTTL),
		LastSeenAt: now,
		CreatedAt:  now,
	}
	if err := s.userRepo.CreateRefreshSession(ctx, session); err != nil {
		return nil, fmt.Errorf("create refresh session: %w", err)
	}

	accessToken, err := security.GenerateAccessToken(user.ID, user.Email, s.jwtSecret, s.tokenTTL)
	if err != nil {
		return nil, fmt.Errorf("generate access token: %w", err)
	}

	if s.auditLogger != nil {
		_ = s.auditLogger.Log(ctx, audit.Entry{
			UserID:        &user.ID,
			Action:        "user.logged_in",
			AggregateType: "user",
			AggregateID:   &user.ID,
			Metadata:      map[string]any{"email": user.Email},
			RequestID:     meta.RequestID,
			IPAddress:     meta.IPAddress,
			UserAgent:     meta.UserAgent,
		})
	}

	return &AuthTokens{
		AccessToken:  accessToken,
		RefreshToken: rawRefreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    int(s.tokenTTL.Seconds()),
		User:         user,
	}, nil
}

func (s *Service) Refresh(ctx context.Context, refreshToken string, meta RequestMeta) (*AuthTokens, error) {
	tokenHash := security.HashToken(refreshToken)
	session, err := s.userRepo.GetRefreshSession(ctx, tokenHash)
	if err != nil {
		return nil, ErrSessionNotFound
	}

	// Revoke old session (Rotation)
	if err := s.userRepo.RevokeRefreshSession(ctx, tokenHash); err != nil {
		return nil, fmt.Errorf("revoke old refresh session: %w", err)
	}

	user, err := s.userRepo.GetUserByID(ctx, session.UserID)
	if err != nil {
		return nil, fmt.Errorf("fetch user for session: %w", err)
	}

	if user.Status == "suspended" {
		return nil, ErrAccountSuspended
	}

	newRefreshToken, err := security.GenerateRandomToken(32)
	if err != nil {
		return nil, fmt.Errorf("generate new refresh token: %w", err)
	}

	now := time.Now().UTC()
	newSession := &RefreshSession{
		ID:         uuid.NewString(),
		UserID:     user.ID,
		TokenHash:  security.HashToken(newRefreshToken),
		DeviceInfo: meta.UserAgent,
		IPAddress:  meta.IPAddress,
		ExpiresAt:  now.Add(s.sessionTTL),
		LastSeenAt: now,
		CreatedAt:  now,
	}
	if err := s.userRepo.CreateRefreshSession(ctx, newSession); err != nil {
		return nil, fmt.Errorf("create rotated refresh session: %w", err)
	}

	accessToken, err := security.GenerateAccessToken(user.ID, user.Email, s.jwtSecret, s.tokenTTL)
	if err != nil {
		return nil, fmt.Errorf("generate access token: %w", err)
	}

	if s.auditLogger != nil {
		_ = s.auditLogger.Log(ctx, audit.Entry{
			UserID:        &user.ID,
			Action:        "session.refreshed",
			AggregateType: "session",
			AggregateID:   &newSession.ID,
			Metadata:      map[string]any{"userId": user.ID},
			RequestID:     meta.RequestID,
			IPAddress:     meta.IPAddress,
			UserAgent:     meta.UserAgent,
		})
	}

	return &AuthTokens{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    int(s.tokenTTL.Seconds()),
		User:         user,
	}, nil
}

func (s *Service) Logout(ctx context.Context, refreshToken string, userID string, meta RequestMeta) error {
	tokenHash := security.HashToken(refreshToken)
	_ = s.userRepo.RevokeRefreshSession(ctx, tokenHash)

	if s.auditLogger != nil && userID != "" {
		_ = s.auditLogger.Log(ctx, audit.Entry{
			UserID:        &userID,
			Action:        "user.logged_out",
			AggregateType: "user",
			AggregateID:   &userID,
			Metadata:      map[string]any{},
			RequestID:     meta.RequestID,
			IPAddress:     meta.IPAddress,
			UserAgent:     meta.UserAgent,
		})
	}
	return nil
}

func (s *Service) VerifyEmail(ctx context.Context, token string, meta RequestMeta) error {
	tokenHash := security.HashToken(token)
	consumed, err := s.userRepo.ConsumeAuthToken(ctx, tokenHash, "email_verification")
	if err != nil {
		return ErrTokenInvalid
	}

	if err := s.userRepo.SetEmailVerified(ctx, consumed.UserID); err != nil {
		return fmt.Errorf("verify user email: %w", err)
	}

	if s.auditLogger != nil {
		_ = s.auditLogger.Log(ctx, audit.Entry{
			UserID:        &consumed.UserID,
			Action:        "email.verified",
			AggregateType: "user",
			AggregateID:   &consumed.UserID,
			Metadata:      map[string]any{},
			RequestID:     meta.RequestID,
			IPAddress:     meta.IPAddress,
			UserAgent:     meta.UserAgent,
		})
	}
	return nil
}

func (s *Service) ForgotPassword(ctx context.Context, email string, meta RequestMeta) (string, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	user, err := s.userRepo.GetUserByEmail(ctx, email)
	if err != nil {
		// Return empty token without error to prevent account enumeration
		return "", nil
	}

	rawToken, err := security.GenerateRandomToken(32)
	if err != nil {
		return "", fmt.Errorf("generate reset token: %w", err)
	}

	authToken := &AuthToken{
		ID:        uuid.NewString(),
		UserID:    user.ID,
		TokenHash: security.HashToken(rawToken),
		TokenType: "password_reset",
		ExpiresAt: time.Now().UTC().Add(1 * time.Hour),
	}
	if err := s.userRepo.CreateAuthToken(ctx, authToken); err != nil {
		return "", fmt.Errorf("store reset token: %w", err)
	}

	return rawToken, nil
}

func (s *Service) ResetPassword(ctx context.Context, token, newPassword string, meta RequestMeta) error {
	if len(newPassword) < 8 {
		return ErrPasswordTooShort
	}

	tokenHash := security.HashToken(token)
	consumed, err := s.userRepo.ConsumeAuthToken(ctx, tokenHash, "password_reset")
	if err != nil {
		return ErrTokenInvalid
	}

	passwordHash, err := security.HashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("hash new password: %w", err)
	}

	if err := s.userRepo.UpdatePassword(ctx, consumed.UserID, passwordHash); err != nil {
		return fmt.Errorf("update user password: %w", err)
	}

	// Invalidate all active sessions for this user for security
	_ = s.userRepo.RevokeAllUserSessions(ctx, consumed.UserID)

	if s.auditLogger != nil {
		_ = s.auditLogger.Log(ctx, audit.Entry{
			UserID:        &consumed.UserID,
			Action:        "password.reset",
			AggregateType: "user",
			AggregateID:   &consumed.UserID,
			Metadata:      map[string]any{},
			RequestID:     meta.RequestID,
			IPAddress:     meta.IPAddress,
			UserAgent:     meta.UserAgent,
		})
	}
	return nil
}

func (s *Service) GetCurrentUser(ctx context.Context, userID string) (*User, error) {
	return s.userRepo.GetUserByID(ctx, userID)
}
