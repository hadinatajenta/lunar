import assert from "node:assert/strict"
import { describe, test } from "node:test"
import type { CreateBranchesResponse, IssueBranchMatch, RepositorySuggestion } from "../branch-types.ts"
import {
  addManualTarget,
  applyCreationResponse,
  clearFailedOutcome,
  createBranchPlan,
  removeTarget,
  resolveExistingBranch,
  seedExistingBranches,
  seedSuggestedTargets,
  setBaseBranch,
  setTargetExcluded
} from "./branch-plan.ts"
import {
  blockedTargets,
  canCreate,
  creatableTargets,
  elsewhereMatches,
  isBranchNameLocked,
  planProgress,
  planSummary,
  primaryAction,
  rowKind
} from "./branch-plan-progress.ts"

const suggestion = (repo: string, isCloned: boolean, reason: string): RepositorySuggestion => ({
  repo,
  score: 9,
  is_cloned: isCloned,
  bitbucket_project: "NME",
  reasons: [{ signal: "epic_siblings", weight: 5, strength: 0.8, text: reason }]
})

const match = (repo: string, branch: string, source: IssueBranchMatch["source"]): IssueBranchMatch => ({
  repo,
  branch,
  source,
  is_current: false,
  is_cloned: source !== "remote",
  bitbucket_project: "NME",
  last_commit: null,
  last_commit_relative: source === "remote" ? null : "3 days ago",
  ahead: null,
  behind: null
})

const planWithThreeTargets = () => {
  const plan = createBranchPlan("CRMMS-77911")
  plan.branchName = "feature/CRMMS-77911-Enhancement-Manage-User"
  seedSuggestedTargets(plan, [
    suggestion("fe-mms", false, "the AC names the route"),
    suggestion("mms-api", true, "the AC names the list API"),
    suggestion("micro-user-mms", true, "the AC names PN grouping")
  ])
  return plan
}

const partialResponse: CreateBranchesResponse = {
  key: "CRMMS-77911",
  branch: "feature/CRMMS-77911-Enhancement-Manage-User",
  created: [{ repo: "mms-api", base_branch: "release/production", head_commit: "abc" }],
  failed: [{ repo: "fe-mms", reason: "the repository is not cloned" }],
  skipped: [{ repo: "micro-user-mms", reason: "the branch already exists: feature/CRMMS-77911-Manage-User-Fix" }]
}

describe("branch plan targets", () => {
  test("a suggested target carries every reason and is never duplicated", () => {
    const plan = createBranchPlan("CRMMS-77911")
    seedSuggestedTargets(plan, [suggestion("mms-api", true, "first reason")])
    seedSuggestedTargets(plan, [suggestion("mms-api", true, "second reason")])
    assert.equal(plan.targets.length, 1)
    assert.deepEqual(plan.targets[0].reasons, ["first reason"])
    assert.equal(plan.targets[0].isManual, false)
  })

  test("a manual target has no reason and can be removed while a suggested one cannot", () => {
    const plan = planWithThreeTargets()
    addManualTarget(plan, "atlas", true)
    const manual = plan.targets.find((target) => target.repo === "atlas")
    assert.deepEqual(manual?.reasons, [])
    assert.equal(manual?.isManual, true)

    setBaseBranch(plan, "atlas", "master")
    removeTarget(plan, "atlas")
    assert.equal(plan.targets.some((target) => target.repo === "atlas"), false)
    assert.equal(plan.bases.atlas, undefined)

    removeTarget(plan, "mms-api")
    assert.equal(plan.targets.some((target) => target.repo === "mms-api"), true)
  })
})

