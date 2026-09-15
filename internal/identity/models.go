package identity

import (
	"errors"
	"time"
)

var (
	ErrUserNotFound       = errors.New("user not found")
	ErrUserAlreadyExists  = errors.New("a user with this email already exists")
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrSessionNotFound    = errors.New("session not found or expired")
	ErrTokenInvalid       = errors.New("token is invalid or expired")
	ErrAccountSuspended   = errors.New("account is suspended")
	ErrPasswordTooShort   = errors.New("password must be at least 8 characters")
)

type User struct {
	ID           string     `json:"id"`
	Email        string     `json:"email"`
	PasswordHash string     `json:"-"`
	Name         string     `json:"name"`
	Status       string     `json:"status"` // "active", "unverified", "suspended"
	VerifiedAt   *time.Time `json:"verifiedAt"`
	CreatedAt    time.Time  `json:"createdAt"`
	UpdatedAt    time.Time  `json:"updatedAt"`
}

type RefreshSession struct {
	ID         string     `json:"id"`
	UserID     string     `json:"userId"`
	TokenHash  string     `json:"-"`
	DeviceInfo string     `json:"deviceInfo"`
	IPAddress  string     `json:"ipAddress"`
	ExpiresAt  time.Time  `json:"expiresAt"`
	RevokedAt  *time.Time `json:"revokedAt"`
	LastSeenAt time.Time  `json:"lastSeenAt"`
	CreatedAt  time.Time  `json:"createdAt"`
}

type AuthToken struct {
	ID         string     `json:"id"`
	UserID     string     `json:"userId"`
	TokenHash  string     `json:"-"`
	TokenType  string     `json:"tokenType"` // "email_verification", "password_reset"
	ExpiresAt  time.Time  `json:"expiresAt"`
	ConsumedAt *time.Time `json:"consumedAt"`
	CreatedAt  time.Time  `json:"createdAt"`
}

type AuthTokens struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken,omitempty"`
	TokenType    string `json:"tokenType"`
	ExpiresIn    int    `json:"expiresIn"`
	User         *User  `json:"user"`
}
