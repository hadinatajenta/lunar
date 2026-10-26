export const mockPushes = [
  {
    id: "push-1",
    repo: "BRI/mms-frontend",
    repo_mark: "FE",
    branch: "feat/auth-refresh",
    commit_count: 3,
    relative_time: "10m ago",
    status: "ready",
    ai_badge: "AI Ready",
    ai_summary: "Authentication session handling refresh with token rotation.",
    suggested_title: "feat(auth): add session rotation"
  },
  {
    id: "push-2",
    repo: "BRI/mms-backend",
    repo_mark: "BE",
    branch: "fix/cors-preflight",
    commit_count: 1,
    relative_time: "25m ago",
    status: "ready",
    ai_badge: "AI Clean",
    ai_summary: "Fix CORS preflight headers on standard net/http mux.",
    suggested_title: "fix(cors): allow credentials and expose headers"
  },
  {
    id: "push-3",
    repo: "BRI/service-map",
    repo_mark: "SM",
    branch: "chore/update-deps",
    commit_count: 2,
    relative_time: "3d ago",
    status: "stale",
    ai_badge: "AI Stale",
    ai_summary: "Dependency updates pending merge for more than 48 hours.",
    suggested_title: "chore: update dependencies"
  }
]

export const mockPRs = [
  {
    id: "#184",
    number: 184,
    repo: "BRI/mms-frontend",
    title: "feat(auth): LUN-421 support biometric and session renewal",
    source_branch: "feat/auth-refresh",
    target_branch: "main",
    status: "open",
    updated_relative: "12m ago",
    author: "Lunar Developer",
    files_count: 4,
    lines_added: 124,
    lines_deleted: 18,
    is_assigned_to_me: true,
    is_ai_flagged: true
  },
  {
    id: "#188",
    number: 188,
    repo: "BRI/mms-backend",
    title: "fix(db): SQLite WAL busy timeout configuration",
    source_branch: "fix/sqlite-wal",
    target_branch: "main",
    status: "open",
    updated_relative: "1h ago",
    author: "Current User",
    files_count: 2,
    lines_added: 42,
    lines_deleted: 6,
    is_assigned_to_me: true,
    is_ai_flagged: true
  },
  {
    id: "#191",
    number: 191,
    repo: "BRI/service-map",
    title: "refactor(fts5): optimize tokenization query speed",
    source_branch: "perf/fts5-tokenize",
    target_branch: "main",
    status: "open",
    updated_relative: "3h ago",
    author: "Alex",
    files_count: 5,
    lines_added: 88,
    lines_deleted: 34,
    is_assigned_to_me: false,
    is_ai_flagged: true
  },
  {
    id: "#195",
    number: 195,
    repo: "BRI/timesheet-core",
    title: "docs: update API contract and swagger specs",
    source_branch: "docs/api-specs",
    target_branch: "main",
    status: "open",
    updated_relative: "5h ago",
    author: "Sarah",
    files_count: 1,
    lines_added: 15,
    lines_deleted: 2,
    is_assigned_to_me: false,
    is_ai_flagged: false
  }
]

export const mockDiff = {
  pr_id: "184",
  repo: "BRI/mms-frontend",
  total_added: 124,
  total_deleted: 18,
  files: [
    {
      old_path: "src/auth/session.ts",
      new_path: "src/auth/session.ts",
      status: "modified",
      additions: 124,
      deletions: 18,
      hunks: [
        {
          header: "@@ -40,6 +40,12 @@",
          lines: [
            "- const prev = null",
            "+ const prev = sessionManager.get()",
            "+ The compatibility layer is implemented cleanly"
          ]
        }
      ]
    }
  ]
}

