import { type Page } from "@playwright/test"

export class IssueDetailModal {
  readonly page: Page

  constructor(page: Page) {
    this.page = page
  }

  get modalRoot() {
    return this.page.getByTestId("issue-detail-modal")
  }

  get issueKey() {
    return this.page.getByTestId("modal-issue-key")
  }

  get issueTitle() {
    return this.page.getByTestId("modal-issue-title")
  }

  get issueStatus() {
    return this.page.getByTestId("modal-issue-status")
  }

  get issuePriority() {
    return this.page.getByTestId("modal-issue-priority")
  }

  get issueDescription() {
    return this.page.getByTestId("modal-issue-description")
  }

  get readMoreButton() {
    return this.page.getByTestId("btn-read-more")
  }

  get closeButton() {
    return this.page.getByTestId("btn-modal-close")
  }

  get modalScrim() {
    return this.page.locator(".modal-backdrop")
  }

  async close() {
    await this.closeButton.click()
  }
}
