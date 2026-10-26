package infrastructure

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"lunar/backend/internal/utdocs/domain"
)

const (
	runDeadline         = 10 * time.Minute
	maxRunOutputBytes   = 512 * 1024
	maxTestOutputBytes  = 8 * 1024
	runOutputCapMarker  = "\n[output capped at 512 KiB]"
	testOutputCapMarker = "\n...[test output truncated]..."
)

var ansiEscapePattern = regexp.MustCompile("\x1b\\[[0-9;?]*[a-zA-Z]")

var errTestCommandFailed = errors.New("the test command exited with a non-zero status")

var hardenedRunEnvironmentVariables = []string{
	"LC_ALL=C",
	"GIT_TERMINAL_PROMPT=0",
	"GIT_OPTIONAL_LOCKS=0",
	"GIT_CONFIG_NOSYSTEM=1",
}

var hardenedRunEnvironmentKeys = map[string]struct{}{
	"LC_ALL":              {},
	"GIT_TERMINAL_PROMPT": {},
	"GIT_OPTIONAL_LOCKS":  {},
	"GIT_CONFIG_NOSYSTEM": {},
}

type ExecutorFunc func(ctx context.Context, dir string, args []string, env []string) ([]byte, error)

type Runner struct {
	executor ExecutorFunc
}

type ParsedGoTest struct {
	Result domain.RunResult
	Tests  []domain.TestResult
	Text   string
}

func NewRunner() *Runner {
	return &Runner{executor: executeGoTest}
}

func NewRunnerWithExecutor(executor ExecutorFunc) *Runner {
	return &Runner{executor: executor}
}

func (r *Runner) Run(ctx context.Context, repositoryPath string, packages []string, testFunctions []string) (domain.RunResult, []domain.TestResult, string, error) {
	if r.executor == nil {
		return domain.RunResult{}, nil, "", errors.New("the go test executor is not configured")
	}
	runRegex := domain.RunRegex(testFunctions)
	if len(packages) == 0 || runRegex == "" {
		return domain.RunResult{Status: domain.RunStatusNotRun, ExecutedPackages: []string{}}, []domain.TestResult{}, "", nil
	}

	executionContext, cancel := context.WithTimeout(ctx, runDeadline)
	defer cancel()

	arguments := []string{"test", "-json", "-count=1", "-timeout", "120s", "-run", runRegex}
	arguments = append(arguments, packages...)
	rawOutput, err := r.executor(executionContext, repositoryPath, arguments, hardenedRunEnvironment())
	if err != nil && needsModuleResolution(string(rawOutput)) {
		restore := snapshotModuleFiles(repositoryPath)
		rawOutput, err = r.executor(executionContext, repositoryPath, withModuleResolution(arguments), hardenedRunEnvironment())
		if restoreError := restore(); restoreError != nil {
			return domain.RunResult{Status: domain.RunStatusFailed, ExecutedPackages: packages}, nil, "", fmt.Errorf("restore the module files: %w", restoreError)
		}
	}
	output, _ := CapRunOutput(StripANSI(string(rawOutput)))
	parsed := ParseGoTestJSONOutput(output)
	parsed.Result.ExecutedPackages = packages
	readable := parsed.Text
	if strings.TrimSpace(readable) == "" {
		readable = output
	}

	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(executionContext.Err(), context.DeadlineExceeded) {
			return domain.RunResult{Status: domain.RunStatusFailed, ExecutedPackages: packages}, parsed.Tests, readable, fmt.Errorf("%w: %v", domain.ErrRunTimeout, err)
		}
		if errors.Is(err, errTestCommandFailed) {
			parsed.Result.Status = domain.RunStatusFailed
			return parsed.Result, parsed.Tests, readable, nil
		}
		return domain.RunResult{Status: domain.RunStatusFailed, ExecutedPackages: packages}, nil, "", fmt.Errorf("execute go test: %w", err)
	}

	return parsed.Result, parsed.Tests, readable, nil
}

func executeGoTest(ctx context.Context, dir string, args []string, env []string) ([]byte, error) {
	command := exec.CommandContext(ctx, "go", args...)
	command.Dir = dir
	command.Env = env

	output, err := command.CombinedOutput()
	if ctx.Err() != nil {
		return output, ctx.Err()
	}

	var exitError *exec.ExitError
	if errors.As(err, &exitError) {
		return output, errTestCommandFailed
	}
	return output, err
}

func StripANSI(value string) string {
	return ansiEscapePattern.ReplaceAllString(value, "")
}

func CapRunOutput(output string) (string, bool) {
	if len(output) <= maxRunOutputBytes {
		return output, false
	}
	return output[:maxRunOutputBytes] + runOutputCapMarker, true
}

