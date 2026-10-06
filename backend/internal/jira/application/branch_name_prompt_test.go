package application

import (
	"strings"
	"testing"

	jiraDomain "lunar/backend/internal/jira/domain"
)

const acceptanceDescription = "h3. Deskripsi Acceptance Criteria (AC)\n\n" +
	"*1. Halaman Utama Manage User (Grouping by PN)*\n" +
	"The page lists every user grouped by personal number.\n\n" +
	"*2. Riwayat Perubahan*\n" +
	"The page shows the change history.\n\n" +
	"h3. Catatan Teknis\n" +
	"Internal notes that are not acceptance criteria."

func TestAcceptanceCriteriaSectionStopsAtTheNextHeading(t *testing.T) {
	section := acceptanceCriteriaSection(acceptanceDescription)
	if !strings.Contains(section, "Halaman Utama Manage User (Grouping by PN)") {
		t.Fatalf("the section must hold the acceptance criteria, got %q", section)
	}
	if strings.Contains(section, "Internal notes") || strings.Contains(section, "Catatan Teknis") {
		t.Fatalf("the section must stop at the next heading, got %q", section)
	}
}

func TestNumberedCriteriaHeadingsDropTheNumbering(t *testing.T) {
	headings := numberedCriteriaHeadings(acceptanceCriteriaSection(acceptanceDescription))
	want := []string{"Halaman Utama Manage User (Grouping by PN)", "Riwayat Perubahan"}
	if len(headings) != len(want) {
		t.Fatalf("headings = %v, want %v", headings, want)
	}
	for index := range want {
		if headings[index] != want[index] {
			t.Fatalf("headings = %v, want %v", headings, want)
		}
	}
}

func TestAcceptanceCriteriaSectionIsEmptyWithoutTheHeading(t *testing.T) {
	if section := acceptanceCriteriaSection("just a description"); section != "" {
		t.Fatalf("section = %q, want an empty section", section)
	}
	if headings := numberedCriteriaHeadings(""); len(headings) != 0 {
		t.Fatalf("headings = %v, want none", headings)
	}
}

func TestBranchNamePromptCarriesTheIssueAndTheCriteria(t *testing.T) {
	issue := &jiraDomain.JiraIssue{
		Key: "CRMMS-77911",
		Fields: jiraDomain.JiraIssueFields{
			Summary:     "Sebagai Administrator, I want to melihat daftar user",
			Description: acceptanceDescription,
		},
	}

	prompt := BranchNamePrompt(issue)
	for _, wanted := range []string{
		"CRMMS-77911",
		"Sebagai Administrator, I want to melihat daftar user",
		"Halaman Utama Manage User (Grouping by PN)",
		dataOpenMarker,
		dataCloseMarker,
	} {
		if !strings.Contains(prompt, wanted) {
			t.Fatalf("the prompt must carry %q, got %q", wanted, prompt)
		}
	}
}

func TestBranchNamePromptIsBoundedForALongDescription(t *testing.T) {
	description := "h3. Deskripsi Acceptance Criteria (AC)\n\n" + strings.Repeat("panjang ", 4000)
	prompt := BranchNamePrompt(&jiraDomain.JiraIssue{
		Key:    "CRMMS-77911",
		Fields: jiraDomain.JiraIssueFields{Description: description},
	})

	if len([]rune(prompt)) > maximumDescriptionRunes+maximumAcceptanceCriteriaRunes+1000 {
		t.Fatalf("the prompt grew to %d runes", len([]rune(prompt)))
	}
}

func TestParseBranchNameResponseNormalisesTheSegments(t *testing.T) {
	category, core, evidence, err := ParseBranchNameResponse(`{"category":"enhancement","core":"manage user","evidence":"Halaman Utama Manage User (Grouping by PN)"}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if category != "Enhancement" || core != "Manage-User" {
		t.Fatalf("category = %q, core = %q", category, core)
	}
	if evidence != "Halaman Utama Manage User (Grouping by PN)" {
		t.Fatalf("evidence = %q", evidence)
	}
}

func TestParseBranchNameResponseRejectsMalformedAnswers(t *testing.T) {
	tests := []struct {
		name   string
		answer string
	}{
		{name: "not json", answer: "Manage-User"},
		{name: "missing fields", answer: `{"category":"Enhancement"}`},
		{name: "refusal", answer: `{"category":"","core":"","evidence":""}`},
		{name: "empty evidence", answer: `{"category":"Enhancement","core":"Manage-User","evidence":"  "}`},
		{name: "category too long", answer: `{"category":"Enhancement Refactor Cleanup","core":"Manage-User","evidence":"Halaman"}`},
		{name: "core too long", answer: `{"category":"Enhancement","core":"Manage User Grouping Detail PN List","evidence":"Halaman"}`},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			if _, _, _, err := ParseBranchNameResponse(testCase.answer); err == nil {
				t.Fatalf("expected an error for %q", testCase.answer)
			}
		})
	}
}

func TestContainsEvidenceIgnoresCaseAndWhitespace(t *testing.T) {
	text := "h3. Deskripsi Acceptance Criteria (AC)\n\n*1. Halaman Utama Manage User\n(Grouping by PN)*"
	if !containsEvidence(text, "Halaman Utama Manage User (Grouping by PN)") {
		t.Fatal("the heading wrapped over two lines must still count as evidence")
	}
	if !containsEvidence(text, "halaman utama manage user") {
		t.Fatal("the evidence check must ignore case")
	}
	if containsEvidence(text, "Riwayat Perubahan") {
		t.Fatal("an absent phrase must not count as evidence")
	}
	if containsEvidence(text, "  ") {
		t.Fatal("a blank phrase must not count as evidence")
	}
}
