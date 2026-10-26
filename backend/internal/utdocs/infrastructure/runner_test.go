package infrastructure

import (
	"context"
	"errors"
	"strings"
	"testing"

	"lunar/backend/internal/utdocs/domain"
)

const sampleGoTestJSON = `{"Action":"run","Package":"lunar/backend/internal/repo","Test":"TestFind"}
{"Action":"output","Package":"lunar/backend/internal/repo","Test":"TestFind","Output":"=== RUN   TestFind\n"}
{"Action":"pass","Package":"lunar/backend/internal/repo","Test":"TestFind","Elapsed":0.12}
{"Action":"run","Package":"lunar/backend/internal/repo","Test":"TestReset"}
{"Action":"output","Package":"lunar/backend/internal/repo","Test":"TestReset","Output":"    dao_test.go:12: boom\n"}
{"Action":"fail","Package":"lunar/backend/internal/repo","Test":"TestReset","Elapsed":0.05}
{"Action":"output","Package":"lunar/backend/internal/repo","Output":"FAIL\n"}
{"Action":"fail","Package":"lunar/backend/internal/repo","Elapsed":0.2}
`

func TestRunnerUsesTheStrictRunRegexAndTheExecutorSeam(t *testing.T) {
	var capturedArgs []string
	var capturedEnv []string

	runner := NewRunnerWithExecutor(func(_ context.Context, _ string, args []string, env []string) ([]byte, error) {
		capturedArgs = append([]string{}, args...)
		capturedEnv = append([]string{}, env...)
		return []byte(sampleGoTestJSON), nil
	})

	result, tests, output, err := runner.Run(context.Background(), "/workspace/micro-payment", []string{"./internal/repo"}, []string{"TestFind", "TestReset"})
	if err != nil {
		t.Fatalf("expected the run to finish, got %v", err)
	}

	expectedArgs := []string{"test", "-json", "-count=1", "-timeout", "120s", "-run", "^(TestFind|TestReset)$", "./internal/repo"}
	if strings.Join(capturedArgs, " ") != strings.Join(expectedArgs, " ") {
		t.Fatalf("expected the fixed argument array, got %+v", capturedArgs)
	}
	if !containsEnvironment(capturedEnv, "LC_ALL=C") || !containsEnvironment(capturedEnv, "GIT_TERMINAL_PROMPT=0") {
		t.Fatalf("expected the hardened environment, got %+v", capturedEnv)
	}

	if result.Status != domain.RunStatusFailed || result.Total != 2 || result.Passed != 1 || result.Failed != 1 {
		t.Fatalf("expected the parsed summary, got %+v", result)
	}
	if result.DurationMs != 170 {
		t.Fatalf("expected the summed duration 170ms, got %d", result.DurationMs)
	}
	if len(result.ExecutedPackages) != 1 || result.ExecutedPackages[0] != "./internal/repo" {
		t.Fatalf("expected the executed package, got %+v", result.ExecutedPackages)
	}

	if len(tests) != 2 {
		t.Fatalf("expected two per-test results, got %+v", tests)
	}
	if tests[0].Name != "TestFind" || tests[0].Status != domain.TestStatusPassed || tests[0].DurationMs != 120 {
		t.Fatalf("expected the passed test, got %+v", tests[0])
	}
	if tests[0].Package != "lunar/backend/internal/repo" {
		t.Fatalf("expected the package reference, got %+v", tests[0])
	}
	if tests[1].Name != "TestReset" || tests[1].Status != domain.TestStatusFailed || tests[1].DurationMs != 50 {
		t.Fatalf("expected the failed test, got %+v", tests[1])
	}
	if !strings.Contains(tests[1].Output, "boom") {
		t.Fatalf("expected the per-test output, got %q", tests[1].Output)
	}
	if output == "" {
		t.Fatal("expected a readable run text for the screenshot")
	}
}

func TestRunnerParsesSkippedTests(t *testing.T) {
	var capturedArgs []string
	runner := NewRunnerWithExecutor(func(_ context.Context, _ string, args []string, _ []string) ([]byte, error) {
		capturedArgs = append([]string{}, args...)
		return []byte(`{"Action":"skip","Package":"p","Test":"TestOnly","Elapsed":0.001}` + "\n"), nil
	})

	result, tests, _, err := runner.Run(context.Background(), "/repo", []string{"./p"}, []string{"TestOnly"})
	if err != nil {
		t.Fatalf("expected the run to finish, got %v", err)
	}
	if len(tests) != 1 || tests[0].Status != domain.TestStatusSkipped {
		t.Fatalf("expected a skipped test, got %+v", tests)
	}
	if result.Status != domain.RunStatusPassed || result.Total != 1 || result.Passed != 0 || result.Failed != 0 {
		t.Fatalf("expected a passed summary with one skipped test, got %+v", result)
	}
	if len(capturedArgs) < 1 || capturedArgs[0] != "test" {
		t.Fatalf("expected the test command, got %+v", capturedArgs)
	}
}