type goTestEvent struct {
	Action  string  `json:"Action"`
	Package string  `json:"Package"`
	Test    string  `json:"Test"`
	Elapsed float64 `json:"Elapsed"`
	Output  string  `json:"Output"`
}

type goTestBuilder struct {
	name       string
	packageRef string
	status     string
	durationMs int64
	output     strings.Builder
}

func ParseGoTestJSONOutput(output string) ParsedGoTest {
	order := make([]string, 0)
	builders := make(map[string]*goTestBuilder)
	packageFailed := false
	var packageElapsed float64
	var packageOutput strings.Builder

	for _, line := range strings.Split(output, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		var event goTestEvent
		if err := json.Unmarshal([]byte(trimmed), &event); err != nil {
			continue
		}

		if event.Test == "" {
			switch event.Action {
			case "output":
				packageOutput.WriteString(event.Output)
			case "fail":
				packageFailed = true
				packageElapsed += event.Elapsed
			case "pass":
				packageElapsed += event.Elapsed
			}
			continue
		}

		key := event.Package + "\x1f" + event.Test
		builder, exists := builders[key]
		if !exists {
			builder = &goTestBuilder{name: event.Test, packageRef: event.Package}
			builders[key] = builder
			order = append(order, key)
		}

		switch event.Action {
		case "output":
			builder.output.WriteString(event.Output)
		case "pass":
			builder.status = domain.TestStatusPassed
			builder.durationMs = int64(math.Round(event.Elapsed * 1000))
		case "fail":
			builder.status = domain.TestStatusFailed
			builder.durationMs = int64(math.Round(event.Elapsed * 1000))
			packageFailed = true
		case "skip":
			builder.status = domain.TestStatusSkipped
			builder.durationMs = int64(math.Round(event.Elapsed * 1000))
		}
	}

	tests := make([]domain.TestResult, 0, len(order))
	for _, key := range order {
		builder := builders[key]
		if builder.status == "" {
			continue
		}
		tests = append(tests, domain.TestResult{
			Name:       builder.name,
			Package:    builder.packageRef,
			Status:     builder.status,
			DurationMs: builder.durationMs,
			Output:     capTestOutput(builder.output.String()),
		})
	}

	result := domain.RunResult{Status: domain.RunStatusPassed, ExecutedPackages: []string{}}
	for _, test := range tests {
		result.Total++
		switch test.Status {
		case domain.TestStatusPassed:
			result.Passed++
		case domain.TestStatusFailed:
			result.Failed++
			result.Status = domain.RunStatusFailed
		}
		result.DurationMs += test.DurationMs
	}
	if packageFailed {
		result.Status = domain.RunStatusFailed
	}
	if result.DurationMs == 0 && packageElapsed > 0 {
		result.DurationMs = int64(math.Round(packageElapsed * 1000))
	}

	return ParsedGoTest{
		Result: result,
		Tests:  tests,
		Text:   renderGoTestText(packageOutput.String(), tests),
	}
}

func renderGoTestText(packageOutput string, tests []domain.TestResult) string {
	if strings.TrimSpace(packageOutput) != "" {
		return packageOutput
	}

	builder := strings.Builder{}
	for _, test := range tests {
		builder.WriteString(testStatusLabel(test.Status))
		builder.WriteString(" ")
		builder.WriteString(test.Name)
		builder.WriteString(" (")
		builder.WriteString(formatSeconds(test.DurationMs))
		builder.WriteString(")\n")
	}
	return builder.String()
}

func testStatusLabel(status string) string {
	switch status {
	case domain.TestStatusPassed:
		return "PASS"
	case domain.TestStatusFailed:
		return "FAIL"
	case domain.TestStatusSkipped:
		return "SKIP"
	default:
		return "????"
	}
}

func formatSeconds(durationMs int64) string {
	return strconv.FormatFloat(float64(durationMs)/1000, 'f', 2, 64) + "s"
}

func capTestOutput(output string) string {
	if len(output) <= maxTestOutputBytes {
		return output
	}
	return output[:maxTestOutputBytes] + testOutputCapMarker
}

func hardenedRunEnvironment() []string {
	environment := make([]string, 0, len(os.Environ())+len(hardenedRunEnvironmentVariables))

	for _, variable := range os.Environ() {
		key, _, found := strings.Cut(variable, "=")
		if !found {
			continue
		}
		if _, overridden := hardenedRunEnvironmentKeys[key]; overridden {
			continue
		}
		environment = append(environment, variable)
	}

	return append(environment, hardenedRunEnvironmentVariables...)
}
