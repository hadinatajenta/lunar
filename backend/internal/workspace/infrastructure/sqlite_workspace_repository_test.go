package infrastructure

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
	"time"

	sharedDatabase "lunar/backend/internal/shared/database"
	sharedErrors "lunar/backend/internal/shared/errors"
	"lunar/backend/internal/workspace/domain"
)

const (
	testUserA = "user-a"
	testUserB = "user-b"
)

func openWorkspaceTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sharedDatabase.OpenDB(filepath.Join(t.TempDir(), "workspace.db"))
	if err != nil {
		t.Fatalf("cannot open test database: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	if err := sharedDatabase.AutoMigrate(db); err != nil {
		t.Fatalf("cannot migrate test database: %v", err)
	}

	return db
}

func seedWorkspaceUsers(t *testing.T, db *sql.DB, userIDs ...string) {
	t.Helper()

	now := time.Now().UTC()
	for _, userID := range userIDs {
		if _, err := db.Exec(
			"INSERT INTO users (id, email, password_hash, salt, full_name, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
			userID,
			userID+"@example.com",
			"password-hash",
			"salt-value",
			userID,
			now,
			now,
		); err != nil {
			t.Fatalf("cannot seed user %q: %v", userID, err)
		}
	}
}

func storedHelperToken(t *testing.T, db *sql.DB, userID string) string {
	t.Helper()

	var helperTokenEnc string
	if err := db.QueryRow("SELECT helper_token_enc FROM workspaces WHERE user_id = ?", userID).Scan(&helperTokenEnc); err != nil {
		t.Fatalf("cannot read stored helper token for %q: %v", userID, err)
	}

	return helperTokenEnc
}

func TestSQLiteWorkspaceRepository_GetReturnsNotFoundForUnknownUser(t *testing.T) {
	db := openWorkspaceTestDB(t)
	seedWorkspaceUsers(t, db, testUserA)

	repository := NewSQLiteWorkspaceRepository(db)
	workspace, err := repository.GetWorkspace(context.Background(), testUserA)

	if !errors.Is(err, sharedErrors.ErrNotFound) {
		t.Fatalf("expected ErrNotFound for an unconfigured workspace, got %v", err)
	}
	if workspace != nil {
		t.Fatalf("expected no workspace, got %+v", workspace)
	}
}

func TestSQLiteWorkspaceRepository_UpsertInsertThenUpdatePreservesCreatedAt(t *testing.T) {
	db := openWorkspaceTestDB(t)
	seedWorkspaceUsers(t, db, testUserA)

	repository := NewSQLiteWorkspaceRepository(db)
	firstCreatedAt := time.Now().UTC().Add(-time.Hour).Truncate(time.Microsecond)
	firstUpdatedAt := time.Now().UTC().Add(-time.Hour).Truncate(time.Microsecond)

	err := repository.UpsertWorkspace(context.Background(), &domain.Workspace{
		UserID:         testUserA,
		Source:         domain.SourceServer,
		RootPath:       "/Users/erendt/BRI",
		HelperURL:      "",
		HelperTokenEnc: "",
		CreatedAt:      firstCreatedAt,
		UpdatedAt:      firstUpdatedAt,
	})
	if err != nil {
		t.Fatalf("cannot insert workspace: %v", err)
	}

	secondUpdatedAt := time.Now().UTC().Truncate(time.Microsecond)
	err = repository.UpsertWorkspace(context.Background(), &domain.Workspace{
		UserID:         testUserA,
		Source:         domain.SourceHelper,
		RootPath:       "/Users/erendt/BCA",
		HelperURL:      "http://127.0.0.1:5199",
		HelperTokenEnc: "",
		CreatedAt:      secondUpdatedAt,
		UpdatedAt:      secondUpdatedAt,
	})
	if err != nil {
		t.Fatalf("cannot update workspace: %v", err)
	}

	workspace, err := repository.GetWorkspace(context.Background(), testUserA)
	if err != nil {
		t.Fatalf("cannot load updated workspace: %v", err)
	}
	if workspace.Source != domain.SourceHelper {
		t.Errorf("expected source %q, got %q", domain.SourceHelper, workspace.Source)
	}
	if workspace.RootPath != "/Users/erendt/BCA" {
		t.Errorf("expected updated root path, got %q", workspace.RootPath)
	}
	if workspace.HelperURL != "http://127.0.0.1:5199" {
		t.Errorf("expected updated helper url, got %q", workspace.HelperURL)
	}
	if !workspace.CreatedAt.Truncate(time.Microsecond).Equal(firstCreatedAt) {
		t.Errorf("expected created_at to stay %v, got %v", firstCreatedAt, workspace.CreatedAt)
	}
	if !workspace.UpdatedAt.Truncate(time.Microsecond).Equal(secondUpdatedAt) {
		t.Errorf("expected updated_at to become %v, got %v", secondUpdatedAt, workspace.UpdatedAt)
	}
}

