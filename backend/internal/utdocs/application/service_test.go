package application

import (
	"context"
	"errors"
	"strings"
	"testing"

	confluenceDomain "lunar/backend/internal/confluence/domain"
	sharedErrors "lunar/backend/internal/shared/errors"
	"lunar/backend/internal/utdocs/domain"
)

type stubScanner struct {
	result  *domain.ScanResult
	err     error
	request domain.ScanRequest
}

func (s *stubScanner) Scan(_ context.Context, request domain.ScanRequest) (*domain.ScanResult, error) {
	s.request = request
	if s.err != nil {
		return nil, s.err
	}
	return s.result, nil
}

type stubRunner struct {
	result         domain.RunResult
	tests          []domain.TestResult
	output         string
	err            error
	calls          int
	repositoryPath string
	packages       []string
	functions      []string
}

func (r *stubRunner) Run(_ context.Context, repositoryPath string, packages []string, testFunctions []string) (domain.RunResult, []domain.TestResult, string, error) {
	r.calls++
	r.repositoryPath = repositoryPath
	r.packages = append([]string{}, packages...)
	r.functions = append([]string{}, testFunctions...)
	if r.err != nil {
		return domain.RunResult{}, nil, "", r.err
	}
	return r.result, r.tests, r.output, nil
}

type stubRenderer struct {
	png     []byte
	err     error
	calls   int
	request domain.ScreenshotRequest
}

func (r *stubRenderer) Render(request domain.ScreenshotRequest) ([]byte, error) {
	r.calls++
	r.request = request
	if r.err != nil {
		return nil, r.err
	}
	return r.png, nil
}

type recordingObserver struct {
	phases []string
	chunks []string
}

func (o *recordingObserver) Phase(phase string) {
	o.phases = append(o.phases, phase)
}

func (o *recordingObserver) Output(chunk string) {
	o.chunks = append(o.chunks, chunk)
}

func safeRepositoryScan() *domain.ScanResult {
	return &domain.ScanResult{
		ChangedFiles: []string{"internal/repo/dao.go"},
		Candidates: []domain.CandidateTestFile{
			{
				Path:          "internal/repo/dao_test.go",
				Package:       "./internal/repo",
				Language:      domain.LanguageGo,
				Content:       "package repo\n\nfunc TestDao(t *testing.T) {}\n",
				TestFunctions: []string{"TestDao"},
			},
		},
		NonTestFiles: []domain.NonTestFile{
			{Path: "internal/repo/dao.go", Content: "package repo\n"},
		},
	}
}

func TestRunServiceExecutesSafeFunctionsAndRendersTheScreenshot(t *testing.T) {
	runner := &stubRunner{
		result: domain.RunResult{Status: domain.RunStatusPassed, Total: 1, Passed: 1, DurationMs: 120},
		tests:  []domain.TestResult{{Name: "TestDao", Package: "lunar/backend/internal/repo", Status: domain.TestStatusPassed, DurationMs: 120}},
		output: "ok  \tlunar/backend/internal/repo\t0.120s\n",
	}
	renderer := &stubRenderer{png: []byte("png-bytes")}
	service := NewRunService(&stubScanner{result: safeRepositoryScan()}, runner, renderer)
	observer := &recordingObserver{}

	outcome, err := service.Execute(context.Background(), RunInput{
		RepositoryPath: "/workspace/micro-payment",
		Repo:           "micro-payment",
		Branch:         "feat/CRMMS-123",
		BaseRef:        "development",
	}, observer)
	if err != nil {
		t.Fatalf("expected the run to succeed, got %v", err)
	}

	if strings.Join(observer.phases, ",") != "scan,safety,run,screenshot" {
		t.Fatalf("expected the four phases in order, got %+v", observer.phases)
	}
	if runner.calls != 1 || len(runner.packages) != 1 || runner.packages[0] != "./internal/repo" {
		t.Fatalf("expected the safe package to run, got %+v", runner.packages)
	}
	if len(runner.functions) != 1 || runner.functions[0] != "TestDao" {
		t.Fatalf("expected only the changed function to run, got %+v", runner.functions)
	}
	if strings.Join(observer.chunks, "") != runner.output {
		t.Fatalf("expected the streamed chunks to rebuild the output, got %q", strings.Join(observer.chunks, ""))
	}

	payload := outcome.Payload
	if payload.Run.Status != domain.RunStatusPassed {
		t.Fatalf("expected a passed run, got %+v", payload.Run)
	}
	if len(payload.Tests) != 1 || payload.Tests[0].File != "internal/repo/dao_test.go" {
		t.Fatalf("expected the test result with its file, got %+v", payload.Tests)
	}
	if string(outcome.Screenshot) != "png-bytes" {
		t.Fatalf("expected the rendered screenshot, got %q", outcome.Screenshot)
	}
	if renderer.request.Command != "go test -count=1 -timeout 120s -run ^(TestDao)$ ./internal/repo" {
		t.Fatalf("expected the strict command in the screenshot, got %q", renderer.request.Command)
	}
}

