import type {
  BranchNameSuggestion,
  CreateBranchesResponse,
  IssueBranchMatch,
  RepositorySuggestion
} from "../branch-types"
import { describeBranchFailure, existingBranchFromSkipReason } from "./branch-failure.ts"

export type ConflictResolution = "use" | "new" | null

export type TargetOutcomeStatus = "created" | "exists" | "failed"

export interface BranchTarget {
  repo: string
  isCloned: boolean
  reasons: string[]
  isManual: boolean
}

export interface ExistingBranch {
  branch: string
  isCloned: boolean
}

export interface TargetOutcome {
  status: TargetOutcomeStatus
  reason: string
  branch: string
  baseBranch: string
}

export interface BranchPlanState {
  issueKey: string
  branchName: string
  branchNameEvidence: string
  branchNameSource: string
  isBranchNameEdited: boolean
  targets: BranchTarget[]
  bases: Record<string, string>
  rememberBases: Record<string, boolean>
  resolutions: Record<string, ConflictResolution>
  outcomes: Record<string, TargetOutcome>
  existing: Record<string, ExistingBranch>
  excluded: Record<string, boolean>
}

export const createBranchPlan = (issueKey: string): BranchPlanState => ({
  issueKey,
  branchName: "",
  branchNameEvidence: "",
  branchNameSource: "",
  isBranchNameEdited: false,
  targets: [],
  bases: {},
  rememberBases: {},
  resolutions: {},
  outcomes: {},
  existing: {},
  excluded: {}
})

export const findTarget = (plan: BranchPlanState, repo: string): BranchTarget | null => {
  return plan.targets.find((target) => target.repo === repo) ?? null
}

export const isSettledOutcome = (outcome: TargetOutcome | undefined): boolean => {
  return outcome !== undefined && (outcome.status === "created" || outcome.status === "exists")
}

export const isTargetExcluded = (plan: BranchPlanState, repo: string): boolean => {
  return plan.excluded[repo] === true
}

export const setTargetExcluded = (
  plan: BranchPlanState,
  repo: string,
  excluded: boolean
): void => {
  plan.excluded[repo] = excluded
}

export const seedSuggestedTargets = (
  plan: BranchPlanState,
  suggestions: RepositorySuggestion[]
): void => {
  for (const suggestion of suggestions) {
    if (findTarget(plan, suggestion.repo) !== null) {
      continue
    }
    plan.targets.push({
      repo: suggestion.repo,
      isCloned: suggestion.is_cloned,
      reasons: suggestion.reasons.map((reason) => reason.text),
      isManual: false
    })
  }
}

export const addManualTarget = (
  plan: BranchPlanState,
  repo: string,
  isCloned: boolean
): void => {
  if (findTarget(plan, repo) !== null) {
    return
  }
  plan.targets.push({ repo, isCloned, reasons: [], isManual: true })
}

export const applyBranchName = (
  plan: BranchPlanState,
  suggestion: BranchNameSuggestion
): void => {
  if (plan.isBranchNameEdited) {
    return
  }
  plan.branchName = suggestion.branch
  plan.branchNameSource = suggestion.source
  plan.branchNameEvidence = suggestion.evidence ?? ""
}

export const editBranchName = (plan: BranchPlanState, branch: string): void => {
  plan.branchName = branch
  plan.isBranchNameEdited = true
}

export const removeTarget = (plan: BranchPlanState, repo: string): void => {
  const target = findTarget(plan, repo)
  if (target === null || !target.isManual) {
    return
  }
  plan.targets = plan.targets.filter((entry) => entry.repo !== repo)
  delete plan.bases[repo]
  delete plan.rememberBases[repo]
  delete plan.resolutions[repo]
  delete plan.outcomes[repo]
  delete plan.excluded[repo]
}

export const setBaseBranch = (plan: BranchPlanState, repo: string, base: string): void => {
  plan.bases[repo] = base
}

export const setRememberBase = (plan: BranchPlanState, repo: string, remember: boolean): void => {
  plan.rememberBases[repo] = remember
}

export const resolveExistingBranch = (
  plan: BranchPlanState,
  repo: string,
  resolution: Exclude<ConflictResolution, null>
): void => {
  plan.resolutions[repo] = resolution
}

export const reconsiderExistingBranch = (plan: BranchPlanState, repo: string): void => {
  delete plan.resolutions[repo]
}

export const seedExistingBranches = (
  plan: BranchPlanState,
  matches: IssueBranchMatch[]
): void => {
  const next: Record<string, ExistingBranch> = {}
  for (const match of matches) {
    const current = next[match.repo]
    if (current === undefined) {
      next[match.repo] = { branch: match.branch, isCloned: match.is_cloned }
      continue
    }
    if (match.branch === plan.branchName && current.branch !== plan.branchName) {
      next[match.repo] = { branch: match.branch, isCloned: match.is_cloned }
    }
  }
  plan.existing = next
}

const writeOutcome = (plan: BranchPlanState, repo: string, outcome: TargetOutcome): void => {
  const current = plan.outcomes[repo]
  if (current !== undefined && current.status === "created") {
    return
  }
  plan.outcomes[repo] = outcome
}

export const applyCreationResponse = (
  plan: BranchPlanState,
  response: CreateBranchesResponse
): void => {
  for (const created of response.created) {
    writeOutcome(plan, created.repo, {
      status: "created",
      reason: "",
      branch: response.branch,
      baseBranch: created.base_branch
    })
  }
  for (const skipped of response.skipped) {
    writeOutcome(plan, skipped.repo, {
      status: "exists",
      reason: skipped.reason,
      branch: existingBranchFromSkipReason(skipped.reason) ?? response.branch,
      baseBranch: ""
    })
  }
  for (const failed of response.failed) {
    writeOutcome(plan, failed.repo, {
      status: "failed",
      reason: describeBranchFailure(failed.reason),
      branch: "",
      baseBranch: ""
    })
  }
}

export const clearFailedOutcome = (plan: BranchPlanState, repo: string): void => {
  if (plan.outcomes[repo]?.status === "failed") {
    delete plan.outcomes[repo]
  }
}