describe("existing branches", () => {
  test("the same name settles the row and a different tail becomes a conflict", () => {
    const plan = planWithThreeTargets()
    seedExistingBranches(plan, [
      match("mms-api", "feature/CRMMS-77911-Enhancement-Manage-User", "both"),
      match("micro-user-mms", "feature/CRMMS-77911-Manage-User-Fix", "local")
    ])
    assert.equal(rowKind(plan, "mms-api"), "existing-same-name")
    assert.equal(rowKind(plan, "micro-user-mms"), "conflict")
    assert.equal(rowKind(plan, "fe-mms"), "pending")
  })

  test("using the existing branch settles it, and Change returns the conflict", () => {
    const plan = planWithThreeTargets()
    seedExistingBranches(plan, [match("micro-user-mms", "feature/CRMMS-77911-Manage-User-Fix", "local")])
    resolveExistingBranch(plan, "micro-user-mms", "use")
    assert.equal(rowKind(plan, "micro-user-mms"), "existing-kept")
    assert.equal(canCreate(plan), false)
    plan.outcomes["mms-api"] = { status: "created", reason: "", branch: "b", baseBranch: "master" }
    assert.equal(primaryAction(plan).label, "Create 1 branch")
  })

  test("a conflict blocks the create until it is resolved", () => {
    const plan = planWithThreeTargets()
    seedExistingBranches(plan, [match("micro-user-mms", "feature/CRMMS-77911-Manage-User-Fix", "local")])
    setBaseBranch(plan, "fe-mms", "release/development")
    setBaseBranch(plan, "mms-api", "release/production")
    setBaseBranch(plan, "micro-user-mms", "development")
    assert.equal(canCreate(plan), false)

    resolveExistingBranch(plan, "micro-user-mms", "new")
    assert.equal(rowKind(plan, "micro-user-mms"), "pending-duplicate")
    assert.equal(canCreate(plan), true)
  })

  test("the existing name wins when it matches the generated one", () => {
    const plan = planWithThreeTargets()
    seedExistingBranches(plan, [
      match("mms-api", "feature/CRMMS-77911-Other", "local"),
      match("mms-api", "feature/CRMMS-77911-Enhancement-Manage-User", "both")
    ])
    assert.equal(plan.existing["mms-api"].branch, "feature/CRMMS-77911-Enhancement-Manage-User")
  })

  test("a match in a repository that is not a target is reported elsewhere", () => {
    const plan = planWithThreeTargets()
    const atlas = match("atlas", "feature/CRMMS-77911-Enhancement-Manage-User", "local")
    assert.deepEqual(elsewhereMatches(plan, [atlas]), [atlas])
    addManualTarget(plan, "atlas", true)
    assert.deepEqual(elsewhereMatches(plan, [atlas]), [])
  })
})

