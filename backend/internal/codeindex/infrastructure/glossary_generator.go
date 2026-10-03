package infrastructure

import (
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"

	"lunar/backend/internal/codeindex/domain"
)

const generatedGlossaryFileName = "glossary.generated.yaml"

const maxValuesPerNote = 25

const maxMembersInTerm = 10

var (
	rEnumAnnotation = regexp.MustCompile(`^[ \t]*//[ \t]*@enum:[ \t]*([A-Za-z0-9_.\-]+)([ \t]*-[ \t]*(.*))?$`)
	rIotaOffset     = regexp.MustCompile(`^iota \+ ([0-9]+)$`)
)

type GlossaryGenerator struct {
	customPath string
}

type enumMember struct {
	name  string
	value string
	label string
}

type enumValueState struct {
	ordinal      int
	inherited    string
	isIota       bool
	hasInherited bool
}

func NewGlossaryGenerator(customPath ...string) *GlossaryGenerator {
	configuredPath := ""
	if len(customPath) > 0 {
		configuredPath = customPath[0]
	}
	return &GlossaryGenerator{customPath: configuredPath}
}

func ResolveGeneratedGlossaryPath(manualGlossaryPath string) string {
	return filepath.Join(filepath.Dir(manualGlossaryPath), generatedGlossaryFileName)
}

func (g *GlossaryGenerator) Generate(chunks map[string][]domain.CodeChunk) (int, error) {
	notes := collectEnumNotes(chunks)
	if len(notes) == 0 {
		return 0, nil
	}
	slices.SortFunc(notes, func(left, right domain.GlossaryNote) int {
		return strings.Compare(left.Term, right.Term)
	})

	contents, err := yaml.Marshal(domain.Glossary{Notes: notes})
	if err != nil {
		return 0, fmt.Errorf("Generate: marshal generated glossary: %w", err)
	}

	manualGlossaryPath, err := ResolveGlossaryPath(g.customPath)
	if err != nil {
		return 0, fmt.Errorf("Generate: %w", err)
	}
	destination := ResolveGeneratedGlossaryPath(manualGlossaryPath)
	if err := os.WriteFile(destination, contents, 0o644); err != nil {
		return 0, fmt.Errorf("Generate: write %s: %w", destination, err)
	}
	return len(notes), nil
}

func collectEnumNotes(chunks map[string][]domain.CodeChunk) []domain.GlossaryNote {
	seenTerms := make(map[string]bool)
	notes := make([]domain.GlossaryNote, 0)

	for _, repository := range slices.Sorted(maps.Keys(chunks)) {
		repositoryChunks := slices.Clone(chunks[repository])
		slices.SortStableFunc(repositoryChunks, func(left, right domain.CodeChunk) int {
			if left.FilePath != right.FilePath {
				return strings.Compare(left.FilePath, right.FilePath)
			}
			return strings.Compare(left.RawContent, right.RawContent)
		})

		for _, chunk := range repositoryChunks {
			if chunk.ChunkType != string(domain.ChunkTypeEnumDef) {
				continue
			}
			note, isCandidate := buildEnumNote(chunk)
			if !isCandidate || seenTerms[note.Term] {
				continue
			}
			seenTerms[note.Term] = true
			notes = append(notes, note)
		}
	}
	return notes
}

func buildEnumNote(chunk domain.CodeChunk) (domain.GlossaryNote, bool) {
	key, annotationDescription, members, isParsed := parseEnumDeclaration(chunk.RawContent)
	if !isParsed {
		return domain.GlossaryNote{}, false
	}

	memberNames := make([]string, 0, len(members))
	for _, member := range members {
		memberNames = append(memberNames, member.name)
	}
	if len(memberNames) > maxMembersInTerm {
		memberNames = append(slices.Clone(memberNames[:maxMembersInTerm]), "...")
	}

	valueLimit := min(len(members), maxValuesPerNote)
	valuePairs := make([]string, 0, valueLimit)
	for _, member := range members[:valueLimit] {
		valuePairs = append(valuePairs, fmt.Sprintf("%s = %s", member.value, member.label))
	}
	if len(members) > maxValuesPerNote {
		valuePairs = append(valuePairs, fmt.Sprintf("... (+%d more values)", len(members)-maxValuesPerNote))
	}

	explanation := fmt.Sprintf(
		"Enum candidate from source code (%s/%s). %s: %s.",
		chunk.RepoName, chunk.FilePath, key, strings.Join(valuePairs, ", "),
	)
	if annotationDescription != "" {
		explanation += " Note: " + annotationDescription + "."
	}

	term := fmt.Sprintf("enum %s %s %s", key, chunk.RepoName, strings.Join(memberNames, " "))
	return domain.GlossaryNote{Term: term, Explanation: explanation}, true
}

