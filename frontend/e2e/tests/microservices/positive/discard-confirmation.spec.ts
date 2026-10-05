import { test, expect } from "../../../fixtures/authenticated.fixture"
import { routeWorkspaceGitActions } from "../../../helpers/route-mock"
import { routeWorkspaceBoard, routeWorkspaceBranches } from "../../../helpers/workspace-board"

const REPOSITORY_NAME = "aurora"

test.describe("Microservices discard confirmation", () => {
  test.beforeEach(async ({ page }) => {
    await routeWorkspaceBoard(page)
    await routeWorkspaceBranches(page)
  })

  test(
    "MS-DISCARD-001 choosing Discard changes opens the confirmation dialog",
    { tag: ["@microservices", "@positive", "@discard", "@p0"] },
    async ({ microservicesPage }) => {
      const gitActions = routeWorkspaceGitActions(microservicesPage.page)
      await microservicesPage.navigateTo()

      await microservicesPage.openDiscardConfirmation(REPOSITORY_NAME)

      await expect(microservicesPage.discardDialog).toHaveAccessibleName("Discard local changes?")
      await expect(microservicesPage.discardDialog).toContainText(
        `Every tracked file in ${REPOSITORY_NAME} is restored to the last commit`
      )
      await expect(microservicesPage.discardDialogButton("Cancel")).toBeVisible()
      await expect(microservicesPage.discardDialogButton("Cancel")).toBeFocused()
      await expect(microservicesPage.discardDialogButton("Discard changes")).toBeVisible()
      expect(gitActions.countFor("discard")).toBe(0)
    }
  )

  test(
    "MS-DISCARD-002 Cancel closes the dialog and sends no request",
    { tag: ["@microservices", "@positive", "@discard", "@p0"] },
    async ({ microservicesPage }) => {
      const gitActions = routeWorkspaceGitActions(microservicesPage.page)
      await microservicesPage.navigateTo()
      await microservicesPage.openDiscardConfirmation(REPOSITORY_NAME)

      await microservicesPage.discardDialogButton("Cancel").click()

      await expect(microservicesPage.discardDialog).toBeHidden()
      await expect(microservicesPage.kebabTrigger(REPOSITORY_NAME)).toBeFocused()
      expect(gitActions.countFor("discard")).toBe(0)
      expect(gitActions.calls).toEqual([])
    }
  )

  test(
    "MS-DISCARD-003 Escape closes the dialog and sends no request",
    { tag: ["@microservices", "@positive", "@discard", "@p0"] },
    async ({ page, microservicesPage }) => {
      const gitActions = routeWorkspaceGitActions(page)
      await microservicesPage.navigateTo()
      await microservicesPage.openDiscardConfirmation(REPOSITORY_NAME)

      await page.keyboard.press("Escape")

      await expect(microservicesPage.discardDialog).toBeHidden()
      await expect(microservicesPage.kebabTrigger(REPOSITORY_NAME)).toBeFocused()
      expect(gitActions.countFor("discard")).toBe(0)
      expect(gitActions.calls).toEqual([])
    }
  )

  test(
    "MS-DISCARD-004 a backdrop click closes the dialog and sends no request",
    { tag: ["@microservices", "@positive", "@discard", "@p1"] },
    async ({ microservicesPage }) => {
      const gitActions = routeWorkspaceGitActions(microservicesPage.page)
      await microservicesPage.navigateTo()
      await microservicesPage.openDiscardConfirmation(REPOSITORY_NAME)

      await microservicesPage.confirmBackdrop.click({ position: { x: 40, y: 40 } })

      await expect(microservicesPage.discardDialog).toBeHidden()
      await expect(microservicesPage.kebabTrigger(REPOSITORY_NAME)).toBeFocused()
      expect(gitActions.countFor("discard")).toBe(0)
      expect(gitActions.calls).toEqual([])
    }
  )

  test(
    "MS-DISCARD-005 confirming sends exactly one discard request",
    { tag: ["@microservices", "@positive", "@discard", "@p0"] },
    async ({ microservicesPage }) => {
      const gitActions = routeWorkspaceGitActions(microservicesPage.page)
      await microservicesPage.navigateTo()
      await microservicesPage.openDiscardConfirmation(REPOSITORY_NAME)

      await microservicesPage.discardDialogButton("Discard changes").click()

      await expect(microservicesPage.noticeBanner).toContainText(
        `Restored 3 files in ${REPOSITORY_NAME}`
      )
      await expect(microservicesPage.discardDialog).toBeHidden()
      expect(gitActions.callsFor("discard")).toEqual([
        { action: "discard", repositoryName: REPOSITORY_NAME }
      ])
      expect(gitActions.countFor("discard")).toBe(1)
    }
  )
})
