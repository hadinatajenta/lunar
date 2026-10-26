package infrastructure

import (
	"encoding/json"
	"fmt"
	"strings"

	bitbucketDomain "lunar/backend/internal/bitbucket/domain"
	sharedErrors "lunar/backend/internal/shared/errors"
)

type aiReviewFinding struct {
	Severity     string `json:"severity"`
	Category     string `json:"category"`
	File         string `json:"file"`
	Line         int    `json:"line"`
	Title        string `json:"title"`
	Explanation  string `json:"explanation"`
	SuggestedFix string `json:"suggested_fix"`
	Confidence   string `json:"confidence"`
	Evidence     string `json:"evidence"`
}

type aiReviewPayload struct {
	Summary        string            `json:"summary"`
	Findings       []aiReviewFinding `json:"findings"`
	Assumptions    []string          `json:"assumptions"`
	Strengths      []string          `json:"strengths"`
	Recommendation string            `json:"recommendation"`
	ClosingNote    string            `json:"closing_note"`
}

var reviewSeverities = map[string]bitbucketDomain.ReviewSeverity{
	string(bitbucketDomain.ReviewSeverityBlocker):    bitbucketDomain.ReviewSeverityBlocker,
	string(bitbucketDomain.ReviewSeveritySuggestion): bitbucketDomain.ReviewSeveritySuggestion,
	string(bitbucketDomain.ReviewSeverityNitpick):    bitbucketDomain.ReviewSeverityNitpick,
}

var reviewCategories = map[string]bool{
	"correctness":   true,
	"security":      true,
	"performance":   true,
	"readability":   true,
	"testing":       true,
	"compatibility": true,
}

func parseReviewPayload(content string) (*aiReviewPayload, error) {
	cleaned := extractJSONObject(content)
	if cleaned == "" {
		return nil, fmt.Errorf("%w: the model returned no JSON object", sharedErrors.ErrBadGateway)
	}

	var payload aiReviewPayload
	decoder := json.NewDecoder(strings.NewReader(cleaned))
	if err := decoder.Decode(&payload); err != nil {
		return nil, fmt.Errorf("%w: the model returned invalid review JSON: %v", sharedErrors.ErrBadGateway, err)
	}

	if strings.TrimSpace(payload.Summary) == "" && len(payload.Findings) == 0 && len(payload.Strengths) == 0 {
		return nil, fmt.Errorf("%w: the review JSON is empty", sharedErrors.ErrBadGateway)
	}

	return &payload, nil
}

func extractJSONObject(content string) string {
	trimmed := strings.TrimSpace(content)
	trimmed = strings.TrimPrefix(trimmed, "```json")
	trimmed = strings.TrimPrefix(trimmed, "```")
	trimmed = strings.TrimSuffix(strings.TrimSpace(trimmed), "```")

	start := strings.Index(trimmed, "{")
	end := strings.LastIndex(trimmed, "}")
	if start < 0 || end <= start {
		return ""
	}
	return trimmed[start : end+1]
}

func normalizeReviewFindings(payload *aiReviewPayload, context reviewContext) ([]bitbucketDomain.ReviewFinding, []string) {
	findings := make([]bitbucketDomain.ReviewFinding, 0, len(payload.Findings))
	assumptions := make([]string, 0)

	for _, raw := range payload.Findings {
		if len(findings) >= reviewMaxFindings {
			assumptions = append(assumptions, fmt.Sprintf("%d further findings were dropped because a review reports at most %d.", len(payload.Findings)-reviewMaxFindings, reviewMaxFindings))
			break
		}

		title := strings.TrimSpace(raw.Title)
		if title == "" {
			continue
		}

		finding := bitbucketDomain.ReviewFinding{
			Severity:     reviewSeverities[strings.ToLower(strings.TrimSpace(raw.Severity))],
			Category:     normalizeReviewCategory(raw.Category),
			File:         strings.TrimSpace(raw.File),
			Line:         raw.Line,
			Title:        title,
			Explanation:  strings.TrimSpace(raw.Explanation),
			SuggestedFix: strings.TrimSpace(raw.SuggestedFix),
			Confidence:   bitbucketDomain.ReviewConfidenceConfirmed,
			Evidence:     strings.TrimSpace(raw.Evidence),
		}
		if finding.Severity == "" {
			finding.Severity = bitbucketDomain.ReviewSeveritySuggestion
		}
		if strings.EqualFold(strings.TrimSpace(raw.Confidence), string(bitbucketDomain.ReviewConfidenceAssumption)) {
			finding.Confidence = bitbucketDomain.ReviewConfidenceAssumption
		}

		if !locationExistsInDiff(finding, context) {
			finding.Confidence = bitbucketDomain.ReviewConfidenceAssumption
			assumptions = append(assumptions, fmt.Sprintf("The line cited by \"%s\" (%s) is not in the diff, so it could not be checked against the change.", title, describeLocation(finding)))
		}

		findings = append(findings, finding)
	}

	return findings, dedupeStrings(append(assumptions, payload.Assumptions...))
}

