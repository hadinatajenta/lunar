package infrastructure

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	sharedErrors "lunar/backend/internal/shared/errors"
)

const (
	compareDiffEmptyPath = "/dev/null"
	compareDiffMaxBytes  = 8 << 20
)

type rawCompareDiffResponse struct {
	Diffs []rawCompareDiff `json:"diffs"`
}

type rawCompareDiff struct {
	Source      *rawCompareDiffPath  `json:"source"`
	Destination *rawCompareDiffPath  `json:"destination"`
	Hunks       []rawCompareDiffHunk `json:"hunks"`
}

type rawCompareDiffPath struct {
	ToString string `json:"toString"`
}

type rawCompareDiffHunk struct {
	SourceLine      int                     `json:"sourceLine"`
	SourceSpan      int                     `json:"sourceSpan"`
	DestinationLine int                     `json:"destinationLine"`
	DestinationSpan int                     `json:"destinationSpan"`
	Segments        []rawCompareDiffSegment `json:"segments"`
}

type rawCompareDiffSegment struct {
	Type  string               `json:"type"`
	Lines []rawCompareDiffLine `json:"lines"`
}

type rawCompareDiffLine struct {
	Source      int    `json:"source"`
	Destination int    `json:"destination"`
	Line        string `json:"line"`
}

func (c *AtlassianBitbucketClient) GetCompareDiff(ctx context.Context, pat string, projectKey string, repoSlug string, sourceBranch string, targetBranch string) (string, error) {
	if strings.TrimSpace(pat) == "" {
		return "", fmt.Errorf("%w: Bitbucket PAT is not configured", sharedErrors.ErrUnauthorized)
	}

	trimmedProjectKey := strings.TrimSpace(projectKey)
	trimmedRepoSlug := strings.TrimSpace(repoSlug)
	trimmedSourceBranch := strings.TrimSpace(sourceBranch)
	trimmedTargetBranch := strings.TrimSpace(targetBranch)
	if trimmedProjectKey == "" || trimmedRepoSlug == "" || trimmedSourceBranch == "" || trimmedTargetBranch == "" {
		return "", fmt.Errorf("%w: project key, repository slug, source branch, and target branch are required", sharedErrors.ErrBadRequest)
	}

	endpoint := fmt.Sprintf(
		"%s/rest/api/1.0/projects/%s/repos/%s/compare/diff?from=%s&to=%s",
		c.baseURL,
		url.PathEscape(trimmedProjectKey),
		url.PathEscape(trimmedRepoSlug),
		url.QueryEscape(trimmedSourceBranch),
		url.QueryEscape(trimmedTargetBranch),
	)

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", err
	}
	request.Header.Set("Authorization", "Bearer "+pat)
	request.Header.Set("Accept", "application/json")
	request.Header.Set("User-Agent", "Lunar-Workspace/1.0")

	response, err := c.httpClient.Do(request)
	if err != nil {
		return "", fmt.Errorf("failed to fetch the branch compare diff from Bitbucket: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode == http.StatusUnauthorized {
		return "", fmt.Errorf("%w: invalid or expired Bitbucket PAT", sharedErrors.ErrUnauthorized)
	}
	if response.StatusCode == http.StatusForbidden {
		return "", fmt.Errorf("%w: access forbidden: your PAT does not have permission for this repository", sharedErrors.ErrForbidden)
	}
	if response.StatusCode == http.StatusNotFound {
		return "", fmt.Errorf(
			"%w: the branch compare was not found in %s/%s (from=%s, to=%s)",
			sharedErrors.ErrNotFound,
			trimmedProjectKey,
			trimmedRepoSlug,
			trimmedSourceBranch,
			trimmedTargetBranch,
		)
	}
	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("bitbucket returned status %d when fetching the compare diff", response.StatusCode)
	}

	var data rawCompareDiffResponse
	if err := json.NewDecoder(io.LimitReader(response.Body, compareDiffMaxBytes)).Decode(&data); err != nil {
		return "", fmt.Errorf("failed to decode Bitbucket compare diff response: %w", err)
	}

	return renderCompareDiff(data), nil
}

