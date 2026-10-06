import assert from "node:assert/strict"
import { describe, test } from "node:test"
import { describeBranchFailure, existingBranchFromSkipReason } from "./branch-failure.ts"

describe("branch failure copy", () => {
  test("every known backend reason becomes an actionable sentence", () => {
    assert.equal(
      describeBranchFailure("the working tree has uncommitted changes; commit or stash them before switching branches"),
      "The working tree has uncommitted changes. Commit or stash them, then retry."
    )
    assert.equal(
      describeBranchFailure("the repository is not cloned"),
      "The repository is not cloned, so no branch can be created there."
    )
    assert.equal(
      describeBranchFailure("the base branch was not found in this repository"),
      "The base branch was not found in this repository. Choose a different base branch."
    )
    assert.equal(
      describeBranchFailure("branch creation is disabled until LUNAR_WORKSPACE_ALLOWED_ROOTS names the workspace root"),
      "The workspace root is outside LUNAR_WORKSPACE_ALLOWED_ROOTS."
    )
  })

  test("an unknown reason is still shown as a sentence rather than hidden", () => {
    assert.equal(describeBranchFailure("something unexpected happened"), "Something unexpected happened.")
    assert.equal(describeBranchFailure("trailing full stop."), "Trailing full stop.")
    assert.equal(describeBranchFailure("   "), "The branch could not be created.")
  })

  test("a skip reason names the branch that carries the key", () => {
    assert.equal(
      existingBranchFromSkipReason("the branch already exists: feature/CRMMS-77911-Manage-User-Fix"),
      "feature/CRMMS-77911-Manage-User-Fix"
    )
    assert.equal(existingBranchFromSkipReason("the branch already exists"), null)
  })
})