func TestRunServiceBlocksWhenEveryCandidateIsBlocked(t *testing.T) {
	scan := &domain.ScanResult{
		ChangedFiles: []string{"internal/seed/reset_test.go"},
		Candidates: []domain.CandidateTestFile{
			{
				Path:          "internal/seed/reset_test.go",
				Package:       "./internal/seed",
				Language:      domain.LanguageGo,
				Content:       "package seed\n\nfunc TestReset(t *testing.T) {\n\tTRUNCATE TABLE users\n}\n",
				TestFunctions: []string{"TestReset"},
			},
		},
		NonTestFiles: []domain.NonTestFile{},
	}
	runner := &stubRunner{}
	service := NewRunService(&stubScanner{result: scan}, runner, &stubRenderer{png: []byte("png")})

	outcome, err := service.Execute(context.Background(), RunInput{RepositoryPath: "/repo", Repo: "repo", Branch: "feat/x", BaseRef: "main"}, &recordingObserver{})
	if err != nil {
		t.Fatalf("expected the run to be reported, got %v", err)
	}
	if runner.calls != 0 {
		t.Fatalf("expected the runner not to be called, got %d calls", runner.calls)
	}
	if outcome.Payload.Run.Status != domain.RunStatusBlocked {
		t.Fatalf("expected a blocked run, got %+v", outcome.Payload.Run)
	}
	if len(outcome.Payload.BlockedFiles) != 1 {
		t.Fatalf("expected the blocked file list, got %+v", outcome.Payload.BlockedFiles)
	}
}

func TestRunServiceReportsNotRunWithoutCandidates(t *testing.T) {
	runner := &stubRunner{}
	service := NewRunService(&stubScanner{result: &domain.ScanResult{ChangedFiles: []string{}, Candidates: []domain.CandidateTestFile{}, NonTestFiles: []domain.NonTestFile{}}}, runner, &stubRenderer{png: []byte("png")})

	outcome, err := service.Execute(context.Background(), RunInput{RepositoryPath: "/repo", Repo: "repo", Branch: "feat/x", BaseRef: "main"}, &recordingObserver{})
	if err != nil {
		t.Fatalf("expected the run to be reported, got %v", err)
	}
	if outcome.Payload.Run.Status != domain.RunStatusNotRun {
		t.Fatalf("expected a not run status, got %+v", outcome.Payload.Run)
	}
	if runner.calls != 0 {
		t.Fatalf("expected the runner not to be called, got %d calls", runner.calls)
	}
}

