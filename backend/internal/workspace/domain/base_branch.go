package domain

import "sort"

type BaseBranchSource string

const (
	BaseBranchSourceRemembered       BaseBranchSource = "remembered"
	BaseBranchSourceBitbucketDefault BaseBranchSource = "bitbucket_default"
	BaseBranchSourceTrusted          BaseBranchSource = "trusted"
	BaseBranchSourceOther            BaseBranchSource = "other"
)

var trustedBaseBranchNames = []string{"master", "main", "release/production", "develop", "development"}

func IsTrustedBaseBranch(name string) bool {
	for _, trusted := range trustedBaseBranchNames {
		if name == trusted {
			return true
		}
	}
	return false
}

type BaseBranchCandidate struct {
	Name               string
	CommitAt           int64
	LastCommitRelative string
}

type BaseBranchCandidates struct {
	IsCloned      bool
	RemoteDefault string
	Branches      []BaseBranchCandidate
}

type BaseBranchOption struct {
	Name               string
	LastCommitRelative string
	Source             BaseBranchSource
}

type BaseBranchOptions struct {
	Repo             string
	IsCloned         bool
	Remembered       string
	BitbucketDefault string
	Options          []BaseBranchOption
	OtherBranchCount int
}

func OrderBaseBranchOptions(candidates []BaseBranchCandidate, remembered string, remoteDefault string) []BaseBranchOption {
	ordered := make([]BaseBranchOption, 0, len(candidates))
	emitted := make(map[string]struct{}, len(candidates))

	appendByName := func(name string) {
		candidate, found := findCandidate(candidates, name)
		if !found {
			return
		}
		if _, done := emitted[candidate.Name]; done {
			return
		}
		emitted[candidate.Name] = struct{}{}
		ordered = append(ordered, BaseBranchOption{
			Name:               candidate.Name,
			LastCommitRelative: candidate.LastCommitRelative,
			Source:             baseBranchSource(candidate.Name, remembered, remoteDefault),
		})
	}

	appendByName(remembered)
	appendByName(remoteDefault)

	for _, group := range groupCandidatesByTrust(candidates) {
		for _, candidate := range group {
			appendByName(candidate.Name)
		}
	}

	return ordered
}

func groupCandidatesByTrust(candidates []BaseBranchCandidate) [][]BaseBranchCandidate {
	trusted := make([]BaseBranchCandidate, 0, len(candidates))
	other := make([]BaseBranchCandidate, 0, len(candidates))

	for _, candidate := range candidates {
		if IsTrustedBaseBranch(candidate.Name) {
			trusted = append(trusted, candidate)
			continue
		}
		other = append(other, candidate)
	}

	sortCandidatesByRecency(trusted)
	sortCandidatesByRecency(other)

	return [][]BaseBranchCandidate{trusted, other}
}

func sortCandidatesByRecency(candidates []BaseBranchCandidate) {
	sort.SliceStable(candidates, func(left int, right int) bool {
		if candidates[left].CommitAt != candidates[right].CommitAt {
			return candidates[left].CommitAt > candidates[right].CommitAt
		}
		return candidates[left].Name < candidates[right].Name
	})
}

func findCandidate(candidates []BaseBranchCandidate, name string) (BaseBranchCandidate, bool) {
	if name == "" {
		return BaseBranchCandidate{}, false
	}
	for _, candidate := range candidates {
		if candidate.Name == name {
			return candidate, true
		}
	}
	return BaseBranchCandidate{}, false
}

func baseBranchSource(name string, remembered string, remoteDefault string) BaseBranchSource {
	switch {
	case name == remembered && name != "":
		return BaseBranchSourceRemembered
	case name == remoteDefault && name != "":
		return BaseBranchSourceBitbucketDefault
	case IsTrustedBaseBranch(name):
		return BaseBranchSourceTrusted
	default:
		return BaseBranchSourceOther
	}
}
