package infrastructure

import (
	"fmt"
	"strings"

	bitbucketDomain "lunar/backend/internal/bitbucket/domain"
)

func writeFindingGroup(builder *strings.Builder, findings []bitbucketDomain.ReviewFinding, severity bitbucketDomain.ReviewSeverity, heading string, hint string) {
	group := make([]bitbucketDomain.ReviewFinding, 0)
	for _, finding := range findings {
		if finding.Severity == severity {
			group = append(group, finding)
		}
	}
	if len(group) == 0 {
		return
	}

	builder.WriteString(fmt.Sprintf("\n### %s\n\n%s\n", heading, hint))
	for index, finding := range group {
		writeFinding(builder, finding, index+1)
	}
}

func writeFinding(builder *strings.Builder, finding bitbucketDomain.ReviewFinding, position int) {
	builder.WriteString(fmt.Sprintf("\n**%d. %s**\n\n", position, findingHeadline(finding)))

	writeLabelledBlock(builder, "Why", finding.Explanation)
	writeLabelledBlock(builder, "Fix", finding.SuggestedFix)

	if finding.Evidence != "" {
		builder.WriteString(fmt.Sprintf("**Evidence:** `%s`\n\n", finding.Evidence))
	}
	if finding.Confidence == bitbucketDomain.ReviewConfidenceAssumption {
		builder.WriteString("**Confidence:** assumption, not confirmed from the diff\n\n")
	}
}

func writeLabelledBlock(builder *strings.Builder, label string, value string) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return
	}
	if strings.Contains(trimmed, "\n") {
		builder.WriteString(fmt.Sprintf("**%s:**\n\n%s\n\n", label, trimmed))
		return
	}
	builder.WriteString(fmt.Sprintf("**%s:** %s\n\n", label, trimmed))
}

func findingHeadline(finding bitbucketDomain.ReviewFinding) string {
	location := describeLocation(finding)
	headline := finding.Title
	if location != "" {
		headline = fmt.Sprintf("`%s` %s", location, headline)
	}
	if finding.Category != "" {
		headline = fmt.Sprintf("%s (%s)", headline, finding.Category)
	}
	return headline
}

func composeFallbackSummary(findings []bitbucketDomain.ReviewFinding, context reviewContext) string {
	counts := map[bitbucketDomain.ReviewSeverity]int{}
	for _, finding := range findings {
		counts[finding.Severity]++
	}

	scope := fmt.Sprintf("Reviewed %d files, +%d -%d.", len(context.Files), context.TotalAdded, context.TotalDeleted)
	if len(findings) == 0 {
		return fmt.Sprintf("%s No blockers, suggestions, or nitpicks were raised against the diff.", scope)
	}

	return fmt.Sprintf("%s Raised %d blockers, %d suggestions, and %d nitpicks.",
		scope,
		counts[bitbucketDomain.ReviewSeverityBlocker],
		counts[bitbucketDomain.ReviewSeveritySuggestion],
		counts[bitbucketDomain.ReviewSeverityNitpick])
}

func writeBulletSection(builder *strings.Builder, heading string, entries []string) {
	items := make([]string, 0, len(entries))
	for _, entry := range entries {
		if trimmed := strings.TrimSpace(entry); trimmed != "" {
			items = append(items, trimmed)
		}
	}
	if len(items) == 0 {
		return
	}

	builder.WriteString(fmt.Sprintf("\n### %s\n\n", heading))
	for _, entry := range items {
		builder.WriteString(fmt.Sprintf("- %s\n", entry))
	}
}

func formatRecommendation(recommendation string) string {
	if recommendation == bitbucketDomain.ReviewRecommendationRequestChanges {
		return "🔴 Request changes"
	}
	return "🟢 Approve"
}

func dedupeStrings(entries []string) []string {
	seen := make(map[string]bool)
	unique := make([]string, 0, len(entries))
	for _, entry := range entries {
		trimmed := strings.TrimSpace(entry)
		if trimmed == "" || seen[trimmed] {
			continue
		}
		seen[trimmed] = true
		unique = append(unique, trimmed)
	}
	return unique
}
