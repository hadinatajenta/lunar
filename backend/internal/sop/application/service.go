package application

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	confluenceDomain "lunar/backend/internal/confluence/domain"
	queryReviewDomain "lunar/backend/internal/queryreview/domain"
	sharedErrors "lunar/backend/internal/shared/errors"
	"lunar/backend/internal/sop/domain"
)

var issueKeyPattern = regexp.MustCompile(`[A-Z][A-Z0-9]+-[0-9]+`)

type DocumentGateway interface {
	GetDocumentDetail(ctx context.Context, userID string, documentID string) (*confluenceDomain.ConfluenceDocument, error)
	GetPageAncestors(ctx context.Context, userID string, pageID string) ([]confluenceDomain.ConfluencePageRef, error)
	GetChildPages(ctx context.Context, userID string, pageID string) ([]confluenceDomain.ConfluencePageRef, error)
	GetPageStorage(ctx context.Context, userID string, pageID string) (string, error)
	UpdatePageStorage(ctx context.Context, userID string, pageID string, storage string) (int, string, error)
	CreatePage(ctx context.Context, userID string, parentPageID string, title string, storage string) (string, int, string, error)
	GetPageVersion(ctx context.Context, userID string, pageID string) (int, error)
}

type MigReader interface {
	ReadMigRepositories(ctx context.Context, userID string, documentID string) ([]queryReviewDomain.MigRepository, error)
}

type HashStore interface {
	Get(ctx context.Context, pageID string, region string) (string, bool, error)
	Set(ctx context.Context, pageID string, region string, hash string) error
}

type Service struct {
	documents DocumentGateway
	mig       MigReader
	hashes    HashStore
	bamboo    map[string]string
	databases map[string]domain.DatabaseTarget
}

func NewService(
	documents DocumentGateway,
	mig MigReader,
	hashes HashStore,
	bamboo map[string]string,
	databases map[string]domain.DatabaseTarget,
) *Service {
	return &Service{
		documents: documents,
		mig:       mig,
		hashes:    hashes,
		bamboo:    bamboo,
		databases: databases,
	}
}

type documentContext struct {
	issueKey          string
	developmentPageID string
}

type existingPage struct {
	pageID string
	title  string
	exists bool
}

func (s *Service) Generate(ctx context.Context, userID string, documentID string, request domain.GenerateRequest) (*domain.GenerateResult, error) {
	if err := s.validate(); err != nil {
		return nil, err
	}
	if strings.TrimSpace(userID) == "" {
		return nil, fmt.Errorf("%w: an authenticated user is required to generate the SOP", sharedErrors.ErrUnauthorized)
	}

	trimmedID := strings.TrimSpace(documentID)
	if trimmedID == "" {
		return nil, fmt.Errorf("%w: document id is required", sharedErrors.ErrBadRequest)
	}

	document, err := s.documents.GetDocumentDetail(ctx, userID, trimmedID)
	if err != nil {
		return nil, err
	}
	if document == nil {
		return nil, fmt.Errorf("%w: document %s was not found", sharedErrors.ErrNotFound, trimmedID)
	}

	resolved, err := s.resolveDocumentContext(ctx, userID, document)
	if err != nil {
		return nil, err
	}

	repositories, err := s.resolveRepositories(ctx, userID, trimmedID, request.Repositories)
	if err != nil {
		return nil, err
	}

	block := BuildDeployBlock(DeployBlockInput{
		IssueKey:    resolved.issueKey,
		Scans:       repositories,
		BambooPlans: s.bamboo,
		Databases:   s.databases,
	})

	deployPage, operasionalPage, err := s.resolveExistingPages(ctx, userID, resolved)
	if err != nil {
		return nil, err
	}

	return &domain.GenerateResult{
		IssueKey:          resolved.issueKey,
		DevelopmentPageID: resolved.developmentPageID,
		Repositories:      repositories,
		Deploy: domain.SopDeploy{
			Title:        deployPage.title,
			PageID:       deployPage.pageID,
			Exists:       deployPage.exists,
			ManagedBlock: block.FullBody(),
		},
		Operasional: operasionalPage,
		Warnings:    collectWarnings(repositories, block.Warnings),
	}, nil
}

