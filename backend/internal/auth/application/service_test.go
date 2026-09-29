package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"lunar/backend/internal/auth/domain"
	"lunar/backend/internal/auth/infrastructure"
	sharedDatabase "lunar/backend/internal/shared/database"
	sharedErrors "lunar/backend/internal/shared/errors"
)

func setupTestService(t *testing.T) (*AuthService, func()) {
	t.Helper()
	db, err := sharedDatabase.OpenDB(":memory:")
	if err != nil {
		t.Fatalf("failed to open in-memory db: %v", err)
	}

	if err := sharedDatabase.AutoMigrate(db); err != nil {
		_ = db.Close()
		t.Fatalf("failed to auto-migrate: %v", err)
	}

	userRepo := infrastructure.NewSQLiteUserRepository(db)
	vaultRepo := infrastructure.NewSQLiteVaultRepository(db)

	encryptionKey := "12345678901234567890123456789012"
	jwtSecret := "test-jwt-secret-key-987654321012"
	jwtTTL := 2 * time.Hour

	service := NewAuthService(userRepo, vaultRepo, encryptionKey, jwtSecret, jwtTTL)

	cleanup := func() {
		_ = db.Close()
	}

	return service, cleanup
}

func TestRegisterSuccess(t *testing.T) {
	service, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()
	req := RegisterRequest{
		Email:    "test@example.com",
		Password: "StrongPassword123!",
		FullName: "Test Engineer",
	}

	res, err := service.Register(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error registering user: %v", err)
	}

	if res.Token == "" {
		t.Fatalf("expected non-empty token")
	}
	if res.User.Email != "test@example.com" {
		t.Fatalf("expected email test@example.com, got %s", res.User.Email)
	}
	if res.User.FullName != "Test Engineer" {
		t.Fatalf("expected name Test Engineer, got %s", res.User.FullName)
	}
	if res.User.ID == "" {
		t.Fatalf("expected non-empty user ID")
	}
}

func TestRegisterDuplicateEmail(t *testing.T) {
	service, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()
	req := RegisterRequest{
		Email:    "duplicate@example.com",
		Password: "StrongPassword123!",
		FullName: "Initial User",
	}

	_, err := service.Register(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error on first registration: %v", err)
	}

	duplicateReq := RegisterRequest{
		Email:    "DUPLICATE@example.com",
		Password: "AnotherPassword123!",
		FullName: "Duplicate User",
	}

	_, err = service.Register(ctx, duplicateReq)
	if !errors.Is(err, sharedErrors.ErrConflict) {
		t.Fatalf("expected ErrConflict on duplicate email, got %v", err)
	}
}

func TestRegisterValidation(t *testing.T) {
	service, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()

	testCases := []struct {
		req RegisterRequest
	}{
		{req: RegisterRequest{Email: "", Password: "ValidPassword123!", FullName: "John"}},
		{req: RegisterRequest{Email: "invalid-email", Password: "ValidPassword123!", FullName: "John"}},
		{req: RegisterRequest{Email: "valid@example.com", Password: "short", FullName: "John"}},
		{req: RegisterRequest{Email: "valid@example.com", Password: "ValidPassword123!", FullName: ""}},
	}

	for _, tc := range testCases {
		_, err := service.Register(ctx, tc.req)
		if !errors.Is(err, sharedErrors.ErrBadRequest) {
			t.Fatalf("expected ErrBadRequest for invalid request, got %v", err)
		}
	}
}

func TestLoginSuccessAndFailure(t *testing.T) {
	service, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()
	email := "login_test@example.com"
	password := "CorrectPassword123!"

	_, err := service.Register(ctx, RegisterRequest{
		Email:    email,
		Password: password,
		FullName: "Login User",
	})
	if err != nil {
		t.Fatalf("failed to register user: %v", err)
	}

	loginRes, err := service.Login(ctx, LoginRequest{
		Email:    email,
		Password: password,
	})
	if err != nil {
		t.Fatalf("expected login to succeed, got %v", err)
	}
	if loginRes.Token == "" {
		t.Fatalf("expected non-empty auth token")
	}
	if loginRes.User.Email != email {
		t.Fatalf("expected email %s, got %s", email, loginRes.User.Email)
	}

	_, err = service.Login(ctx, LoginRequest{
		Email:    email,
		Password: "WrongPassword!",
	})
	if !errors.Is(err, sharedErrors.ErrUnauthorized) {
		t.Fatalf("expected ErrUnauthorized on incorrect password, got %v", err)
	}

	_, err = service.Login(ctx, LoginRequest{
		Email:    "unknown@example.com",
		Password: password,
	})
	if !errors.Is(err, sharedErrors.ErrUnauthorized) {
		t.Fatalf("expected ErrUnauthorized on unknown email, got %v", err)
	}
}

