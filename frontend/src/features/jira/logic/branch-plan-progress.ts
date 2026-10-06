import type { IssueBranchMatch } from "../branch-types"
import {
  findTarget,
  isSettledOutcome,
  isTargetExcluded,
  type BranchPlanState,
  type BranchTarget
} from "./branch-plan.ts"

export type TargetRowKind =
  | "outcome"
  | "existing-same-name"
  | "existing-kept"
  | "conflict"
  | "pending-duplicate"
  | "pending"

export interface PlanProgress {
  total: number
  settled: number
  failed: number
  pending: number
}

export interface PlanSummary {
  countLabel: string
  stateLabel: string
  detail: string
  tone: "ok" | "partial"
}

export interface PrimaryAction {
  label: string
  enabled: boolean
}

const isRowCreatable = (plan: BranchPlanState, repo: string): boolean => {
  if (isSettledOutcome(plan.outcomes[repo]) || isTargetExcluded(plan, repo)) {
    return false
  }
  const existing = plan.existing[repo]
  if (existing === undefined) {
    return true
  }
  if (existing.branch === plan.branchName) {
    return false
  }
  return plan.resolutions[repo] === "new"
}

export const rowKind = (plan: BranchPlanState, repo: string): TargetRowKind => {
  if (plan.outcomes[repo] !== undefined) {
    return "outcome"
  }
  const existing = plan.existing[repo]
  if (existing === undefined) {
    return "pending"
  }
  if (existing.branch === plan.branchName) {
    return "existing-same-name"
  }
  const resolution = plan.resolutions[repo] ?? null
  if (resolution === "use") {
    return "existing-kept"
  }
  if (resolution === "new") {
    return "pending-duplicate"
  }
  return "conflict"
}

export const creatableTargets = (plan: BranchPlanState): BranchTarget[] => {
  return plan.targets.filter((target) => isRowCreatable(plan, target.repo))
}

export const blockedTargets = (plan: BranchPlanState): BranchTarget[] => {
  return plan.targets.filter((target) => {
    if (isSettledOutcome(plan.outcomes[target.repo]) || isTargetExcluded(plan, target.repo)) {
      return false
    }
    return !isRowCreatable(plan, target.repo)
  })
}

export const missingBaseTargets = (plan: BranchPlanState): BranchTarget[] => {
  return creatableTargets(plan).filter((target) => (plan.bases[target.repo] ?? "").length === 0)
}

export const planProgress = (plan: BranchPlanState): PlanProgress => {
  let settled = 0
  let failed = 0
  for (const target of plan.targets) {
    const outcome = plan.outcomes[target.repo]
    if (isSettledOutcome(outcome)) {
      settled += 1
      continue
    }
    if (outcome?.status === "failed") {
      failed += 1
    }
  }
  return {
    total: plan.targets.length,
    settled,
    failed,
    pending: plan.targets.length - settled - failed
  }
}

export const planSummary = (plan: BranchPlanState): PlanSummary | null => {
  const progress = planProgress(plan)
  if (progress.settled === 0) {
    return null
  }
  if (progress.failed === 0) {
    return {
      countLabel: `${progress.settled} of ${progress.total}`,
      stateLabel: "created",
      detail: "",
      tone: "ok"
    }
  }
  return {
    countLabel: `${progress.settled} of ${progress.total}`,
    stateLabel: `created · ${progress.failed} failed`,
    detail: "The branches that were created are kept. Fix the reason below and retry only the rest.",
    tone: "partial"
  }
}

export const primaryAction = (plan: BranchPlanState): PrimaryAction => {
  const creatable = creatableTargets(plan)
  const progress = planProgress(plan)
  const pendingCreatable = creatable.filter((target) => plan.outcomes[target.repo] === undefined)
  let label = progress.settled > 0 ? "All created" : "Create branch"
  if (creatable.length > 0) {
    label = pendingCreatable.length === 0 && progress.failed > 0
      ? `Retry ${progress.failed} failed`
      : `Create ${creatable.length} branch${creatable.length === 1 ? "" : "es"}`
  }
  return { label, enabled: canCreate(plan) }
}

export const canCreate = (plan: BranchPlanState): boolean => {
  const creatable = creatableTargets(plan)
  if (plan.branchName.trim().length === 0 || creatable.length === 0) {
    return false
  }
  if (blockedTargets(plan).length > 0) {
    return false
  }
  return missingBaseTargets(plan).length === 0
}

export const isBranchNameLocked = (plan: BranchPlanState): boolean => {
  return planProgress(plan).settled > 0
}

export const elsewhereMatches = (
  plan: BranchPlanState,
  matches: IssueBranchMatch[]
): IssueBranchMatch[] => {
  return matches.filter((match) => findTarget(plan, match.repo) === null)
}