func (s *Service) Apply(ctx context.Context, userID string, documentID string, request domain.ApplyRequest) (*domain.ApplyResult, error) {
	if err := s.validate(); err != nil {
		return nil, err
	}
	if strings.TrimSpace(userID) == "" {
		return nil, fmt.Errorf("%w: an authenticated user is required to apply the SOP", sharedErrors.ErrUnauthorized)
	}

	trimmedID := strings.TrimSpace(documentID)
	if trimmedID == "" {
		return nil, fmt.Errorf("%w: document id is required", sharedErrors.ErrBadRequest)
	}
	if err := validateDeployMode(request.Deploy.Mode); err != nil {
		return nil, err
	}

	document, err := s.documents.GetDocumentDetail(ctx, userID, trimmedID)
	if err != nil {
		return nil, err
	}
	if document == nil {
		return nil, fmt.Errorf("%w: document %s was not found", sharedErrors.ErrNotFound, trimmedID)
	}

	resolved, err := s.resolveDocumentContext(ctx, userID, document)
	if err != nil {
		return nil, err
	}

	deployPage, operasionalPage, err := s.resolveExistingPages(ctx, userID, resolved)
	if err != nil {
		return nil, err
	}

	block, err := s.resolveBlock(ctx, userID, trimmedID, request)
	if err != nil {
		return nil, err
	}

	deployResult, err := s.applyDeployPage(ctx, userID, resolved, deployPage, block)
	if err != nil {
		return nil, err
	}

	operasionalResult, createdChildren, err := s.applyOperasional(ctx, userID, resolved, operasionalPage, request.Operasional.CreateChildren)
	if err != nil {
		return nil, err
	}

	return &domain.ApplyResult{
		DeployPage:      deployResult,
		OperasionalPage: operasionalResult,
		CreatedChildren: createdChildren,
	}, nil
}

func (s *Service) validate() error {
	if s.documents == nil || s.mig == nil {
		return fmt.Errorf("%w: the SOP service is not configured", sharedErrors.ErrBadGateway)
	}
	return nil
}

func validateDeployMode(mode string) error {
	trimmed := strings.TrimSpace(mode)
	if trimmed == "" || trimmed == domain.DeployModeCreate || trimmed == domain.DeployModeUpdate {
		return nil
	}
	return fmt.Errorf("%w: deploy mode must be create or update", sharedErrors.ErrBadRequest)
}

func (s *Service) resolveBlock(ctx context.Context, userID string, documentID string, request domain.ApplyRequest) (DeployBlock, error) {
	if body := strings.TrimSpace(request.Deploy.ManagedBlock); body != "" {
		block, ok := ParseDeployBlock(body)
		if !ok {
			return DeployBlock{}, fmt.Errorf("%w: the managed block is missing the database markers", sharedErrors.ErrBadRequest)
		}
		return block, nil
	}

	repositories, err := s.resolveRepositories(ctx, userID, documentID, request.Repositories)
	if err != nil {
		return DeployBlock{}, err
	}
	return BuildDeployBlock(DeployBlockInput{
		Scans:       repositories,
		BambooPlans: s.bamboo,
		Databases:   s.databases,
	}), nil
}

func (s *Service) resolveDocumentContext(ctx context.Context, userID string, document *confluenceDomain.ConfluenceDocument) (documentContext, error) {
	ancestors, err := s.documents.GetPageAncestors(ctx, userID, document.ID)
	if err != nil {
		return documentContext{}, err
	}

	issueKey := extractIssueKey(append([]string{document.Title}, ancestorTitles(ancestors)...)...)
	developmentPage, ok := selectDevelopmentPage(ancestors, issueKey)
	if !ok {
		return documentContext{}, fmt.Errorf("%w: the development page was not found among the document ancestors", sharedErrors.ErrNotFound)
	}
	if issueKey == "" {
		issueKey = issueKeyFromDevelopmentTitle(developmentPage.Title)
	}
	if issueKey == "" {
		return documentContext{}, fmt.Errorf("%w: the issue key could not be resolved from the document", sharedErrors.ErrBadRequest)
	}

	return documentContext{issueKey: issueKey, developmentPageID: developmentPage.ID}, nil
}