func TestRunServiceReportsChangedNonTestFiles(t *testing.T) {
	scan := &domain.ScanResult{
		ChangedFiles: []string{"internal/seed/reset.go"},
		Candidates:   []domain.CandidateTestFile{},
		NonTestFiles: []domain.NonTestFile{
			{Path: "internal/seed/reset.go", Content: "package seed\n\nfunc Reset(db *sql.DB) {\n\tdb.Exec(\"TRUNCATE users\")\n}\n"},
		},
	}
	service := NewRunService(&stubScanner{result: scan}, &stubRunner{}, &stubRenderer{png: []byte("png")})

	outcome, err := service.Execute(context.Background(), RunInput{RepositoryPath: "/repo", Repo: "repo", Branch: "feat/x", BaseRef: "main"}, &recordingObserver{})
	if err != nil {
		t.Fatalf("expected the run to be reported, got %v", err)
	}
	if len(outcome.Payload.BlockedFiles) != 1 || outcome.Payload.BlockedFiles[0].Path != "internal/seed/reset.go" {
		t.Fatalf("expected the changed non-test file to be reported, got %+v", outcome.Payload.BlockedFiles)
	}
}

func TestRunServiceReturnsScanAndRunnerFailures(t *testing.T) {
	scannerService := NewRunService(&stubScanner{err: errors.New("cannot read the branch")}, &stubRunner{}, &stubRenderer{})
	if _, err := scannerService.Execute(context.Background(), RunInput{RepositoryPath: "/repo"}, &recordingObserver{}); err == nil {
		t.Fatal("expected a scan failure")
	}

	runnerService := NewRunService(&stubScanner{result: safeRepositoryScan()}, &stubRunner{err: errors.New("go is missing")}, &stubRenderer{})
	if _, err := runnerService.Execute(context.Background(), RunInput{RepositoryPath: "/repo"}, &recordingObserver{}); err == nil {
		t.Fatal("expected a runner failure")
	}

	rendererService := NewRunService(&stubScanner{result: safeRepositoryScan()}, &stubRunner{result: domain.RunResult{Status: domain.RunStatusPassed}}, &stubRenderer{err: errors.New("cannot encode")})
	if _, err := rendererService.Execute(context.Background(), RunInput{RepositoryPath: "/repo"}, &recordingObserver{}); err == nil {
		t.Fatal("expected a renderer failure")
	}
}

type stubDocumentLookup struct {
	document *confluenceDomain.ConfluenceDocument
	err      error
	userID   string
	id       string
}

func (l *stubDocumentLookup) GetDocumentDetail(_ context.Context, userID string, documentID string) (*confluenceDomain.ConfluenceDocument, error) {
	l.userID = userID
	l.id = documentID
	if l.err != nil {
		return nil, l.err
	}
	return l.document, nil
}

type stubDocumentGenerator struct {
	generated *domain.GeneratedContent
	err       error
	userID    string
	model     string
	title     string
	request   domain.DocumentRequest
	calls     int
}

func (g *stubDocumentGenerator) Generate(_ context.Context, userID string, model string, documentTitle string, request domain.DocumentRequest) (*domain.GeneratedContent, error) {
	g.calls++
	g.userID = userID
	g.model = model
	g.title = documentTitle
	g.request = request
	if g.err != nil {
		return nil, g.err
	}
	return g.generated, nil
}

func validDocumentRequest() domain.DocumentRequest {
	return domain.DocumentRequest{
		Model: "DeepSeek-V4 Flash",
		Repositories: []domain.DocumentRepository{
			{
				Repository: "micro-payment",
				Branch:     "feat/CRMMS-123",
				BaseRef:    "development",
				TestFiles: []domain.TestFile{
					{Path: "internal/repo/dao_test.go", Package: "./internal/repo", Language: domain.LanguageGo, Classification: domain.ClassificationSafe, TestFunctions: []string{"TestDao"}},
				},
				BlockedFiles: []domain.TestFile{
					{Path: "internal/seed/reset_test.go", Classification: domain.ClassificationBlocked, Reason: "contains destructive or database markers: TRUNCATE"},
					{Path: "internal/seed/seed_test.go", Classification: domain.ClassificationBlocked, Reason: "its package ./internal/seed does not run because the package contains a blocked test file"},
				},
				Run:   domain.RunResult{Status: domain.RunStatusPassed, Total: 1, Passed: 1, DurationMs: 120},
				Tests: []domain.TestResult{{Name: "TestDao", Package: "lunar/backend/internal/repo", File: "internal/repo/dao_test.go", Status: domain.TestStatusPassed, DurationMs: 120}},
			},
		},
	}
}

