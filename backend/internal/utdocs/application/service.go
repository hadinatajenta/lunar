package application

import (
	"context"
	"errors"
	"fmt"
	"strings"

	confluenceDomain "lunar/backend/internal/confluence/domain"
	sharedErrors "lunar/backend/internal/shared/errors"
	"lunar/backend/internal/utdocs/domain"
)

const (
	PhaseScan       = "scan"
	PhaseSafety     = "safety"
	PhaseRun        = "run"
	PhaseScreenshot = "screenshot"

	outputChunkSize    = 8192
	outputExcerptLimit = 4000
)

type Scanner interface {
	Scan(ctx context.Context, request domain.ScanRequest) (*domain.ScanResult, error)
}

type TestRunner interface {
	Run(ctx context.Context, repositoryPath string, packages []string, testFunctions []string) (domain.RunResult, []domain.TestResult, string, error)
}

type ScreenshotRenderer interface {
	Render(request domain.ScreenshotRequest) ([]byte, error)
}

type RunObserver interface {
	Phase(phase string)
	Output(chunk string)
}

type RunInput struct {
	RepositoryPath string
	Repo           string
	Branch         string
	BaseRef        string
}

type RunOutcome struct {
	Payload    domain.RunPayload
	Screenshot []byte
}

type RunService struct {
	scanner  Scanner
	runner   TestRunner
	renderer ScreenshotRenderer
}

func NewRunService(scanner Scanner, runner TestRunner, renderer ScreenshotRenderer) *RunService {
	return &RunService{scanner: scanner, runner: runner, renderer: renderer}
}

func (s *RunService) Execute(ctx context.Context, input RunInput, observer RunObserver) (*RunOutcome, error) {
	if s.scanner == nil || s.runner == nil || s.renderer == nil {
		return nil, errors.New("the unit test run service is not configured")
	}

	observer.Phase(PhaseScan)
	scan, err := s.scanner.Scan(ctx, domain.ScanRequest{
		RepositoryPath: input.RepositoryPath,
		Branch:         input.Branch,
		BaseRef:        input.BaseRef,
	})
	if err != nil {
		return nil, fmt.Errorf("scan the branch: %w", err)
	}

	observer.Phase(PhaseSafety)
	classification := ClassifyCandidates(scan.Candidates)
	blockedFiles := append(classification.BlockedFiles, ClassifyNonTestFiles(scan.NonTestFiles)...)
	selectedFunctions, filesByFunction := SelectedTestFunctions(classification)

	observer.Phase(PhaseRun)
	runResult := domain.RunResult{
		Status:           domain.RunStatusNotRun,
		ExecutedPackages: emptyStrings(classification.ExecutedPackages),
	}
	executedPackages := classification.ExecutedPackages
	if len(selectedFunctions) == 0 {
		executedPackages = nil
		runResult.ExecutedPackages = emptyStrings(nil)
	}
	tests := []domain.TestResult{}
	output := ""
	isEverythingBlocked := len(classification.ExecutedPackages) == 0 && len(scan.Candidates) > 0
	if isEverythingBlocked {
		runResult.Status = domain.RunStatusBlocked
	}
	if len(executedPackages) > 0 {
		runResult, tests, output, err = s.runner.Run(ctx, input.RepositoryPath, executedPackages, selectedFunctions)
		if err != nil {
			return nil, fmt.Errorf("run the unit tests: %w", err)
		}
		runResult.ExecutedPackages = emptyStrings(executedPackages)
		tests = withTestFiles(tests, filesByFunction)
	}
	runResult.OutputExcerpt = excerptOutput(output, outputExcerptLimit)
	emitOutputChunks(observer, output)

	observer.Phase(PhaseScreenshot)
	command := domain.TestCommand(executedPackages, selectedFunctions)
	screenshot, err := s.renderer.Render(domain.ScreenshotRequest{
		Repo:    input.Repo,
		Branch:  input.Branch,
		Command: command,
		Result:  string(runResult.Status),
		Passed:  runResult.Passed,
		Total:   runResult.Total,
		Output:  output,
	})
	if err != nil {
		return nil, fmt.Errorf("render the run screenshot: %w", err)
	}

	return &RunOutcome{
		Payload: domain.RunPayload{
			Repo:              input.Repo,
			Branch:            input.Branch,
			BaseRef:           input.BaseRef,
			ChangedFiles:      emptyStrings(scan.ChangedFiles),
			TestFiles:         emptyTestFiles(classification.TestFiles),
			BlockedFiles:      emptyBlockedFiles(blockedFiles),
			Run:               runResult,
			Tests:             tests,
			UntestedFunctions: emptyUntestedFunctions(scan.UntestedFunctions),
		},
		Screenshot: screenshot,
	}, nil
}

func withTestFiles(tests []domain.TestResult, filesByFunction map[string]string) []domain.TestResult {
	annotated := make([]domain.TestResult, 0, len(tests))
	for _, test := range tests {
		if test.File == "" {
			test.File = filesByFunction[test.Name]
		}
		annotated = append(annotated, test)
	}
	return annotated
}