func TestSQLiteWorkspaceRepository_IsolatesUsers(t *testing.T) {
	db := openWorkspaceTestDB(t)
	seedWorkspaceUsers(t, db, testUserA, testUserB)

	repository := NewSQLiteWorkspaceRepository(db)
	now := time.Now().UTC().Truncate(time.Microsecond)

	err := repository.UpsertWorkspace(context.Background(), &domain.Workspace{
		UserID:         testUserA,
		Source:         domain.SourceHelper,
		RootPath:       "/Users/erendt/BRI",
		HelperURL:      "http://127.0.0.1:5199",
		HelperTokenEnc: "ciphertext-for-user-a",
		CreatedAt:      now,
		UpdatedAt:      now,
	})
	if err != nil {
		t.Fatalf("cannot insert workspace for user A: %v", err)
	}

	workspaceForB, err := repository.GetWorkspace(context.Background(), testUserB)
	if !errors.Is(err, sharedErrors.ErrNotFound) {
		t.Fatalf("expected user B to have no workspace, got %v", err)
	}
	if workspaceForB != nil {
		t.Fatalf("expected no workspace for user B, got %+v", workspaceForB)
	}

	if storedHelperToken(t, db, testUserA) != "ciphertext-for-user-a" {
		t.Fatal("expected user A helper token to be stored untouched")
	}
}

func TestSQLiteWorkspaceRepository_StoresOpaqueHelperTokenPerUser(t *testing.T) {
	db := openWorkspaceTestDB(t)
	seedWorkspaceUsers(t, db, testUserA, testUserB)

	repository := NewSQLiteWorkspaceRepository(db)
	now := time.Now().UTC().Truncate(time.Microsecond)

	for _, userID := range []string{testUserA, testUserB} {
		err := repository.UpsertWorkspace(context.Background(), &domain.Workspace{
			UserID:         userID,
			Source:         domain.SourceHelper,
			RootPath:       "/Users/erendt/BRI",
			HelperTokenEnc: "ciphertext-" + userID,
			CreatedAt:      now,
			UpdatedAt:      now,
		})
		if err != nil {
			t.Fatalf("cannot insert workspace for %q: %v", userID, err)
		}
	}

	workspaceForA, err := repository.GetWorkspace(context.Background(), testUserA)
	if err != nil {
		t.Fatalf("cannot load workspace for user A: %v", err)
	}
	if workspaceForA.HelperTokenEnc != "ciphertext-"+testUserA {
		t.Errorf("expected user A ciphertext, got %q", workspaceForA.HelperTokenEnc)
	}

	workspaceForB, err := repository.GetWorkspace(context.Background(), testUserB)
	if err != nil {
		t.Fatalf("cannot load workspace for user B: %v", err)
	}
	if workspaceForB.HelperTokenEnc != "ciphertext-"+testUserB {
		t.Errorf("expected user B ciphertext, got %q", workspaceForB.HelperTokenEnc)
	}
}
