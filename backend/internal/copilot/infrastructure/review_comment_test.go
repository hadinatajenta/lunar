package infrastructure

import (
	"strings"
	"testing"

	bitbucketDomain "lunar/backend/internal/bitbucket/domain"
)

func TestComposeReviewCommentGroupsBySeverity(t *testing.T) {
	context := buildReviewContext(reviewFixturePR(), reviewFixtureDiff())
	review := bitbucketDomain.AICodeReview{
		PRID:           "#184",
		Model:          "DeepSeek Flash",
		Summary:        "The rotation path is nearly correct.",
		Recommendation: bitbucketDomain.ReviewRecommendationRequestChanges,
		Findings: []bitbucketDomain.ReviewFinding{
			{
				Severity:     bitbucketDomain.ReviewSeverityBlocker,
				Category:     "security",
				File:         "src/auth/session.ts",
				Line:         11,
				Title:        "Refresh token is not invalidated on rotation",
				Explanation:  "The previous token stays valid after rotation.",
				SuggestedFix: "Call invalidate(refresh) after write(refresh).",
				Confidence:   bitbucketDomain.ReviewConfidenceConfirmed,
				Evidence:     "const next = write(refresh)",
			},
			{
				Severity:   bitbucketDomain.ReviewSeverityNitpick,
				Category:   "readability",
				File:       "src/auth/session.ts",
				Line:       10,
				Title:      "Name the argument after what it holds",
				Confidence: bitbucketDomain.ReviewConfidenceAssumption,
			},
		},
		Assumptions: []string{"No integration test for the rotation endpoint appears in the diff."},
		Strengths:   []string{"The added test file covers the happy path (src/auth/session.test.ts)."},
	}

	comment := composeReviewComment(review, context, "Worth a second pass before merging.")

	requiredFragments := []string{
		"## Code review: feat(auth): rotate refresh tokens",
		"**Reviewed:** 2 files, +4 -1",
		"**Verdict:** 🔴 Request changes",
		"### 🔴 Blocker",
		"### 🟢 Nitpick",
		"`src/auth/session.ts:11`",
		"**Why:** The previous token stays valid after rotation.",
		"**Fix:** Call invalidate(refresh) after write(refresh).",
		"**Evidence:** `const next = write(refresh)`",
		"**Confidence:** assumption, not confirmed from the diff",
		"### ✅ What was done well",
		"### ⚠️ Assumptions",
		"Worth a second pass before merging.",
		"Reviewed with DeepSeek Flash.",
	}
	for _, fragment := range requiredFragments {
		if !strings.Contains(comment, fragment) {
			t.Errorf("expected the composed comment to contain %q, got:\n%s", fragment, comment)
		}
	}

	if strings.Contains(comment, "### 🟡 Suggestion") {
		t.Error("an empty severity group must not be printed")
	}
	if strings.Contains(comment, "—") {
		t.Error("the composed comment must not use em dashes")
	}
}

func TestComposeReviewCommentKeepsACodeExampleOnItsOwnBlock(t *testing.T) {
	context := buildReviewContext(reviewFixturePR(), reviewFixtureDiff())
	review := bitbucketDomain.AICodeReview{
		PRID:           "#184",
		Summary:        "Summary.",
		Recommendation: bitbucketDomain.ReviewRecommendationRequestChanges,
		Findings: []bitbucketDomain.ReviewFinding{
			{
				Severity:     bitbucketDomain.ReviewSeverityBlocker,
				File:         "src/auth/session.ts",
				Line:         11,
				Title:        "Invalidate the previous token",
				SuggestedFix: "```ts\nawait invalidate(refresh)\n```",
			},
		},
	}

	comment := composeReviewComment(review, context, "")

	if !strings.Contains(comment, "**Fix:**\n\n```ts\nawait invalidate(refresh)\n```") {
		t.Errorf("expected the code example to keep its own block, got:\n%s", comment)
	}
	if strings.Contains(comment, "**Fix:** ```") {
		t.Error("a fenced code example must not start on the label line")
	}
}

func TestComposeReviewCommentStatesWhenNothingWasFound(t *testing.T) {
	context := buildReviewContext(reviewFixturePR(), reviewFixtureDiff())
	review := bitbucketDomain.AICodeReview{
		PRID:           "#184",
		Model:          "DeepSeek Flash",
		Summary:        "The change is small and correct.",
		Recommendation: bitbucketDomain.ReviewRecommendationApprove,
		Strengths:      []string{"The token rotation is covered by a new test."},
	}

	comment := composeReviewComment(review, context, "")
	if !strings.Contains(comment, "No issues were found in the reviewed diff.") {
		t.Errorf("expected an explicit empty-findings statement, got:\n%s", comment)
	}
	if !strings.Contains(comment, "**Verdict:** 🟢 Approve") {
		t.Errorf("expected an approve verdict, got:\n%s", comment)
	}
	if strings.Contains(comment, "### ⚠️ Assumptions") {
		t.Error("an empty assumptions section must not be printed")
	}
}

func TestComposeReviewCommentReportsFilesThatWereNotFullyRead(t *testing.T) {
	context := buildReviewContext(reviewFixturePR(), reviewFixtureDiff())
	context.TruncatedFile = []string{"src/large/generated.ts (partially read)"}
	review := bitbucketDomain.AICodeReview{
		PRID:           "#184",
		Summary:        "Summary.",
		Recommendation: bitbucketDomain.ReviewRecommendationApprove,
		Truncated:      true,
	}

	comment := composeReviewComment(review, context, "")
	if !strings.Contains(comment, "**Not fully read:** src/large/generated.ts (partially read)") {
		t.Errorf("expected the unread files to be named, got:\n%s", comment)
	}
}

func TestComposeFallbackSummaryCountsFindings(t *testing.T) {
	context := buildReviewContext(reviewFixturePR(), reviewFixtureDiff())
	findings := []bitbucketDomain.ReviewFinding{
		{Severity: bitbucketDomain.ReviewSeverityBlocker},
		{Severity: bitbucketDomain.ReviewSeverityBlocker},
		{Severity: bitbucketDomain.ReviewSeverityNitpick},
	}

	summary := composeFallbackSummary(findings, context)
	if !strings.Contains(summary, "Reviewed 2 files, +4 -1.") {
		t.Errorf("expected the scope in the fallback summary, got %q", summary)
	}
	if !strings.Contains(summary, "2 blockers") {
		t.Errorf("expected the blocker count in the fallback summary, got %q", summary)
	}

	empty := composeFallbackSummary(nil, context)
	if !strings.Contains(empty, "No blockers, suggestions, or nitpicks were raised") {
		t.Errorf("expected an explicit empty statement, got %q", empty)
	}
}

func TestDedupeStringsKeepsOrderAndDropsBlanks(t *testing.T) {
	unique := dedupeStrings([]string{" first ", "", "second", "first", "  "})
	if len(unique) != 2 || unique[0] != "first" || unique[1] != "second" {
		t.Errorf("unexpected dedupe result: %v", unique)
	}
}
