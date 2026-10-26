package application

import (
	"context"
	"testing"

	confluenceDomain "lunar/backend/internal/confluence/domain"
)

func TestReadMigRepositoriesReturnsEveryRepository(t *testing.T) {
	feCompareURL := "https://bitbucket.example.com/projects/NME/repos/fe-mms/compare/diff?sourceBranch=feat%2FCRMMS-1&targetBranch=development"
	mmsCompareURL := "https://bitbucket.example.com/projects/NME/repos/mms/compare/diff?sourceBranch=feat%2FCRMMS-1&targetBranch=development"
	storage := `<table><tbody>
<tr><th>Repository</th><th>Branch</th><th>Compare</th></tr>
<tr><td>fe-mms</td><td>feat/CRMMS-1</td><td>` + feCompareURL + `</td></tr>
<tr><td>mms</td><td>feat/CRMMS-1</td><td>` + mmsCompareURL + `</td></tr>
<tr><td>micro-payment</td><td>feat/CRMMS-1</td><td>` + microCompareURL + `</td></tr>
</tbody></table>`

	provider := &stubDocumentProvider{
		document:  &confluenceDomain.ConfluenceDocument{ID: "999", Title: "CRMMS-1 Query Review"},
		ancestors: []confluenceDomain.ConfluencePageRef{{ID: "2", Title: "CRMMS-1-Development"}},
		children:  []confluenceDomain.ConfluencePageRef{{ID: "10", Title: "CRMMS-1-MIG"}},
		storage:   storage,
	}
	service := newMigResolverService(provider)

	repositories, err := service.ReadMigRepositories(context.Background(), "usr-1", "999")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(repositories) != 3 {
		t.Fatalf("expected all three repositories including the excluded ones, got %+v", repositories)
	}
	names := []string{repositories[0].Repository, repositories[1].Repository, repositories[2].Repository}
	if names[0] != "fe-mms" || names[1] != "mms" || names[2] != "micro-payment" {
		t.Fatalf("unexpected repository order %+v", names)
	}
	if repositories[0].Branch != "feat/CRMMS-1" || repositories[0].TargetBranch != "development" {
		t.Fatalf("unexpected branches %+v", repositories[0])
	}
	if repositories[0].CompareURL != feCompareURL {
		t.Fatalf("expected the compare url, got %q", repositories[0].CompareURL)
	}
}

func TestReadMigRepositoriesUsesTheCompareURLSlugOverTheCellText(t *testing.T) {
	storage := `<table><tbody>
<tr><th>Repository</th><th>Branch</th><th>Compare</th></tr>
<tr><td>everest https://bitbucket.example.com/projects/NME/repos/everest/browse</td><td>feat/CRMMS-1</td><td>` + microCompareURL + `</td></tr>
</tbody></table>`

	provider := &stubDocumentProvider{
		document:  &confluenceDomain.ConfluenceDocument{ID: "999", Title: "CRMMS-1 Query Review"},
		ancestors: []confluenceDomain.ConfluencePageRef{{ID: "2", Title: "CRMMS-1-Development"}},
		children:  []confluenceDomain.ConfluencePageRef{{ID: "10", Title: "CRMMS-1-MIG"}},
		storage:   storage,
	}
	service := newMigResolverService(provider)

	repositories, err := service.ReadMigRepositories(context.Background(), "usr-1", "999")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(repositories) != 1 || repositories[0].Repository != "micro-payment" {
		t.Fatalf("expected the compare URL slug, got %+v", repositories)
	}
}

func TestReadMigRepositoriesRequiresAnAuthenticatedUser(t *testing.T) {
	service := newMigResolverService(&stubDocumentProvider{})

	if _, err := service.ReadMigRepositories(context.Background(), "", "999"); err == nil {
		t.Fatal("expected an error for an empty user")
	}
}
