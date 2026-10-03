package application

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"lunar/backend/internal/agent/domain"
	codeindexdomain "lunar/backend/internal/codeindex/domain"
)

const (
	domainContextHeader          = "## REPOSITORY & DOMAIN KNOWLEDGE (AUTO-GENERATED)"
	historyWindowSize            = 10
	maxAssistantHistoryLength    = 2500
	truncationSuffix             = "..."
	requiredToolChoice           = "required"
	autoToolChoice               = "auto"
	noneToolChoice               = "none"
	synthesisPhaseName           = "synthesis"
	agentRole                    = "assistant"
	toolRole                     = "tool"
	userRole                     = "user"
	systemRole                   = "system"
	reasoningFallbackText        = "Evaluating tools for verification."
	requiredEvidenceReminderText = "WARNING: Every technical analysis and verification must be grounded in real evidence from source code, git diffs, or the knowledge base. You have not called any tool yet. Call the relevant tool immediately before drawing conclusions."
)

const baseSystemPrompt = `You are an AI engineering copilot with cross-domain expertise, helping developers with:
1. Architecture & Source Code: tracing API request flows, broker events, dependency graphs, and local git diffs.
2. Jira Ticket Management: listing active issues, reading acceptance criteria, creating and updating subtasks, and transitioning issue status.
3. Bitbucket Code Review: reading pull request diffs to mitigate defect and security risk.
4. DBA Query Review: inspecting database queries, target tables, and pull request comparison links.

OPERATIONAL RULES:
- Always use the available tools to gather concrete evidence from source code, git diffs, or the knowledge base before concluding any technical answer.
- Only serve questions and tasks inside the scope of this workspace and its engineering operations.
- Refuse general chit-chat, poetry, recipes, school assignments, politics, and anything outside that scope.
- Answer in professional English using standard technical terminology.

OUTPUT FORMAT RULES:
- Never use LaTeX mathematical notation. Use plain text arrows such as '->' or the arrow character, or connective words.
- Use correctly ordered numbered lists. Never repeat a number or duplicate numbering.
- For endpoint flow or service integration questions, include a visual summary as a fenced 'flow' code block containing a single 'node -> node -> node' chain with at least three nodes.`

var domainExpertiseOrder = []string{
	DomainServiceMap,
	DomainJira,
	DomainBitbucket,
	DomainQueryReview,
}

var domainExpertise = map[string]string{
	DomainServiceMap: `SERVICE MAP EXPERTISE:
- Trace request flows across microservices, HTTP clients, message broker topics, and route definitions.
- Prefer indexed code search over guessing; read file content before quoting it.
- Cite repository, file, and snippet for every architectural claim.`,
	DomainJira: `JIRA EXPERTISE:
- Read the ticket and its acceptance criteria before proposing an implementation.
- Derive subtask summaries from real activity evidence rather than assumptions.
- Confirm the parent issue and squad before creating or updating a subtask.`,
	DomainBitbucket: `BITBUCKET EXPERTISE:
- Read the pull request diff before commenting on correctness, security, or performance.
- Separate blocking defects from stylistic suggestions.`,
	DomainQueryReview: `QUERY REVIEW EXPERTISE:
- Inspect the query, its target tables, and the comparison link before judging index usage.
- Report the database, table, and action type for every reviewed statement.`,
}

const defaultSynthesisPrompt = `Based on all the code and dependency analysis above, produce a comprehensive, structured, and accurate answer. Include service names, endpoints, broker topics, and the request flow step by step. When relevant, add a sequence diagram using mermaid syntax. Do not use LaTeX mathematical notation; use '->' or the arrow character instead.`

func BuildSystemPrompt(domains []string, dynamicContext string) string {
	sections := []string{baseSystemPrompt}

	selectedDomains := domains
	if len(selectedDomains) == 0 {
		selectedDomains = domainExpertiseOrder
	}

	allowedDomains := make(map[string]bool, len(selectedDomains))
	for _, domainName := range selectedDomains {
		allowedDomains[strings.ToLower(strings.TrimSpace(domainName))] = true
	}

	for _, domainName := range domainExpertiseOrder {
		if !allowedDomains[domainName] {
			continue
		}
		sections = append(sections, domainExpertise[domainName])
	}

	return ComposeSystemPrompt(strings.Join(sections, "\n\n"), dynamicContext)
}

func ComposeSystemPrompt(basePrompt string, dynamicContext string) string {
	context := strings.TrimSpace(dynamicContext)
	if context == "" {
		return basePrompt
	}
	return basePrompt + "\n\n" + domainContextHeader + "\n" + context + "\n"
}

func DefaultSynthesisPrompt() string {
	return defaultSynthesisPrompt
}

func buildInitialMessages(question codeindexdomain.AgentQuestion) []domain.ChatMessage {
	systemPrompt := BuildSystemPrompt(question.Domains, "")
	messages := []domain.ChatMessage{{Role: systemRole, Content: domain.TextMessage(systemPrompt)}}

	history := question.History
	if len(history) > historyWindowSize {
		history = history[len(history)-historyWindowSize:]
	}
	for _, turn := range history {
		role := strings.ToLower(strings.TrimSpace(turn.Role))
		if role != userRole && role != agentRole {
			continue
		}
		content := strings.TrimSpace(turn.Content)
		if content == "" {
			continue
		}
		if role == agentRole && len(content) > maxAssistantHistoryLength {
			content = content[:maxAssistantHistoryLength] + truncationSuffix
		}
		messages = append(messages, domain.ChatMessage{Role: role, Content: domain.TextMessage(content)})
	}

	messages = append(messages, domain.ChatMessage{Role: userRole, Content: domain.TextMessage(question.Question)})
	return messages
}