func renderCompareDiff(response rawCompareDiffResponse) string {
	var builder strings.Builder

	for _, diff := range response.Diffs {
		oldPath := compareDiffPath(diff.Source)
		newPath := compareDiffPath(diff.Destination)
		if oldPath == "" && newPath == "" {
			continue
		}
		if len(diff.Hunks) == 0 {
			continue
		}

		builder.WriteString(fmt.Sprintf("diff --git a/%s b/%s\n", compareDiffFallbackPath(oldPath, newPath), compareDiffFallbackPath(newPath, oldPath)))
		builder.WriteString(fmt.Sprintf("--- %s\n", markCompareDiffPath(oldPath, "a")))
		builder.WriteString(fmt.Sprintf("+++ %s\n", markCompareDiffPath(newPath, "b")))
		for _, hunk := range diff.Hunks {
			writeCompareDiffHunk(&builder, hunk)
		}
	}

	return builder.String()
}

func compareDiffPath(file *rawCompareDiffPath) string {
	if file == nil {
		return ""
	}
	return strings.TrimPrefix(strings.TrimSpace(file.ToString), "/")
}

func compareDiffFallbackPath(filePath string, fallback string) string {
	if filePath == "" {
		return fallback
	}
	return filePath
}

func markCompareDiffPath(filePath string, prefix string) string {
	if filePath == "" {
		return compareDiffEmptyPath
	}
	return prefix + "/" + filePath
}

func writeCompareDiffHunk(builder *strings.Builder, hunk rawCompareDiffHunk) {
	oldStart, oldCount, newStart, newCount := compareDiffHunkRanges(hunk)
	if oldCount == 0 && newCount == 0 {
		return
	}

	builder.WriteString(fmt.Sprintf("@@ -%d,%d +%d,%d @@\n", oldStart, oldCount, newStart, newCount))
	for _, segment := range hunk.Segments {
		prefix := compareDiffSegmentPrefix(segment.Type)
		if prefix == "" {
			continue
		}
		for _, line := range segment.Lines {
			builder.WriteString(prefix)
			builder.WriteString(line.Line)
			builder.WriteString("\n")
		}
	}
}

func compareDiffSegmentPrefix(segmentType string) string {
	switch strings.ToUpper(strings.TrimSpace(segmentType)) {
	case "CONTEXT":
		return " "
	case "ADDED":
		return "+"
	case "REMOVED":
		return "-"
	default:
		return ""
	}
}

func compareDiffHunkRanges(hunk rawCompareDiffHunk) (int, int, int, int) {
	oldStart, newStart := 0, 0
	oldCount, newCount := 0, 0

	for _, segment := range hunk.Segments {
		for _, line := range segment.Lines {
			switch strings.ToUpper(strings.TrimSpace(segment.Type)) {
			case "CONTEXT":
				if oldStart == 0 && line.Source > 0 {
					oldStart = line.Source
				}
				if newStart == 0 && line.Destination > 0 {
					newStart = line.Destination
				}
				oldCount++
				newCount++
			case "ADDED":
				if newStart == 0 && line.Destination > 0 {
					newStart = line.Destination
				}
				newCount++
			case "REMOVED":
				if oldStart == 0 && line.Source > 0 {
					oldStart = line.Source
				}
				oldCount++
			}
		}
	}

	if oldStart == 0 {
		oldStart = maxCompareDiffLine(hunk.SourceLine)
	}
	if newStart == 0 {
		newStart = maxCompareDiffLine(hunk.DestinationLine)
	}
	if oldCount == 0 {
		oldStart = 0
	}
	if newCount == 0 {
		newStart = 0
	}

	return oldStart, oldCount, newStart, newCount
}

func maxCompareDiffLine(line int) int {
	if line < 0 {
		return 0
	}
	return line
}