func TestRunnerReportsAFailedExitAsFailed(t *testing.T) {
	runner := NewRunnerWithExecutor(func(_ context.Context, _ string, _ []string, _ []string) ([]byte, error) {
		return []byte(`{"Action":"fail","Package":"p","Test":"TestBuild","Elapsed":0.01}` + "\n"), errTestCommandFailed
	})

	result, _, _, err := runner.Run(context.Background(), "/repo", []string{"./p"}, []string{"TestBuild"})
	if err != nil {
		t.Fatalf("expected a failed result instead of an error, got %v", err)
	}
	if result.Status != domain.RunStatusFailed {
		t.Fatalf("expected the exit code to force a failed status, got %+v", result)
	}
}

func TestRunnerReportsTimeouts(t *testing.T) {
	runner := NewRunnerWithExecutor(func(_ context.Context, _ string, _ []string, _ []string) ([]byte, error) {
		return nil, context.DeadlineExceeded
	})

	_, _, _, err := runner.Run(context.Background(), "/repo", []string{"./p"}, []string{"TestFind"})
	if !errors.Is(err, domain.ErrRunTimeout) {
		t.Fatalf("expected a timeout error, got %v", err)
	}
}

func TestRunnerReturnsIncompleteExecutions(t *testing.T) {
	runner := NewRunnerWithExecutor(func(_ context.Context, _ string, _ []string, _ []string) ([]byte, error) {
		return nil, errors.New("executable not found")
	})

	_, _, _, err := runner.Run(context.Background(), "/repo", []string{"./p"}, []string{"TestFind"})
	if err == nil || errors.Is(err, domain.ErrRunTimeout) {
		t.Fatalf("expected a plain execution error, got %v", err)
	}
}

func TestRunnerCapsTheOutput(t *testing.T) {
	runner := NewRunnerWithExecutor(func(_ context.Context, _ string, _ []string, _ []string) ([]byte, error) {
		return []byte(strings.Repeat("x", maxRunOutputBytes+4096)), nil
	})

	_, _, output, err := runner.Run(context.Background(), "/repo", []string{"./p"}, []string{"TestFind"})
	if err != nil {
		t.Fatalf("expected the run to pass, got %v", err)
	}
	if len(output) != maxRunOutputBytes+len(runOutputCapMarker) {
		t.Fatalf("expected the capped output length, got %d", len(output))
	}
	if !strings.Contains(output, "[output capped at 512 KiB]") {
		t.Fatal("expected the cap marker")
	}
}

func TestRunnerWithoutAFunctionSelectionDoesNotExecute(t *testing.T) {
	called := false
	runner := NewRunnerWithExecutor(func(_ context.Context, _ string, _ []string, _ []string) ([]byte, error) {
		called = true
		return nil, nil
	})

	result, tests, output, err := runner.Run(context.Background(), "/repo", []string{"./p"}, []string{})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if called {
		t.Fatal("expected the executor not to be called")
	}
	if result.Status != domain.RunStatusNotRun || output != "" || len(tests) != 0 {
		t.Fatalf("expected a not run result, got %+v %q %+v", result, output, tests)
	}
}

func TestRunnerWithoutPackagesDoesNotExecute(t *testing.T) {
	called := false
	runner := NewRunnerWithExecutor(func(_ context.Context, _ string, _ []string, _ []string) ([]byte, error) {
		called = true
		return nil, nil
	})

	result, _, _, err := runner.Run(context.Background(), "/repo", []string{}, []string{"TestFind"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if called {
		t.Fatal("expected the executor not to be called")
	}
	if result.Status != domain.RunStatusNotRun {
		t.Fatalf("expected a not run result, got %+v", result)
	}
}

func TestRunRegexQuotesTheSelectedFunctions(t *testing.T) {
	if got := domain.RunRegex([]string{"TestA", "TestB"}); got != "^(TestA|TestB)$" {
		t.Fatalf("expected the strict anchored regex, got %q", got)
	}
	if got := domain.RunRegex([]string{"TestA.B"}); got != `^(TestA\.B)$` {
		t.Fatalf("expected the function name to be quoted, got %q", got)
	}
	if got := domain.RunRegex(nil); got != "" {
		t.Fatalf("expected no regex without functions, got %q", got)
	}
}

func TestParseGoTestJSONOutputIgnoresNonJSONLines(t *testing.T) {
	raw := "go: downloading example.com/dep\n" + sampleGoTestJSON + "not json\n"

	parsed := ParseGoTestJSONOutput(raw)
	if parsed.Result.Total != 2 {
		t.Fatalf("expected two parsed tests, got %+v", parsed.Result)
	}
}

func containsEnvironment(environment []string, expected string) bool {
	for _, variable := range environment {
		if variable == expected {
			return true
		}
	}
	return false
}
