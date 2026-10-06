package application

import (
	"context"
	"errors"
	"strings"
	"testing"

	"lunar/backend/internal/workspace/domain"
)

func TestCatalogueMergesLocalAndRemoteRepositoriesWithCounts(t *testing.T) {
	catalogue, err := testCatalogueService(
		stubCatalogueLocal{repositories: testLocalRepositories("mms-api", "mocash_api")},
		stubCatalogueRemote{repositories: testRemoteRepositories("NME", "mms-api", "atlas")},
	).Catalogue(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("catalogue failed: %v", err)
	}

	if catalogue.LocalRepositoryCount != 2 || catalogue.RemoteRepositoryCount != 2 {
		t.Fatalf("unexpected counts: %+v", catalogue)
	}
	if catalogue.RemoteUnavailable {
		t.Fatal("a successful remote listing must not be reported as unavailable")
	}

	expected := []CataloguedRepository{
		{Repo: "atlas", Project: "NME", IsCloned: false},
		{Repo: "mms-api", Project: "NME", IsCloned: true},
		{Repo: "mocash_api", Project: "", IsCloned: true},
	}
	if len(catalogue.Repositories) != len(expected) {
		t.Fatalf("expected %d repositories, got %+v", len(expected), catalogue.Repositories)
	}
	for index, entry := range expected {
		if catalogue.Repositories[index] != entry {
			t.Fatalf("expected %+v at %d, got %+v", entry, index, catalogue.Repositories[index])
		}
	}
}

func TestCatalogueSearchFiltersByNameAndKeepsTheTotals(t *testing.T) {
	service := testCatalogueService(
		stubCatalogueLocal{repositories: testLocalRepositories("mms-api")},
		stubCatalogueRemote{repositories: testRemoteRepositories("NME", "mms-api", "atlas")},
	)

	searched, err := service.Search(context.Background(), "user-1", "ATLAS")
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}
	if len(searched.Repositories) != 1 || searched.Repositories[0].Repo != "atlas" {
		t.Fatalf("expected the search to filter, got %+v", searched.Repositories)
	}
	if searched.LocalRepositoryCount != 1 || searched.RemoteRepositoryCount != 2 {
		t.Fatalf("the counts state the whole catalogue, not the filtered page: %+v", searched)
	}

	empty, err := service.Search(context.Background(), "user-1", "nothing-matches-this")
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}
	if len(empty.Repositories) != 0 {
		t.Fatalf("expected no match, got %+v", empty.Repositories)
	}
	if empty.LocalRepositoryCount != 1 || empty.RemoteRepositoryCount != 2 {
		t.Fatalf("an empty search still reports what exists: %+v", empty)
	}
}

func TestCatalogueDegradesToTheLocalTierWhenBitbucketIsUnavailable(t *testing.T) {
	catalogue, err := testCatalogueService(
		stubCatalogueLocal{repositories: testLocalRepositories("mms-api")},
		stubCatalogueRemote{err: errors.New("bitbucket is unreachable")},
	).Catalogue(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("a remote failure must not fail the catalogue: %v", err)
	}

	if !catalogue.RemoteUnavailable {
		t.Fatal("an unreachable Bitbucket must be reported as unavailable")
	}
	if catalogue.RemoteRepositoryCount != 0 || catalogue.LocalRepositoryCount != 1 {
		t.Fatalf("unexpected counts: %+v", catalogue)
	}
	if len(catalogue.Repositories) != 1 || catalogue.Repositories[0].Repo != "mms-api" {
		t.Fatalf("the local repositories must survive, got %+v", catalogue.Repositories)
	}
}

func TestCatalogueListsRemoteRepositoriesWithoutAWorkspace(t *testing.T) {
	service := NewRepositoryCatalogueService(
		stubSuggestionRoots{err: ErrWorkspaceNotUsable},
		stubCatalogueLocal{repositories: testLocalRepositories("mms-api")},
		stubCatalogueRemote{repositories: testRemoteRepositories("NME", "atlas")},
	)

	catalogue, err := service.Catalogue(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("catalogue failed: %v", err)
	}
	if catalogue.LocalRepositoryCount != 0 {
		t.Fatalf("no workspace means no local repositories, got %+v", catalogue)
	}
	if catalogue.RemoteRepositoryCount != 1 || len(catalogue.Repositories) != 1 {
		t.Fatalf("the remote picker must still work, got %+v", catalogue.Repositories)
	}
}

func TestCatalogueReportsALocalScanFailure(t *testing.T) {
	_, err := testCatalogueService(stubCatalogueLocal{err: errors.New("git is unavailable")}, nil).Catalogue(context.Background(), "user-1")
	if err == nil {
		t.Fatal("expected the local scan failure to be reported")
	}
	if !strings.Contains(err.Error(), "git is unavailable") {
		t.Fatalf("the failure must be wrapped with its cause, got %v", err)
	}
}

func TestCatalogueRequiresALocalReader(t *testing.T) {
	service := NewRepositoryCatalogueService(stubSuggestionRoots{rootPath: "/workspace"}, nil, nil)

	if _, err := service.Catalogue(context.Background(), "user-1"); !errors.Is(err, ErrServerReaderUnavailable) {
		t.Fatalf("expected an unavailable reader, got %v", err)
	}
}

func TestCatalogueReusesTheLocalListingForTheCacheTTL(t *testing.T) {
	local := &countingLocalRepositories{}
	service := testCatalogueService(local, nil)

	if _, err := service.Catalogue(context.Background(), "user-1"); err != nil {
		t.Fatalf("first read failed: %v", err)
	}
	if _, err := service.Catalogue(context.Background(), "user-1"); err != nil {
		t.Fatalf("second read failed: %v", err)
	}
	if local.calls != 1 {
		t.Fatalf("expected the cached listing to serve both reads, got %d", local.calls)
	}
}

type countingLocalRepositories struct {
	calls int
}

func (c *countingLocalRepositories) ListRepositories(_ context.Context, _ string) ([]domain.RepositorySummary, error) {
	c.calls++
	return testLocalRepositories("mms-api"), nil
}