func emitOutputChunks(observer RunObserver, output string) {
	for len(output) > 0 {
		end := outputChunkSize
		if end > len(output) {
			end = len(output)
		}
		observer.Output(output[:end])
		output = output[end:]
	}
}

func excerptOutput(output string, limit int) string {
	if len(output) <= limit {
		return output
	}
	return output[:limit] + "\n...[output excerpt truncated]..."
}

func emptyStrings(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}

func emptyTestFiles(values []domain.TestFile) []domain.TestFile {
	if values == nil {
		return []domain.TestFile{}
	}
	return values
}

func emptyBlockedFiles(values []domain.BlockedFile) []domain.BlockedFile {
	if values == nil {
		return []domain.BlockedFile{}
	}
	return values
}

func emptyUntestedFunctions(values []domain.UntestedFunction) []domain.UntestedFunction {
	if values == nil {
		return []domain.UntestedFunction{}
	}
	return values
}

type DocumentLookup interface {
	GetDocumentDetail(ctx context.Context, userID string, documentID string) (*confluenceDomain.ConfluenceDocument, error)
}

type DocumentGenerator interface {
	Generate(ctx context.Context, userID string, model string, documentTitle string, request domain.DocumentRequest) (*domain.GeneratedContent, error)
}

type DocumentService struct {
	documents DocumentLookup
	generator DocumentGenerator
}

func NewDocumentService(documents DocumentLookup, generator DocumentGenerator) *DocumentService {
	return &DocumentService{documents: documents, generator: generator}
}

func (s *DocumentService) Generate(ctx context.Context, userID string, documentID string, request domain.DocumentRequest) (*domain.DocumentResult, error) {
	if s.documents == nil || s.generator == nil {
		return nil, errors.New("the unit test document service is not configured")
	}
	if strings.TrimSpace(userID) == "" {
		return nil, fmt.Errorf("%w: an authenticated user is required to generate the document", sharedErrors.ErrUnauthorized)
	}

	trimmedID := strings.TrimSpace(documentID)
	if trimmedID == "" {
		return nil, fmt.Errorf("%w: document id is required", sharedErrors.ErrBadRequest)
	}
	modelName := strings.TrimSpace(request.Model)
	if modelName == "" {
		return nil, fmt.Errorf("%w: a model is required", sharedErrors.ErrBadRequest)
	}
	if err := validateDocumentRepositories(request.Repositories); err != nil {
		return nil, err
	}

	document, err := s.documents.GetDocumentDetail(ctx, userID, trimmedID)
	if err != nil {
		return nil, fmt.Errorf("load the document: %w", err)
	}
	if document == nil {
		return nil, fmt.Errorf("%w: document %s was not found", sharedErrors.ErrNotFound, trimmedID)
	}

	generated, err := s.generator.Generate(ctx, userID, modelName, document.Title, request)
	if err != nil {
		return nil, fmt.Errorf("generate the unit test document: %w", err)
	}

	title := strings.TrimSpace(generated.Title)
	if title == "" {
		title = fallbackDocumentTitle(request)
	}
	summary := strings.TrimSpace(generated.Summary)
	caption := strings.TrimSpace(generated.Caption)
	if caption == "" {
		caption = fallbackCaption(request)
	}

	rows, mockTests := buildDocumentContent(request.Repositories, generated)

	return &domain.DocumentResult{
		DocumentID:        trimmedID,
		Model:             modelName,
		PromptVersion:     domain.PromptVersion,
		Title:             title,
		Summary:           summary,
		Table:             BuildNormalTestScenarioTable(rows),
		Rows:              rows,
		MockTests:         mockTests,
		Markdown:          composeMarkdown(title, summary, rows, mockTests),
		BlockedNotice:     blockedNotice(request.Repositories),
		ScreenshotCaption: caption,
	}, nil
}

func validateDocumentRepositories(repositories []domain.DocumentRepository) error {
	if len(repositories) == 0 {
		return fmt.Errorf("%w: at least one repository is required", sharedErrors.ErrBadRequest)
	}
	if len(repositories) > maxDocumentRepositories {
		return fmt.Errorf("%w: at most %d repositories are allowed", sharedErrors.ErrBadRequest, maxDocumentRepositories)
	}
	for _, repository := range repositories {
		if strings.TrimSpace(repository.Repository) == "" || strings.TrimSpace(repository.Branch) == "" || strings.TrimSpace(repository.BaseRef) == "" {
			return fmt.Errorf("%w: repository, branch, and base_ref are required for every repository", sharedErrors.ErrBadRequest)
		}
	}
	return nil
}