describe("partial creation", () => {
  test("a partial result reports every target and never rolls back the created one", () => {
    const plan = planWithThreeTargets()
    setBaseBranch(plan, "fe-mms", "release/development")
    setBaseBranch(plan, "mms-api", "release/production")
    setBaseBranch(plan, "micro-user-mms", "development")
    applyCreationResponse(plan, partialResponse)

    assert.equal(plan.outcomes["mms-api"].status, "created")
    assert.equal(plan.outcomes["mms-api"].baseBranch, "release/production")
    assert.equal(plan.outcomes["fe-mms"].status, "failed")
    assert.equal(plan.outcomes["fe-mms"].reason, "The repository is not cloned, so no branch can be created there.")
    assert.equal(plan.outcomes["micro-user-mms"].status, "exists")
    assert.equal(plan.outcomes["micro-user-mms"].branch, "feature/CRMMS-77911-Manage-User-Fix")

    const progress = planProgress(plan)
    assert.deepEqual(progress, { total: 3, settled: 2, failed: 1, pending: 0 })
  })

  test("the summary never claims completion while a row failed", () => {
    const plan = planWithThreeTargets()
    applyCreationResponse(plan, partialResponse)
    const summary = planSummary(plan)
    assert.equal(summary?.countLabel, "2 of 3")
    assert.equal(summary?.stateLabel, "created · 1 failed")
    assert.equal(summary?.tone, "partial")
    assert.match(summary?.detail ?? "", /retry only the rest/)
    assert.equal(primaryAction(plan).label, "Retry 1 failed")
  })

  test("a created branch is never overwritten by a later result for the same repository", () => {
    const plan = planWithThreeTargets()
    applyCreationResponse(plan, partialResponse)
    applyCreationResponse(plan, {
      key: "CRMMS-77911",
      branch: "feature/CRMMS-77911-Enhancement-Manage-User",
      created: [],
      failed: [{ repo: "mms-api", reason: "the repository is not cloned" }],
      skipped: []
    })
    assert.equal(plan.outcomes["mms-api"].status, "created")
    assert.equal(planProgress(plan).settled, 2)
  })

  test("retrying one failed row leaves the settled rows alone", () => {
    const plan = planWithThreeTargets()
    applyCreationResponse(plan, partialResponse)
    clearFailedOutcome(plan, "fe-mms")
    assert.equal(plan.outcomes["fe-mms"], undefined)
    assert.equal(rowKind(plan, "fe-mms"), "pending")
    assert.equal(planProgress(plan).settled, 2)
    assert.equal(primaryAction(plan).label, "Create 1 branch")

    setBaseBranch(plan, "fe-mms", "release/development")
    assert.equal(canCreate(plan), true)
    applyCreationResponse(plan, {
      key: "CRMMS-77911",
      branch: "feature/CRMMS-77911-Enhancement-Manage-User",
      created: [{ repo: "fe-mms", base_branch: "release/development", head_commit: "def" }],
      failed: [],
      skipped: []
    })
    assert.equal(planProgress(plan).settled, 3)
    assert.equal(planSummary(plan)?.tone, "ok")
    assert.equal(primaryAction(plan).label, "All created")
    assert.equal(primaryAction(plan).enabled, false)
  })

  test("the branch name locks once anything exists", () => {
    const plan = planWithThreeTargets()
    assert.equal(isBranchNameLocked(plan), false)
    applyCreationResponse(plan, partialResponse)
    assert.equal(isBranchNameLocked(plan), true)
  })

  test("create stays disabled until every creatable row has a base", () => {
    const plan = planWithThreeTargets()
    assert.equal(primaryAction(plan).label, "Create 3 branches")
    assert.equal(primaryAction(plan).enabled, false)
    setBaseBranch(plan, "fe-mms", "release/development")
    setBaseBranch(plan, "mms-api", "release/production")
    assert.equal(canCreate(plan), false)
    setBaseBranch(plan, "micro-user-mms", "development")
    assert.equal(canCreate(plan), true)
    assert.deepEqual(primaryAction(plan), { label: "Create 3 branches", enabled: true })
  })

  test("an unticked target is left out and never blocks the other rows", () => {
    const plan = planWithThreeTargets()
    setTargetExcluded(plan, "micro-user-mms", true)
    setBaseBranch(plan, "fe-mms", "release/development")
    setBaseBranch(plan, "mms-api", "release/production")
    assert.equal(primaryAction(plan).label, "Create 2 branches")
    assert.equal(canCreate(plan), true)
    assert.deepEqual(creatableTargets(plan).map((target) => target.repo), ["fe-mms", "mms-api"])
  })

  test("an unticked conflict does not block the create either", () => {
    const plan = planWithThreeTargets()
    seedExistingBranches(plan, [match("micro-user-mms", "feature/CRMMS-77911-Manage-User-Fix", "local")])
    setTargetExcluded(plan, "micro-user-mms", true)
    setBaseBranch(plan, "fe-mms", "release/development")
    setBaseBranch(plan, "mms-api", "release/production")
    assert.deepEqual(blockedTargets(plan), [])
    assert.equal(canCreate(plan), true)
  })

  test("an empty branch name keeps the create disabled", () => {
    const plan = planWithThreeTargets()
    plan.branchName = ""
    setBaseBranch(plan, "fe-mms", "release/development")
    setBaseBranch(plan, "mms-api", "release/production")
    setBaseBranch(plan, "micro-user-mms", "development")
    assert.equal(canCreate(plan), false)
  })
})
