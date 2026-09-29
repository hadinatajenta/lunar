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

export interface AICodeReview {
  pr_id: string
  summary: string
  findings: string[]
  generated_comment: string
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
