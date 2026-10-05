import { test, expect } from "../../../fixtures/authenticated.fixture"
import { routeWorkspaceGitActions } from "../../../helpers/route-mock"

const REPOSITORY_NAME = "aurora"
const MENU_ITEM_LABELS = ["Pull", "Fetch", "Stash changes local", "Discard changes", "Details"]

test.describe("Microservices repository actions menu", () => {
  test(
    "MS-MENU-001 the kebab trigger is a button that opens the repository actions menu",
    { tag: ["@microservices", "@positive", "@actions-menu", "@p0"] },
    async ({ microservicesPage }) => {
      routeWorkspaceGitActions(microservicesPage.page)
      await microservicesPage.navigateTo()

      const trigger = microservicesPage.kebabTrigger(REPOSITORY_NAME)
      await expect(trigger).toHaveRole("button")
      await expect(trigger).toHaveAttribute("aria-haspopup", "menu")
      await expect(trigger).toHaveAttribute("aria-expanded", "false")
      await expect(trigger).toHaveAccessibleName(`Repository actions for ${REPOSITORY_NAME}`)
      await expect(trigger.locator("svg circle")).toHaveCount(3)
      await expect(microservicesPage.actionMenu).toBeHidden()

      await microservicesPage.openRepositoryActions(REPOSITORY_NAME)

      await expect(trigger).toHaveAttribute("aria-expanded", "true")
      await expect(microservicesPage.actionMenu).toHaveRole("menu")
      await expect(microservicesPage.actionMenu).toHaveAccessibleName(
        `Actions for ${REPOSITORY_NAME}`
      )
      await expect(microservicesPage.actionMenuItems).toHaveCount(MENU_ITEM_LABELS.length)
    }
  )

  test(
    "MS-MENU-002 the menu lists Pull, Fetch, Stash changes local, Discard changes and Details in order",
    { tag: ["@microservices", "@positive", "@actions-menu", "@p0"] },
    async ({ microservicesPage }) => {
      routeWorkspaceGitActions(microservicesPage.page)
      await microservicesPage.navigateTo()
      await microservicesPage.openRepositoryActions(REPOSITORY_NAME)

      await expect(microservicesPage.actionMenuItems).toHaveText(MENU_ITEM_LABELS)
      expect(await microservicesPage.actionMenuItems.allInnerTexts()).toEqual(MENU_ITEM_LABELS)
    }
  )

  test(
    "MS-MENU-003 Details toggles the expand panel and reflects aria-expanded",
    { tag: ["@microservices", "@positive", "@actions-menu", "@p0"] },
    async ({ microservicesPage }) => {
      routeWorkspaceGitActions(microservicesPage.page)
      await microservicesPage.navigateTo()
      const expandPanel = microservicesPage.expandPanel(REPOSITORY_NAME)
      await expect(expandPanel).toBeHidden()

      await microservicesPage.openRepositoryActions(REPOSITORY_NAME)
      await expect(microservicesPage.actionMenuItem("Details")).toHaveAttribute(
        "aria-expanded",
        "false"
      )
      await microservicesPage.actionMenuItem("Details").click()
      await expect(expandPanel).toBeVisible()

      await microservicesPage.openRepositoryActions(REPOSITORY_NAME)
      await expect(microservicesPage.actionMenuItem("Details")).toHaveAttribute(
        "aria-expanded",
        "true"
      )
      await microservicesPage.actionMenuItem("Details").click()
      await expect(expandPanel).toBeHidden()

      await microservicesPage.openRepositoryActions(REPOSITORY_NAME)
      await expect(microservicesPage.actionMenuItem("Details")).toHaveAttribute(
        "aria-expanded",
        "false"
      )
    }
  )

  test(
    "MS-MENU-004 Pull sends exactly one pull request",
    { tag: ["@microservices", "@positive", "@actions-menu", "@p0"] },
    async ({ microservicesPage }) => {
      const gitActions = routeWorkspaceGitActions(microservicesPage.page)
      await microservicesPage.navigateTo()

      await microservicesPage.openRepositoryActions(REPOSITORY_NAME)
      await microservicesPage.actionMenuItem("Pull").click()

      await expect(microservicesPage.noticeBanner).toContainText(
        `Pulled ${REPOSITORY_NAME} to development`
      )
      expect(gitActions.callsFor("pull")).toEqual([
        { action: "pull", repositoryName: REPOSITORY_NAME }
      ])
      expect(gitActions.countFor("pull")).toBe(1)
      await expect(microservicesPage.actionMenu).toBeHidden()
    }
  )

  test(
    "MS-MENU-005 Fetch sends exactly one fetch request",
    { tag: ["@microservices", "@positive", "@actions-menu", "@p0"] },
    async ({ microservicesPage }) => {
      const gitActions = routeWorkspaceGitActions(microservicesPage.page)
      await microservicesPage.navigateTo()

      await microservicesPage.openRepositoryActions(REPOSITORY_NAME)
      await microservicesPage.actionMenuItem("Fetch").click()

      await expect(microservicesPage.noticeBanner).toContainText(
        `Fetched ${REPOSITORY_NAME}: 2 ahead, 1 behind`
      )
      expect(gitActions.callsFor("fetch")).toEqual([
        { action: "fetch", repositoryName: REPOSITORY_NAME }
      ])
      expect(gitActions.countFor("fetch")).toBe(1)
    }
  )

  test(
    "MS-MENU-006 Stash changes local sends exactly one stash request",
    { tag: ["@microservices", "@positive", "@actions-menu", "@p0"] },
    async ({ microservicesPage }) => {
      const gitActions = routeWorkspaceGitActions(microservicesPage.page)
      await microservicesPage.navigateTo()

      await microservicesPage.openRepositoryActions(REPOSITORY_NAME)
      await microservicesPage.actionMenuItem("Stash changes local").click()

      await expect(microservicesPage.noticeBanner).toContainText(
        `Stashed local changes in ${REPOSITORY_NAME}: 1 stash entry`
      )
      expect(gitActions.callsFor("stash")).toEqual([
        { action: "stash", repositoryName: REPOSITORY_NAME }
      ])
      expect(gitActions.countFor("stash")).toBe(1)
    }
  )
})
