package application

import (
	"context"
	"strings"
	"testing"

	jiraDomain "lunar/backend/internal/jira/domain"
)

const summaryOnlyIssueSummary = "Preparation & Regression Deployment | Report Menu - Implementasi Summary"

const summaryOnlyModelAnswer = `{"category":"Report","category_choice":"observed","core":"Menu-Ikhtisar","evidence":"Preparation & Regression Deployment | Report Menu - Implementasi Summary"}`

const descriptionOnlyModelAnswer = `{"category":"Enhancement","category_choice":"observed","core":"Input-MDR-Program","evidence":"Pilihan Tipe MDR dan Input Form Data Program"}`

func summaryOnlyIssue() *jiraDomain.JiraIssue {
	return &jiraDomain.JiraIssue{
		Key:    "CRMMS-77759",
		Fields: jiraDomain.JiraIssueFields{Summary: summaryOnlyIssueSummary},
	}
}

func TestSuggestNamesFromTheSummaryWhenTheDescriptionIsEmpty(t *testing.T) {
	issues := &stubBranchNameIssues{issue: summaryOnlyIssue()}
	model := &stubBranchNameModel{answers: []string{summaryOnlyModelAnswer}}
	service := NewBranchNameService(issues, model, observedVocabulary())

	suggestion, err := service.Suggest(context.Background(), "user-a", "CRMMS-77759")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if suggestion.Source != jiraDomain.BranchNameSourceModel {
		t.Fatalf("source = %q, want the model", suggestion.Source)
	}
	if suggestion.EvidenceSource != EvidenceSourceSummary {
		t.Fatalf("evidence source = %q, want the summary", suggestion.EvidenceSource)
	}
	if suggestion.Evidence != summaryOnlyIssueSummary {
		t.Fatalf("evidence = %q", suggestion.Evidence)
	}
	if suggestion.Branch != "feature/CRMMS-77759-Report-Menu-Ikhtisar" {
		t.Fatalf("branch = %q", suggestion.Branch)
	}
	if !strings.Contains(model.prompts[0], summaryOnlyIssueSummary) {
		t.Fatalf("the prompt must carry the summary, got %q", model.prompts[0])
	}
}

func TestSuggestNamesFromTheDescriptionWithoutAcceptanceCriteriaHeadings(t *testing.T) {
	issues := &stubBranchNameIssues{issue: &jiraDomain.JiraIssue{
		Key: "CRMMS-77739",
		Fields: jiraDomain.JiraIssueFields{
			Summary:     "Sebagai Maker & Approval di New MMS, I want to mengajukan program perubahan MDR secara bulk",
			Description: "*Deskripsi Acceptance Criteria :*\n\n*Pilihan Tipe MDR dan Input Form Data Program*\n\nThe form offers the new type.",
		},
	}}
	model := &stubBranchNameModel{answers: []string{descriptionOnlyModelAnswer}}
	service := NewBranchNameService(issues, model, observedVocabulary())

	suggestion, err := service.Suggest(context.Background(), "user-a", "CRMMS-77739")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if suggestion.Source != jiraDomain.BranchNameSourceModel {
		t.Fatalf("source = %q, want the model", suggestion.Source)
	}
	if suggestion.EvidenceSource != EvidenceSourceDescription {
		t.Fatalf("evidence source = %q, want the description", suggestion.EvidenceSource)
	}
	if suggestion.Branch != "feature/CRMMS-77739-Enhancement-Input-MDR-Program" {
		t.Fatalf("branch = %q", suggestion.Branch)
	}
}

func TestSuggestRejectsSummaryEvidenceThatIsNotInTheIssue(t *testing.T) {
	issues := &stubBranchNameIssues{issue: summaryOnlyIssue()}
	model := &stubBranchNameModel{answers: []string{`{"category":"Report","category_choice":"observed","core":"Menu-Ikhtisar","evidence":"Totally Invented Phrase"}`}}
	service := NewBranchNameService(issues, model, observedVocabulary())

	suggestion, err := service.Suggest(context.Background(), "user-a", "CRMMS-77759")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if suggestion.Source != jiraDomain.BranchNameSourceFallback {
		t.Fatalf("evidence outside the issue is malformed, source = %q", suggestion.Source)
	}
	if suggestion.EvidenceSource != EvidenceSourceSummary {
		t.Fatalf("the fallback names from the summary, evidence source = %q", suggestion.EvidenceSource)
	}
}

func TestEvidenceSourceLocatesThePhraseInTheIssue(t *testing.T) {
	issue := &jiraDomain.JiraIssue{
		Key: "CRMMS-77911",
		Fields: jiraDomain.JiraIssueFields{
			Summary:     "Sebagai Administrator, I want to melihat daftar user",
			Description: acceptanceDescription,
		},
	}

	tests := []struct {
		name     string
		evidence string
		want     string
	}{
		{name: "acceptance criteria section", evidence: "Halaman Utama Manage User (Grouping by PN)", want: EvidenceSourceAcceptanceCriteria},
		{name: "summary", evidence: "melihat daftar user", want: EvidenceSourceSummary},
		{name: "description outside the criteria", evidence: "Internal notes that are not acceptance criteria.", want: EvidenceSourceDescription},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			got, err := evidenceSource(issue, testCase.evidence)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != testCase.want {
				t.Fatalf("evidence source = %q, want %q", got, testCase.want)
			}
		})
	}

	if _, err := evidenceSource(issue, "a phrase that is nowhere"); err == nil {
		t.Fatal("evidence outside the issue must be rejected")
	}
}
