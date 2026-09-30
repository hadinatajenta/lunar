import { type Page, type Locator } from "@playwright/test"

export class ChatWindow {
  readonly page: Page

  constructor(page: Page) {
    this.page = page
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

  get emptyHistoryNotice() {
    return this.page.locator(".empty-history-notice")
  }

  get emptyTitle() {
    return this.page.locator(".empty-title")
  }

  historyDeleteButton(item: Locator) {
    return item.locator(".history-delete-btn")
  }

  async sendMessage(text: string) {
    await this.messageTextarea.fill(text)
    await this.sendButton.click()
  }
}