func validGeneratedContent() *domain.GeneratedContent {
	return &domain.GeneratedContent{
		Title:   "Unit tests for Payment API",
		Summary: "The changed repository layer is covered by mock based tests.",
		Rows: []domain.GeneratedRow{
			{TestFunctionName: "TestDao", Function: "Find payment", Scenario: "Mencari data pembayaran", ExpectedResult: "Data pembayaran ditemukan", InputData: "ID pembayaran"},
		},
		Caption: "UT run for micro-payment on feat/CRMMS-123",
	}
}

func TestDocumentServiceAssemblesTheDocument(t *testing.T) {
	lookup := &stubDocumentLookup{document: &confluenceDomain.ConfluenceDocument{ID: "123456", Title: "Payment API"}}
	generator := &stubDocumentGenerator{generated: validGeneratedContent()}
	service := NewDocumentService(lookup, generator)

	result, err := service.Generate(context.Background(), "user-1", "123456", validDocumentRequest())
	if err != nil {
		t.Fatalf("expected the document to be generated, got %v", err)
	}

	if result.DocumentID != "123456" || result.Model != "DeepSeek-V4 Flash" || result.PromptVersion != domain.PromptVersion {
		t.Fatalf("expected the frozen identity fields, got %+v", result)
	}
	if lookup.userID != "user-1" || lookup.id != "123456" {
		t.Fatalf("expected the document lookup to use the user and document, got %q %q", lookup.userID, lookup.id)
	}
	if generator.title != "Payment API" || generator.model != "DeepSeek-V4 Flash" {
		t.Fatalf("expected the document title and model to reach the generator, got %q %q", generator.title, generator.model)
	}
	if len(result.Rows) != 1 {
		t.Fatalf("expected one merged row, got %+v", result.Rows)
	}
	row := result.Rows[0]
	if row.Function != "Find payment" || row.Scenario != "Mencari data pembayaran" || row.Result != domain.ResultPassed {
		t.Fatalf("expected the merged row, got %+v", row)
	}
	if row.ScreenCapture != domain.ScreenCaptureMarker {
		t.Fatalf("expected the screen capture marker, got %q", row.ScreenCapture)
	}
	if !strings.Contains(result.Table, "Normal Test Scenario") || !strings.Contains(result.Table, "Find payment") {
		t.Fatalf("expected the table to carry the scenario, got %q", result.Table)
	}
	if result.BlockedNotice != "2 test files were not executed; each one is listed with its reason" {
		t.Fatalf("expected the blocked notice, got %q", result.BlockedNotice)
	}
	if result.ScreenshotCaption != "UT run for micro-payment on feat/CRMMS-123" {
		t.Fatalf("expected the generated caption, got %q", result.ScreenshotCaption)
	}
}

func TestDocumentServiceNeverTakesAResultFromTheModel(t *testing.T) {
	lookup := &stubDocumentLookup{document: &confluenceDomain.ConfluenceDocument{ID: "123456", Title: "Payment API"}}
	generated := validGeneratedContent()
	generated.Rows = append(generated.Rows, domain.GeneratedRow{TestFunctionName: "TestInvented", Function: "Invented"})
	generator := &stubDocumentGenerator{generated: generated}
	service := NewDocumentService(lookup, generator)

	result, err := service.Generate(context.Background(), "user-1", "123456", validDocumentRequest())
	if err != nil {
		t.Fatalf("expected the document to be generated, got %v", err)
	}
	if len(result.Rows) != 1 || result.Rows[0].TestFunctionName != "TestDao" {
		t.Fatalf("expected only the payload test function, got %+v", result.Rows)
	}
	if result.Rows[0].Result != domain.ResultPassed {
		t.Fatalf("expected the system owned result, got %q", result.Rows[0].Result)
	}
}

