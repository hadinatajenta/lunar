import type { IssueBranchMatch } from "../branch-types"

export type IssueBranchPillTone = "none" | "found" | "warning"

export interface IssueBranchPillState {
  text: string
  tone: IssueBranchPillTone
}

export interface IssuePlanProgress {
  total: number
  settled: number
  failed: number
}

export const issueBranchPill = (
  matches: IssueBranchMatch[],
  progress: IssuePlanProgress | null
): IssueBranchPillState => {
  if (progress !== null && progress.total > 0 && progress.failed > 0) {
    return { text: `${progress.settled} of ${progress.total} created`, tone: "warning" }
  }
  if (progress !== null && progress.total > 0 && progress.settled === progress.total) {
    return { text: `${progress.settled} created`, tone: "found" }
  }
  if (matches.length === 0) {
    return { text: "no branch", tone: "none" }
  }
  if (matches.every((match) => match.source === "remote")) {
    return { text: "not cloned", tone: "warning" }
  }
  return { text: `${matches.length} branch`, tone: "found" }
}
