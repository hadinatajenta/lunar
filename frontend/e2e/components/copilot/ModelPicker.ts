import { type Page } from "@playwright/test"

export class ModelPicker {
  readonly page: Page

  constructor(page: Page) {
    this.page = page
  }

  get modelTrigger() {
    return this.page.locator(".model-trigger")
  }

  get modelMenu() {
    return this.page.locator(".model-menu")
  }

  get modelTriggerLabel() {
    return this.page.locator(".model-trigger-label")
  }

  get disabledModelOptions() {
    return this.page.getByTestId("model-option-disabled")
  }

  modelOption(label: string) {
    return this.page.locator(`.model-option:has-text("${label}")`)
  }

  async selectModel(label: string) {
    await this.modelTrigger.click()
    await this.modelOption(label).click()
  }
}
