export interface PushBranch {
  id: string
  repo: string
  repo_mark: string
  branch: string
  commit_count: number
  relative_time: string
  status: string
  ai_badge: string
  ai_summary: string
  suggested_title: string
}

export interface PullRequest {
  id: string
  number: number
  repo: string
  title: string
  description: string
  source_branch: string
  target_branch: string
  status: string
  updated_relative: string
  author: string
  files_count: number
  lines_added: number
  lines_deleted: number
  is_assigned_to_me: boolean
  is_ai_flagged: boolean
}

export interface DiffHunk {
  header: string
  old_start: number
  old_lines: number
  new_start: number
  new_lines: number
  lines: string[]
}

export interface DiffFile {
  old_path: string
  new_path: string
  status: string
  hunks: DiffHunk[]
  additions: number
  deletions: number
}

export interface PRDiff {
  pr_id: string
  repo: string
  total_added: number
  total_deleted: number
  files: DiffFile[]
}

export type ReviewSeverity = "blocker" | "suggestion" | "nitpick"

export type ReviewConfidence = "confirmed" | "assumption"

export type ReviewRecommendation = "approve" | "request_changes"

export interface ReviewFinding {
  severity: ReviewSeverity
  category: string
  file: string
  line: number
  title: string
  explanation: string
  suggested_fix: string
  confidence: ReviewConfidence
  evidence: string
}

export interface AICodeReview {
  pr_id: string
  model: string
  prompt_version: string
  summary: string
  findings: ReviewFinding[]
  assumptions: string[]
  strengths: string[]
  recommendation: ReviewRecommendation
  generated_comment: string
  truncated: boolean
}

export interface PRComment {
  id: string
  pr_id: string
  author: string
  content: string
  created_at: string
}

export interface CreatePRPayload {
  repo: string
  source_branch: string
  target_branch: string
  title: string
  description: string
}
