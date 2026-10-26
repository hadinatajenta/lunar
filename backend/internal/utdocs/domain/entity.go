package domain

import (
	"errors"
	"regexp"
	"strings"
)

var ErrRunTimeout = errors.New("the unit test run exceeded its deadline")

const (
	LanguageGo   = "go"
	LanguageNode = "node"
	LanguagePHP  = "php"

	PromptVersion = "utdocs-v1"
)

type Classification string

const (
	ClassificationSafe    Classification = "safe"
	ClassificationBlocked Classification = "blocked"
)

type RunStatus string

const (
	RunStatusPassed  RunStatus = "passed"
	RunStatusFailed  RunStatus = "failed"
	RunStatusBlocked RunStatus = "blocked"
	RunStatusNotRun  RunStatus = "not_run"
)

const (
	TestStatusPassed  = "passed"
	TestStatusFailed  = "failed"
	TestStatusSkipped = "skipped"
)

const (
	ResultPassed  = "Passed"
	ResultFailed  = "Failed"
	ResultSkipped = "Skipped"
	ResultNotRun  = "Not Run"
)

const ScreenCaptureMarker = "[[screen-capture]]"

type TestFile struct {
	Path           string         `json:"path"`
	Package        string         `json:"package"`
	Language       string         `json:"language"`
	Classification Classification `json:"classification"`
	Reason         string         `json:"reason"`
	TestFunctions  []string       `json:"test_functions"`
}

type TestResult struct {
	Name       string `json:"name"`
	Package    string `json:"package"`
	File       string `json:"file"`
	Status     string `json:"status"`
	DurationMs int64  `json:"duration_ms"`
	Output     string `json:"output"`
}

type BlockedFile struct {
	Path           string         `json:"path"`
	Classification Classification `json:"classification"`
	Reason         string         `json:"reason"`
}

type UntestedFunction struct {
	Name    string `json:"name"`
	File    string `json:"file"`
	Package string `json:"package"`
}

type RunResult struct {
	Status           RunStatus `json:"status"`
	ExecutedPackages []string  `json:"executed_packages"`
	Total            int       `json:"total"`
	Passed           int       `json:"passed"`
	Failed           int       `json:"failed"`
	DurationMs       int64     `json:"duration_ms"`
	OutputExcerpt    string    `json:"output_excerpt"`
}

type RunPayload struct {
	RunID             string             `json:"run_id"`
	Repo              string             `json:"repo"`
	Branch            string             `json:"branch"`
	BaseRef           string             `json:"base_ref"`
	ChangedFiles      []string           `json:"changed_files"`
	TestFiles         []TestFile         `json:"test_files"`
	BlockedFiles      []BlockedFile      `json:"blocked_files"`
	Run               RunResult          `json:"run"`
	Tests             []TestResult       `json:"tests"`
	UntestedFunctions []UntestedFunction `json:"untested_functions"`
	Stashed           bool               `json:"stashed"`
	Restored          bool               `json:"restored"`
	ScreenshotURL     string             `json:"screenshot_url"`
}

type ScanRequest struct {
	RepositoryPath string
	Branch         string
	BaseRef        string
}

type CandidateTestFile struct {
	Path          string
	Package       string
	Language      string
	Content       string
	TestFunctions []string
}

type NonTestFile struct {
	Path    string
	Content string
}

type ScanResult struct {
	ChangedFiles      []string
	Candidates        []CandidateTestFile
	NonTestFiles      []NonTestFile
	UntestedFunctions []UntestedFunction
}

type ScreenshotRequest struct {
	Repo    string
	Branch  string
	Command string
	Result  string
	Passed  int
	Total   int
	Output  string
}

type DocumentRepository struct {
	Repository        string             `json:"repository"`
	Branch            string             `json:"branch"`
	BaseRef           string             `json:"base_ref"`
	TestFiles         []TestFile         `json:"test_files"`
	BlockedFiles      []TestFile         `json:"blocked_files"`
	Run               RunResult          `json:"run"`
	Tests             []TestResult       `json:"tests"`
	UntestedFunctions []UntestedFunction `json:"untested_functions"`
}

