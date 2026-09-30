import { type Page } from "@playwright/test"

export class ToolsModal {
  readonly page: Page

  constructor(page: Page) {
    this.page = page
  }

  get settingsGearButton() {
    return this.page.locator(".history-settings-btn")
  }

  get activeToolsModalTitle() {
    return this.page.locator(".modal-title")
  }

  get doneButton() {
    return this.page.locator(".done-btn")
  }

  domainRow(toolName: string) {
    return this.page.locator(`.domain-row:has-text("${toolName}")`)
  }

  async openModal() {
    await this.settingsGearButton.click()
  }

  async closeModal() {
    await this.doneButton.click()
  }
}