func (s *Service) resolveRepositories(ctx context.Context, userID string, documentID string, provided []domain.RepositoryScan) ([]domain.RepositoryScan, error) {
	if len(provided) > 0 {
		return normalizeScans(provided), nil
	}

	repositories, err := s.mig.ReadMigRepositories(ctx, userID, documentID)
	if err != nil {
		return nil, err
	}

	scans := make([]domain.RepositoryScan, 0, len(repositories))
	for _, repository := range repositories {
		scans = append(scans, domain.RepositoryScan{
			Repository:    strings.TrimSpace(repository.Repository),
			Branch:        strings.TrimSpace(repository.Branch),
			BaseRef:       strings.TrimSpace(repository.TargetBranch),
			DatabaseFiles: []domain.DatabaseFile{},
			Tables:        []domain.DatabaseTable{},
			Warnings:      []string{},
		})
	}
	return scans, nil
}

func (s *Service) resolveExistingPages(ctx context.Context, userID string, resolved documentContext) (existingPage, domain.SopOperasional, error) {
	children, err := s.documents.GetChildPages(ctx, userID, resolved.developmentPageID)
	if err != nil {
		return existingPage{}, domain.SopOperasional{}, err
	}

	deployTitle := resolved.issueKey + "-SOP Deploy"
	operasionalTitle := resolved.issueKey + "-SOP Operasional"

	deployPage := existingPage{title: deployTitle}
	if ref, ok := findChildByTitle(children, deployTitle); ok {
		deployPage.pageID = ref.ID
		deployPage.exists = true
	}

	operasional := domain.SopOperasional{Title: operasionalTitle, Children: []domain.SopOperasionalChild{}}
	operasionalRef, exists := findChildByTitle(children, operasionalTitle)
	if !exists {
		for _, title := range operasionalChildTitles {
			operasional.Children = append(operasional.Children, domain.SopOperasionalChild{Title: title, Exists: false})
		}
		return deployPage, operasional, nil
	}

	operasional.PageID = operasionalRef.ID
	operasional.Exists = true
	childRefs, err := s.documents.GetChildPages(ctx, userID, operasionalRef.ID)
	if err != nil {
		return existingPage{}, domain.SopOperasional{}, err
	}
	for _, title := range operasionalChildTitles {
		_, childExists := findChildByTitle(childRefs, title)
		operasional.Children = append(operasional.Children, domain.SopOperasionalChild{Title: title, Exists: childExists})
	}
	return deployPage, operasional, nil
}

func (s *Service) applyDeployPage(
	ctx context.Context,
	userID string,
	resolved documentContext,
	page existingPage,
	block DeployBlock,
) (domain.DeployApplyResult, error) {
	if !page.exists {
		pageID, version, pageURL, err := s.documents.CreatePage(ctx, userID, resolved.developmentPageID, page.title, block.FullBody())
		if err != nil {
			return domain.DeployApplyResult{}, fmt.Errorf("create the SOP Deploy page: %w", err)
		}
		if err := s.storeRegionHashes(ctx, pageID, block); err != nil {
			return domain.DeployApplyResult{}, err
		}
		return domain.DeployApplyResult{ID: pageID, Version: version, URL: pageURL, Action: domain.DeployActionCreated}, nil
	}

	storage, err := s.documents.GetPageStorage(ctx, userID, page.pageID)
	if err != nil {
		return domain.DeployApplyResult{}, err
	}

	if !HasRegion(storage, domain.DeployDatabaseMarker) || !HasRegion(storage, domain.DeployRollbackMarker) {
		body := block.FullBody()
		version, pageURL, err := s.updateWithRetry(ctx, userID, page.pageID, func(current string) string {
			return PrependBlock(current, body)
		})
		if err != nil {
			return domain.DeployApplyResult{}, err
		}
		if err := s.storeRegionHashes(ctx, page.pageID, block); err != nil {
			return domain.DeployApplyResult{}, err
		}
		return domain.DeployApplyResult{ID: page.pageID, Version: version, URL: pageURL, Action: domain.DeployActionUpdated}, nil
	}

	databaseChanged, err := s.regionChanged(ctx, page.pageID, domain.RegionDatabase, block.DatabaseHash)
	if err != nil {
		return domain.DeployApplyResult{}, err
	}
	rollbackChanged, err := s.regionChanged(ctx, page.pageID, domain.RegionRollback, block.RollbackHash)
	if err != nil {
		return domain.DeployApplyResult{}, err
	}
	if !databaseChanged && !rollbackChanged {
		version, _ := s.documents.GetPageVersion(ctx, userID, page.pageID)
		return domain.DeployApplyResult{
			ID:      page.pageID,
			Version: version,
			URL:     s.resolvePageURL(ctx, userID, page.pageID),
			Action:  domain.DeployActionAlreadyUpToDate,
		}, nil
	}

	version, pageURL, err := s.updateWithRetry(ctx, userID, page.pageID, func(current string) string {
		updated := current
		if databaseChanged {
			updated, _ = ReplaceRegion(updated, block.DatabaseRegion, domain.DeployDatabaseMarker)
		}
		if rollbackChanged {
			updated, _ = ReplaceRegion(updated, block.RollbackRegion, domain.DeployRollbackMarker)
		}
		return updated
	})
	if err != nil {
		return domain.DeployApplyResult{}, err
	}
	if err := s.storeRegionHashes(ctx, page.pageID, block); err != nil {
		return domain.DeployApplyResult{}, err
	}
	return domain.DeployApplyResult{ID: page.pageID, Version: version, URL: pageURL, Action: domain.DeployActionUpdated}, nil
}

