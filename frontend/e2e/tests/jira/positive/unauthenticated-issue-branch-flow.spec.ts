import { test, expect } from "../../../fixtures/authenticated.fixture"
import { contrastRatio, parseColor } from "../../../helpers/contrast"
import {
  AUTH_TOKEN_KEY,
  BRANCH_NAME,
  ISSUE_KEY,
  chooseBase,
  match,
  openIssue,
  routeBranchFlow,
  suggestion
} from "../../../helpers/jira-branch-flow"

test.beforeEach(async ({ page }) => {
  await page.addInitScript((storageKey) => {
    window.localStorage.setItem(storageKey, "e2e-branch-flow-token")
  }, AUTH_TOKEN_KEY)
})

test.describe("Jira issue branch flow", () => {
  test(
    "JIRA-BRANCH-001 the Branches section names the repository and when it was last touched",
    { tag: ["@jira", "@positive", "@branch-flow"] },
    async ({ page, jiraPage }) => {
      await routeBranchFlow(page, {
        matches: [
          match({ repo: "mms-api", branch: BRANCH_NAME, source: "both", lastCommitRelative: "3 days ago" }),
          match({ repo: "atlas", branch: "feature/CRMMS-77911-Fix-Something", source: "local", lastCommitRelative: "2 weeks ago" }),
          match({ repo: "fe-mms", branch: "feature/CRMMS-77911-Frontend", source: "remote" })
        ],
        suggestions: [suggestion({ repo: "mms-api", isCloned: true, reason: "the AC names the list API the page calls" })]
      })
      await jiraPage.goto()
      await openIssue(page)

      const rows = page.getByTestId("issue-branch-row")
      await expect(rows).toHaveCount(3)
      await expect(rows.first()).toContainText("feature/CRMMS-77911-Fix-Something")
      await expect(rows.first()).toContainText("in atlas")
      await expect(rows.first()).toContainText("2 weeks ago")
      const remoteRow = rows.filter({ hasText: "fe-mms" })
      await expect(remoteRow).toContainText("feature/CRMMS-77911-Frontend")
      await expect(remoteRow).toContainText("not cloned")
      await expect(page.getByTestId("branch-remote-notice")).toContainText(
        "This branch exists on Bitbucket in fe-mms, which is not in your workspace."
      )
    }
  )

  test(
    "JIRA-BRANCH-002 the base picker separates another team's branches and states the dependency",
    { tag: ["@jira", "@positive", "@branch-flow"] },
    async ({ page, jiraPage }) => {
      await routeBranchFlow(page, {
        matches: [],
        suggestions: [suggestion({ repo: "mms-api", isCloned: true, reason: "the AC names the list API the page calls" })]
      })
      await jiraPage.goto()
      await openIssue(page)

      const row = page.getByTestId("branch-target-mms-api")
      await expect(row).toContainText("the AC names the list API the page calls")
      await row.getByRole("button", { name: /^Base branch for mms-api/ }).click()

      const popup = page.getByTestId("base-branch-popup")
      await expect(popup).toBeVisible()
      await expect(page.getByTestId("base-branch-search")).toBeVisible()
      await expect(popup).toContainText("Other teams' branches")

      await page.getByTestId("base-branch-search").fill("crmms-77745")
      await expect(popup).toContainText("feature/CRMMS-77745-Input-MDR-QRIS-Statis")
      await expect(popup).not.toContainText("release/production")

      await page.getByTestId("base-branch-option-feature/CRMMS-77745-Input-MDR-QRIS-Statis").click()
      await expect(page.getByTestId("branch-cross-team-note")).toContainText(
        "will branch from another team's branch, which has not reached production yet."
      )
    }
  )

  test(
    "JIRA-BRANCH-003 Add repository is a left aligned solid control that reaches 4.5 to 1",
    { tag: ["@jira", "@positive", "@branch-flow"] },
    async ({ page, jiraPage }) => {
      await routeBranchFlow(page, {
        matches: [],
        suggestions: [suggestion({ repo: "mms-api", isCloned: true, reason: "the AC names the list API" })]
      })
      await jiraPage.goto()
      await openIssue(page)

      const addButton = page.getByTestId("add-repository")
      await expect(addButton).toBeVisible()
      await expect(addButton).toHaveText(/\+\s*Add repository/)

      const rowBox = await page.getByTestId("branch-target-mms-api").boundingBox()
      const addBox = await addButton.boundingBox()
      expect(rowBox).not.toBeNull()
      expect(addBox).not.toBeNull()
      expect(Math.abs((addBox?.x ?? 0) - (rowBox?.x ?? 0))).toBeLessThan(2)
      expect(addBox?.width ?? 0).toBeLessThan((rowBox?.width ?? 0) / 2)

      const style = await addButton.evaluate((element) => {
        const computed = window.getComputedStyle(element)
        return {
          color: computed.color,
          background: computed.backgroundColor,
          borderStyle: computed.borderTopStyle,
          borderWidth: computed.borderTopWidth
        }
      })
      expect(style.borderStyle).toBe("solid")
      expect(style.borderWidth).not.toBe("0px")
      const foreground = parseColor(style.color)
      const background = parseColor(style.background)
      expect(foreground).not.toBeNull()
      expect(background).not.toBeNull()
      expect(contrastRatio(foreground!, background!)).toBeGreaterThanOrEqual(4.5)

      await addButton.click()
      await page.getByTestId("repository-search").fill("atl")
      await page.getByTestId("repository-option-atlas").click()

      const manualRow = page.getByTestId("branch-target-atlas")
      await expect(manualRow).toBeVisible()
      await expect(manualRow).toContainText("atlas")
      await expect(manualRow.getByTestId("branch-target-remove-atlas")).toBeVisible()
      await expect(manualRow).not.toContainText("added by you")
      await expect(manualRow.locator(".sub-line")).toHaveCount(0)
      await expect(page.getByTestId("branch-target-include-atlas")).toHaveCount(0)
    }
  )

  test(
    "JIRA-BRANCH-004 an existing branch settles, conflicts, or is offered for another repository",
    { tag: ["@jira", "@positive", "@branch-flow"] },
    async ({ page, jiraPage }) => {
      await routeBranchFlow(page, {
        matches: [
          match({ repo: "mms-api", branch: BRANCH_NAME, source: "both" }),
          match({ repo: "micro-user-mms", branch: "feature/CRMMS-77911-Manage-User-Fix", source: "local" }),
          match({ repo: "atlas", branch: "feature/CRMMS-77911-Enhancement-Manage-User", source: "local" })
        ],
        suggestions: [
          suggestion({ repo: "mms-api", isCloned: true, reason: "the AC names the list API" }),
          suggestion({ repo: "micro-user-mms", isCloned: true, reason: "the AC names PN grouping" })
        ]
      })
      await jiraPage.goto()
      await openIssue(page)

      await expect(page.getByTestId("branch-target-mms-api")).toHaveAttribute("data-state", "existing-same-name")
      await expect(page.getByTestId("branch-target-mms-api")).toContainText("Already exists with this name, left alone")

      const conflictRow = page.getByTestId("branch-target-micro-user-mms")
      await expect(conflictRow).toHaveAttribute("data-state", "conflict")
      await expect(conflictRow).toContainText("A branch for this issue already exists here, under a different name.")
      await expect(conflictRow).toContainText("feature/CRMMS-77911-Manage-User-Fix")
      await conflictRow.getByTestId("branch-target-use-existing-micro-user-mms").click()
      await expect(conflictRow).toHaveAttribute("data-state", "existing-kept")
      await conflictRow.getByTestId("branch-target-change-micro-user-mms").click()
      await expect(conflictRow).toHaveAttribute("data-state", "conflict")
      await conflictRow.getByTestId("branch-target-create-anyway-micro-user-mms").click()
      await expect(conflictRow).toHaveAttribute("data-state", "pending-duplicate")
      await expect(conflictRow).toContainText("Creating another splits the work for this issue across two branches.")

      await expect(page.getByTestId("branch-existing-elsewhere")).toContainText("atlas")
      await page.getByTestId("branch-existing-elsewhere-add-atlas").click()
      await expect(page.getByTestId("branch-target-atlas")).toHaveAttribute("data-state", "existing-same-name")
      await expect(page.getByTestId("branch-existing-elsewhere")).toHaveCount(0)
    }
  )

  test(
    "JIRA-BRANCH-005 a partial create keeps what was created, reports every row and locks the name",
    { tag: ["@jira", "@positive", "@branch-flow"] },
    async ({ page, jiraPage }) => {
      const flow = await routeBranchFlow(page, {
        matches: [],
        suggestions: [
          suggestion({ repo: "fe-mms", isCloned: false, reason: "the AC names the route /manage-user" }),
          suggestion({ repo: "mms-api", isCloned: true, reason: "the AC names the list API" }),
          suggestion({ repo: "micro-user-mms", isCloned: true, reason: "the AC names PN grouping" })
        ],
        created: {
          key: ISSUE_KEY,
          branch: BRANCH_NAME,
          created: [{ repo: "mms-api", base_branch: "release/production", head_commit: "abc123" }],
          failed: [
            { repo: "fe-mms", reason: "the repository is not cloned" },
            { repo: "micro-user-mms", reason: "the working tree has uncommitted changes; commit or stash them before switching branches" }
          ],
          skipped: []
        }
      })
      await jiraPage.goto()
      await openIssue(page)

      await chooseBase(page, "fe-mms", "release/production")
      await chooseBase(page, "mms-api", "release/production")
      await chooseBase(page, "micro-user-mms", "pre/development")
      await expect(page.getByTestId("branch-not-cloned-note")).toContainText("fe-mms is not cloned")

      const createButton = page.getByTestId("branch-create")
      await expect(createButton).toHaveText("Create 3 branches")
      await expect(createButton).toBeEnabled()
      await createButton.click()

      const summary = page.getByTestId("branch-create-summary")
      await expect(summary).toContainText("1 of 3")
      await expect(summary).toContainText("created · 2 failed")
      await expect(summary).toContainText("The branches that were created are kept.")
      await expect(createButton).toHaveText("Retry 2 failed")

      const createdRow = page.getByTestId("branch-target-mms-api")
      await expect(createdRow).toHaveAttribute("data-state", "outcome")
      await expect(createdRow).toContainText("Created")
      await expect(createdRow).toContainText(`${BRANCH_NAME} from release/production`)

      expect(flow.createRequests.length).toBe(1)
      expect(flow.createRequests[0].branch).toBe(BRANCH_NAME)
      expect(flow.createRequests[0].targets).toEqual([
        { repo: "fe-mms", base_branch: "release/production", remember_base: false },
        { repo: "mms-api", base_branch: "release/production", remember_base: false },
        { repo: "micro-user-mms", base_branch: "pre/development", remember_base: false }
      ])

      const failedRow = page.getByTestId("branch-target-fe-mms")
      await expect(failedRow).toContainText("The repository is not cloned, so no branch can be created there.")
      const retryButton = failedRow.getByTestId("branch-target-retry-fe-mms")
      await expect(retryButton).toBeVisible()
      await retryButton.click()
      await expect.poll(() => flow.createRequests.length).toBe(2)
      expect(flow.createRequests[1].targets).toEqual([
        { repo: "fe-mms", base_branch: "release/production", remember_base: false }
      ])
      await expect(page.getByTestId("branch-target-micro-user-mms")).toContainText(
        "The working tree has uncommitted changes. Commit or stash them, then retry."
      )

      const nameInput = page.getByTestId("branch-name-input")
      await expect(nameInput).toBeDisabled()
      await expect(page.getByTestId("branch-name-locked")).toContainText(
        "The name is fixed because branches already exist with it."
      )

      await page.getByTestId("btn-close-modal").click()
      const card = page.getByTestId("kanban-card-CRMMS-77911")
      await expect(card.getByTestId("issue-branch-pill")).toHaveText("1 of 3 created")
    }
  )

  test(
    "JIRA-BRANCH-006 an empty suggestion still leads forward and names the unavailable evidence",
    { tag: ["@jira", "@positive", "@branch-flow"] },
    async ({ page, jiraPage }) => {
      await routeBranchFlow(page, {
        matches: [],
        suggestions: [],
        unavailable: [
          { signal: "code_index", reason: "the repository is not cloned, so its files cannot be searched" }
        ]
      })
      await jiraPage.goto()
      await openIssue(page)

      await expect(page.getByTestId("branch-suggestion-empty")).toContainText(
        "Lunar could not suggest a repository from the acceptance criteria. Choose one yourself."
      )
      await expect(page.getByTestId("add-repository")).toBeVisible()
      await expect(page.getByTestId("branch-create-form")).toHaveCount(0)

      const unavailable = page.getByTestId("branch-signals-unavailable")
      await expect(unavailable).toContainText("the repository is not cloned, so its files cannot be searched")
      await expect(unavailable).not.toContainText("code_index")
    }
  )
})
