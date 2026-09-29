import { type Page, expect } from "@playwright/test"

export class CopilotPage {
  readonly page: Page

  constructor(page: Page) {
    this.page = page
  }

  get emptyHistoryNotice() {
    return this.page.locator(".empty-history-notice")
  }

  get emptyTitle() {
    return this.page.locator(".empty-title")
  }

  get settingsGearButton() {
    return this.page.locator(".history-settings-btn")
  }

  get activeToolsModalTitle() {
    return this.page.locator(".modal-title")
  }

  domainRow(toolName: string) {
    return this.page.locator(`.domain-row:has-text("${toolName}")`)
  }

  get doneButton() {
    return this.page.locator(".done-btn")
  }

  get thinkingToggle() {
    return this.page.locator(".thinking-toggle")
  }

  get effortTrigger() {
    return this.page.locator(".effort-trigger")
  }

  get effortMenu() {
    return this.page.locator(".effort-menu")
  }

  effortOption(label: string) {
    return this.page.locator(`.effort-option:has-text("${label}")`)
  }

  get modelTrigger() {
    return this.page.locator(".model-trigger")
  }

  get modelMenu() {
    return this.page.locator(".model-menu")
  }

  modelOption(label: string) {
    return this.page.locator(`.model-option:has-text("${label}")`)
  }

  get modelTriggerLabel() {
    return this.page.locator(".model-trigger-label")
  }

  get messageTextarea() {
    return this.page.locator("textarea")
  }

  get sendButton() {
    return this.page.locator(".send-btn")
  }

  get thoughtSection() {
    return this.page.locator(".thought-section")
  }

  get thoughtTitle() {
    return this.page.locator(".thought-title")
  }

  get thoughtContent() {
    return this.page.locator(".thought-content")
  }

  get thoughtToggle() {
    return this.page.locator(".thought-toggle")
  }

  get lastAssistantMessage() {
    return this.page.locator(".message.assistant").last()
  }

  get firstSourceChip() {
    return this.page.locator(".source-chip").first()
  }

  get firstHistoryItem() {
    return this.page.locator(".history-item").first()
  }

  historyDeleteButton(historyItem: ReturnType<Page["locator"]>) {
    return historyItem.locator(".history-delete-btn")
  }

  async navigateTo() {
    await this.page.goto("/copilot")
    await expect(this.page).toHaveURL(/\/copilot/)
  }
}
