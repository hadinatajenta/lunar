import { type Page } from "@playwright/test"

export class ConfluencePage {
  readonly page: Page

  constructor(page: Page) {
    this.page = page
  }

  get statCards() {
    return this.page.getByTestId("stat-card")
  }

  get searchInput() {
    return this.page.getByTestId("doc-search")
  }

  get filterChips() {
    return this.page.getByTestId("doc-filter")
  }

  get documentCards() {
    return this.page.locator('[data-testid="doc-card"]')
  }

  get visibleCount() {
    return this.page.getByTestId("doc-visible-count")
  }

  get showMoreButton() {
    return this.page.getByTestId("doc-show-more")
  }

  get detailTitle() {
    return this.page.getByTestId("detail-title")
  }

  get detailTypeBadge() {
    return this.page.getByTestId("detail-type-badge")
  }

  get detailStatus() {
    return this.page.getByTestId("detail-status")
  }

  get detailDescription() {
    return this.page.getByTestId("detail-description")
  }

  get detailMeta() {
    return this.page.getByTestId("detail-meta")
  }

  get detailLastEditor() {
    return this.page.getByTestId("detail-last-editor")
  }

  get detailReadMore() {
    return this.page.getByTestId("detail-read-more")
  }

  get detailBody() {
    return this.page.getByTestId("detail-body")
  }

  get detailBackButton() {
    return this.page.getByTestId("detail-back")
  }

  get detailOpenConfluenceButton() {
    return this.page.getByTestId("detail-open-confluence")
  }

  get instantApplyButton() {
    return this.page
      .getByTestId("detail-apply")
      .or(this.page.getByRole("button", { name: "Instant apply" }))
  }

  get copyDescriptionButton() {
    return this.page.getByTestId("detail-copy")
  }

  get generateAiButton() {
    return this.page.getByTestId("detail-generate")
  }

  get globalToast() {
    return this.page.getByTestId("global-toast")
  }

  get noPatBanner() {
    return this.page.getByTestId("banner-no-confluence-pat")
  }

  get errorBanner() {
    return this.page.getByTestId("banner-confluence-error")
  }

  get vpnBanner() {
    return this.page.getByTestId("banner-confluence-vpn")
  }

  get emptyState() {
    return this.page.getByTestId("doc-empty")
  }

  get commentInput() {
    return this.page.getByTestId("detail-comment-input")
  }

  get commentPostButton() {
    return this.page.getByTestId("detail-comment-post")
  }

  get commentItems() {
    return this.page.locator('[data-testid="comment-item"]')
  }

  async navigateTo() {
    await this.page.goto("/confluence")
  }

  async search(query: string) {
    await this.searchInput.fill(query)
  }

  async clearSearch() {
    await this.searchInput.clear()
  }

  async clickCard(index = 0) {
    await this.documentCards.nth(index).click()
  }

  async clickCardByTitle(title: string) {
    await this.documentCards.filter({ hasText: title }).click()
  }
}
