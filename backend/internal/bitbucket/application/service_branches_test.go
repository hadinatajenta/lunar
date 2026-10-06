package application

import (
	"context"
	"errors"
	"testing"
	"time"

	authApp "lunar/backend/internal/auth/application"
	authDomain "lunar/backend/internal/auth/domain"
	"lunar/backend/internal/shared/crypto"
	sharedErrors "lunar/backend/internal/shared/errors"
)

const testEncryptionKey = "test-encryption-key"

type stubVaultRepository struct {
	secrets *authDomain.UserSecrets
}

var _ authDomain.VaultRepository = (*stubVaultRepository)(nil)

func (s *stubVaultRepository) SaveSecrets(ctx context.Context, secrets *authDomain.UserSecrets) error {
	s.secrets = secrets
	return nil
}

func (s *stubVaultRepository) GetSecretsByUserID(ctx context.Context, userID string) (*authDomain.UserSecrets, error) {
	if s.secrets == nil {
		return nil, sharedErrors.ErrNotFound
	}

	return s.secrets, nil
}

func newServiceWithStoredPAT(t *testing.T, pat string) (*BitbucketService, *mockBitbucketRepo) {
	t.Helper()

	encryptedPAT, err := crypto.EncryptSecret(pat, testEncryptionKey)
	if err != nil {
		t.Fatalf("cannot encrypt the stored PAT: %v", err)
	}

	vault := &stubVaultRepository{secrets: &authDomain.UserSecrets{
		UserID:            "usr-1",
		BitbucketPATEnc:   encryptedPAT,
		BitbucketUsername: "erendt",
	}}
	authService := authApp.NewAuthService(nil, vault, testEncryptionKey, "jwt-secret", time.Minute)

	mockRepo := &mockBitbucketRepo{branches: []string{"main"}, defaultBranch: "master"}
	return NewBitbucketService(mockRepo, authService, nil), mockRepo
}

func TestRemoteBranchScannerResolvesTheCallersPATForEveryMethod(t *testing.T) {
	service, mockRepo := newServiceWithStoredPAT(t, "personal-pat")
	var scanner RemoteBranchScanner = service
	ctx := context.Background()

	if _, err := service.SearchBranches(ctx, "usr-1", "NME/mms-api", "CRMMS-77911"); err != nil {
		t.Fatalf("cannot search branches: %v", err)
	}
	searchBranchesPAT := mockRepo.lastPAT

	branches, err := scanner.ListBranches(ctx, "usr-1", "NME/mms-api")
	if err != nil {
		t.Fatalf("cannot list branches: %v", err)
	}
	if mockRepo.lastPAT != "personal-pat" || mockRepo.lastPAT != searchBranchesPAT {
		t.Fatalf("expected the stored PAT %q, got %q", searchBranchesPAT, mockRepo.lastPAT)
	}
	if mockRepo.lastRepo != "NME/mms-api" {
		t.Fatalf("expected the repository to be forwarded, got %q", mockRepo.lastRepo)
	}
	if len(branches) != 1 || branches[0] != "main" {
		t.Fatalf("unexpected branches %+v", branches)
	}

	defaultBranch, err := scanner.DefaultBranch(ctx, "usr-1", "NME/mms-api")
	if err != nil {
		t.Fatalf("cannot read the default branch: %v", err)
	}
	if mockRepo.lastPAT != "personal-pat" {
		t.Fatalf("expected the stored PAT, got %q", mockRepo.lastPAT)
	}
	if defaultBranch != "master" {
		t.Fatalf("unexpected default branch %q", defaultBranch)
	}
}

func TestListBranchesRequiresARepository(t *testing.T) {
	service := NewBitbucketService(&mockBitbucketRepo{}, nil, nil)

	_, err := service.ListBranches(context.Background(), "usr-1", "  ")
	if !errors.Is(err, sharedErrors.ErrBadRequest) {
		t.Fatalf("expected a bad request error, got %v", err)
	}

	_, err = service.DefaultBranch(context.Background(), "usr-1", "")
	if !errors.Is(err, sharedErrors.ErrBadRequest) {
		t.Fatalf("expected a bad request error, got %v", err)
	}
}