func (a *Agent) determineToolChoice(question string, round int, executedTools []string, hasTools bool) (string, bool) {
	if !hasTools {
		return noneToolChoice, false
	}
	if a.ToolChoiceFunc != nil {
		return a.ToolChoiceFunc(question, round, executedTools)
	}
	if isNonDomainQuestion(question) {
		return noneToolChoice, false
	}
	if len(executedTools) == 0 {
		return requiredToolChoice, true
	}
	return autoToolChoice, false
}

func isNonDomainQuestion(question string) bool {
	normalized := strings.ToLower(strings.TrimSpace(question))

	conversational := []string{
		"hello", "hi", "hey",
		"good morning", "good afternoon", "good evening",
		"how are you", "thank you", "thanks",
		"who are you", "what can you do", "what features",
	}
	for _, greeting := range conversational {
		if matchesGreeting(normalized, greeting) {
			return true
		}
	}

	offTopicKeywords := []string{
		"poem", "poetry", "song lyrics", "short story", "joke", "fairy tale",
		"recipe", "how to cook", "how to bake",
		"horoscope", "zodiac", "today's weather",
		"who is the president", "politics", "election",
		"health tips", "medicine for", "disease symptoms",
		"college assignment", "thesis", "school homework",
		"make a game", "build a calculator", "flutter app", "python script",
		"ignore previous instructions", "forget your system prompt", "pretend you are",
	}
	for _, keyword := range offTopicKeywords {
		if strings.Contains(normalized, keyword) {
			return true
		}
	}
	return false
}

func matchesGreeting(normalizedQuestion string, greeting string) bool {
	if normalizedQuestion == greeting {
		return true
	}
	for _, separator := range []string{" ", ",", "!", "?"} {
		if strings.HasPrefix(normalizedQuestion, greeting+separator) {
			return true
		}
	}
	return false
}

var toolActionLabels = map[string]string{
	"search_code":              "Searching indexed code",
	"get_file_content":         "Reading source file",
	"get_route_details":        "Tracing route and downstream calls",
	"get_service_contract":     "Inspecting service manifest",
	"find_symbol_references":   "Finding symbol references",
	"get_service_routes":       "Listing service routes",
	"get_service_dependencies": "Mapping service dependency graph",
	"list_services":            "Listing services",
	"get_domain_knowledge":     "Searching domain knowledge",
	"get_git_diff":             "Reading local git diff",
	"grep_raw_fallback":        "Grepping source tree",
	"list_my_jira_issues":      "Listing assigned Jira issues",
	"get_jira_issue_detail":    "Reading Jira issue detail",
	"get_active_sprint_issues": "Listing active sprint issues",
	"create_jira_subtask":      "Creating Jira subtask",
	"update_jira_subtask":      "Updating Jira subtask",
	"transition_jira_issue":    "Transitioning Jira issue",
	"get_pull_request_diff":    "Reading pull request diff",
	"get_query_review_context": "Loading query review context",
}

func formatToolAction(call domain.ToolCall) string {
	label, exists := toolActionLabels[call.Function.Name]
	if !exists {
		return "Running " + call.Function.Name
	}

	arguments := make(map[string]interface{})
	if err := json.Unmarshal([]byte(call.Function.Arguments), &arguments); err != nil || len(arguments) == 0 {
		return label
	}
	return label + " (" + formatToolArguments(arguments) + ")"
}

func formatToolArguments(arguments map[string]interface{}) string {
	keys := make([]string, 0, len(arguments))
	for key := range arguments {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s=%v", key, arguments[key]))
	}
	return strings.Join(parts, ", ")
}

func (s *runState) synthesize(
	ctx context.Context,
	client domain.AIClient,
	rounds int,
	isTruncated bool,
	startedAt time.Time,
) (*codeindexdomain.AgentAnswer, error) {
	s.messages = append(s.messages, domain.ChatMessage{Role: userRole, Content: domain.TextMessage(DefaultSynthesisPrompt())})
	if err := s.emit(Event{Type: EventPhaseChange, Payload: PhaseChangePayload{Phase: synthesisPhaseName}}); err != nil {
		return nil, fmt.Errorf("emit phase change: %w", err)
	}

	var answerBuilder strings.Builder
	filter := newContentFilter()
	onChunk := func(chunk string) error {
		visible := filter.accept(chunk)
		if visible == "" {
			return nil
		}
		answerBuilder.WriteString(visible)
		if err := s.emit(Event{Type: EventTextDelta, Payload: TextDeltaPayload{Text: visible}}); err != nil {
			return fmt.Errorf("emit text delta: %w", err)
		}
		return nil
	}
	onReasoning := func(chunk string) error {
		payload := ReasoningDeltaPayload{Text: chunk}
		if err := s.emit(Event{Type: EventReasoningDelta, Payload: payload}); err != nil {
			return fmt.Errorf("emit reasoning delta: %w", err)
		}
		return nil
	}

	if err := streamAnswer(ctx, client, s.messages, s.model, onReasoning, onChunk); err != nil {
		return nil, fmt.Errorf("stream final answer: %w", err)
	}
	return s.completeRun(rounds, isTruncated, answerBuilder.String(), startedAt)
}