type DocumentRequest struct {
	Model        string               `json:"model"`
	Repositories []DocumentRepository `json:"repositories"`
}

type GeneratedRow struct {
	TestFunctionName string `json:"testFunctionName"`
	Function         string `json:"function"`
	Scenario         string `json:"scenario"`
	ExpectedResult   string `json:"expectedResult"`
	InputData        string `json:"inputData"`
}

type GeneratedMockTest struct {
	TestFunctionName string `json:"testFunctionName"`
	Function         string `json:"function"`
	Scenario         string `json:"scenario"`
	ExpectedResult   string `json:"expectedResult"`
	InputData        string `json:"inputData"`
	Code             string `json:"code"`
}

type GeneratedContent struct {
	Title     string              `json:"title"`
	Summary   string              `json:"summary"`
	Rows      []GeneratedRow      `json:"rows"`
	MockTests []GeneratedMockTest `json:"mockTests"`
	Caption   string              `json:"caption"`
}

type DocumentRow struct {
	No               int    `json:"no"`
	Repository       string `json:"repository"`
	Function         string `json:"function"`
	Scenario         string `json:"scenario"`
	ExpectedResult   string `json:"expected_result"`
	InputData        string `json:"input_data"`
	Result           string `json:"result"`
	ScreenCapture    string `json:"screen_capture"`
	TestFunctionName string `json:"test_function_name"`
}

type DocumentMockTest struct {
	TestFunctionName string `json:"test_function_name"`
	Function         string `json:"function"`
	Scenario         string `json:"scenario"`
	ExpectedResult   string `json:"expected_result"`
	InputData        string `json:"input_data"`
	Code             string `json:"code"`
}

type DocumentResult struct {
	DocumentID        string             `json:"document_id"`
	Model             string             `json:"model"`
	PromptVersion     string             `json:"prompt_version"`
	Title             string             `json:"title"`
	Summary           string             `json:"summary"`
	Table             string             `json:"table"`
	Rows              []DocumentRow      `json:"rows"`
	MockTests         []DocumentMockTest `json:"mock_tests"`
	Markdown          string             `json:"markdown"`
	BlockedNotice     string             `json:"blocked_notice"`
	ScreenshotCaption string             `json:"screenshot_caption"`
}

type ApplyRequest struct {
	Repositories []ApplyRepository `json:"repositories"`
}

type ApplyRepository struct {
	Repository string          `json:"repository"`
	Screenshot ApplyScreenshot `json:"screenshot"`
	Rows       []ApplyRow      `json:"rows"`
	MockTests  []ApplyMockTest `json:"mock_tests"`
}

type ApplyMockTest struct {
	Function       string `json:"function"`
	Scenario       string `json:"scenario"`
	ExpectedResult string `json:"expected_result"`
	InputData      string `json:"input_data"`
	Code           string `json:"code"`
}

type ApplyScreenshot struct {
	Filename      string `json:"filename"`
	ContentBase64 string `json:"content_base64"`
}

type ApplyRow struct {
	Function       string `json:"function"`
	Scenario       string `json:"scenario"`
	ExpectedResult string `json:"expected_result"`
	InputData      string `json:"input_data"`
	Result         string `json:"result"`
}

type ApplyResult struct {
	PageID  string `json:"page_id"`
	Version int    `json:"version"`
	URL     string `json:"url"`
}

func TestCommand(packages []string, testFunctions []string) string {
	command := "go test -count=1 -timeout 120s"
	if len(testFunctions) > 0 {
		command += " -run " + RunRegex(testFunctions)
	}
	if len(packages) > 0 {
		command += " " + strings.Join(packages, " ")
	}
	return command
}

func RunRegex(testFunctions []string) string {
	quoted := make([]string, 0, len(testFunctions))
	for _, function := range testFunctions {
		trimmed := strings.TrimSpace(function)
		if trimmed == "" {
			continue
		}
		quoted = append(quoted, regexp.QuoteMeta(trimmed))
	}
	if len(quoted) == 0 {
		return ""
	}
	return "^(" + strings.Join(quoted, "|") + ")$"
}