func TestDocumentServiceFallsBackOnEmptyGeneratedFields(t *testing.T) {
	lookup := &stubDocumentLookup{document: &confluenceDomain.ConfluenceDocument{ID: "123456", Title: "Payment API"}}
	generator := &stubDocumentGenerator{generated: &domain.GeneratedContent{}}
	service := NewDocumentService(lookup, generator)

	request := validDocumentRequest()
	request.Repositories[0].BlockedFiles = []domain.TestFile{}
	result, err := service.Generate(context.Background(), "user-1", "123456", request)
	if err != nil {
		t.Fatalf("expected the document to be generated, got %v", err)
	}

	if result.Title != "Unit test documentation for micro-payment (feat/CRMMS-123)" {
		t.Fatalf("expected the fallback title, got %q", result.Title)
	}
	if result.ScreenshotCaption != "UT run for micro-payment on feat/CRMMS-123" {
		t.Fatalf("expected the fallback caption, got %q", result.ScreenshotCaption)
	}
	if result.BlockedNotice != "" {
		t.Fatalf("expected no blocked notice, got %q", result.BlockedNotice)
	}
	if len(result.Rows) != 1 || result.Rows[0].Function != "TestDao" {
		t.Fatalf("expected a fallback function name from the test, got %+v", result.Rows)
	}
}

func TestDocumentServiceValidatesTheRequest(t *testing.T) {
	service := NewDocumentService(&stubDocumentLookup{document: &confluenceDomain.ConfluenceDocument{}}, &stubDocumentGenerator{generated: &domain.GeneratedContent{}})

	if _, err := service.Generate(context.Background(), "", "123456", validDocumentRequest()); !errors.Is(err, sharedErrors.ErrUnauthorized) {
		t.Fatalf("expected an unauthorized error without a user, got %v", err)
	}
	if _, err := service.Generate(context.Background(), "user-1", " ", validDocumentRequest()); !errors.Is(err, sharedErrors.ErrBadRequest) {
		t.Fatalf("expected a bad request for a missing document id, got %v", err)
	}

	withoutModel := validDocumentRequest()
	withoutModel.Model = " "
	if _, err := service.Generate(context.Background(), "user-1", "123456", withoutModel); !errors.Is(err, sharedErrors.ErrBadRequest) {
		t.Fatalf("expected a bad request without a model, got %v", err)
	}

	withoutRepositories := validDocumentRequest()
	withoutRepositories.Repositories = nil
	if _, err := service.Generate(context.Background(), "user-1", "123456", withoutRepositories); !errors.Is(err, sharedErrors.ErrBadRequest) {
		t.Fatalf("expected a bad request without repositories, got %v", err)
	}

	withoutBranch := validDocumentRequest()
	withoutBranch.Repositories[0].Branch = ""
	if _, err := service.Generate(context.Background(), "user-1", "123456", withoutBranch); !errors.Is(err, sharedErrors.ErrBadRequest) {
		t.Fatalf("expected a bad request without a branch, got %v", err)
	}
}

func TestDocumentServicePropagatesFailures(t *testing.T) {
	lookupService := NewDocumentService(&stubDocumentLookup{err: sharedErrors.ErrNotFound}, &stubDocumentGenerator{generated: &domain.GeneratedContent{}})
	if _, err := lookupService.Generate(context.Background(), "user-1", "123456", validDocumentRequest()); !errors.Is(err, sharedErrors.ErrNotFound) {
		t.Fatalf("expected the lookup failure to propagate, got %v", err)
	}

	generatorService := NewDocumentService(&stubDocumentLookup{document: &confluenceDomain.ConfluenceDocument{ID: "123456"}}, &stubDocumentGenerator{err: errors.New("the provider is down")})
	if _, err := generatorService.Generate(context.Background(), "user-1", "123456", validDocumentRequest()); err == nil {
		t.Fatal("expected the generator failure to propagate")
	}
}

