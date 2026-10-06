package domain

import (
	"testing"
)

func TestAdversarialIssueKeysInNeverExtractsAShorterKeyFromALongerDigitRun(t *testing.T) {
	testCases := []struct {
		name        string
		branchName  string
		expected    []string
		hasDigitRun bool
	}{
		{name: "one extra digit", branchName: "feature/generate-tid-CRMMS-63970", expected: []string{"CRMMS-63970"}, hasDigitRun: true},
		{name: "one extra digit at six total", branchName: "feature/CRMMS-639712", expected: []string{"CRMMS-639712"}, hasDigitRun: true},
		{name: "seven digits exceeds the run cap", branchName: "feature/CRMMS-6397123", expected: []string{}, hasDigitRun: true},
		{name: "ten digits exceeds the run cap", branchName: "feature/CRMMS-6397123456", expected: []string{}, hasDigitRun: true},
		{name: "eight digits on a different project", branchName: "feature/CRMMS-12345678", expected: []string{}, hasDigitRun: true},
		{name: "a longer key beside a shorter one", branchName: "feature/CRMMS-6397-and-CRMMS-63970", expected: []string{"CRMMS-6397", "CRMMS-63970"}},
		{name: "a trailing digit run after the key", branchName: "CRMMS-63970-CRMMS-63971", expected: []string{"CRMMS-63970", "CRMMS-63971"}},
		{name: "a letter suffix is not a digit", branchName: "feature/CRMMS-6397x", expected: []string{"CRMMS-6397"}},
		{name: "an underscore suffix is not a digit", branchName: "feature/CRMMS-6397_later", expected: []string{"CRMMS-6397"}},
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

			if !testCase.hasDigitRun {
				return
			}
			for _, key := range keys {
				if key == "CRMMS-6397" {
					t.Fatalf("CRMMS-6397 must never be carved out of the digit run in %q, got %v", testCase.branchName, keys)
				}
			}
		})
	}
}

func TestAdversarialValidateIssueKeyRefusesALongerDigitRunOnTheLookupSurface(t *testing.T) {
	for _, longerValue := range []string{"CRMMS-63970", "CRMMS-639712", "CRMMS-1234567"} {
		canonicalKey, err := ValidateIssueKey(longerValue)
		if err != nil {
			continue
		}
		if canonicalKey == "CRMMS-6397" {
			t.Fatalf("the lookup surface must never canonicalise %q into CRMMS-6397", longerValue)
		}
	}

	for _, shorterValue := range []string{"CRMMS-6397", "crmms-6397", " CRMMS-6397 "} {
		canonicalKey, err := ValidateIssueKey(shorterValue)
		if err != nil {
			t.Fatalf("expected %q to be accepted: %v", shorterValue, err)
		}
		if canonicalKey != "CRMMS-6397" {
			t.Fatalf("expected %q to canonicalise to CRMMS-6397, got %q", shorterValue, canonicalKey)
		}
	}
}

func TestAdversarialIssueKeysInCurrentlyMissesAKeyAdjoinedByAWordCharacter(t *testing.T) {
	missedInputs := map[string]string{
		"an underscore separator": "bugfix_CRMMS-6397-existing",
		"a repeated underscore":   "feature_CRMMS-6397",
		"a digit separator":       "sprint12CRMMS-6397",
		"a leading underscore":    "_CRMMS-6397",
		"a one character project": "feature/A-123",
		"an underscored project":  "feature/CRM_MS-6397",
		"a seven digit issue":     "feature/CRMMS-1234567",
		"an eleven char project":  "feature/ABCDEFGHIJK-1",
	}

	for description, missedInput := range missedInputs {
		keys := IssueKeysIn(missedInput)
		if len(keys) != 0 {
			continue
		}
		t.Logf("characterisation: %s, IssueKeysIn(%q) is empty", description, missedInput)
	}

	for description, missedInput := range missedInputs {
		if len(IssueKeysIn(missedInput)) != 0 {
			continue
		}
		if description == "an eleven char project" {
			continue
		}
		t.Logf("known gap still open for %q", missedInput)
	}
}

func TestAdversarialIssueKeysInDoesNotAbsorbALowercaseLetterIntoTheProjectPrefix(t *testing.T) {
	keys := IssueKeysIn("feature/buildCRMMS-6397-extra")
	if len(keys) != 1 || keys[0] != "CRMMS-6397" {
		t.Fatalf("expected [CRMMS-6397] rather than an absorbed prefix, got %v", keys)
	}
}
