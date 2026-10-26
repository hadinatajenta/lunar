import assert from "node:assert/strict"
import { test } from "node:test"
import { createBranchPlan, isTargetExcluded, seedSuggestedTargets } from "./branch-plan.ts"
import { emptyBranchMessage } from "./branch-groups.ts"

test("qa a low confidence suggestion is still seeded as an included target", () => {
  const plan = createBranchPlan("CRMMS-77911")
  seedSuggestedTargets(plan, [
    {
      repo: "atlas",
      score: 1,
      is_cloned: true,
      bitbucket_project: "NME",
      reasons: [{ signal: "model", weight: 1, strength: 1, text: "the model guessed" }]
    }
  ])

  assert.equal(plan.targets.length, 1)
  assert.equal(isTargetExcluded(plan, "atlas"), false)
})

test("qa a partial remote check is rendered as a completed Bitbucket check", () => {
  const message = emptyBranchMessage("CRMMS-77911", {
    local_repository_count: 36,
    remote_repository_count: 36,
    coverage_percent: 100,
    verified_remotely: false,
    remote_repositories_checked: 5
  })

  assert.equal(
    message,
    "No branch found in the 36 repositories Lunar can see, or in the 5 most likely repositories checked on Bitbucket."
  )
})