func TestVaultSecretLifecycle(t *testing.T) {
	service, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()
	authRes, err := service.Register(ctx, RegisterRequest{
		Email:    "vault@example.com",
		Password: "VaultPassword123!",
		FullName: "Vault User",
	})
	if err != nil {
		t.Fatalf("failed to register: %v", err)
	}
	userID := authRes.User.ID

	initialRedacted, err := service.GetRedactedSecrets(ctx, userID)
	if err != nil {
		t.Fatalf("failed to get empty secrets: %v", err)
	}
	if initialRedacted.HasJiraPAT || initialRedacted.HasBitbucketPAT || initialRedacted.HasAIKeys {
		t.Fatalf("expected all secret flags to be false initially")
	}

	saveInput := domain.SaveSecretsInput{
		JiraPAT:           "jira-pat-plain-text-12345",
		JiraUsername:      "jira.developer",
		BitbucketPAT:      "bitbucket-pat-secret-value",
		BitbucketUsername: "bb.developer",
		ConfluencePAT:     "confluence-pat-secret-value",
		AIKeys: map[string]string{
			"deepseek": "sk-deepseek-test-key",
			"gemini":   "sk-gemini-test-key",
		},
	}

	if err := service.SaveUserSecrets(ctx, userID, saveInput); err != nil {
		t.Fatalf("failed to save secrets: %v", err)
	}

	redacted, err := service.GetRedactedSecrets(ctx, userID)
	if err != nil {
		t.Fatalf("failed to get redacted secrets: %v", err)
	}

	if !redacted.HasJiraPAT || !redacted.HasBitbucketPAT || !redacted.HasConfluencePAT || !redacted.HasAIKeys {
		t.Fatalf("expected all secret flags to be true after saving")
	}
	if redacted.JiraUsername != "jira.developer" {
		t.Fatalf("expected jira username jira.developer, got %s", redacted.JiraUsername)
	}
	if redacted.BitbucketUsername != "bb.developer" {
		t.Fatalf("expected bitbucket username bb.developer, got %s", redacted.BitbucketUsername)
	}
	if len(redacted.ConfiguredAIProviders) != 2 {
		t.Fatalf("expected 2 configured providers, got %d", len(redacted.ConfiguredAIProviders))
	}

	decrypted, err := service.GetDecryptedSecrets(ctx, userID)
	if err != nil {
		t.Fatalf("failed to get decrypted secrets: %v", err)
	}

	if decrypted.JiraPAT != "jira-pat-plain-text-12345" {
		t.Fatalf("decrypted Jira PAT mismatch")
	}
	if decrypted.BitbucketPAT != "bitbucket-pat-secret-value" {
		t.Fatalf("decrypted Bitbucket PAT mismatch")
	}
	if decrypted.ConfluencePAT != "confluence-pat-secret-value" {
		t.Fatalf("decrypted Confluence PAT mismatch")
	}
	if decrypted.AIKeys["deepseek"] != "sk-deepseek-test-key" {
		t.Fatalf("decrypted DeepSeek key mismatch")
	}
	if decrypted.AIKeys["gemini"] != "sk-gemini-test-key" {
		t.Fatalf("decrypted Gemini key mismatch")
	}

	updateInput := domain.SaveSecretsInput{
		BitbucketPAT: "new-bb-pat-secret",
	}
	if err := service.SaveUserSecrets(ctx, userID, updateInput); err != nil {
		t.Fatalf("failed to update single secret: %v", err)
	}

	updatedDecrypted, err := service.GetDecryptedSecrets(ctx, userID)
	if err != nil {
		t.Fatalf("failed to get updated decrypted secrets: %v", err)
	}
	if updatedDecrypted.BitbucketPAT != "new-bb-pat-secret" {
		t.Fatalf("expected updated Bitbucket PAT, got %s", updatedDecrypted.BitbucketPAT)
	}
	if updatedDecrypted.JiraPAT != "jira-pat-plain-text-12345" {
		t.Fatalf("expected unchanged Jira PAT, got %s", updatedDecrypted.JiraPAT)
	}
}

func TestGetProfile(t *testing.T) {
	service, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()
	authRes, err := service.Register(ctx, RegisterRequest{
		Email:    "profile@example.com",
		Password: "ProfilePassword123!",
		FullName: "Profile User",
	})
	if err != nil {
		t.Fatalf("failed to register: %v", err)
	}

	profile, err := service.GetProfile(ctx, authRes.User.ID)
	if err != nil {
		t.Fatalf("expected GetProfile to succeed, got %v", err)
	}
	if profile.Email != "profile@example.com" {
		t.Fatalf("expected email profile@example.com, got %s", profile.Email)
	}

	_, err = service.GetProfile(ctx, "non-existent-user-id")
	if !errors.Is(err, sharedErrors.ErrNotFound) {
		t.Fatalf("expected ErrNotFound for unknown ID, got %v", err)
	}
}
