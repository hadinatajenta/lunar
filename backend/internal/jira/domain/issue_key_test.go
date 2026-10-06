package domain

import (
	"strings"
	"testing"
)

func TestIssueKeyExpressionIsTheContractPatternWithoutTheLookahead(t *testing.T) {
	if IssueKeyPattern != `(?i)\b[A-Z][A-Z0-9]{1,9}-[0-9]{1,6}(?![0-9])` {
		t.Fatalf("the contract key pattern changed: %q", IssueKeyPattern)
	}

	expected := strings.TrimSuffix(IssueKeyPattern, `(?![0-9])`)
	if issueKeyExpression.String() != expected {
		t.Fatalf("the compiled expression %q must be the contract pattern without its lookahead", issueKeyExpression.String())
	}
}

func TestIssueKeysInKeepsALongerKeySeparateFromItsPrefix(t *testing.T) {
	keys := IssueKeysIn("feature/generate-tid-CRMMS-63970")
	if len(keys) != 1 || keys[0] != "CRMMS-63970" {
		t.Fatalf("expected only CRMMS-63970, got %v", keys)
	}

	for _, key := range keys {
		if key == "CRMMS-6397" {
			t.Fatal("CRMMS-6397 must not match the longer CRMMS-63970 key")
		}
	}

	keys = IssueKeysIn("feature/generate-tid-CRMMS-6397")
	if len(keys) != 1 || keys[0] != "CRMMS-6397" {
		t.Fatalf("expected only CRMMS-6397, got %v", keys)
	}
}

func TestIssueKeysInExtractsEveryKeyFromABranchName(t *testing.T) {
	testCases := []struct {
		name       string
		branchName string
		expected   []string
	}{
		{name: "a plain key", branchName: "feature/CRMMS-77911", expected: []string{"CRMMS-77911"}},
		{name: "a key after a prefix", branchName: "feature/generate-tid-CRMMS-6397", expected: []string{"CRMMS-6397"}},
		{name: "a longer key", branchName: "feature/CRMMS-63970-handle-tid", expected: []string{"CRMMS-63970"}},
		{name: "a key without a trailing separator", branchName: "CRMMS-63970", expected: []string{"CRMMS-63970"}},
		{name: "a lowercase key", branchName: "feature/crmms-6397-fix", expected: []string{"CRMMS-6397"}},
		{name: "a mixed case key", branchName: "feature/Crmms-6397-Fix", expected: []string{"CRMMS-6397"}},
		{name: "two keys", branchName: "feature/CRMMS-6397-and-CRMMS-77745", expected: []string{"CRMMS-6397", "CRMMS-77745"}},
		{name: "a repeated key", branchName: "feature/CRMMS-6397-CRMMS-6397", expected: []string{"CRMMS-6397"}},
		{name: "a key followed by a letter", branchName: "feature/CRMMS-6397x", expected: []string{"CRMMS-6397"}},
		{name: "a key with seven digits", branchName: "feature/CRMMS-1234567", expected: []string{}},
		{name: "a branch without a key", branchName: "feature/main", expected: []string{}},
		{name: "a plain branch", branchName: "develop", expected: []string{}},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			keys := IssueKeysIn(testCase.branchName)
			if len(keys) != len(testCase.expected) {
				t.Fatalf("expected %v, got %v", testCase.expected, keys)
			}
			for index, expectedKey := range testCase.expected {
				if keys[index] != expectedKey {
					t.Fatalf("expected %v, got %v", testCase.expected, keys)
				}
			}
		})
	}
}

func TestValidateIssueKeyAcceptsThePatternAndCanonicalisesCase(t *testing.T) {
	testCases := []struct {
		name     string
		value    string
		expected string
		invalid  bool
	}{
		{name: "a key", value: "CRMMS-77911", expected: "CRMMS-77911"},
		{name: "a lowercase key", value: "crmms-77911", expected: "CRMMS-77911"},
		{name: "a padded key", value: "  CRMMS-77911  ", expected: "CRMMS-77911"},
		{name: "the shortest key", value: "AB-1", expected: "AB-1"},
		{name: "an empty value", value: "", invalid: true},
		{name: "a key without a number", value: "CRMMS-", invalid: true},
		{name: "a key without a project", value: "-6397", invalid: true},
		{name: "a seven digit key", value: "CRMMS-1234567", invalid: true},
		{name: "a value with a suffix", value: "CRMMS-6397-extra", invalid: true},
		{name: "a value with a prefix", value: "feature/CRMMS-6397", invalid: true},
		{name: "an embedded space", value: "CRM MS-6397", invalid: true},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			canonicalKey, err := ValidateIssueKey(testCase.value)
			if testCase.invalid {
				if err == nil {
					t.Fatalf("expected %q to be refused, got %q", testCase.value, canonicalKey)
				}
				return
			}

			if err != nil {
				t.Fatalf("expected %q to be accepted: %v", testCase.value, err)
			}
			if canonicalKey != testCase.expected {
				t.Fatalf("expected %q, got %q", testCase.expected, canonicalKey)
			}
		})
	}
}
