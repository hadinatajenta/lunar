import { test, expect } from "../../../fixtures/authenticated.fixture"
import { CollectionExportPage } from "../../../pages/CollectionExportPage"
import { routeCollectionExport } from "../../../helpers/export-stream"
import { routeSecrets, routeSystemConfig } from "../../../helpers/route-mock"
import { routeWorkspaceBoard } from "../../../helpers/workspace-board"

const REPOSITORY_NAME = "atlas"
const CONFIGURED_PROVIDER = "opencode-go"

test.describe("Microservices Postman export keyboard access", () => {
  test.beforeEach(async ({ page }) => {
    await routeSecrets(page, { hasAiKeys: true, configuredAiProviders: [CONFIGURED_PROVIDER] })
    await routeSystemConfig(page, [])
      await routeWorkspaceBoard(page)
  })

  test(
    "MS-EXP-014 the model picker is reachable, operable and dismissible without a mouse",
    { tag: ["@microservices", "@positive", "@export", "@accessibility", "@p0"] },
    async ({ page, microservicesPage }) => {
      routeCollectionExport(page)
      const exportPage = new CollectionExportPage(page)
      await microservicesPage.navigateTo()
      await microservicesPage.openRepositoryActions(REPOSITORY_NAME)
      await microservicesPage.actionMenuItem("Details").click()
      await expect(microservicesPage.expandPanel(REPOSITORY_NAME)).toBeVisible()

      const entry = exportPage.entryButton(REPOSITORY_NAME)
      await exportPage.focusWithTabKey(entry)
      await expect(entry).toBeFocused()

      await page.keyboard.press("Enter")
      await expect(exportPage.modelPicker).toBeVisible()
      await expect(exportPage.modelRadios.first()).toBeFocused()

      await page.keyboard.press("ArrowDown")
      await expect(exportPage.modelRadios.nth(1)).toBeChecked()
      await expect(exportPage.modelRadios.nth(1)).toBeFocused()

      await page.keyboard.press("Escape")
      await expect(exportPage.modelPicker).toHaveCount(0)
      await expect(entry).toBeFocused()
    }
  )

  test(
    "MS-EXP-015 the download modal focuses the download button and closes with Escape",
    { tag: ["@microservices", "@positive", "@export", "@accessibility", "@p0"] },
    async ({ page, microservicesPage }) => {
      routeCollectionExport(page)
      const exportPage = new CollectionExportPage(page)
      await microservicesPage.navigateTo()
      await exportPage.openModelPicker(REPOSITORY_NAME)
      await exportPage.startExport()
      await expect(exportPage.downloadModal).toBeVisible()

      await expect(exportPage.downloadButton).toBeFocused()

      await page.keyboard.press("Escape")

      await expect(exportPage.downloadModal).toBeHidden()
      await expect(exportPage.modelPicker).toHaveCount(0)
      await expect(exportPage.entryButton(REPOSITORY_NAME)).toBeFocused()
    }
  )
})