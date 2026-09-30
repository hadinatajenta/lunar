package app

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"lunar/backend/internal/auth/application"
	"lunar/backend/internal/config"
)

func TestApplicationBootstrapAndSeedUser(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_app.db")

	cfg := &config.Config{
		Port:             "0",
		DBPath:           dbPath,
		JWTSecret:        "test-jwt-secret-at-least-32-chars-long",
		JWTTTL:           time.Hour,
		EncryptionKey:    "test-aes-key-at-least-32-chars-long",
		CORSOrigin:       "*",
		SeedUserEmail:    "developer@lunar.dev",
		SeedUserPassword: "12345678",
		SeedUserName:     "Lunar Developer",
	}

	app, err := NewApplication(cfg)
	if err != nil {
		t.Fatalf("failed to create application: %v", err)
	}
	defer func() {
		_ = app.Shutdown(context.Background())
	}()

	userRepo := app.DB
	if userRepo == nil {
		t.Fatal("expected database connection to be initialized")
	}

	authService := application.NewAuthService(
		nil,
		nil,
		cfg.EncryptionKey,
		cfg.JWTSecret,
		cfg.JWTTTL,
	)

	var email, fullName, passwordHash, salt string
	err = app.DB.QueryRow("SELECT email, full_name, password_hash, salt FROM users WHERE email = ?", cfg.SeedUserEmail).Scan(&email, &fullName, &passwordHash, &salt)
	if err != nil {
		t.Fatalf("seed user not found in database: %v", err)
	}

	if email != "developer@lunar.dev" {
		t.Errorf("expected email developer@lunar.dev, got %s", email)
	}
	if fullName != "Lunar Developer" {
		t.Errorf("expected full name Lunar Developer, got %s", fullName)
	}

	_ = authService
}
