package application

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	confluenceDomain "lunar/backend/internal/confluence/domain"
	"lunar/backend/internal/queryreview/domain"
	sharedErrors "lunar/backend/internal/shared/errors"
)

const migTitleSuffix = "-MIG"

var issueKeyPattern = regexp.MustCompile(`[A-Z][A-Z0-9]+-[0-9]+`)

type migResolution struct {
	pageID    string
	pageTitle string
	rows      []migTableRow
	skipped   []domain.SkippedRepository
}

func (s *QueryReviewService) readMig(ctx context.Context, userID string, document *confluenceDomain.ConfluenceDocument) (migResolution, error) {
	if s.documents == nil {
		return migResolution{}, fmt.Errorf("%w: the document source is not configured", sharedErrors.ErrBadGateway)
	}

	ancestors, err := s.documents.GetPageAncestors(ctx, userID, document.ID)
	if err != nil {
		return migResolution{}, err
	}
	if len(ancestors) == 0 {
		return migResolution{}, noCompareLinkError(document.ID)
	}

	parent := ancestors[len(ancestors)-1]
	children, err := s.documents.GetChildPages(ctx, userID, parent.ID)
	if err != nil {
		return migResolution{}, err
	}

	issueKey := extractIssueKey(document.Title, parent.Title)
	migPage, ok := selectMigPage(children, issueKey)
	if !ok {
		return migResolution{}, noCompareLinkError(document.ID)
	}

	storage, err := s.documents.GetPageStorage(ctx, userID, migPage.ID)
	if err != nil {
		return migResolution{}, err
	}

	rows, parseSkipped := parseMigTables(storage)

	return migResolution{
		pageID:    migPage.ID,
		pageTitle: migPage.Title,
		rows:      rows,
		skipped:   parseSkipped,
	}, nil
}

func (s *QueryReviewService) resolveMig(ctx context.Context, userID string, document *confluenceDomain.ConfluenceDocument) (migResolution, error) {
	resolution, err := s.readMig(ctx, userID, document)
	if err != nil {
		return migResolution{}, err
	}

	included, excludedSkipped := filterExcludedRepositories(resolution.rows)
	resolution.rows = included
	resolution.skipped = append(resolution.skipped, excludedSkipped...)
	return resolution, nil
}

func selectMigPage(children []confluenceDomain.ConfluencePageRef, issueKey string) (confluenceDomain.ConfluencePageRef, bool) {
	for _, page := range children {
		title := strings.ToUpper(strings.TrimSpace(page.Title))
		if !strings.HasSuffix(title, migTitleSuffix) {
			continue
		}
		if issueKey != "" && !strings.Contains(title, issueKey) {
			continue
		}
		return page, true
	}
	return confluenceDomain.ConfluencePageRef{}, false
}

func extractIssueKey(titles ...string) string {
	for _, title := range titles {
		if match := issueKeyPattern.FindString(strings.ToUpper(title)); match != "" {
			return match
		}
	}
	return ""
}

func noCompareLinkError(documentID string) error {
	return fmt.Errorf("%w: the document %s has no sibling MIG page, provide source.compare_url (%w)", sharedErrors.ErrBadRequest, documentID, domain.ErrNoCompareLink)
}
