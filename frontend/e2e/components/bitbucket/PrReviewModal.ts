import { type Page } from "@playwright/test"

export class PrReviewModal {
  readonly page: Page

  constructor(page: Page) {
    this.page = page
  }

  get modalRoot() {
    return this.page.getByTestId("pr-review-modal")
  }

  get modelSelector() {
    return this.page.getByTestId("select-review-model")
  }

  get noAiWarning() {
    return this.page.getByTestId("banner-review-no-ai")
  }

  get diffContainer() {
    return this.page.getByTestId("diff-container")
  }

  get generateAiReviewButton() {
    return this.page.getByTestId("btn-generate-ai-review")
  }

  get findingsContainer() {
    return this.page.getByTestId("ai-review-findings")
  }

  get commentTextarea() {
    return this.page.getByTestId("input-review-comment")
  }

  get postCommentButton() {
    return this.page.getByTestId("btn-post-comment")
  }

  get approvePrButton() {
    return this.page.getByTestId("btn-approve-pr")
  }

  get toastReviewAction() {
    return this.page.getByTestId("toast-review-action")
  }
}
