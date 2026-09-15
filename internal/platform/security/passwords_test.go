package security

import (
	"testing"
)

func TestHashAndVerifyPassword(t *testing.T) {
	// Use small parameters for fast test execution
	testParams := &Argon2Params{
		Memory:      1024,
		Iterations:  1,
		Parallelism: 1,
		SaltLength:  16,
		KeyLength:   32,
	}

	password := "SecretPassword123!"
	hash, err := HashPasswordWithParams(password, testParams)
	if err != nil {
		t.Fatalf("unexpected error hashing password: %v", err)
	}

	match, err := VerifyPassword(password, hash)
	if err != nil {
		t.Fatalf("unexpected error verifying correct password: %v", err)
	}
	if !match {
		t.Fatalf("expected password to match hash")
	}

	match, err = VerifyPassword("WrongPassword", hash)
	if err != nil {
		t.Fatalf("unexpected error verifying incorrect password: %v", err)
	}
	if match {
		t.Fatalf("expected incorrect password to not match hash")
	}
}

func TestVerifyPasswordInvalidHash(t *testing.T) {
	_, err := VerifyPassword("SecretPassword123!", "invalid_hash_string")
	if err == nil {
		t.Fatalf("expected error on malformed hash")
	}
}