func parseEnumDeclaration(raw string) (string, string, []enumMember, bool) {
	lines := strings.Split(strings.TrimSpace(raw), "\n")
	annotationKey := ""
	annotationDescription := ""
	bodyStart := 0
	for bodyStart < len(lines) && strings.HasPrefix(strings.TrimSpace(lines[bodyStart]), "//") {
		matches := rEnumAnnotation.FindStringSubmatch(strings.TrimSpace(lines[bodyStart]))
		if matches != nil && annotationKey == "" {
			annotationKey = strings.TrimSpace(matches[1])
			annotationDescription = strings.TrimSpace(matches[3])
		}
		bodyStart++
	}
	if bodyStart >= len(lines) {
		return "", "", nil, false
	}

	body := strings.Join(lines[bodyStart:], "\n")
	openIndex := strings.Index(body, "(")
	closeIndex := strings.LastIndex(body, ")")
	memberLines := make([]string, 0)
	if openIndex >= 0 && closeIndex > openIndex {
		memberLines = strings.Split(body[openIndex+1:closeIndex], "\n")
	} else {
		single := strings.TrimSpace(body)
		if strings.HasPrefix(single, "const ") || strings.HasPrefix(single, "const\t") {
			single = strings.TrimSpace(single[len("const"):])
		}
		if single != "" {
			memberLines = []string{single}
		}
	}

	members := collectEnumMembers(memberLines)
	if len(members) == 0 {
		return "", "", nil, false
	}

	if annotationKey == "" {
		annotationKey = deriveEnumKey(members)
	}
	if annotationKey == "" {
		return "", "", nil, false
	}
	return annotationKey, annotationDescription, members, true
}

func collectEnumMembers(lines []string) []enumMember {
	members := make([]enumMember, 0)
	state := enumValueState{}

	for _, line := range lines {
		code, comment := splitTrailingComment(line)
		if code == "" {
			continue
		}

		assignmentIndex := strings.Index(code, "=")
		if assignmentIndex < 0 {
			name := firstNameToken(code)
			switch {
			case state.isIota:
				state.ordinal++
				members = append(members, newEnumMember(name, strconv.Itoa(state.ordinal), comment))
			case state.hasInherited:
				members = append(members, newEnumMember(name, state.inherited, comment))
			}
			continue
		}

		name := firstNameToken(code[:assignmentIndex])
		value, nextState, isResolved := state.resolved(strings.TrimSpace(code[assignmentIndex+1:]))
		if name == "" || !isResolved {
			state = enumValueState{}
			continue
		}
		state = nextState
		members = append(members, newEnumMember(name, value, comment))
	}

	return members
}

func newEnumMember(name, value, comment string) enumMember {
	if comment == "" {
		return enumMember{name: name, value: value, label: name}
	}
	return enumMember{name: name, value: value, label: comment}
}

func (s enumValueState) resolved(rightHandSide string) (string, enumValueState, bool) {
	switch {
	case rightHandSide == "iota":
		return "0", enumValueState{isIota: true}, true
	case rIotaOffset.MatchString(rightHandSide):
		offset, err := strconv.Atoi(rIotaOffset.FindStringSubmatch(rightHandSide)[1])
		if err != nil {
			return "", enumValueState{}, false
		}
		return strconv.Itoa(offset), enumValueState{ordinal: offset, isIota: true}, true
	case isQuotedLiteral(rightHandSide):
		value := rightHandSide[1 : len(rightHandSide)-1]
		return value, enumValueState{inherited: value, hasInherited: true}, true
	default:
		if _, err := strconv.Atoi(rightHandSide); err != nil {
			return "", enumValueState{}, false
		}
		return rightHandSide, enumValueState{inherited: rightHandSide, hasInherited: true}, true
	}
}

func splitTrailingComment(line string) (string, string) {
	trimmed := strings.TrimSpace(line)
	if commentIndex := strings.LastIndex(trimmed, "//"); commentIndex >= 0 {
		return strings.TrimSpace(trimmed[:commentIndex]), strings.TrimSpace(trimmed[commentIndex+2:])
	}
	return trimmed, ""
}

func isQuotedLiteral(value string) bool {
	if len(value) < 2 {
		return false
	}
	return value[len(value)-1] == value[0] && strings.ContainsRune("\"'`", rune(value[0]))
}

func firstNameToken(value string) string {
	fields := strings.Fields(value)
	if len(fields) == 0 {
		return ""
	}
	return fields[0]
}

func deriveEnumKey(members []enumMember) string {
	if len(members) < 2 {
		return ""
	}

	prefix := splitCamelCase(members[0].name)
	for _, member := range members[1:] {
		words := splitCamelCase(member.name)
		limit := min(len(prefix), len(words))
		shared := make([]string, 0, limit)
		for index := 0; index < limit && prefix[index] == words[index]; index++ {
			shared = append(shared, prefix[index])
		}
		prefix = shared
		if len(prefix) == 0 {
			return ""
		}
	}
	return strings.ToLower(strings.Join(prefix, "_"))
}
