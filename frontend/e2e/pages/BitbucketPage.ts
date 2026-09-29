import { type Page, expect } from "@playwright/test"

export class BitbucketPage {
  readonly page: Page

  constructor(page: Page) {
    this.page = page
  }

  get pageHeading() {
    return this.page.getByRole("heading", { level: 1 })
  }

  get pushCards() {
    return this.page.locator(".push-card")
  }

  get prTableRows() {
    return this.page.locator("table.pr-table tbody tr")
  }

  get errorBanner() {
    return this.page.getByTestId("banner-bitbucket-error")
  }

  get noPatWarningBanner() {
    return this.page.getByTestId("banner-no-pat")
  }

  pushCardAt(index: number) {
    return this.page.getByTestId("push-card").nth(index)
  }

  pushFilterButton(filter: "all" | "ready" | "stale") {
    return this.page.getByTestId(`push-filter-${filter}`)
  }

  prFilterButton(filter: "all" | "ai" | "mine") {
    return this.page.getByTestId(`pr-filter-${filter}`)
  }

  createPrButton(pushId: string) {
    return this.page.getByTestId(`btn-create-pr-${pushId}`)
  }

  reviewButton(prNumber: string | number) {
    return this.page.getByTestId(`btn-review-${prNumber}`)
  }

  get createPrModal() {
    return this.page.getByTestId("create-pr-modal")
  }

  get createPrSourceBranch() {
    return this.page.getByTestId("create-pr-source")
  }

  get submitPrButton() {
    return this.page.getByTestId("btn-submit-pr")
  }

  get reviewModal() {
    return this.page.getByTestId("review-modal")
  }

  get reviewModalTitle() {
    return this.page.getByTestId("review-modal-title")
  }

  get aiEmptyState() {
    return this.page.getByTestId("ai-empty")
  }

  get toggleDiffButton() {
    return this.page.getByTestId("btn-toggle-diff")
  }

  get diffViewer() {
    return this.page.getByTestId("diff-viewer")
  }

  get triggerAiReviewButton() {
    return this.page.getByTestId("btn-trigger-ai-review")
  }

  get aiFindings() {
    return this.page.getByTestId("ai-findings")
  }

  get aiSummary() {
    return this.page.getByTestId("ai-summary")
  }

  get reviewCommentInput() {
    return this.page.getByTestId("input-review-comment")
  }

  get sendCommentButton() {
    return this.page.getByTestId("btn-send-comment")
  }

  approveButton() {
    return this.page.getByTestId("btn-action-approve")
  }

  get globalToast() {
    return this.page.getByTestId("global-toast")
  }

  get emptyStates() {
    return this.page.locator(".empty-state")
  }

  async navigateTo() {
    await this.page.goto("/bitbucket")
    await expect(this.page).toHaveURL(/\/bitbucket/)
    await expect(this.pageHeading).toContainText("Bitbucket")
  }

  async assertErrorBannerContains(text: string) {
    await expect(this.errorBanner).toBeVisible()
    await expect(this.errorBanner).toContainText(text)
  }
}
