import { test, expect } from "../../../fixtures/authenticated.fixture"
import { CollectionExportPage } from "../../../pages/CollectionExportPage"
import { routeSecrets, routeSystemConfig } from "../../../helpers/route-mock"
import { routeWorkspaceBoard } from "../../../helpers/workspace-board"

const REPOSITORY_NAME = "atlas"
const LOCKED_REASON = "No AI provider key is configured"
const MODEL_GROUP_NAME = "Choose AI model"
const CONFIGURED_PROVIDER = "opencode-go"
const CONFIGURED_MODEL_COUNT = 3

const buildSecretsResponse = (hasConfiguredModels: boolean): string =>
  JSON.stringify({
    user_id: "test-user-id",
    has_jira_pat: true,
    jira_username: "developer",
    has_bitbucket_pat: true,
    bitbucket_username: "developer",
    has_confluence_pat: true,
    has_ai_keys: hasConfiguredModels,
    configured_ai_providers: hasConfiguredModels ? [CONFIGURED_PROVIDER] : [],
    updated_at: "Just now"
  })

test.describe("Microservices Postman export locked state", () => {
  test(
    "MS-EXP-013 without a provider key the model control stays visible, disabled and explained",
    { tag: ["@microservices", "@positive", "@export", "@lock", "@p0"] },
    async ({ page, microservicesPage }) => {
      await routeSecrets(page, { hasAiKeys: false, configuredAiProviders: [] })
      await routeSystemConfig(page, [])
      await routeWorkspaceBoard(page)
      const exportPage = new CollectionExportPage(page)
      await microservicesPage.navigateTo()
      await exportPage.expandRepository(REPOSITORY_NAME)

      const entry = exportPage.entryButton(REPOSITORY_NAME)
      await expect(entry).toBeVisible()
      await expect(entry).toBeEnabled()

      await entry.click()
      await expect(exportPage.modelPicker).toBeVisible()
      await expect(exportPage.modelPicker.locator(".model-loading")).toBeHidden()

      const lockedControl = exportPage.unavailableModelControl
      await expect(lockedControl).toBeVisible()
      await expect(lockedControl).toBeDisabled()
      await expect(lockedControl).toHaveText("No AI model available")
      await expect(exportPage.modelPicker.locator("input[type='radio']:not([disabled])")).toHaveCount(0)

      const lockedGroup = exportPage.modelGroup
      await expect(lockedGroup).toBeVisible()
      await expect(lockedGroup).toHaveAccessibleName(MODEL_GROUP_NAME)
      await expect(lockedGroup).toBeDisabled()
      await expect(lockedGroup.locator(".model-option.is-unavailable")).toHaveCount(1)

      await expect(exportPage.lockNotice).toBeVisible()
      await expect(exportPage.lockNotice).toContainText(LOCKED_REASON)
      await expect(exportPage.lockNotice).toContainText("locked")
      await expect(exportPage.settingsLink).toBeVisible()
      await expect(exportPage.settingsLink).toHaveAttribute("href", "/settings")

      await expect(exportPage.exportButton).toBeDisabled()
      await expect(exportPage.downloadModal).toHaveCount(0)

      await exportPage.settingsLink.click()
      await expect(page).toHaveURL(/\/settings$/)
    }
  )

  test(
    "MS-EXP-016 the model control keeps the same accessible name once a provider key unlocks it",
    { tag: ["@microservices", "@positive", "@export", "@lock", "@accessibility", "@p0"] },
    async ({ page, microservicesPage }) => {
      let secretsResponseHasModels = false
      await page.route(/\/api\/auth\/secrets/, async (route) => {
        if (route.request().method() !== "GET") {
          await route.continue()
          return
        }
        await route.fulfill({
          status: 200,
          contentType: "application/json",
          body: buildSecretsResponse(secretsResponseHasModels)
        })
      })
      await routeSystemConfig(page, [])
      await routeWorkspaceBoard(page)
      const exportPage = new CollectionExportPage(page)
      await microservicesPage.navigateTo()
      await exportPage.openModelPicker(REPOSITORY_NAME)

      const lockedGroup = exportPage.modelGroup
      await expect(lockedGroup).toBeVisible()
      await expect(lockedGroup).toHaveAccessibleName(MODEL_GROUP_NAME)
      await expect(lockedGroup).toBeDisabled()
      await expect(exportPage.modelPicker.locator("input[type='radio']:not([disabled])")).toHaveCount(0)
      await expect(exportPage.unavailableModelControl).toBeVisible()
      await expect(exportPage.unavailableModelControl).toBeDisabled()

      secretsResponseHasModels = true
      await page.keyboard.press("Escape")
      await expect(exportPage.modelPicker).toHaveCount(0)

      await exportPage.openModelPicker(REPOSITORY_NAME)

      const unlockedGroup = exportPage.modelGroup
      await expect(unlockedGroup).toBeVisible()
      await expect(unlockedGroup).toHaveAccessibleName(MODEL_GROUP_NAME)
      await expect(unlockedGroup).toBeEnabled()
      await expect(exportPage.modelRadios).toHaveCount(CONFIGURED_MODEL_COUNT)
      await expect(unlockedGroup.locator(".model-option.is-unavailable")).toHaveCount(0)
      await expect(exportPage.lockNotice).toHaveCount(0)
    }
  )
})
