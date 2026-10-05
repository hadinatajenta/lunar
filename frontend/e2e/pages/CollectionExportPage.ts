import { type Locator, type Page, expect } from "@playwright/test"
import { MicroservicesPage } from "./MicroservicesPage"

export const EXPORT_STAGE_LABELS = [
  "Selecting the service files",
  "Building the analysis prompt",
  "Asking the AI model",
  "Validating the collection"
]

export const EXPORT_STAGE_DETAILS = [
  "selected 37 files from atlas",
  "built a prompt with the gin appendix",
  "awaiting the provider",
  "validating"
]

export class CollectionExportPage {
  readonly page: Page
  readonly microservices: MicroservicesPage

  constructor(page: Page) {
    this.page = page
    this.microservices = new MicroservicesPage(page)
  }

  get modelPicker(): Locator {
    return this.page.getByTestId("model-picker-overlay")
  }

  get downloadModal(): Locator {
    return this.page.getByTestId("collection-download-overlay")
  }

  get modelGroup(): Locator {
    return this.modelPicker.getByRole("radiogroup", { name: "Choose AI model" })
  }

  get modelRadios(): Locator {
    return this.modelGroup.locator("input[type='radio']")
  }

  get stageRows(): Locator {
    return this.modelPicker.locator(".stage-row")
  }

  get stageNames(): Locator {
    return this.modelPicker.locator(".stage-name")
  }

  get stageDetails(): Locator {
    return this.modelPicker.locator(".stage-detail")
  }

  get exportButton(): Locator {
    return this.modelPicker.locator(".footer-btn.is-ai")
  }

  get errorAlert(): Locator {
    return this.modelPicker.getByRole("alert")
  }

  get lockNotice(): Locator {
    return this.modelPicker.locator(".model-lock")
  }

  get settingsLink(): Locator {
    return this.modelPicker.getByRole("link", { name: "Open Settings" })
  }

  get unavailableModelControl(): Locator {
    return this.modelPicker.locator(".model-option.is-unavailable")
  }

  get downloadButton(): Locator {
    return this.downloadModal.getByRole("button", { name: "Download JSON" })
  }

  get closeDownloadButton(): Locator {
    return this.downloadModal.locator(".footer-btn.is-secondary")
  }

  get downloadFileName(): Locator {
    return this.downloadModal.locator(".download-file")
  }

  get downloadStatValues(): Locator {
    return this.downloadModal.locator(".download-stat-value")
  }

  entryButton(repositoryName: string): Locator {
    return this.page.getByRole("button", { name: `Choose an AI model for ${repositoryName}` })
  }

  modelOption(modelName: string): Locator {
    return this.modelPicker.locator(".model-option").filter({ hasText: modelName })
  }

  async expandRepository(repositoryName: string): Promise<void> {
    await this.microservices.openRepositoryActions(repositoryName)
    await this.microservices.actionMenuItem("Details").click()
    await expect(this.microservices.expandPanel(repositoryName)).toBeVisible()
  }

  async openModelPicker(repositoryName: string): Promise<void> {
    const isExpanded = await this.microservices.expandPanel(repositoryName).isVisible()
    if (!isExpanded) {
      await this.expandRepository(repositoryName)
    }
    await this.entryButton(repositoryName).click()
    await expect(this.modelPicker).toBeVisible()
    await expect(this.modelPicker.locator(".model-loading")).toBeHidden()
  }

  async focusWithTabKey(target: Locator, pressLimit = 40): Promise<void> {
    for (let pressCount = 0; pressCount < pressLimit; pressCount += 1) {
      await this.page.keyboard.press("Tab")
      if (await target.evaluate((element) => element === document.activeElement)) {
        return
      }
    }
    throw new Error("the control was not reachable with the Tab key")
  }

  async startExport(): Promise<void> {
    await expect(this.exportButton).toBeEnabled()
    await this.exportButton.click()
  }
}
