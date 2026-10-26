package application

import (
	"context"
	"fmt"
	"strings"
	"testing"

	confluenceDomain "lunar/backend/internal/confluence/domain"
	queryReviewDomain "lunar/backend/internal/queryreview/domain"
	sharedErrors "lunar/backend/internal/shared/errors"
	"lunar/backend/internal/sop/domain"
)

type createdPageRecord struct {
	parentID string
	title    string
	storage  string
}

type updateRecord struct {
	pageID  string
	storage string
}

type stubGateway struct {
	documents      map[string]*confluenceDomain.ConfluenceDocument
	ancestors      map[string][]confluenceDomain.ConfluencePageRef
	children       map[string][]confluenceDomain.ConfluencePageRef
	storage        map[string]string
	versions       map[string]int
	created        []createdPageRecord
	updates        []updateRecord
	uploads        []string
	updateFailures int
	updateCalls    int
	nextPage       int
}

func newStubGateway() *stubGateway {
	return &stubGateway{
		documents: map[string]*confluenceDomain.ConfluenceDocument{},
		ancestors: map[string][]confluenceDomain.ConfluencePageRef{},
		children:  map[string][]confluenceDomain.ConfluencePageRef{},
		storage:   map[string]string{},
		versions:  map[string]int{},
	}
}

func (g *stubGateway) GetDocumentDetail(_ context.Context, _ string, documentID string) (*confluenceDomain.ConfluenceDocument, error) {
	return g.documents[documentID], nil
}

func (g *stubGateway) GetPageAncestors(_ context.Context, _ string, pageID string) ([]confluenceDomain.ConfluencePageRef, error) {
	return g.ancestors[pageID], nil
}

func (g *stubGateway) GetChildPages(_ context.Context, _ string, pageID string) ([]confluenceDomain.ConfluencePageRef, error) {
	return g.children[pageID], nil
}

func (g *stubGateway) GetPageStorage(_ context.Context, _ string, pageID string) (string, error) {
	return g.storage[pageID], nil
}

func (g *stubGateway) UpdatePageStorage(_ context.Context, _ string, pageID string, storage string) (int, string, error) {
	g.updateCalls++
	if g.updateFailures > 0 {
		g.updateFailures--
		return 0, "", fmt.Errorf("%w: page changed", sharedErrors.ErrConflict)
	}
	g.storage[pageID] = storage
	g.versions[pageID]++
	g.updates = append(g.updates, updateRecord{pageID: pageID, storage: storage})
	return g.versions[pageID], "https://wiki.example.com/pages/" + pageID, nil
}

func (g *stubGateway) UploadAttachment(_ context.Context, _ string, _ string, filename string, _ []byte, _ string) (string, error) {
	g.uploads = append(g.uploads, filename)
	return "attachment-" + filename, nil
}

func (g *stubGateway) CreatePage(_ context.Context, _ string, parentPageID string, title string, storage string) (string, int, string, error) {
	g.nextPage++
	pageID := fmt.Sprintf("page-%d", g.nextPage)
	g.documents[pageID] = &confluenceDomain.ConfluenceDocument{ID: pageID, Title: title, URL: "https://wiki.example.com/pages/" + pageID}
	g.storage[pageID] = storage
	g.versions[pageID] = 1
	g.created = append(g.created, createdPageRecord{parentID: parentPageID, title: title, storage: storage})
	g.children[parentPageID] = append(g.children[parentPageID], confluenceDomain.ConfluencePageRef{ID: pageID, Title: title})
	return pageID, 1, g.documents[pageID].URL, nil
}

func (g *stubGateway) GetPageVersion(_ context.Context, _ string, pageID string) (int, error) {
	return g.versions[pageID], nil
}

type stubMigReader struct {
	repositories []queryReviewDomain.MigRepository
	err          error
}

func (s *stubMigReader) ReadMigRepositories(_ context.Context, _ string, _ string) ([]queryReviewDomain.MigRepository, error) {
	return s.repositories, s.err
}

type stubHashStore struct {
	values map[string]string
}

func newStubHashStore() *stubHashStore {
	return &stubHashStore{values: map[string]string{}}
}

func regionKey(pageID string, region string) string {
	return pageID + "\x1f" + region
}

func (s *stubHashStore) Get(_ context.Context, pageID string, region string) (string, bool, error) {
	value, exists := s.values[regionKey(pageID, region)]
	return value, exists, nil
}

func (s *stubHashStore) Set(_ context.Context, pageID string, region string, hash string) error {
	s.values[regionKey(pageID, region)] = hash
	return nil
}