func normalizeReviewCategory(category string) string {
	normalized := strings.ToLower(strings.TrimSpace(category))
	if reviewCategories[normalized] {
		return normalized
	}
	return ""
}

func locationExistsInDiff(finding bitbucketDomain.ReviewFinding, context reviewContext) bool {
	if finding.File == "" {
		return false
	}

	lines, ok := context.ValidLines[finding.File]
	if !ok {
		return false
	}
	if finding.Line == 0 {
		return true
	}
	return lines[finding.Line]
}

func describeLocation(finding bitbucketDomain.ReviewFinding) string {
	if finding.File == "" {
		return "no file given"
	}
	if finding.Line == 0 {
		return finding.File
	}
	return fmt.Sprintf("%s:%d", finding.File, finding.Line)
}

func normalizeRecommendation(payload *aiReviewPayload, findings []bitbucketDomain.ReviewFinding) string {
	recommendation := strings.ToLower(strings.TrimSpace(payload.Recommendation))
	if recommendation == bitbucketDomain.ReviewRecommendationApprove && hasBlocker(findings) {
		return bitbucketDomain.ReviewRecommendationRequestChanges
	}
	if recommendation == bitbucketDomain.ReviewRecommendationRequestChanges {
		return recommendation
	}
	if hasBlocker(findings) {
		return bitbucketDomain.ReviewRecommendationRequestChanges
	}
	return bitbucketDomain.ReviewRecommendationApprove
}

func hasBlocker(findings []bitbucketDomain.ReviewFinding) bool {
	for _, finding := range findings {
		if finding.Severity == bitbucketDomain.ReviewSeverityBlocker {
			return true
		}
	}
	return false
}

func composeReviewComment(review bitbucketDomain.AICodeReview, context reviewContext, closingNote string) string {
	var builder strings.Builder

	builder.WriteString(fmt.Sprintf("## Code review: %s\n\n", fallbackText(context.Title, fallbackText(context.PRID, "pull request"))))
	builder.WriteString(fmt.Sprintf("**Pull request:** %s in `%s`, `%s` into `%s`\n", fallbackText(context.PRID, "unknown"), fallbackText(context.Repository, "unknown"), fallbackText(context.SourceBranch, "unknown"), fallbackText(context.TargetBranch, "unknown")))
	builder.WriteString(fmt.Sprintf("**Reviewed:** %d files, +%d -%d\n", len(context.Files), context.TotalAdded, context.TotalDeleted))
	builder.WriteString(fmt.Sprintf("**Verdict:** %s\n", formatRecommendation(review.Recommendation)))
	if review.Truncated {
		builder.WriteString(fmt.Sprintf("**Not fully read:** %s\n", strings.Join(context.TruncatedFile, ", ")))
	}

	builder.WriteString("\n")
	builder.WriteString(review.Summary)
	builder.WriteString("\n")

	writeFindingGroup(&builder, review.Findings, bitbucketDomain.ReviewSeverityBlocker, "🔴 Blocker", "Must be fixed before merging.")
	writeFindingGroup(&builder, review.Findings, bitbucketDomain.ReviewSeveritySuggestion, "🟡 Suggestion", "Should be fixed.")
	writeFindingGroup(&builder, review.Findings, bitbucketDomain.ReviewSeverityNitpick, "🟢 Nitpick", "Optional, or a style preference.")

	if len(review.Findings) == 0 {
		builder.WriteString("\n### Findings\n\nNo issues were found in the reviewed diff.\n")
	}

	writeBulletSection(&builder, "✅ What was done well", review.Strengths)
	writeBulletSection(&builder, "⚠️ Assumptions", review.Assumptions)

	if note := strings.TrimSpace(closingNote); note != "" {
		builder.WriteString("\n")
		builder.WriteString(note)
		builder.WriteString("\n")
	}

	builder.WriteString(fmt.Sprintf("\n---\n\nReviewed with %s. Findings marked as assumptions could not be confirmed from the diff alone.\n", fallbackText(review.Model, "Lunar Copilot")))
	return builder.String()
}