func (s *Service) updateWithRetry(ctx context.Context, userID string, pageID string, transform func(string) string) (int, string, error) {
	storage, err := s.documents.GetPageStorage(ctx, userID, pageID)
	if err != nil {
		return 0, "", err
	}

	version, pageURL, err := s.documents.UpdatePageStorage(ctx, userID, pageID, transform(storage))
	if err == nil {
		return version, pageURL, nil
	}
	if !errors.Is(err, sharedErrors.ErrConflict) {
		return 0, "", err
	}

	refetched, refetchErr := s.documents.GetPageStorage(ctx, userID, pageID)
	if refetchErr != nil {
		return 0, "", refetchErr
	}
	version, pageURL, retryErr := s.documents.UpdatePageStorage(ctx, userID, pageID, transform(refetched))
	if retryErr != nil {
		return 0, "", fmt.Errorf("%w: the SOP Deploy page could not be updated after a version conflict", sharedErrors.ErrBadGateway)
	}
	return version, pageURL, nil
}

func (s *Service) regionChanged(ctx context.Context, pageID string, region string, hash string) (bool, error) {
	if s.hashes == nil {
		return true, nil
	}
	stored, found, err := s.hashes.Get(ctx, pageID, region)
	if err != nil {
		return false, err
	}
	return !found || stored != hash, nil
}

func (s *Service) applyOperasional(
	ctx context.Context,
	userID string,
	resolved documentContext,
	page domain.SopOperasional,
	createChildren bool,
) (domain.OperasionalApplyResult, []string, error) {
	pageID := page.PageID
	version := 0
	pageURL := ""
	created := make([]string, 0)

	if !page.Exists {
		newID, newVersion, newURL, err := s.documents.CreatePage(ctx, userID, resolved.developmentPageID, page.Title, BuildOperasionalParentBlock())
		if err != nil {
			return domain.OperasionalApplyResult{}, nil, fmt.Errorf("create the SOP Operasional page: %w", err)
		}
		pageID = newID
		version = newVersion
		pageURL = newURL
	} else {
		version, _ = s.documents.GetPageVersion(ctx, userID, pageID)
		pageURL = s.resolvePageURL(ctx, userID, pageID)
	}

	if createChildren {
		children, err := s.documents.GetChildPages(ctx, userID, pageID)
		if err != nil {
			return domain.OperasionalApplyResult{}, nil, err
		}
		for _, title := range operasionalChildTitles {
			if _, exists := findChildByTitle(children, title); exists {
				continue
			}
			if _, _, _, err := s.documents.CreatePage(ctx, userID, pageID, title, BuildOperasionalChildBlock(title)); err != nil {
				return domain.OperasionalApplyResult{}, nil, fmt.Errorf("create the SOP Operasional child %s: %w", title, err)
			}
			created = append(created, title)
		}
	}

	return domain.OperasionalApplyResult{ID: pageID, Version: version, URL: pageURL}, created, nil
}