func newServiceTestFixture() (*stubGateway, *stubMigReader, *stubHashStore, *Service) {
	gateway := newStubGateway()
	gateway.documents["999"] = &confluenceDomain.ConfluenceDocument{ID: "999", Title: "CRMMS-78064 Query Review", URL: "https://wiki.example.com/pages/999"}
	gateway.ancestors["999"] = []confluenceDomain.ConfluencePageRef{{ID: "100", Title: "CRMMS-78064-Development"}}

	mig := &stubMigReader{repositories: []queryReviewDomain.MigRepository{
		{Repository: "everest", Branch: "feature/CRMMS-77914", TargetBranch: "development", CompareURL: "https://bitbucket/compare"},
	}}
	hashes := newStubHashStore()

	service := NewService(gateway, mig, hashes, map[string]string{}, map[string]domain.DatabaseTarget{})
	return gateway, mig, hashes, service
}

func deployBlockWithStatement(sql string) DeployBlock {
	scan := scanWithCreateStatement()
	scan.DatabaseFiles[0].Statements[0].SQL = sql
	return BuildDeployBlock(DeployBlockInput{
		IssueKey:    "CRMMS-1",
		Scans:       []domain.RepositoryScan{scan},
		BambooPlans: map[string]string{"everest": "https://bamboo/plan/EVEREST"},
		Databases:   map[string]domain.DatabaseTarget{"log_program_mdr": {IP: "10.1.2.3", Database: "mms"}},
	})
}

