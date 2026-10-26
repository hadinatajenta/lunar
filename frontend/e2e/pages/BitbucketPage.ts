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

  get reviewModalOverlay() {
    return this.page.getByTestId("review-modal-overlay")
  }

  get reviewModal() {
    return this.page.getByTestId("review-modal")
  }

  get closeReviewButton() {
    return this.page.getByTestId("btn-close-review")
  }

  get reviewModalId() {
    return this.page.getByTestId("review-modal-id")
  }

  get reviewModalTitle() {
    return this.page.getByTestId("review-modal-title")
  }

  get reviewModalRepo() {
    return this.page.getByTestId("review-modal-repo")
  }

  get reviewModalBranch() {
    return this.page.getByTestId("review-modal-branch")
  }

  get reviewModalAuthor() {
    return this.page.getByTestId("review-modal-author")
  }

  get reviewModalFiles() {
    return this.page.getByTestId("review-modal-files")
  }

  get reviewFindingsTab() {
    return this.page.getByTestId("tab-review-findings")
  }

  get reviewCommentTab() {
    return this.page.getByTestId("tab-review-comment")
  }

  get toggleDiffButton() {
    return this.page.getByTestId("btn-toggle-diff")
  }

  get diffViewer() {
    return this.page.getByTestId("diff-viewer")
  }

  get diffFileList() {
    return this.page.getByTestId("diff-file-list")
  }

  diffFile(index: number) {
    return this.page.getByTestId(`diff-file-${index}`)
  }

  diffFileToggle(index: number) {
    return this.page.getByTestId(`diff-file-toggle-${index}`)
  }

  get aiPanel() {
    return this.page.getByTestId("ai-panel")
  }

  get noAiKeysBanner() {
    return this.page.getByTestId("banner-no-ai-keys")
  }

  get aiEmptyState() {
    return this.page.getByTestId("ai-empty")
  }

  get aiLoading() {
    return this.page.getByTestId("ai-loading")
  }

  get aiError() {
    return this.page.getByTestId("ai-error")
  }

  get retryAiReviewButton() {
    return this.page.getByTestId("btn-retry-ai-review")
  }

  get aiSummary() {
    return this.page.getByTestId("ai-summary")
  }

  get aiRecommendation() {
    return this.page.getByTestId("ai-recommendation")
  }

  get aiFindings() {
    return this.page.getByTestId("ai-findings")
  }

  get aiGroupBlocker() {
    return this.page.getByTestId("ai-group-blocker")
  }

  get aiGroupSuggestion() {
    return this.page.getByTestId("ai-group-suggestion")
  }

  get aiGroupNitpick() {
    return this.page.getByTestId("ai-group-nitpick")
  }

  aiFinding(index: number) {
    return this.page.getByTestId(`ai-finding-${index}`)
  }

  get aiAssumptions() {
    return this.page.getByTestId("ai-assumptions")
  }

  get aiStrengths() {
    return this.page.getByTestId("ai-strengths")
  }

  get aiTruncatedNotice() {
    return this.page.getByTestId("ai-truncated-notice")
  }

  get triggerAiReviewButton() {
    return this.page.getByTestId("btn-trigger-ai-review")
  }

  get reviewCommentInput() {
    return this.page.getByTestId("input-review-comment")
  }

  get toggleCommentPreviewButton() {
    return this.page.getByTestId("btn-toggle-comment-preview")
  }

  get commentPreview() {
    return this.page.getByTestId("comment-preview")
  }

  get sendCommentButton() {
    return this.page.getByTestId("btn-send-comment")
  }

  get prReviewModelTrigger() {
    return this.page.getByTestId("pr-review-model-trigger")
  }

  get prReviewModelRootOption() {
    return this.page.getByTestId("review-modal").getByTestId("model-root-option-model")
  }

  get prReviewModelMenu() {
    return this.page.getByTestId("review-modal").locator(".model-menu")
  }

  prReviewModelOption(modelName: string) {
    return this.prReviewModelMenu.getByRole("button", { name: modelName, exact: true })
  }

  async openReviewModelPicker() {
    await this.prReviewModelTrigger.click()
    await this.prReviewModelRootOption.click()
  }

  approveButton() {
    return this.page.getByTestId("btn-action-approve")
  }

  needsWorkButton() {
    return this.page.getByTestId("btn-action-needs-work")
  }

  declineButton() {
    return this.page.getByTestId("btn-action-decline")
  }

  get confirmAcceptButton() {
    return this.page.getByTestId("confirm-accept")
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
