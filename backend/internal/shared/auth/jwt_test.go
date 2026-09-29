package auth

import (
	"errors"
	"strings"
	"testing"
	"time"

	sharedErrors "lunar/backend/internal/shared/errors"
)

func TestGenerateAndValidateToken(t *testing.T) {
	secret := "super-secure-jwt-signing-key-12345"
	userID := "usr-123e4567-e89b"
	email := "developer@example.com"
	ttl := 1 * time.Hour

	token, err := GenerateToken(userID, email, secret, ttl)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	claims, err := ValidateToken(token, secret)
	if err != nil {
		t.Fatalf("failed to validate valid token: %v", err)
	}

	if claims.UserID != userID {
		t.Fatalf("expected userID %s, got %s", userID, claims.UserID)
	}
	if claims.Email != email {
		t.Fatalf("expected email %s, got %s", email, claims.Email)
	}
	if claims.ExpiresAt <= claims.IssuedAt {
		t.Fatalf("expected expiresAt to be after issuedAt")
	}
}

func TestValidateTokenExpired(t *testing.T) {
	secret := "super-secure-jwt-signing-key-12345"
	userID := "usr-expired"
	email := "expired@example.com"
	ttl := -1 * time.Minute

	token, err := GenerateToken(userID, email, secret, ttl)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	_, err = ValidateToken(token, secret)
	if !errors.Is(err, sharedErrors.ErrTokenExpired) {
		t.Fatalf("expected ErrTokenExpired, got %v", err)
	}
}

func TestValidateTokenTamperedSignature(t *testing.T) {
	secret := "super-secure-jwt-signing-key-12345"
	token, err := GenerateToken("user-1", "user1@example.com", secret, time.Hour)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		t.Fatalf("invalid token format")
	}

	tamperedSig := parts[2] + "x"
	tamperedToken := parts[0] + "." + parts[1] + "." + tamperedSig

	_, err = ValidateToken(tamperedToken, secret)
	if !errors.Is(err, sharedErrors.ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken on tampered signature, got %v", err)
	}
}

func TestValidateTokenTamperedPayload(t *testing.T) {
	secret := "super-secure-jwt-signing-key-12345"
	token, err := GenerateToken("user-1", "user1@example.com", secret, time.Hour)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	parts := strings.Split(token, ".")
	tamperedPayload := parts[1] + "a"
	tamperedToken := parts[0] + "." + tamperedPayload + "." + parts[2]

	_, err = ValidateToken(tamperedToken, secret)
	if !errors.Is(err, sharedErrors.ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken on tampered payload, got %v", err)
	}
}

func TestValidateTokenWrongSecret(t *testing.T) {
	secretA := "secret-key-alpha-12345678"
	secretB := "secret-key-bravo-12345678"

	token, err := GenerateToken("user-1", "user1@example.com", secretA, time.Hour)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	_, err = ValidateToken(token, secretB)
	if !errors.Is(err, sharedErrors.ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken with wrong secret, got %v", err)
	}
}

func TestValidateTokenMalformed(t *testing.T) {
	secret := "super-secure-jwt-signing-key-12345"

	malformedTokens := []string{
		"",
		"header.payload",
		"singlepart",
		"a.b.c.d",
	}

	for _, malformed := range malformedTokens {
		_, err := ValidateToken(malformed, secret)
		if !errors.Is(err, sharedErrors.ErrInvalidToken) {
			t.Fatalf("expected ErrInvalidToken for %q, got %v", malformed, err)
		}
	}
}

func TestEmptySecretHandling(t *testing.T) {
	_, err := GenerateToken("user-1", "user1@example.com", "", time.Hour)
	if !errors.Is(err, sharedErrors.ErrInvalidKey) {
		t.Fatalf("expected ErrInvalidKey on empty generate secret, got %v", err)
	}

	_, err = ValidateToken("a.b.c", "")
	if !errors.Is(err, sharedErrors.ErrInvalidKey) {
		t.Fatalf("expected ErrInvalidKey on empty validate secret, got %v", err)
	}
}
