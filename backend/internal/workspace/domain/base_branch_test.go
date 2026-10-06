package domain

import "testing"

func TestOrderBaseBranchOptionsKeepsEveryBranchAndOrdersByHint(t *testing.T) {
	candidates := []BaseBranchCandidate{
		{Name: "another/other", CommitAt: 400, LastCommitRelative: "1 day ago"},
		{Name: "feature/CRMMS-77911-Other-Team", CommitAt: 500, LastCommitRelative: "5 hours ago"},
		{Name: "develop", CommitAt: 200, LastCommitRelative: "2 days ago"},
		{Name: "master", CommitAt: 300, LastCommitRelative: "3 days ago"},
		{Name: "main", CommitAt: 100, LastCommitRelative: "1 week ago"},
	}

	options := OrderBaseBranchOptions(candidates, "feature/CRMMS-77911-Other-Team", "main")

	expectedNames := []string{"feature/CRMMS-77911-Other-Team", "main", "master", "develop", "another/other"}
	expectedSources := []BaseBranchSource{
		BaseBranchSourceRemembered,
		BaseBranchSourceBitbucketDefault,
		BaseBranchSourceTrusted,
		BaseBranchSourceTrusted,
		BaseBranchSourceOther,
	}

	if len(options) != len(expectedNames) {
		t.Fatalf("expected %d options, got %d: %+v", len(expectedNames), len(options), options)
	}

	for index, expectedName := range expectedNames {
		if options[index].Name != expectedName {
			t.Fatalf("expected option %d to be %q, got %q", index, expectedName, options[index].Name)
		}
		if options[index].Source != expectedSources[index] {
			t.Fatalf("expected %q to be %q, got %q", expectedName, expectedSources[index], options[index].Source)
		}
		if options[index].LastCommitRelative == "" {
			t.Fatalf("expected %q to carry its last commit", expectedName)
		}
	}
}

func TestOrderBaseBranchOptionsNeverExcludesAFeatureBranch(t *testing.T) {
	candidates := []BaseBranchCandidate{
		{Name: "main", CommitAt: 900, LastCommitRelative: "1 hour ago"},
		{Name: "bugfix/CRMMS-1-hotfix", CommitAt: 800, LastCommitRelative: "2 hours ago"},
		{Name: "feature/CRMMS-1-in-flight", CommitAt: 700, LastCommitRelative: "3 hours ago"},
	}

	options := OrderBaseBranchOptions(candidates, "", "main")

	names := make(map[string]BaseBranchSource, len(options))
	for _, option := range options {
		names[option.Name] = option.Source
	}

	if len(options) != len(candidates) {
		t.Fatalf("expected every branch to be offered, got %d of %d", len(options), len(candidates))
	}
	if names["feature/CRMMS-1-in-flight"] != BaseBranchSourceOther {
		t.Fatalf("expected the feature branch to be offered as other, got %q", names["feature/CRMMS-1-in-flight"])
	}
	if names["bugfix/CRMMS-1-hotfix"] != BaseBranchSourceOther {
		t.Fatalf("expected the bugfix branch to be offered as other, got %q", names["bugfix/CRMMS-1-hotfix"])
	}
}

func TestOrderBaseBranchOptionsEmitsEachBranchOnce(t *testing.T) {
	candidates := []BaseBranchCandidate{
		{Name: "main", CommitAt: 100, LastCommitRelative: "1 day ago"},
		{Name: "main", CommitAt: 100, LastCommitRelative: "1 day ago"},
	}

	options := OrderBaseBranchOptions(candidates, "main", "main")

	if len(options) != 1 {
		t.Fatalf("expected a single option for a duplicated branch, got %d: %+v", len(options), options)
	}
	if options[0].Source != BaseBranchSourceRemembered {
		t.Fatalf("expected the remembered source to win over the remote default, got %q", options[0].Source)
	}
}

func TestOrderBaseBranchOptionsIgnoresAnAbsentRememberedBranch(t *testing.T) {
	candidates := []BaseBranchCandidate{
		{Name: "main", CommitAt: 100, LastCommitRelative: "1 day ago"},
	}

	options := OrderBaseBranchOptions(candidates, "deleted-branch", "main")

	if len(options) != 1 || options[0].Name != "main" {
		t.Fatalf("expected only the existing branch, got %+v", options)
	}
	if options[0].Source != BaseBranchSourceBitbucketDefault {
		t.Fatalf("expected the remote default to stay the remote default, got %q", options[0].Source)
	}
}

func TestIsTrustedBaseBranch(t *testing.T) {
	trusted := []string{"master", "main", "release/production", "develop", "development"}
	for _, name := range trusted {
		if !IsTrustedBaseBranch(name) {
			t.Errorf("expected %q to be trusted", name)
		}
	}

	untrusted := []string{"init", "pre/development", "release/development", "feature/master"}
	for _, name := range untrusted {
		if IsTrustedBaseBranch(name) {
			t.Errorf("expected %q not to be trusted", name)
		}
	}
}

func TestParseRepositoryCoordinates(t *testing.T) {
	testCases := []struct {
		name        string
		remoteURL   string
		project     string
		slug        string
		shouldParse bool
	}{
		{"ssh url", "ssh://git@bitbucket.bri.co.id:7999/nme/mms-api.git", "NME", "mms-api", true},
		{"scp like", "git@bitbucket.bri.co.id:nme/mms-api.git", "NME", "mms-api", true},
		{"https with scm", "https://bitbucket.bri.co.id/scm/nme/mms-api.git", "NME", "mms-api", true},
		{"https without suffix", "https://bitbucket.bri.co.id/scm/aurora/aurora", "AURORA", "aurora", true},
		{"local path", "/tmp/repositories/scratch.git", "", "", false},
		{"file url", "file:///tmp/repositories/scratch.git", "", "", false},
		{"single segment", "ssh://git@bitbucket.bri.co.id/mms-api.git", "", "", false},
		{"empty", "", "", "", false},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			coordinates, found := ParseRepositoryCoordinates(testCase.remoteURL)
			if found != testCase.shouldParse {
				t.Fatalf("expected found %v for %q, got %v", testCase.shouldParse, testCase.remoteURL, found)
			}
			if !found {
				return
			}
			if coordinates.Project != testCase.project || coordinates.Slug != testCase.slug {
				t.Fatalf("expected %s/%s, got %s", testCase.project, testCase.slug, coordinates.String())
			}
		})
	}
}
