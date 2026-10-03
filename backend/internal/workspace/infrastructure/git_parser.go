package infrastructure

import (
	"fmt"
	"strconv"
	"strings"

	"lunar/backend/internal/workspace/domain"
)

const (
	porcelainArrow        = " -> "
	numstatArrow          = " => "
	commitFieldDelimiter  = "\x1f"
	porcelainPrefixLength = 3
	shortHashLength       = 7
)

type porcelainEntry struct {
	Status       domain.ChangeStatus
	Path         string
	OriginalPath string
}

type lineDelta struct {
	Added   int
	Deleted int
}

func parsePorcelainStatus(output string) []porcelainEntry {
	lines := strings.Split(output, "\n")
	entries := make([]porcelainEntry, 0, len(lines))

	for _, line := range lines {
		if len(line) <= porcelainPrefixLength || line[2] != ' ' {
			continue
		}

		status := mapPorcelainStatus(line[:2])
		path := normalizeGitPath(line[porcelainPrefixLength:])
		originalPath := ""

		if status == domain.StatusRenamed {
			if original, renamed, found := splitRenamePath(path); found {
				originalPath = normalizeGitPath(original)
				path = normalizeGitPath(renamed)
			}
		}

		entries = append(entries, porcelainEntry{Status: status, Path: path, OriginalPath: originalPath})
	}

	return entries
}

func mapPorcelainStatus(code string) domain.ChangeStatus {
	primary := code[0]
	if primary == ' ' || primary == '?' {
		primary = code[1]
	}

	switch primary {
	case 'A':
		return domain.StatusAdded
	case 'D':
		return domain.StatusDeleted
	case 'R', 'C':
		return domain.StatusRenamed
	case '?':
		return domain.StatusUntracked
	default:
		return domain.StatusModified
	}
}

func splitRenamePath(path string) (string, string, bool) {
	isQuoted := false
	for index := 0; index+len(porcelainArrow) <= len(path); index++ {
		switch {
		case path[index] == '\\' && isQuoted:
			index++
		case path[index] == '"':
			isQuoted = !isQuoted
		case !isQuoted && strings.HasPrefix(path[index:], porcelainArrow):
			return path[:index], path[index+len(porcelainArrow):], true
		}
	}
	return "", "", false
}

func normalizeGitPath(rawPath string) string {
	trimmed := strings.TrimSpace(rawPath)
	if len(trimmed) >= 2 && strings.HasPrefix(trimmed, `"`) && strings.HasSuffix(trimmed, `"`) {
		if unquoted, err := strconv.Unquote(trimmed); err == nil {
			return unquoted
		}
	}
	return trimmed
}

func parseNumstat(output string) map[string]lineDelta {
	deltas := make(map[string]lineDelta)

	for _, line := range strings.Split(output, "\n") {
		fields := strings.SplitN(line, "\t", 3)
		if len(fields) != 3 {
			continue
		}

		delta := lineDelta{
			Added:   parseNumstatCount(fields[0]),
			Deleted: parseNumstatCount(fields[1]),
		}

		path, originalPath := splitNumstatPath(fields[2])
		deltas[path] = delta
		if originalPath != "" {
			deltas[originalPath] = delta
		}
	}

	return deltas
}

func parseNumstatCount(value string) int {
	count, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return 0
	}
	return count
}

func splitNumstatPath(rawPath string) (string, string) {
	path := normalizeGitPath(rawPath)
	if !strings.Contains(path, numstatArrow) {
		return path, ""
	}

	openIndex := strings.Index(path, "{")
	closeIndex := strings.LastIndex(path, "}")
	if openIndex >= 0 && closeIndex > openIndex {
		inner := path[openIndex+1 : closeIndex]
		if arrowIndex := strings.Index(inner, numstatArrow); arrowIndex >= 0 {
			prefix := path[:openIndex]
			suffix := path[closeIndex+1:]
			original := prefix + inner[:arrowIndex] + suffix
			renamed := prefix + inner[arrowIndex+len(numstatArrow):] + suffix
			return renamed, original
		}
	}

	arrowIndex := strings.LastIndex(path, numstatArrow)
	return path[arrowIndex+len(numstatArrow):], path[:arrowIndex]
}

func parseCommitInfo(output string) (domain.CommitInfo, error) {
	fields := strings.Split(strings.TrimRight(output, "\n"), commitFieldDelimiter)
	if len(fields) != 4 {
		return domain.CommitInfo{}, fmt.Errorf("unexpected commit output")
	}

	return domain.CommitInfo{
		Hash:         abbreviateHash(fields[0]),
		Author:       fields[1],
		RelativeTime: fields[2],
		Subject:      fields[3],
	}, nil
}

func abbreviateHash(hash string) string {
	if len(hash) <= shortHashLength {
		return hash
	}
	return hash[:shortHashLength]
}