func buildDocumentContent(repositories []domain.DocumentRepository, generated *domain.GeneratedContent) ([]domain.DocumentRow, []domain.DocumentMockTest) {
	generatedRows := []domain.GeneratedRow{}
	generatedMockTests := []domain.GeneratedMockTest{}
	if generated != nil {
		generatedRows = generated.Rows
		generatedMockTests = generated.MockTests
	}

	rowsByFunction := make(map[string][]domain.GeneratedRow)
	for _, generatedRow := range generatedRows {
		key := strings.TrimSpace(generatedRow.TestFunctionName)
		if key == "" {
			continue
		}
		rowsByFunction[key] = append(rowsByFunction[key], generatedRow)
	}

	usedMockTests := make([]bool, len(generatedMockTests))
	rows := make([]domain.DocumentRow, 0)
	mockTests := make([]domain.DocumentMockTest, 0)

	for _, repository := range repositories {
		repositoryName := strings.TrimSpace(repository.Repository)
		for _, test := range repository.Tests {
			generatedRow := nextGeneratedRow(rowsByFunction, test.Name)
			rows = append(rows, domain.DocumentRow{
				Repository:       repositoryName,
				Function:         firstNonEmpty(generatedRow.Function, test.Name),
				Scenario:         generatedRow.Scenario,
				ExpectedResult:   generatedRow.ExpectedResult,
				InputData:        generatedRow.InputData,
				Result:           resultForTestStatus(test.Status),
				ScreenCapture:    domain.ScreenCaptureMarker,
				TestFunctionName: test.Name,
			})
			if len(rows) >= maxDocumentRows {
				return rows, mockTests
			}
		}

		for _, untested := range repository.UntestedFunctions {
			functionName := strings.TrimSpace(untested.Name)
			if functionName == "" {
				continue
			}
			mockTest, index := takeGeneratedMockTest(generatedMockTests, usedMockTests, functionName)
			if index >= 0 {
				usedMockTests[index] = true
			}
			rows = append(rows, domain.DocumentRow{
				Repository:       repositoryName,
				Function:         firstNonEmpty(mockTest.Function, functionName),
				Scenario:         mockTest.Scenario,
				ExpectedResult:   mockTest.ExpectedResult,
				InputData:        mockTest.InputData,
				Result:           domain.ResultNotRun,
				ScreenCapture:    "",
				TestFunctionName: functionName,
			})
			if code := strings.TrimSpace(mockTest.Code); code != "" {
				mockTests = append(mockTests, domain.DocumentMockTest{
					TestFunctionName: functionName,
					Function:         firstNonEmpty(mockTest.Function, functionName),
					Scenario:         mockTest.Scenario,
					ExpectedResult:   mockTest.ExpectedResult,
					InputData:        mockTest.InputData,
					Code:             code,
				})
			}
			if len(rows) >= maxDocumentRows {
				return rows, mockTests
			}
		}
	}
	return rows, mockTests
}

func takeGeneratedMockTest(mockTests []domain.GeneratedMockTest, used []bool, functionName string) (domain.GeneratedMockTest, int) {
	for index, mockTest := range mockTests {
		if used[index] {
			continue
		}
		if strings.TrimSpace(mockTest.TestFunctionName) == functionName || strings.TrimSpace(mockTest.Function) == functionName {
			return mockTest, index
		}
	}
	return domain.GeneratedMockTest{}, -1
}

func nextGeneratedRow(rowsByFunction map[string][]domain.GeneratedRow, name string) domain.GeneratedRow {
	queue := rowsByFunction[name]
	if len(queue) == 0 {
		return domain.GeneratedRow{}
	}
	next := queue[0]
	rowsByFunction[name] = queue[1:]
	return next
}

func resultForTestStatus(status string) string {
	switch status {
	case domain.TestStatusPassed:
		return domain.ResultPassed
	case domain.TestStatusFailed:
		return domain.ResultFailed
	case domain.TestStatusSkipped:
		return domain.ResultSkipped
	default:
		return ""
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func fallbackDocumentTitle(request domain.DocumentRequest) string {
	return fmt.Sprintf("Unit test documentation for %s (%s)", request.Repositories[0].Repository, request.Repositories[0].Branch)
}

func fallbackCaption(request domain.DocumentRequest) string {
	return fmt.Sprintf("UT run for %s on %s", request.Repositories[0].Repository, request.Repositories[0].Branch)
}

func composeMarkdown(title string, summary string, rows []domain.DocumentRow, mockTests []domain.DocumentMockTest) string {
	builder := strings.Builder{}
	builder.WriteString("# ")
	builder.WriteString(title)
	builder.WriteString("\n")
	if summary != "" {
		builder.WriteString("\n")
		builder.WriteString(summary)
		builder.WriteString("\n")
	}
	builder.WriteString("\n")
	builder.WriteString(renderMarkdownTable(rows))
	appendMarkdownMockTests(&builder, mockTests)
	return builder.String()
}

func appendMarkdownMockTests(builder *strings.Builder, mockTests []domain.DocumentMockTest) {
	for _, mockTest := range mockTests {
		builder.WriteString("\n\n## ")
		builder.WriteString(firstNonEmpty(mockTest.Function, mockTest.TestFunctionName))
		builder.WriteString("\n\n```go\n")
		builder.WriteString(strings.TrimSpace(mockTest.Code))
		builder.WriteString("\n```\n")
	}
}

func blockedNotice(repositories []domain.DocumentRepository) string {
	total := 0
	for _, repository := range repositories {
		total += len(repository.BlockedFiles)
	}
	if total == 0 {
		return ""
	}
	return fmt.Sprintf("%d test files were not executed; each one is listed with its reason", total)
}
