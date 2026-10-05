import { test, expect } from "../../../fixtures/authenticated.fixture"
import { CollectionExportPage } from "../../../pages/CollectionExportPage"
import { routeSecrets, routeSystemConfig } from "../../../helpers/route-mock"
import { routeWorkspaceBoard } from "../../../helpers/workspace-board"

const REPOSITORY_NAME = "atlas"
const CONFIGURED_PROVIDER = "opencode-go"
const CONFIGURED_MODEL_NAMES = [
  "DeepSeek V4.1 Flash (Go)",
  "MiMo V2.6 Flash (Go)",
  "Space Bunny Free (Go)"
]
const UNCONFIGURED_MODEL_NAMES = [
  "DeepSeek-V4 Pro (Thinking)",
  "DeepSeek Flash",
  "GPT-6 Sol",
  "Claude Sonnet 5.5",
  "Gemini 3.1 Pro",
  "Xiaomi MiMo-V2.5 Pro"
]
const FREE_MODEL_NAME = "Space Bunny Free (Go)"
const PAID_MODEL_NAMES = ["DeepSeek V4.1 Flash (Go)", "MiMo V2.6 Flash (Go)"]

test.describe("Microservices Postman export model picker", () => {
  test.beforeEach(async ({ page }) => {
    await routeSecrets(page, { hasAiKeys: true, configuredAiProviders: [CONFIGURED_PROVIDER] })
    await routeSystemConfig(page, [])
      await routeWorkspaceBoard(page)
  })

  test(
    "MS-EXP-001 the Choose AI model control is enabled and lists only the configured models",
    { tag: ["@microservices", "@positive", "@export", "@p0"] },
    async ({ page, microservicesPage }) => {
      const exportPage = new CollectionExportPage(page)
      await microservicesPage.navigateTo()
      await exportPage.expandRepository(REPOSITORY_NAME)

      const entry = exportPage.entryButton(REPOSITORY_NAME)
      await expect(entry).toBeVisible()
      await expect(entry).toBeEnabled()
      await expect(entry).toHaveText(/Choose AI model/)

      await entry.click()
      await expect(exportPage.modelPicker).toBeVisible()
      await expect(exportPage.modelPicker.locator(".model-loading")).toBeHidden()
      await expect(exportPage.modelGroup).toBeVisible()
      await expect(exportPage.modelGroup).toHaveAttribute("aria-disabled", "false")
      await expect(exportPage.modelRadios).toHaveCount(CONFIGURED_MODEL_NAMES.length)
      for (const radio of await exportPage.modelRadios.all()) {
        await expect(radio).toBeEnabled()
      }
      expect(await exportPage.modelPicker.locator(".model-option-name").allInnerTexts()).toEqual(
        CONFIGURED_MODEL_NAMES
      )
      for (const modelName of UNCONFIGURED_MODEL_NAMES) {
        await expect(exportPage.modelOption(modelName)).toHaveCount(0)
      }
      await expect(exportPage.exportButton).toBeEnabled()
    }
  )

  test(
    "MS-EXP-002 free models carry a Free badge",
    { tag: ["@microservices", "@positive", "@export", "@p1"] },
    async ({ page, microservicesPage }) => {
      const exportPage = new CollectionExportPage(page)
      await microservicesPage.navigateTo()
      await exportPage.openModelPicker(REPOSITORY_NAME)

      const freeOption = exportPage.modelOption(FREE_MODEL_NAME)
      await expect(freeOption.locator(".model-option-badge")).toBeVisible()
      await expect(freeOption.locator(".model-option-badge")).toHaveText("Free")
      await expect(freeOption.locator("input[type='radio']")).toHaveValue(
        "opencode-go-space-bunny-free"
      )
      await expect(freeOption.locator("input[type='radio']")).toHaveAttribute(
        "aria-label",
        "Space Bunny Free (Go), free"
      )
      for (const paidModelName of PAID_MODEL_NAMES) {
        await expect(exportPage.modelOption(paidModelName).locator(".model-option-badge")).toHaveCount(0)
      }
    }
  )

  test(
    "MS-EXP-003 the modal exposes no effort control and no text that names an effort setting",
    { tag: ["@microservices", "@positive", "@export", "@p0"] },
    async ({ page, microservicesPage }) => {
      const exportPage = new CollectionExportPage(page)
      await microservicesPage.navigateTo()
      await exportPage.openModelPicker(REPOSITORY_NAME)

      await expect(exportPage.modelGroup).toBeVisible()
      await expect(exportPage.modelPicker.locator("[data-testid*='effort']")).toHaveCount(0)
      await expect(exportPage.modelPicker.locator("select, [role='combobox'], [role='slider']")).toHaveCount(0)
      await expect(exportPage.modelPicker.getByText(/effort/i)).toHaveCount(0)
      await expect(exportPage.modelPicker.getByText(/reasoning/i)).toHaveCount(0)
      await expect(exportPage.modelPicker.getByText(/\b(low|medium|high)\b/i)).toHaveCount(0)

      const modalText = await exportPage.modelPicker.innerText()
      expect(modalText).not.toMatch(/effort/i)
      expect(modalText).not.toMatch(/reasoning/i)
      expect(modalText).not.toMatch(/\b(low|medium|high)\b/i)
      expect(modalText).not.toMatch(/\bmedium\b/i)
    }
  )
})