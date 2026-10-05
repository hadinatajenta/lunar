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

  get noAiKeysGate() {
    return this.page.getByTestId("banner-no-ai-keys")
  }

  get goToSettingsButton() {
    return this.page.getByTestId("btn-go-to-settings")
  }

  get disabledModelOptions() {
    return this.page.getByTestId("model-option-disabled")
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

  get modelRootOptionModel() {
    return this.page.getByTestId("model-root-option-model")
  }

  get modelRootOptionEffort() {
    return this.page.getByTestId("model-root-option-effort")
  }

  get modelBackButton() {
    return this.page.getByTestId("model-picker-back")
  }

  get modelSearchInput() {
    return this.page.getByTestId("model-search-input")
  }

  effortOption(effortId: string) {
    return this.page.getByTestId(`effort-option-${effortId}`)
  }

  get composer() {
    return this.page.locator(".chat-composer")
  }

  get composerInput() {
    return this.page.getByTestId("composer-input")
  }

  get composerSendButton() {
    return this.page.getByTestId("composer-send")
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

  get reasoningPanel() {
    return this.page.locator(".reasoning-panel")
  }

  get reasoningTitle() {
    return this.page.locator(".reasoning-title")
  }

  get reasoningContent() {
    return this.page.locator(".reasoning-body")
  }

  get reasoningToggle() {
    return this.page.locator(".reasoning-toggle")
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
