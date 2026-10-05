import { test, expect } from "../../../fixtures/authenticated.fixture"
import { CollectionExportPage } from "../../../pages/CollectionExportPage"
import { routeSecrets, routeSystemConfig } from "../../../helpers/route-mock"
import { routeWorkspaceBoard } from "../../../helpers/workspace-board"

const REPOSITORY_NAME = "atlas"
const LOCKED_REASON = "No AI provider key is configured"

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
      await expect(exportPage.modelGroup).toHaveCount(0)

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
})