export const mockAIReview = {
  pr_id: "#184",
  model: "DeepSeek-V4 Pro (Thinking)",
  prompt_version: "pr-review-v2",
  summary:
    "The session refresh path reads the previous session through the manager but never persists the rotated token, so a failed write would leave memory and storage out of step.",
  findings: [
    {
      severity: "blocker",
      category: "correctness",
      file: "src/auth/session.ts",
      line: 40,
      title: "Rotate the session only after the new token is persisted",
      explanation:
        "The previous session is read before the rotated token reaches storage, so a failed write leaves the in-memory session ahead of what the next request will load.",
      suggested_fix: "Await the storage write, then assign the new session to the manager.",
      confidence: "confirmed",
      evidence: "+ const prev = sessionManager.get()"
    },
    {
      severity: "suggestion",
      category: "testing",
      file: "src/auth/session.ts",
      line: 41,
      title: "Cover the compatibility layer with a rejection test",
      explanation:
        "The diff states that the compatibility layer is implemented cleanly but adds no test that exercises a rejected refresh token.",
      suggested_fix: "Add a unit test that returns an expired refresh token and asserts the fallback sign-out.",
      confidence: "assumption",
      evidence: "+ The compatibility layer is implemented cleanly"
    },
    {
      severity: "nitpick",
      category: "readability",
      file: "src/auth/session.ts",
      line: 40,
      title: "Name the previous session explicitly",
      explanation: "prev does not say what the manager returns once the session carries both tokens.",
      suggested_fix: "Rename prev to previousSession.",
      confidence: "confirmed",
      evidence: "+ const prev = sessionManager.get()"
    }
  ],
  assumptions: [
    "The refresh token is rotated server side, so the client only needs to persist the new value."
  ],
  strengths: [
    "The compatibility layer keeps the existing session shape, so no call site needs to change."
  ],
  recommendation: "request_changes",
  generated_comment: [
    "## Code review: feat(auth): LUN-421 support biometric and session renewal",
    "",
    "**Pull request:** #184 in `BRI/mms-frontend`, `feat/auth-refresh` into `main`",
    "**Reviewed:** 1 file, +124 -18",
    "**Verdict:** 🔴 Request changes",
    "",
    "The session refresh path reads the previous session through the manager but never persists the rotated token, so a failed write would leave memory and storage out of step.",
    "",
    "### 🔴 Blocker",
    "",
    "**1. `src/auth/session.ts:40` Rotate the session only after the new token is persisted (correctness)**",
    "",
    "- Why: The previous session is read before the rotated token reaches storage, so a failed write leaves the in-memory session ahead of what the next request will load.",
    "- Fix: Await the storage write, then assign the new session to the manager.",
    "- Evidence: `+ const prev = sessionManager.get()`",
    "",
    "### 🟡 Suggestion",
    "",
    "**1. `src/auth/session.ts:41` Cover the compatibility layer with a rejection test (testing)**",
    "",
    "- Why: The diff states that the compatibility layer is implemented cleanly but adds no test that exercises a rejected refresh token.",
    "- Fix: Add a unit test that returns an expired refresh token and asserts the fallback sign-out.",
    "- Evidence: `+ The compatibility layer is implemented cleanly`",
    "- Confidence: assumption, not confirmed from the diff",
    "",
    "### 🟢 Nitpick",
    "",
    "**1. `src/auth/session.ts:40` Name the previous session explicitly (readability)**",
    "",
    "- Why: prev does not say what the manager returns once the session carries both tokens.",
    "- Fix: Rename prev to previousSession.",
    "- Evidence: `+ const prev = sessionManager.get()`",
    "",
    "### ✅ What was done well",
    "",
    "- The compatibility layer keeps the existing session shape, so no call site needs to change.",
    "",
    "### ⚠️ Assumptions",
    "",
    "- The refresh token is rotated server side, so the client only needs to persist the new value.",
    "",
    "---",
    "",
    "Reviewed with DeepSeek-V4 Pro (Thinking). Findings marked as assumptions could not be confirmed from the diff alone."
  ].join("\n"),
  truncated: false
}
