export const mockConfiguredSecrets = {
  user_id: "test-user-id",
  has_jira_pat: true,
  jira_username: "developer",
  has_bitbucket_pat: true,
  bitbucket_username: "developer",
  has_confluence_pat: true,
  has_ai_keys: true,
  configured_ai_providers: ["deepseek"],
  updated_at: "Just now"
}

export const mockUnconfiguredSecrets = {
  user_id: "test-user-id",
  has_jira_pat: true,
  jira_username: "developer",
  has_bitbucket_pat: true,
  bitbucket_username: "developer",
  has_confluence_pat: true,
  has_ai_keys: false,
  configured_ai_providers: [],
  updated_at: "Just now"
}

export const mockChatSuccessResponse = {
  session_id: "session-test-1",
  message: {
    id: "msg-ast-test-1",
    chat_id: "session-test-1",
    role: "assistant",
    content: "Analyzed pull request #184 and linked Jira issue LUN-421. Architecture boundaries and lint constraints are verified cleanly.",
    reasoning: "Deep reasoning pass initiated. Evaluated session renewal flow and verified zero comments rule.",
    thinking_duration_ms: 1240,
    sources: [
      { type: "jira", label: "Jira BRI · Active Issues" },
      { type: "bitbucket", label: "Bitbucket BRI · PRs & Diffs" }
    ],
    created_at: "Just now"
  }
}