func generatedContentWithMockTest() *domain.GeneratedContent {
	return &domain.GeneratedContent{
		Title:   "Unit tests for Payment API",
		Summary: "The changed repository layer is covered by mock based tests.",
		Rows: []domain.GeneratedRow{
			{TestFunctionName: "TestDao", Function: "Find payment", Scenario: "Mencari data pembayaran", ExpectedResult: "Data pembayaran ditemukan", InputData: "ID pembayaran"},
		},
		MockTests: []domain.GeneratedMockTest{
			{
				TestFunctionName: "TestSave",
				Function:         "Save",
				Scenario:         "Menyimpan data pembayaran dengan mock repository",
				ExpectedResult:   "Data pembayaran tersimpan",
				InputData:        "Objek pembayaran dan mock repository",
				Code:             "func TestSave(t *testing.T) {\n\tmockRepo := new(MockRepository)\n\tmockRepo.On(\"Save\", mock.Anything).Return(nil)\n\tmockRepo.AssertExpectations(t)\n}",
			},
		},
		Caption: "UT run for micro-payment on feat/CRMMS-123",
	}
}

func TestDocumentServiceMergesExecutedTestsAndUntestedFunctions(t *testing.T) {
	lookup := &stubDocumentLookup{document: &confluenceDomain.ConfluenceDocument{ID: "123456", Title: "Payment API"}}
	generator := &stubDocumentGenerator{generated: generatedContentWithMockTest()}
	service := NewDocumentService(lookup, generator)

	request := validDocumentRequest()
	request.Repositories[0].UntestedFunctions = []domain.UntestedFunction{
		{Name: "Save", File: "internal/repo/dao.go", Package: "./internal/repo"},
	}

	result, err := service.Generate(context.Background(), "user-1", "123456", request)
	if err != nil {
		t.Fatalf("expected the document to be generated, got %v", err)
	}

	if len(result.Rows) != 2 {
		t.Fatalf("expected one executed row and one untested row, got %+v", result.Rows)
	}
	executed := result.Rows[0]
	if executed.Result != domain.ResultPassed || executed.ScreenCapture != domain.ScreenCaptureMarker {
		t.Fatalf("expected the executed row to keep its result and screenshot, got %+v", executed)
	}
	untested := result.Rows[1]
	if untested.TestFunctionName != "Save" || untested.Function != "Save" {
		t.Fatalf("expected the untested function row, got %+v", untested)
	}
	if untested.Result != domain.ResultNotRun {
		t.Fatalf("expected the Not Run result, got %q", untested.Result)
	}
	if untested.ScreenCapture != "" {
		t.Fatalf("expected an empty screen capture, got %q", untested.ScreenCapture)
	}
	if untested.Scenario != "Menyimpan data pembayaran dengan mock repository" || untested.ExpectedResult != "Data pembayaran tersimpan" || untested.InputData != "Objek pembayaran dan mock repository" {
		t.Fatalf("expected the AI scenario fields, got %+v", untested)
	}

	if len(result.MockTests) != 1 {
		t.Fatalf("expected one mock test, got %+v", result.MockTests)
	}
	if result.MockTests[0].TestFunctionName != "Save" || result.MockTests[0].Function != "Save" {
		t.Fatalf("expected the mock test bound to the untested function, got %+v", result.MockTests[0])
	}
	if result.MockTests[0].Code == "" {
		t.Fatalf("expected the generated code, got %+v", result.MockTests[0])
	}

	if !strings.Contains(result.Table, domain.ResultNotRun) {
		t.Fatalf("expected the Not Run result in the table, got %q", result.Table)
	}
	if !strings.Contains(result.Markdown, "```go") || !strings.Contains(result.Markdown, "func TestSave") {
		t.Fatalf("expected one fenced go block per mock test, got %q", result.Markdown)
	}
}

