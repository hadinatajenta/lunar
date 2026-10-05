import { test, expect } from "../../../fixtures/authenticated.fixture"
import { routeWorkspaceGitActions } from "../../../helpers/route-mock"
import {
  BOARD_TALL_REPOSITORY_NAMES,
  routeWorkspaceBoard,
  routeWorkspaceBranches
} from "../../../helpers/workspace-board"

const REPOSITORY_NAME = "aurora"

test.describe("Microservices branch picker", () => {
  test.beforeEach(async ({ page }) => {
    await routeWorkspaceBoard(page)
    await routeWorkspaceBranches(page)
  })

  test(
    "MS-BRANCH-001 the branch pill is a button that opens a named branch popup",
    { tag: ["@microservices", "@positive", "@branch-picker", "@p0"] },
    async ({ microservicesPage }) => {
      routeWorkspaceGitActions(microservicesPage.page)
      await microservicesPage.navigateTo()

      const pill = microservicesPage.branchPill(REPOSITORY_NAME)
      await expect(pill).toHaveRole("button")
      await expect(pill).toHaveAttribute("aria-haspopup", "listbox")
      await expect(pill).toHaveAttribute("aria-expanded", "false")
      await expect(pill).toHaveAccessibleName(/^Branch .+ for aurora\. Choose a different branch$/)

      await microservicesPage.openBranchPicker(REPOSITORY_NAME)

      await expect(pill).toHaveAttribute("aria-expanded", "true")
      await expect(microservicesPage.branchPopup).toBeVisible()
      await expect(microservicesPage.branchList).toHaveRole("listbox")
      await expect(microservicesPage.branchList).toHaveAccessibleName("Branches")
      await expect(microservicesPage.branchSearchInput).toBeVisible()
      await expect(microservicesPage.branchSearchInput).toHaveAccessibleName("Search branches")
    }
  )

  test(
    "MS-BRANCH-002 the popup lists the current branch first and shows at most five rows",
    { tag: ["@microservices", "@positive", "@branch-picker", "@p0"] },
    async ({ microservicesPage }) => {
      routeWorkspaceGitActions(microservicesPage.page)
      await microservicesPage.navigateTo()
      const currentBranch = await microservicesPage.currentBranchName(REPOSITORY_NAME)
      await microservicesPage.openBranchPicker(REPOSITORY_NAME)

      const branchNames = await microservicesPage.branchRowNames()
      expect(branchNames.length).toBeGreaterThan(5)
      expect(branchNames[0]).toBe(currentBranch)

      const firstBranchRow = microservicesPage.branchRows().first()
      await expect(firstBranchRow).toHaveAttribute("aria-selected", "true")
      await expect(firstBranchRow).toHaveAccessibleName(`${currentBranch} (current branch)`)
      await expect(microservicesPage.branchRow(currentBranch)).toHaveAttribute("aria-selected", "true")

      const visibleRowCount = await microservicesPage.countVisibleBranchRows()
      expect(visibleRowCount).toBeGreaterThanOrEqual(1)
      expect(visibleRowCount).toBeLessThanOrEqual(5)

      const scrollMetrics = await microservicesPage.branchList.evaluate((listElement) => ({
        clientHeight: listElement.clientHeight,
        scrollHeight: listElement.scrollHeight
      }))
      expect(scrollMetrics.scrollHeight).toBeGreaterThan(scrollMetrics.clientHeight)
    }
  )

  test(
    "MS-BRANCH-003 the search field filters branches case-insensitively",
    { tag: ["@microservices", "@positive", "@branch-picker", "@p0"] },
    async ({ microservicesPage }) => {
      routeWorkspaceGitActions(microservicesPage.page)
      await microservicesPage.navigateTo()
      await microservicesPage.openBranchPicker(REPOSITORY_NAME)

      const branchNames = await microservicesPage.branchRowNames()
      const uppercaseSegment = branchNames
        .map((branchName) => branchName.match(/[A-Z]{3,}/)?.[0])
        .find((segment) => segment !== undefined)
      if (uppercaseSegment === undefined) {
        throw new Error("aurora exposes no uppercase branch segment to search for")
      }
      const searchQuery = uppercaseSegment.toLowerCase()
      const expectedNames = branchNames.filter((branchName) =>
        branchName.toLowerCase().includes(searchQuery)
      )
      expect(expectedNames.length).toBeGreaterThan(0)
      expect(expectedNames.some((branchName) => branchName.includes(uppercaseSegment))).toBe(true)

      await microservicesPage.branchSearchInput.fill(searchQuery)

      await expect(microservicesPage.branchRows()).toHaveCount(expectedNames.length)
      expect(await microservicesPage.branchRowNames()).toEqual(expectedNames)
    }
  )

  test(
    "MS-BRANCH-004 Escape closes the popup and returns focus to the pill",
    { tag: ["@microservices", "@positive", "@branch-picker", "@p1"] },
    async ({ page, microservicesPage }) => {
      routeWorkspaceGitActions(page)
      await microservicesPage.navigateTo()
      const pill = microservicesPage.branchPill(REPOSITORY_NAME)
      await microservicesPage.openBranchPicker(REPOSITORY_NAME)
      await expect(microservicesPage.branchPopup).toBeVisible()

      await page.keyboard.press("Escape")

      await expect(microservicesPage.branchPopup).toBeHidden()
      await expect(pill).toHaveAttribute("aria-expanded", "false")
      await expect(pill).toBeFocused()
    }
  )

  test(
    "MS-BRANCH-005 clicking outside the popup closes it",
    { tag: ["@microservices", "@positive", "@branch-picker", "@p1"] },
    async ({ microservicesPage }) => {
      routeWorkspaceGitActions(microservicesPage.page)
      await microservicesPage.navigateTo()
      const pill = microservicesPage.branchPill(REPOSITORY_NAME)
      await microservicesPage.openBranchPicker(REPOSITORY_NAME)
      await expect(microservicesPage.branchPopup).toBeVisible()

      await microservicesPage.pageHeading.click()

      await expect(microservicesPage.branchPopup).toBeHidden()
      await expect(pill).toHaveAttribute("aria-expanded", "false")
    }
  )

  test(
    "MS-BRANCH-006 the popup opens below the pill with room below and above the last row",
    { tag: ["@microservices", "@positive", "@branch-picker", "@p1"] },
    async ({ page, microservicesPage }) => {
      await routeWorkspaceBoard(page, BOARD_TALL_REPOSITORY_NAMES)
      routeWorkspaceGitActions(page)
      await microservicesPage.navigateTo()

      const trigger = microservicesPage.branchPill(REPOSITORY_NAME)
      await microservicesPage.openBranchPicker(REPOSITORY_NAME)
      const triggerBox = await microservicesPage.boundingBoxOf(trigger)
      const dropdownBox = await microservicesPage.boundingBoxOf(microservicesPage.branchPopup)
      await expect(microservicesPage.branchPopup).toHaveClass(/is-dropdown/)
      expect(dropdownBox.y).toBeGreaterThanOrEqual(triggerBox.y + triggerBox.height)

      await page.keyboard.press("Escape")
      await expect(microservicesPage.branchPopup).toBeHidden()

      await microservicesPage.scrollToPageBottom()
      const lastPill = microservicesPage.serviceRows.last().locator(".branch-pill")
      await expect(lastPill).toBeVisible()
      await lastPill.click()
      await microservicesPage.waitForBranchList()

      const lastTriggerBox = await microservicesPage.boundingBoxOf(lastPill)
      const dropupBox = await microservicesPage.boundingBoxOf(microservicesPage.branchPopup)
      await expect(microservicesPage.branchPopup).toHaveClass(/is-dropup/)
      expect(dropupBox.y + dropupBox.height).toBeLessThanOrEqual(lastTriggerBox.y)
    }
  )

  test(
    "MS-BRANCH-007 choosing a branch sends exactly one checkout request",
    { tag: ["@microservices", "@positive", "@branch-picker", "@p0"] },
    async ({ microservicesPage }) => {
      const gitActions = routeWorkspaceGitActions(microservicesPage.page)
      await microservicesPage.navigateTo()
      const currentBranch = await microservicesPage.currentBranchName(REPOSITORY_NAME)
      await microservicesPage.openBranchPicker(REPOSITORY_NAME)

      const branchNames = await microservicesPage.branchRowNames()
      const targetBranch = branchNames.find((branchName) => branchName !== currentBranch)
      if (targetBranch === undefined) {
        throw new Error("aurora exposes no branch other than the current one")
      }
      await expect(microservicesPage.branchRow(targetBranch)).toBeVisible()

      await microservicesPage.branchRow(targetBranch).click()

      await expect(microservicesPage.noticeBanner).toContainText(
        `Switched ${REPOSITORY_NAME} to ${targetBranch}`
      )
      expect(gitActions.calls).toEqual([
        { action: "checkout", repositoryName: REPOSITORY_NAME, branch: targetBranch }
      ])
      expect(gitActions.countFor("checkout")).toBe(1)
      await expect(microservicesPage.branchPopup).toBeHidden()
    }
  )
})
