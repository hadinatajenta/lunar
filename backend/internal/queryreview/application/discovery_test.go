package application

import "testing"

func TestParseCompareCandidateStripsTheRefsHeadsPrefix(t *testing.T) {
	candidate, ok := parseCompareCandidate("https://bitbucket.example.com/projects/NME/repos/micro-payment/compare/diff?sourceBranch=refs%2Fheads%2Ffeature%2FCRMMS-1&targetBranch=refs%2Fheads%2Fdevelopment")
	if !ok {
		t.Fatal("expected a compare candidate")
	}
	if candidate.SourceBranch != "feature/CRMMS-1" || candidate.TargetBranch != "development" {
		t.Fatalf("expected stripped refs, got %s -> %s", candidate.SourceBranch, candidate.TargetBranch)
	}
}

func TestParseCompareCandidateReadsACompareURL(t *testing.T) {
	candidate, ok := parseCompareCandidate("https://bitbucket.example.com/projects/NME/repos/micro-payment/compare/diff?sourceBranch=feat%2Fquery-review&targetBranch=development")
	if !ok {
		t.Fatal("expected a compare candidate")
	}
	if candidate.ProjectKey != "NME" || candidate.RepoSlug != "micro-payment" {
		t.Fatalf("expected NME/micro-payment, got %s/%s", candidate.ProjectKey, candidate.RepoSlug)
	}
	if candidate.SourceBranch != "feat/query-review" || candidate.TargetBranch != "development" {
		t.Fatalf("expected feat/query-review -> development, got %s -> %s", candidate.SourceBranch, candidate.TargetBranch)
	}
	if candidate.PullRequestID != "" {
		t.Fatalf("expected a compare candidate, got a pull request candidate %+v", candidate)
	}
	if candidate.Origin != "https://bitbucket.example.com" {
		t.Fatalf("expected the origin, got %q", candidate.Origin)
	}
}

func TestParseCompareCandidateReadsAPullRequestURL(t *testing.T) {
	candidate, ok := parseCompareCandidate("https://bitbucket.example.com/projects/NME/repos/micro-payment/pull-requests/421/overview")
	if !ok {
		t.Fatal("expected a pull request candidate")
	}
	if candidate.PullRequestID != "421" {
		t.Fatalf("expected pull request 421, got %+v", candidate)
	}
	if candidate.SourceBranch != "" || candidate.TargetBranch != "" {
		t.Fatalf("expected a pull request candidate without branches, got %+v", candidate)
	}
}

func TestParseCompareCandidateMapsFromAndToParameters(t *testing.T) {
	candidate, ok := parseCompareCandidate("https://bitbucket.example.com/projects/NME/repos/micro-payment/compare/diff?from=development&to=hotfix%2Fpn")
	if !ok {
		t.Fatal("expected a compare candidate")
	}
	if candidate.SourceBranch != "hotfix/pn" || candidate.TargetBranch != "development" {
		t.Fatalf("expected to=source and from=target, got %s -> %s", candidate.SourceBranch, candidate.TargetBranch)
	}
}

func TestParseCompareCandidateUnescapesHTML(t *testing.T) {
	candidate, ok := parseCompareCandidate("https://bitbucket.example.com/projects/NME/repos/micro-payment/compare/diff?sourceBranch=feat%2Fx&amp;targetBranch=development")
	if !ok {
		t.Fatal("expected the entity-encoded candidate to parse")
	}
	if candidate.TargetBranch != "development" {
		t.Fatalf("expected the unescaped target branch, got %q", candidate.TargetBranch)
	}
}

func TestParseCompareCandidateIgnoresUnrelatedLinks(t *testing.T) {
	for _, rawLink := range []string{
		"https://confluence.example.com/display/DEV/Query-Review",
		"https://bitbucket.example.com/projects/NME/repos/micro-payment/compare",
		"not a url",
		"",
	} {
		if _, ok := parseCompareCandidate(rawLink); ok {
			t.Fatalf("expected %q to be ignored", rawLink)
		}
	}
}

func TestLinkCandidateReadsTheFirstParsableLinkInACell(t *testing.T) {
	cell := `<a href="https://confluence.example.com/display/DEV">page</a><a href="https://bitbucket.example.com/projects/NME/repos/micro-payment/compare/diff?sourceBranch=feat%2Fx&amp;targetBranch=development">compare</a>`

	candidate, ok := linkCandidate(cell)
	if !ok {
		t.Fatal("expected the compare link to be found")
	}
	if candidate.RepoSlug != "micro-payment" || candidate.SourceBranch != "feat/x" {
		t.Fatalf("unexpected candidate %+v", candidate)
	}
}

func TestLinkCandidateReadsAPlainURL(t *testing.T) {
	cell := `https://bitbucket.example.com/projects/NME/repos/micro-payment/compare/diff?sourceBranch=feat%2Fx&targetBranch=development`

	if _, ok := linkCandidate(cell); !ok {
		t.Fatal("expected the plain URL to be found")
	}
}

func TestLinkCandidateReturnsFalseWithoutAParsableLink(t *testing.T) {
	if _, ok := linkCandidate("<td>no link here</td>"); ok {
		t.Fatal("expected no candidate")
	}
}