func (s *Service) storeRegionHashes(ctx context.Context, pageID string, block DeployBlock) error {
	if s.hashes == nil {
		return nil
	}
	if err := s.hashes.Set(ctx, pageID, domain.RegionDatabase, block.DatabaseHash); err != nil {
		return fmt.Errorf("store the SOP database region hash: %w", err)
	}
	if err := s.hashes.Set(ctx, pageID, domain.RegionRollback, block.RollbackHash); err != nil {
		return fmt.Errorf("store the SOP rollback region hash: %w", err)
	}
	return nil
}

func (s *Service) resolvePageURL(ctx context.Context, userID string, pageID string) string {
	if strings.TrimSpace(pageID) == "" {
		return ""
	}
	document, err := s.documents.GetDocumentDetail(ctx, userID, pageID)
	if err != nil || document == nil {
		return ""
	}
	return document.URL
}

func normalizeScans(scans []domain.RepositoryScan) []domain.RepositoryScan {
	normalized := make([]domain.RepositoryScan, 0, len(scans))
	for _, scan := range scans {
		if scan.DatabaseFiles == nil {
			scan.DatabaseFiles = []domain.DatabaseFile{}
		}
		if scan.Warnings == nil {
			scan.Warnings = []string{}
		}
		if scan.Tables == nil || len(scan.Tables) == 0 {
			scan.Tables = domain.MergeDatabaseTables(scan.DatabaseFiles)
		}
		normalized = append(normalized, scan)
	}
	return normalized
}

func collectWarnings(scans []domain.RepositoryScan, blockWarnings []string) []string {
	warnings := make([]string, 0)
	seen := make(map[string]struct{})
	appendWarning := func(warning string) {
		trimmed := strings.TrimSpace(warning)
		if trimmed == "" {
			return
		}
		if _, exists := seen[trimmed]; exists {
			return
		}
		seen[trimmed] = struct{}{}
		warnings = append(warnings, trimmed)
	}

	for _, scan := range scans {
		for _, warning := range scan.Warnings {
			appendWarning(warning)
		}
	}
	for _, warning := range blockWarnings {
		appendWarning(warning)
	}
	return warnings
}

func extractIssueKey(titles ...string) string {
	for _, title := range titles {
		if match := issueKeyPattern.FindString(strings.ToUpper(title)); match != "" {
			return match
		}
	}
	return ""
}

func ancestorTitles(ancestors []confluenceDomain.ConfluencePageRef) []string {
	titles := make([]string, 0, len(ancestors))
	for _, ancestor := range ancestors {
		titles = append(titles, ancestor.Title)
	}
	return titles
}

func selectDevelopmentPage(ancestors []confluenceDomain.ConfluencePageRef, issueKey string) (confluenceDomain.ConfluencePageRef, bool) {
	if issueKey != "" {
		expected := strings.ToUpper(issueKey + "-Development")
		for _, ancestor := range ancestors {
			if strings.ToUpper(strings.TrimSpace(ancestor.Title)) == expected {
				return ancestor, true
			}
		}
	}

	for _, ancestor := range ancestors {
		if strings.HasSuffix(strings.ToUpper(strings.TrimSpace(ancestor.Title)), "-DEVELOPMENT") {
			return ancestor, true
		}
	}
	return confluenceDomain.ConfluencePageRef{}, false
}

func issueKeyFromDevelopmentTitle(title string) string {
	trimmed := strings.TrimSpace(title)
	trimmed = strings.TrimSuffix(trimmed, "-Development")
	trimmed = strings.TrimSuffix(trimmed, "-development")
	return extractIssueKey(trimmed)
}

func findChildByTitle(children []confluenceDomain.ConfluencePageRef, title string) (confluenceDomain.ConfluencePageRef, bool) {
	expected := strings.ToUpper(strings.TrimSpace(title))
	for _, child := range children {
		if strings.ToUpper(strings.TrimSpace(child.Title)) == expected {
			return child, true
		}
	}
	return confluenceDomain.ConfluencePageRef{}, false
}
