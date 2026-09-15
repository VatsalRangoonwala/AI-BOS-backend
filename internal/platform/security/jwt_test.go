package security

import (
	"testing"
	"time"
)

func TestJWTGenerationAndValidation(t *testing.T) {
	secret := []byte("super-secret-key-for-unit-testing-only-12345")
	userID := "019318b7-6f89-73e4-8461-9c60e47087cb"
	email := "test@example.com"

	token, err := GenerateAccessToken(userID, email, secret, time.Minute)
	if err != nil {
		t.Fatalf("unexpected error generating token: %v", err)
	}

	claims, err := ValidateAccessToken(token, secret)
	if err != nil {
		t.Fatalf("unexpected error validating token: %v", err)
	}
	if claims.UserID != userID {
		t.Fatalf("got UserID %q, want %q", claims.UserID, userID)
	}
	if claims.Email != email {
		t.Fatalf("got Email %q, want %q", claims.Email, email)
	}

	// Validate with wrong secret
	_, err = ValidateAccessToken(token, []byte("wrong-secret-key-1234567890"))
	if err == nil {
		t.Fatalf("expected error validating with wrong secret")
	}

	// Expired token
	expiredToken, err := GenerateAccessToken(userID, email, secret, -time.Second)
	if err != nil {
		t.Fatalf("unexpected error generating expired token: %v", err)
	}
	_, err = ValidateAccessToken(expiredToken, secret)
	if err == nil {
		t.Fatalf("expected error validating expired token")
	}
}

func TestGenerateAndHashTokens(t *testing.T) {
	token, err := GenerateRandomToken(32)
	if err != nil {
		t.Fatalf("unexpected error generating random token: %v", err)
	}
	if len(token) != 64 {
		t.Fatalf("expected hex string of length 64, got %d", len(token))
	}

	hash1 := HashToken(token)
	hash2 := HashToken(token)
	if hash1 != hash2 {
		t.Fatalf("expected deterministic hash output")
	}
}
