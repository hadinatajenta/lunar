import type { Locator, Page } from "@playwright/test"
import { test, expect } from "../../../fixtures/authenticated.fixture"
import { routeWorkspaceGitActions } from "../../../helpers/route-mock"
import { escapeRegExpPattern } from "../../../pages/MicroservicesPage"

const REPOSITORY_NAME = "aurora"
const TAB_PRESS_LIMIT = 40
const MENU_ITEM_LABELS = ["Pull", "Fetch", "Stash changes local", "Discard changes", "Details"]

async function focusWithTabKey(page: Page, target: Locator): Promise<void> {
  for (let pressCount = 0; pressCount < TAB_PRESS_LIMIT; pressCount += 1) {
    await page.keyboard.press("Tab")
    if (await target.evaluate((element) => element === document.activeElement)) {
      return
    }
  }
  throw new Error("the control was not reachable with the Tab key")
}

test.describe("Microservices keyboard and accessible names", () => {
  test(
    "MS-A11Y-001 the branch picker is operable with the keyboard alone",
    { tag: ["@microservices", "@positive", "@accessibility", "@p0"] },
    async ({ page, microservicesPage }) => {
      const gitActions = routeWorkspaceGitActions(page)
      await microservicesPage.navigateTo()
      const pill = microservicesPage.branchPill(REPOSITORY_NAME)

      await focusWithTabKey(page, pill)
      await expect(pill).toBeFocused()

      await page.keyboard.press("Enter")
      await microservicesPage.waitForBranchList()
      await expect(microservicesPage.branchSearchInput).toBeFocused()

      await page.keyboard.press("ArrowDown")
      await expect(microservicesPage.branchRows().first()).toBeFocused()
      await page.keyboard.press("ArrowDown")
      await expect(microservicesPage.branchRows().nth(1)).toBeFocused()

      const branchNames = await microservicesPage.branchRowNames()
      const targetBranch = branchNames[1]
      const currentBranch = await microservicesPage.currentBranchName(REPOSITORY_NAME)
      expect(targetBranch).not.toBe(currentBranch)

      await page.keyboard.press("Enter")

      await expect(microservicesPage.noticeBanner).toContainText(
        `Switched ${REPOSITORY_NAME} to ${targetBranch}`
      )
      expect(gitActions.calls).toEqual([
        { action: "checkout", repositoryName: REPOSITORY_NAME, branch: targetBranch }
      ])
      await expect(microservicesPage.branchPopup).toBeHidden()
      await expect(pill).toBeFocused()
    }
  )

  test(
    "MS-A11Y-002 the repository actions menu is operable with the keyboard alone",
    { tag: ["@microservices", "@positive", "@accessibility", "@p0"] },
    async ({ page, microservicesPage }) => {
      const gitActions = routeWorkspaceGitActions(page)
      await microservicesPage.navigateTo()
      const trigger = microservicesPage.kebabTrigger(REPOSITORY_NAME)

      await focusWithTabKey(page, trigger)
      await expect(trigger).toBeFocused()

      await page.keyboard.press("Enter")
      await expect(microservicesPage.actionMenu).toBeVisible()
      await expect(microservicesPage.actionMenuItem("Pull")).toBeFocused()

      await page.keyboard.press("ArrowDown")
      await expect(microservicesPage.actionMenuItem("Fetch")).toBeFocused()
      await page.keyboard.press("ArrowUp")
      await expect(microservicesPage.actionMenuItem("Pull")).toBeFocused()
      await page.keyboard.press("ArrowUp")
      await expect(microservicesPage.actionMenuItem("Details")).toBeFocused()

      await page.keyboard.press("Escape")

      await expect(microservicesPage.actionMenu).toBeHidden()
      await expect(trigger).toBeFocused()
      expect(gitActions.calls).toEqual([])
    }
  )

  test(
    "MS-A11Y-003 every branch and repository control exposes an accessible name",
    { tag: ["@microservices", "@positive", "@accessibility", "@p1"] },
    async ({ page, microservicesPage }) => {
      const gitActions = routeWorkspaceGitActions(page)
      await microservicesPage.navigateTo()

      await expect(microservicesPage.branchPill(REPOSITORY_NAME)).toHaveAccessibleName(
        /^Branch .+ for aurora\. Choose a different branch$/
      )
      await expect(microservicesPage.kebabTrigger(REPOSITORY_NAME)).toHaveAccessibleName(
        `Repository actions for ${REPOSITORY_NAME}`
      )

      await microservicesPage.openBranchPicker(REPOSITORY_NAME)
      await expect(microservicesPage.branchSearchInput).toHaveAccessibleName("Search branches")
      const branchRows = await microservicesPage.branchRows().all()
      expect(branchRows.length).toBeGreaterThan(0)
      for (const branchRow of branchRows) {
        const branchName = await branchRow.locator(".branch-row-name").innerText()
        await expect(branchRow).toHaveAccessibleName(new RegExp(`^${escapeRegExpPattern(branchName)}`))
      }
      await page.keyboard.press("Escape")
      await expect(microservicesPage.branchPopup).toBeHidden()

      await microservicesPage.openRepositoryActions(REPOSITORY_NAME)
      for (const menuItemLabel of MENU_ITEM_LABELS) {
        await expect(microservicesPage.actionMenuItem(menuItemLabel)).toHaveAccessibleName(
          menuItemLabel
        )
      }

      await microservicesPage.actionMenuItem("Discard changes").click()
      await expect(microservicesPage.discardDialog).toBeVisible()
      await expect(microservicesPage.discardDialogButton("Cancel")).toHaveAccessibleName("Cancel")
      await expect(microservicesPage.discardDialogButton("Discard changes")).toHaveAccessibleName(
        "Discard changes"
      )
      expect(gitActions.calls).toEqual([])
    }
  )
})
