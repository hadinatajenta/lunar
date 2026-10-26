package transport

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	confluenceDomain "lunar/backend/internal/confluence/domain"
	"lunar/backend/internal/queryreview/domain"
	sharedAuth "lunar/backend/internal/shared/auth"
)

const migTargetsStorage = `<table><tbody>
<tr><th>Repository</th><th>Branch</th><th>Compare</th></tr>
<tr><td>micro-payment</td><td>feat/CRMMS-1</td><td>https://bitbucket.example.com/projects/NME/repos/micro-payment/compare/diff?sourceBranch=feat%2FCRMMS-1&amp;targetBranch=development</td></tr>
<tr><td>fe-mms</td><td>feat/CRMMS-1</td><td>https://bitbucket.example.com/projects/NME/repos/fe-mms/compare/diff?sourceBranch=feat%2FCRMMS-1&amp;targetBranch=development</td></tr>
<tr><td>mms</td><td>feat/CRMMS-1</td><td>https://bitbucket.example.com/projects/NME/repos/mms/compare/diff?sourceBranch=feat%2FCRMMS-1&amp;targetBranch=development</td></tr>
</tbody></table>`

func performMigTargets(t *testing.T, handler *QueryReviewHandler, userID string) *httptest.ResponseRecorder {
	t.Helper()

	request := httptest.NewRequest(http.MethodGet, "/api/confluence/documents/123/mig-targets", nil)
	request.SetPathValue("id", "123")
	if userID != "" {
		ctx := sharedAuth.ContextWithUser(request.Context(), &sharedAuth.UserContext{UserID: userID, Email: "developer@example.com"})
		request = request.WithContext(ctx)
	}

	recorder := httptest.NewRecorder()
	handler.MigTargets(recorder, request)
	return recorder
}

func TestMigTargetsReturnsTheMappedTargetsAndSkipsExcludedRepositories(t *testing.T) {
	fixture := newHandlerFixture()
	fixture.documents.storage = migTargetsStorage

	recorder := performMigTargets(t, fixture.handler, "usr-1")
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", recorder.Code, recorder.Body.String())
	}

	var payload struct {
		MigPageTitle        string                     `json:"mig_page_title"`
		Targets             []domain.MigTarget         `json:"targets"`
		SkippedRepositories []domain.SkippedRepository `json:"skipped_repositories"`
	}
	if err := json.NewDecoder(recorder.Body).Decode(&payload); err != nil {
		t.Fatalf("cannot decode the response: %v", err)
	}
	if payload.MigPageTitle != "CRMMS-1-MIG" {
		t.Fatalf("expected the MIG page title, got %q", payload.MigPageTitle)
	}
	if len(payload.Targets) != 1 || payload.Targets[0].Repository != "micro-payment" || payload.Targets[0].Branch != "feat/CRMMS-1" {
		t.Fatalf("expected the single mapped target, got %+v", payload.Targets)
	}
	if len(payload.SkippedRepositories) != 2 {
		t.Fatalf("expected the two excluded repositories, got %+v", payload.SkippedRepositories)
	}
	for _, skipped := range payload.SkippedRepositories {
		if skipped.Repository != "fe-mms" && skipped.Repository != "mms" {
			t.Fatalf("unexpected skipped repository %+v", skipped)
		}
		if skipped.Reason != "excluded repository" {
			t.Fatalf("expected the exclusion reason, got %q", skipped.Reason)
		}
	}
}

func TestMigTargetsPrefersTheCompareURLSlugOverTheCellText(t *testing.T) {
	fixture := newHandlerFixture()
	fixture.documents.storage = `<table><tbody>
<tr><th>Repository</th><th>Branch</th><th>Compare</th></tr>
<tr><td>Everest Service</td><td>feat/CRMMS-2</td><td>https://bitbucket.example.com/projects/NME/repos/everest/compare/diff?sourceBranch=feat%2FCRMMS-2&amp;targetBranch=release%2Fdevelopment</td></tr>
</tbody></table>`

	recorder := performMigTargets(t, fixture.handler, "usr-1")
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", recorder.Code, recorder.Body.String())
	}

	var payload struct {
		Targets []domain.MigTarget `json:"targets"`
	}
	if err := json.NewDecoder(recorder.Body).Decode(&payload); err != nil {
		t.Fatalf("cannot decode the response: %v", err)
	}
	if len(payload.Targets) != 1 {
		t.Fatalf("expected one target, got %+v", payload.Targets)
	}
	if payload.Targets[0].Repository != "everest" {
		t.Fatalf("expected the URL slug everest, got %q", payload.Targets[0].Repository)
	}
	if payload.Targets[0].BaseRef != "release/development" {
		t.Fatalf("expected the target branch as base ref, got %q", payload.Targets[0].BaseRef)
	}
}

func TestMigTargetsReturns422WhenThereIsNoMigPage(t *testing.T) {
	fixture := newHandlerFixture()
	fixture.documents.children = []confluenceDomain.ConfluencePageRef{{ID: "200", Title: "Design"}}

	recorder := performMigTargets(t, fixture.handler, "usr-1")
	if recorder.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d: %s", recorder.Code, recorder.Body.String())
	}
	payload := decodeHandlerError(t, recorder)
	if payload.Code != domain.ErrorCodeNoCompareLink {
		t.Fatalf("expected the %s code, got %q", domain.ErrorCodeNoCompareLink, payload.Code)
	}
}

func TestMigTargetsRequiresAnAuthenticatedUser(t *testing.T) {
	fixture := newHandlerFixture()

	recorder := performMigTargets(t, fixture.handler, "")
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", recorder.Code, recorder.Body.String())
	}
}