func TestDocumentServiceIgnoresInventedMockTests(t *testing.T) {
	lookup := &stubDocumentLookup{document: &confluenceDomain.ConfluenceDocument{ID: "123456", Title: "Payment API"}}
	generated := generatedContentWithMockTest()
	generated.MockTests = append(generated.MockTests, domain.GeneratedMockTest{Function: "Invented", Code: "func TestInvented(t *testing.T) {}"})
	generator := &stubDocumentGenerator{generated: generated}
	service := NewDocumentService(lookup, generator)

	request := validDocumentRequest()
	request.Repositories[0].UntestedFunctions = []domain.UntestedFunction{{Name: "Save", File: "internal/repo/dao.go", Package: "./internal/repo"}}

	result, err := service.Generate(context.Background(), "user-1", "123456", request)
	if err != nil {
		t.Fatalf("expected the document to be generated, got %v", err)
	}
	if len(result.MockTests) != 1 || result.MockTests[0].Function != "Save" {
		t.Fatalf("expected only the payload untested function, got %+v", result.MockTests)
	}
	if strings.Contains(result.Markdown, "TestInvented") {
		t.Fatalf("expected the invented mock test to be ignored, got %q", result.Markdown)
	}
}

func TestDocumentServiceReportsUntestedFunctionsWithoutAGeneratedMockTest(t *testing.T) {
	lookup := &stubDocumentLookup{document: &confluenceDomain.ConfluenceDocument{ID: "123456", Title: "Payment API"}}
	generator := &stubDocumentGenerator{generated: &domain.GeneratedContent{Title: "Unit tests", Summary: "Mock based coverage."}}
	service := NewDocumentService(lookup, generator)

	request := validDocumentRequest()
	request.Repositories[0].Tests = []domain.TestResult{}
	request.Repositories[0].UntestedFunctions = []domain.UntestedFunction{{Name: "Save", File: "internal/repo/dao.go", Package: "./internal/repo"}}

	result, err := service.Generate(context.Background(), "user-1", "123456", request)
	if err != nil {
		t.Fatalf("expected the document to be generated, got %v", err)
	}
	if len(result.Rows) != 1 || result.Rows[0].Result != domain.ResultNotRun {
		t.Fatalf("expected a single Not Run row, got %+v", result.Rows)
	}
	if len(result.MockTests) != 0 {
		t.Fatalf("expected no mock test without generated code, got %+v", result.MockTests)
	}
	if strings.Contains(result.Table, noScenarioMessage) {
		t.Fatalf("expected a real row instead of the empty message, got %q", result.Table)
	}
}

func TestDocumentServiceUsesTheEmptyMessageOnlyWhenNothingExists(t *testing.T) {
	lookup := &stubDocumentLookup{document: &confluenceDomain.ConfluenceDocument{ID: "123456", Title: "Payment API"}}
	generator := &stubDocumentGenerator{generated: &domain.GeneratedContent{Title: "Unit tests", Summary: "Mock based coverage."}}
	service := NewDocumentService(lookup, generator)

	request := validDocumentRequest()
	request.Repositories[0].Tests = []domain.TestResult{}
	request.Repositories[0].UntestedFunctions = []domain.UntestedFunction{}

	result, err := service.Generate(context.Background(), "user-1", "123456", request)
	if err != nil {
		t.Fatalf("expected the document to be generated, got %v", err)
	}
	if !strings.Contains(result.Table, noScenarioMessage) {
		t.Fatalf("expected the empty message, got %q", result.Table)
	}
}