func TestGenerateResolvesTheIssueKeyAndThePlan(t *testing.T) {
	_, _, _, service := newServiceTestFixture()

	result, err := service.Generate(context.Background(), "usr-1", "999", domain.GenerateRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.IssueKey != "CRMMS-78064" || result.DevelopmentPageID != "100" {
		t.Fatalf("unexpected resolution %+v", result)
	}
	if len(result.Repositories) != 1 || result.Repositories[0].Repository != "everest" {
		t.Fatalf("expected the MIG repository, got %+v", result.Repositories)
	}
	if result.Deploy.Title != "CRMMS-78064-SOP Deploy" || result.Deploy.Exists {
		t.Fatalf("unexpected deploy plan %+v", result.Deploy)
	}
	for _, expected := range []string{domain.DeployDatabaseMarker.Start, babIIApplicationHeading, babIIIRollbackHeading, domain.DeployRollbackMarker.End, noDatabaseChangeMessage} {
		if !strings.Contains(result.Deploy.ManagedBlock, expected) {
			t.Fatalf("expected the managed block to contain %q, got %s", expected, result.Deploy.ManagedBlock)
		}
	}
	if result.Operasional.Title != "CRMMS-78064-SOP Operasional" || result.Operasional.Exists {
		t.Fatalf("unexpected operasional plan %+v", result.Operasional)
	}
	if len(result.Operasional.Children) != 3 {
		t.Fatalf("expected three operasional children, got %+v", result.Operasional.Children)
	}
}

func TestApplyCreatesTheDeployPageAndTheOperasionalChildren(t *testing.T) {
	gateway, _, hashes, service := newServiceTestFixture()
	block := deployBlockWithStatement("CREATE TABLE log_program_mdr (id BIGINT)")

	result, err := service.Apply(context.Background(), "usr-1", "999", domain.ApplyRequest{
		Deploy:      domain.ApplyDeployRequest{Mode: domain.DeployModeCreate, ManagedBlock: block.FullBody()},
		Operasional: domain.ApplyOperasionalRequest{CreateChildren: true},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.DeployPage.Action != domain.DeployActionCreated || result.DeployPage.ID != "page-1" {
		t.Fatalf("unexpected deploy result %+v", result.DeployPage)
	}
	if result.OperasionalPage.ID != "page-2" {
		t.Fatalf("unexpected operasional result %+v", result.OperasionalPage)
	}
	if strings.Join(result.CreatedChildren, ",") != "Maintenance,Monitoring,Troubleshooting" {
		t.Fatalf("unexpected created children %+v", result.CreatedChildren)
	}
	if len(gateway.created) != 5 {
		t.Fatalf("expected the deploy page, the parent, and three children, got %+v", gateway.created)
	}
	if gateway.created[0].parentID != "100" || gateway.created[0].storage != block.FullBody() {
		t.Fatalf("expected the full deploy body under the development page, got %+v", gateway.created[0])
	}
	if hashes.values[regionKey("page-1", domain.RegionDatabase)] != block.DatabaseHash {
		t.Fatalf("expected the database region hash to be stored, got %+v", hashes.values)
	}
	if hashes.values[regionKey("page-1", domain.RegionRollback)] != block.RollbackHash {
		t.Fatalf("expected the rollback region hash to be stored, got %+v", hashes.values)
	}
	if len(gateway.uploads) != 0 {
		t.Fatalf("expected no attachment upload, got %+v", gateway.uploads)
	}
}

func TestApplyPrependsTheBlockWhenThePageHasNoMarkers(t *testing.T) {
	gateway, _, hashes, service := newServiceTestFixture()
	gateway.children["100"] = []confluenceDomain.ConfluencePageRef{{ID: "200", Title: "CRMMS-78064-SOP Deploy"}}
	gateway.storage["200"] = "<p>Screenshot manual</p>"
	gateway.versions["200"] = 3
	block := deployBlockWithStatement("CREATE TABLE log_program_mdr (id BIGINT)")

	result, err := service.Apply(context.Background(), "usr-1", "999", domain.ApplyRequest{
		Deploy: domain.ApplyDeployRequest{Mode: domain.DeployModeUpdate, ManagedBlock: block.FullBody()},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.DeployPage.Action != domain.DeployActionUpdated || result.DeployPage.Version != 4 {
		t.Fatalf("unexpected deploy result %+v", result.DeployPage)
	}
	updated := gateway.storage["200"]
	if !strings.HasPrefix(updated, block.FullBody()) {
		t.Fatalf("expected the full body to be prepended, got %s", updated)
	}
	if !strings.Contains(updated, "<p>Screenshot manual</p>") {
		t.Fatalf("expected the manual content to be kept, got %s", updated)
	}
	if hashes.values[regionKey("200", domain.RegionDatabase)] != block.DatabaseHash || hashes.values[regionKey("200", domain.RegionRollback)] != block.RollbackHash {
		t.Fatalf("expected both region hashes to be stored, got %+v", hashes.values)
	}
	if len(gateway.uploads) != 0 {
		t.Fatalf("expected no attachment upload, got %+v", gateway.uploads)
	}
}

func TestApplyReplacesOnlyTheChangedRegion(t *testing.T) {
	gateway, _, hashes, service := newServiceTestFixture()
	gateway.children["100"] = []confluenceDomain.ConfluencePageRef{{ID: "200", Title: "CRMMS-78064-SOP Deploy"}}

	original := deployBlockWithStatement("CREATE TABLE log_program_mdr (id BIGINT)")
	updated := deployBlockWithStatement("CREATE TABLE log_program_mdr (id BIGINT, name TEXT)")
	gateway.storage["200"] = original.FullBody() + "\n<p>Catatan manual</p>"
	gateway.versions["200"] = 1
	hashes.values[regionKey("200", domain.RegionDatabase)] = original.DatabaseHash
	hashes.values[regionKey("200", domain.RegionRollback)] = original.RollbackHash

	result, err := service.Apply(context.Background(), "usr-1", "999", domain.ApplyRequest{
		Deploy: domain.ApplyDeployRequest{ManagedBlock: updated.FullBody()},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.DeployPage.Action != domain.DeployActionUpdated || result.DeployPage.Version != 2 {
		t.Fatalf("unexpected deploy result %+v", result.DeployPage)
	}
	stored := gateway.storage["200"]
	if !strings.Contains(stored, updated.DatabaseRegion) {
		t.Fatalf("expected the database region to be updated, got %s", stored)
	}
	if strings.Contains(stored, original.DatabaseRegion) {
		t.Fatalf("expected the old database region to be gone, got %s", stored)
	}
	if !strings.Contains(stored, original.RollbackRegion) {
		t.Fatalf("expected the rollback region to be preserved, got %s", stored)
	}
	if !strings.Contains(stored, original.Middle) {
		t.Fatalf("expected BAB II and BAB III to be preserved, got %s", stored)
	}
	if !strings.Contains(stored, "<p>Catatan manual</p>") {
		t.Fatalf("expected unrelated content to be preserved, got %s", stored)
	}
	if hashes.values[regionKey("200", domain.RegionDatabase)] != updated.DatabaseHash {
		t.Fatalf("expected the database hash to be updated, got %+v", hashes.values)
	}
}

func TestApplySkipsWhenBothRegionsAreUnchanged(t *testing.T) {
	gateway, _, hashes, service := newServiceTestFixture()
	gateway.children["100"] = []confluenceDomain.ConfluencePageRef{{ID: "200", Title: "CRMMS-78064-SOP Deploy"}}
	block := deployBlockWithStatement("CREATE TABLE log_program_mdr (id BIGINT)")
	gateway.storage["200"] = block.FullBody()
	gateway.versions["200"] = 5
	hashes.values[regionKey("200", domain.RegionDatabase)] = block.DatabaseHash
	hashes.values[regionKey("200", domain.RegionRollback)] = block.RollbackHash

	result, err := service.Apply(context.Background(), "usr-1", "999", domain.ApplyRequest{
		Deploy: domain.ApplyDeployRequest{ManagedBlock: block.FullBody()},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.DeployPage.Action != domain.DeployActionAlreadyUpToDate {
		t.Fatalf("expected an idempotent skip, got %+v", result.DeployPage)
	}
	if gateway.updateCalls != 0 {
		t.Fatalf("expected no write, got %d update calls", gateway.updateCalls)
	}
	if result.DeployPage.Version != 5 {
		t.Fatalf("expected the current version, got %d", result.DeployPage.Version)
	}
}

func TestApplyRetriesOnceOnAConflict(t *testing.T) {
	gateway, _, hashes, service := newServiceTestFixture()
	gateway.children["100"] = []confluenceDomain.ConfluencePageRef{{ID: "200", Title: "CRMMS-78064-SOP Deploy"}}
	original := deployBlockWithStatement("CREATE TABLE log_program_mdr (id BIGINT)")
	updated := deployBlockWithStatement("CREATE TABLE log_program_mdr (id BIGINT, name TEXT)")
	gateway.storage["200"] = original.FullBody()
	gateway.versions["200"] = 2
	hashes.values[regionKey("200", domain.RegionDatabase)] = original.DatabaseHash
	hashes.values[regionKey("200", domain.RegionRollback)] = original.RollbackHash
	gateway.updateFailures = 1

	result, err := service.Apply(context.Background(), "usr-1", "999", domain.ApplyRequest{
		Deploy: domain.ApplyDeployRequest{ManagedBlock: updated.FullBody()},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.DeployPage.Action != domain.DeployActionUpdated || result.DeployPage.Version != 3 {
		t.Fatalf("unexpected deploy result %+v", result.DeployPage)
	}
	if gateway.updateCalls != 2 {
		t.Fatalf("expected two update attempts, got %d", gateway.updateCalls)
	}
}

func TestApplyFailsWhenTheConflictPersists(t *testing.T) {
	gateway, _, hashes, service := newServiceTestFixture()
	gateway.children["100"] = []confluenceDomain.ConfluencePageRef{{ID: "200", Title: "CRMMS-78064-SOP Deploy"}}
	original := deployBlockWithStatement("CREATE TABLE log_program_mdr (id BIGINT)")
	updated := deployBlockWithStatement("CREATE TABLE log_program_mdr (id BIGINT, name TEXT)")
	gateway.storage["200"] = original.FullBody()
	gateway.versions["200"] = 2
	hashes.values[regionKey("200", domain.RegionDatabase)] = original.DatabaseHash
	hashes.values[regionKey("200", domain.RegionRollback)] = original.RollbackHash
	gateway.updateFailures = 2

	_, err := service.Apply(context.Background(), "usr-1", "999", domain.ApplyRequest{
		Deploy: domain.ApplyDeployRequest{ManagedBlock: updated.FullBody()},
	})
	if err == nil {
		t.Fatal("expected an error when the conflict persists")
	}
	if !strings.Contains(err.Error(), "version conflict") {
		t.Fatalf("expected a safe conflict message, got %v", err)
	}
}

func TestApplySkipsExistingOperasionalChildren(t *testing.T) {
	gateway, _, _, service := newServiceTestFixture()
	gateway.children["100"] = []confluenceDomain.ConfluencePageRef{{ID: "300", Title: "CRMMS-78064-SOP Operasional"}}
	gateway.children["300"] = []confluenceDomain.ConfluencePageRef{{ID: "301", Title: "Maintenance"}}
	gateway.storage["300"] = BuildOperasionalParentBlock()
	gateway.versions["300"] = 4

	result, err := service.Apply(context.Background(), "usr-1", "999", domain.ApplyRequest{
		Operasional: domain.ApplyOperasionalRequest{CreateChildren: true},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if strings.Join(result.CreatedChildren, ",") != "Monitoring,Troubleshooting" {
		t.Fatalf("expected only the missing children, got %+v", result.CreatedChildren)
	}
	if result.OperasionalPage.ID != "300" || result.OperasionalPage.Version != 4 {
		t.Fatalf("unexpected operasional result %+v", result.OperasionalPage)
	}
}

func TestGenerateRejectsAnEmptyUser(t *testing.T) {
	_, _, _, service := newServiceTestFixture()

	if _, err := service.Generate(context.Background(), "", "999", domain.GenerateRequest{}); err == nil {
		t.Fatal("expected an error for an empty user")
	}
}
