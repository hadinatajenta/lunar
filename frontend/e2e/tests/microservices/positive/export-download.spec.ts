import type { Download } from "@playwright/test"
import { test, expect } from "../../../fixtures/authenticated.fixture"
import { CollectionExportPage } from "../../../pages/CollectionExportPage"
import { routeCollectionExport } from "../../../helpers/export-stream"
import { routeSecrets, routeSystemConfig } from "../../../helpers/route-mock"
import { routeWorkspaceBoard } from "../../../helpers/workspace-board"

const REPOSITORY_NAME = "atlas"
const CONFIGURED_PROVIDER = "opencode-go"
const COLLECTION_FILE_NAME = "atlas-postman-collection.json"
const FREE_MODEL_NAME = "Space Bunny Free (Go)"
const FREE_MODEL_ID = "opencode-go-space-bunny-free"
const MODEL_PREFERENCE_KEY = "lunar_microservices_model_preferences"

const readDownloadedText = async (download: Download): Promise<string> => {
  const fileStream = await download.createReadStream()
  const chunks: Buffer[] = []
  for await (const chunk of fileStream) {
    chunks.push(chunk as Buffer)
  }
  return Buffer.concat(chunks).toString("utf8")
}

const readStoredPreferences = (page: import("@playwright/test").Page): Promise<Record<string, string>> =>
  page.evaluate(
    (storageKey) => JSON.parse(localStorage.getItem(storageKey) ?? "{}") as Record<string, string>,
    MODEL_PREFERENCE_KEY
  )

test.describe("Microservices Postman export download", () => {
  test.beforeEach(async ({ page }) => {
    await routeSecrets(page, { hasAiKeys: true, configuredAiProviders: [CONFIGURED_PROVIDER] })
    await routeSystemConfig(page, [])
      await routeWorkspaceBoard(page)
  })

  test(
    "MS-EXP-007 the download button produces atlas-postman-collection.json built in the browser",
    { tag: ["@microservices", "@positive", "@export", "@p0"] },
    async ({ page, microservicesPage }) => {
      routeCollectionExport(page)
      const exportPage = new CollectionExportPage(page)
      await microservicesPage.navigateTo()
      await exportPage.openModelPicker(REPOSITORY_NAME)
      await exportPage.startExport()
      await expect(exportPage.downloadModal).toBeVisible()

      const downloadPromise = page.waitForEvent("download")
      await exportPage.downloadButton.click()
      const download = await downloadPromise

      expect(download.suggestedFilename()).toBe(COLLECTION_FILE_NAME)
      expect(download.url().startsWith("blob:")).toBe(true)

      const downloadedText = await readDownloadedText(download)
      const collection = JSON.parse(downloadedText) as {
        info: { name: string }
        item: unknown[]
        variable: unknown[]
      }
      expect(collection.info.name).toBe(REPOSITORY_NAME)
      expect(collection.item).toEqual([])
      expect(collection.variable).toEqual([])
      expect(Object.keys(collection).sort()).toEqual(["info", "item", "variable"])
      expect(downloadedText).not.toMatch(/effort/i)
      expect(downloadedText).not.toMatch(/reasoning/i)
    }
  )

  test(
    "MS-EXP-008 Escape dismisses the download modal without re-fetching the export or the providers",
    { tag: ["@microservices", "@positive", "@export", "@p0"] },
    async ({ page, microservicesPage }) => {
      const providerRequests: string[] = []
      page.on("request", (request) => {
        if (new URL(request.url()).pathname === "/api/auth/secrets") {
          providerRequests.push(request.url())
        }
      })
      const recorder = routeCollectionExport(page)
      const exportPage = new CollectionExportPage(page)
      await microservicesPage.navigateTo()
      await exportPage.openModelPicker(REPOSITORY_NAME)
      await exportPage.startExport()
      await expect(exportPage.downloadModal).toBeVisible()

      const providerRequestCount = providerRequests.length
      expect(providerRequestCount).toBeGreaterThan(0)
      expect(await readStoredPreferences(page)).toEqual({
        [REPOSITORY_NAME]: "opencode-go-deepseek-v4.1-flash"
      })

      await page.keyboard.press("Escape")

      await expect(exportPage.downloadModal).toBeHidden()
      await expect(exportPage.modelPicker).toHaveCount(0)
      expect(providerRequests.length).toBe(providerRequestCount)
      expect(recorder.requests.length).toBe(1)
      expect(await readStoredPreferences(page)).toEqual({
        [REPOSITORY_NAME]: "opencode-go-deepseek-v4.1-flash"
      })
    }
  )

  test(
    "MS-EXP-009 Close dismisses the download modal and the saved model choice survives the next open",
    { tag: ["@microservices", "@positive", "@export", "@p1"] },
    async ({ page, microservicesPage }) => {
      routeCollectionExport(page)
      const exportPage = new CollectionExportPage(page)
      await microservicesPage.navigateTo()
      await exportPage.openModelPicker(REPOSITORY_NAME)

      await exportPage.modelOption(FREE_MODEL_NAME).click()
      await expect(exportPage.modelOption(FREE_MODEL_NAME).locator("input[type='radio']")).toBeChecked()
      expect(await readStoredPreferences(page)).toEqual({ [REPOSITORY_NAME]: FREE_MODEL_ID })

      await exportPage.startExport()
      await expect(exportPage.downloadModal).toBeVisible()
      await expect(exportPage.downloadFileName).toHaveText(COLLECTION_FILE_NAME)

      await exportPage.closeDownloadButton.click()

      await expect(exportPage.downloadModal).toBeHidden()
      await expect(exportPage.modelPicker).toHaveCount(0)
      expect(await readStoredPreferences(page)).toEqual({ [REPOSITORY_NAME]: FREE_MODEL_ID })

      await exportPage.openModelPicker(REPOSITORY_NAME)
      await expect(exportPage.modelOption(FREE_MODEL_NAME)).toHaveClass(/is-selected/)
      await expect(exportPage.modelOption(FREE_MODEL_NAME).locator("input[type='radio']")).toBeChecked()
    }
  )
})