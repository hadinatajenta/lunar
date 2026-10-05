import { type Locator, type Page } from "@playwright/test"
import { test, expect } from "../../../fixtures/authenticated.fixture"
import { CollectionExportPage } from "../../../pages/CollectionExportPage"
import { routeCollectionExport } from "../../../helpers/export-stream"
import { routeSecrets, routeSystemConfig } from "../../../helpers/route-mock"
import { routeWorkspaceBoard } from "../../../helpers/workspace-board"

const REPOSITORY_NAME = "atlas"
const CONFIGURED_PROVIDER = "opencode-go"

const DARK_THEME = "dark"
const LIGHT_THEME = "light"

const DARK_MODAL_SURFACE = "rgb(16, 18, 23)"
const DARK_MODAL_TEXT = "rgb(244, 246, 248)"
const DARK_LOCK_NOTICE_TEXT = "rgb(187, 169, 132)"
const LIGHT_MODAL_SURFACE = "rgb(255, 255, 255)"
const LIGHT_MODAL_TEXT = "rgb(15, 23, 42)"
const LIGHT_LOCK_NOTICE_TEXT = "rgb(180, 83, 9)"

interface ModalColours {
  surface: string
  text: string
}

const readComputedColour = (target: Locator, cssProperty: string): Promise<string> =>
  target.evaluate((element, property) => window.getComputedStyle(element).getPropertyValue(property), cssProperty)

const readPickerColours = async (exportPage: CollectionExportPage): Promise<ModalColours> => ({
  surface: await readComputedColour(exportPage.modelPicker.locator(".modal"), "background-color"),
  text: await readComputedColour(exportPage.modelPicker.locator(".modal-title"), "color")
})

const readDownloadModalColours = async (exportPage: CollectionExportPage): Promise<ModalColours> => ({
  surface: await readComputedColour(exportPage.downloadModal.locator(".modal"), "background-color"),
  text: await readComputedColour(exportPage.downloadModal.locator(".download-file"), "color")
})

const switchToLightTheme = async (page: Page): Promise<void> => {
  await page.getByTestId("theme-toggle").click()
  await expect(page.locator("html")).toHaveAttribute("data-theme", LIGHT_THEME)
}

test.describe("Microservices Postman export theme surfaces", () => {
  test(
    "MS-EXP-017 the model picker and the download modal take their surface and text colours from the active theme",
    { tag: ["@microservices", "@positive", "@export", "@theme", "@p0"] },
    async ({ page, microservicesPage }) => {
      await routeSecrets(page, { hasAiKeys: true, configuredAiProviders: [CONFIGURED_PROVIDER] })
      await routeSystemConfig(page, [])
      await routeWorkspaceBoard(page)
      routeCollectionExport(page)
      const exportPage = new CollectionExportPage(page)
      await microservicesPage.navigateTo()
      await expect(page.locator("html")).toHaveAttribute("data-theme", DARK_THEME)

      await exportPage.openModelPicker(REPOSITORY_NAME)
      const darkPicker = await readPickerColours(exportPage)
      expect(darkPicker).toEqual({ surface: DARK_MODAL_SURFACE, text: DARK_MODAL_TEXT })

      await exportPage.startExport()
      await expect(exportPage.downloadModal).toBeVisible()
      const darkDownloadModal = await readDownloadModalColours(exportPage)
      expect(darkDownloadModal).toEqual({ surface: DARK_MODAL_SURFACE, text: DARK_MODAL_TEXT })

      await page.keyboard.press("Escape")
      await expect(exportPage.downloadModal).toBeHidden()
      await switchToLightTheme(page)

      await exportPage.openModelPicker(REPOSITORY_NAME)
      const lightPicker = await readPickerColours(exportPage)
      expect(lightPicker).toEqual({ surface: LIGHT_MODAL_SURFACE, text: LIGHT_MODAL_TEXT })

      await exportPage.startExport()
      await expect(exportPage.downloadModal).toBeVisible()
      const lightDownloadModal = await readDownloadModalColours(exportPage)
      expect(lightDownloadModal).toEqual({ surface: LIGHT_MODAL_SURFACE, text: LIGHT_MODAL_TEXT })

      expect(lightPicker.surface).not.toBe(darkPicker.surface)
      expect(lightPicker.text).not.toBe(darkPicker.text)
      expect(lightDownloadModal.surface).not.toBe(darkDownloadModal.surface)
      expect(lightDownloadModal.text).not.toBe(darkDownloadModal.text)
    }
  )

  test(
    "MS-EXP-018 the locked model picker keeps its lock notice legible in both themes",
    { tag: ["@microservices", "@positive", "@export", "@theme", "@lock", "@p1"] },
    async ({ page, microservicesPage }) => {
      await routeSecrets(page, { hasAiKeys: false, configuredAiProviders: [] })
      await routeSystemConfig(page, [])
      await routeWorkspaceBoard(page)
      const exportPage = new CollectionExportPage(page)
      await microservicesPage.navigateTo()
      await expect(page.locator("html")).toHaveAttribute("data-theme", DARK_THEME)

      await exportPage.openModelPicker(REPOSITORY_NAME)
      await expect(exportPage.lockNotice).toBeVisible()
      const darkNoticeColour = await readComputedColour(exportPage.lockNotice, "color")
      expect(darkNoticeColour).toBe(DARK_LOCK_NOTICE_TEXT)

      await page.keyboard.press("Escape")
      await expect(exportPage.modelPicker).toHaveCount(0)
      await switchToLightTheme(page)

      await exportPage.openModelPicker(REPOSITORY_NAME)
      await expect(exportPage.lockNotice).toBeVisible()
      const lightNoticeColour = await readComputedColour(exportPage.lockNotice, "color")
      expect(lightNoticeColour).toBe(LIGHT_LOCK_NOTICE_TEXT)
      expect(lightNoticeColour).not.toBe(darkNoticeColour)
    }
  )
})
