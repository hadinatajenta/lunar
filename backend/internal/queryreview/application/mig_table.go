package application

import (
	"html"
	"regexp"
	"strings"

	"lunar/backend/internal/queryreview/domain"
)

var (
	htmlTablePattern    = regexp.MustCompile(`(?is)<table\b[^>]*>(.*?)</table>`)
	htmlRowPattern      = regexp.MustCompile(`(?is)<tr\b[^>]*>(.*?)</tr>`)
	htmlCellPattern     = regexp.MustCompile(`(?is)<t[dh]\b[^>]*>(.*?)</t[dh]>`)
	htmlTagStripPattern = regexp.MustCompile(`(?is)<[^>]*>`)
)

var excludedRepositorySlugs = []string{"fe-mms", "mms"}

type migTableRow struct {
	repository    string
	projectKey    string
	repoSlug      string
	origin        string
	compareURL    string
	sourceBranch  string
	targetBranch  string
	pullRequestID string
}

func parseMigTables(storage string) ([]migTableRow, []domain.SkippedRepository) {
	rows := make([]migTableRow, 0)
	skipped := make([]domain.SkippedRepository, 0)

	for _, table := range htmlTablePattern.FindAllStringSubmatch(storage, -1) {
		tableRows := htmlRowPattern.FindAllStringSubmatch(table[1], -1)
		if len(tableRows) == 0 {
			continue
		}

		repositoryIndex, branchIndex, linkIndex, ok := migColumnIndexes(migCells(tableRows[0][1]))
		if !ok {
			continue
		}

		for _, rawRow := range tableRows[1:] {
			row, reason := migRowFromCells(migCells(rawRow[1]), rawRow[1], repositoryIndex, branchIndex, linkIndex)
			if reason != "" {
				skipped = append(skipped, domain.SkippedRepository{Repository: row.repository, Reason: reason})
				continue
			}
			rows = append(rows, row)
		}
	}

	return rows, skipped
}

func filterExcludedRepositories(rows []migTableRow) ([]migTableRow, []domain.SkippedRepository) {
	included := make([]migTableRow, 0, len(rows))
	skipped := make([]domain.SkippedRepository, 0)

	for _, row := range rows {
		if isExcludedRepository(row.repository) || isExcludedRepository(row.repoSlug) {
			displayName := row.repository
			if displayName == "" {
				displayName = row.repoSlug
			}
			skipped = append(skipped, domain.SkippedRepository{Repository: displayName, Reason: "excluded repository"})
			continue
		}
		included = append(included, row)
	}

	return included, skipped
}

func isExcludedRepository(repository string) bool {
	trimmed := strings.TrimSpace(repository)
	for _, slug := range excludedRepositorySlugs {
		if strings.EqualFold(trimmed, slug) {
			return true
		}
	}
	return false
}

func migColumnIndexes(headerCells []string) (int, int, int, bool) {
	repositoryIndex := -1
	branchIndex := -1
	linkIndex := -1

	for index, cell := range headerCells {
		name := strings.ToLower(migCellText(cell))
		switch {
		case name == "repository":
			repositoryIndex = index
		case name == "branch":
			branchIndex = index
		case linkIndex == -1 && (strings.Contains(name, "compare") || strings.Contains(name, "link")):
			linkIndex = index
		}
	}

	if repositoryIndex == -1 || branchIndex == -1 || linkIndex == -1 {
		return 0, 0, 0, false
	}
	return repositoryIndex, branchIndex, linkIndex, true
}

func migRowFromCells(cells []string, rowHTML string, repositoryIndex int, branchIndex int, linkIndex int) (migTableRow, string) {
	repository := migCellText(migCellAt(cells, repositoryIndex))
	branch := normalizeBranchRef(migCellText(migCellAt(cells, branchIndex)))
	candidate, ok := linkCandidate(migCellAt(cells, linkIndex))
	if !ok {
		candidate, ok = linkCandidate(rowHTML)
	}
	if !ok {
		return migTableRow{repository: repository}, "no parsable compare link"
	}

	repoSlug := candidate.RepoSlug
	if repository == "" {
		repository = repoSlug
	}
	sourceBranch := normalizeBranchRef(candidate.SourceBranch)
	if sourceBranch == "" {
		sourceBranch = branch
	}
	targetBranch := normalizeBranchRef(candidate.TargetBranch)
	if targetBranch == "" {
		targetBranch = branch
	}

	if repository == "" || sourceBranch == "" || targetBranch == "" {
		return migTableRow{repository: repository}, "incomplete repository row"
	}

	return migTableRow{
		repository:    repository,
		projectKey:    candidate.ProjectKey,
		repoSlug:      repoSlug,
		origin:        candidate.Origin,
		compareURL:    candidate.CompareURL,
		sourceBranch:  sourceBranch,
		targetBranch:  targetBranch,
		pullRequestID: candidate.PullRequestID,
	}, ""
}

func migCellAt(cells []string, index int) string {
	if index < 0 || index >= len(cells) {
		return ""
	}
	return cells[index]
}

func migCells(rowHTML string) []string {
	matches := htmlCellPattern.FindAllStringSubmatch(rowHTML, -1)
	cells := make([]string, 0, len(matches))
	for _, match := range matches {
		cells = append(cells, match[1])
	}
	return cells
}

func migCellText(cellHTML string) string {
	withoutTags := htmlTagStripPattern.ReplaceAllString(cellHTML, " ")
	decoded := html.UnescapeString(withoutTags)
	return strings.Join(strings.Fields(decoded), " ")
}
