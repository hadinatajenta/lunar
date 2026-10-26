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

  get noAiKeysBanner() {
    return this.page.getByTestId("banner-no-ai-keys")
  }

  get queryPanelHeading() {
    return this.page.getByTestId("ai-query-heading")
  }

  get queryIdleState() {
    return this.page.getByTestId("ai-query-idle")
  }

  get queryLoading() {
    return this.page.getByTestId("ai-query-loading")
  }

  get queryError() {
    return this.page.getByTestId("ai-query-error")
  }

  get queryManualUrlInput() {
    return this.page.getByTestId("ai-query-manual-url")
  }

  get queryRetryButton() {
    return this.page.getByTestId("ai-query-retry")
  }

  get queryResult() {
    return this.page.getByTestId("ai-query-result")
  }

  get queryMigPage() {
    return this.page.getByTestId("ai-query-mig-page")
  }

  get querySkippedRepositories() {
    return this.page.getByTestId("ai-query-skipped-repositories")
  }

  get queryRepositories() {
    return this.page.locator('[data-testid="ai-query-repository"]')
  }

  get queryRepositoryNames() {
    return this.page.locator('[data-testid="ai-query-repository-name"]')
  }

  get queryRepositoryStatuses() {
    return this.page.locator('[data-testid="ai-query-repository-status"]')
  }

  get queryRepositoryBranches() {
    return this.page.locator('[data-testid="ai-query-repository-branches"]')
  }

  get queryRepositoryError() {
    return this.page.getByTestId("ai-query-repository-error")
  }

  get queryNoQueriesNote() {
    return this.page.getByTestId("ai-query-repository-no-queries")
  }

  get queryCompareLinks() {
    return this.page.locator('[data-testid="ai-query-repository-compare"]')
  }

  get querySummary() {
    return this.page.getByTestId("ai-query-summary").first()
  }

  get queryTable() {
    return this.page.getByTestId("ai-query-table").first()
  }

  get queryTables() {
    return this.page.getByTestId("ai-query-table")
  }

  get queryTableHeaders() {
    return this.queryTable.locator("th")
  }

  get queryRows() {
    return this.page.locator('[data-testid="ai-query-row"]')
  }

  get queryIndexCells() {
    return this.page.locator('[data-testid="ai-query-index"]')
  }

  get queryCopyStatementButtons() {
    return this.page.locator('[data-testid="ai-query-copy-statement"]')
  }

  get queryCopyTableButton() {
    return this.page.getByTestId("ai-query-copy-table")
  }

  get queryRepositoryCopyButtons() {
    return this.page.locator('[data-testid="ai-query-copy-repository"]')
  }

  get queryCopyJsonButton() {
    return this.page.getByTestId("ai-query-copy-json")
  }

  get queryApplyError() {
    return this.page.getByTestId("ai-query-apply-error")
  }

  get queryApplyRetryButton() {
    return this.page.getByTestId("ai-query-apply-retry")
  }

  get queryEmptyState() {
    return this.page.getByTestId("ai-query-empty")
  }

  get utMigPage() {
    return this.page.getByTestId("ai-ut-mig-page")
  }

  get utTargets() {
    return this.page.locator('[data-testid="ai-ut-target"]')
  }

  get utSkippedRepositories() {
    return this.page.getByTestId("ai-ut-skipped-repositories")
  }

  get utCurrentRepository() {
    return this.page.getByTestId("ai-ut-current-repository")
  }

  get utPhases() {
    return this.page.getByTestId("ai-ut-phases")
  }

  get utOutput() {
    return this.page.getByTestId("ai-ut-output")
  }

  get utSummaries() {
    return this.page.locator('[data-testid="ai-ut-summary"]')
  }

  get utStatus() {
    return this.page.getByTestId("ai-ut-status").first()
  }

  get utCounts() {
    return this.page.getByTestId("ai-ut-counts").first()
  }

  get utWorktree() {
    return this.page.getByTestId("ai-ut-worktree").first()
  }

  get utBlocked() {
    return this.page.getByTestId("ai-ut-blocked").first()
  }

  get utScreenshot() {
    return this.page.locator('[data-testid="ai-ut-screenshot"]').first()
  }

  get utScreenshotStrip() {
    return this.page.getByTestId("ai-ut-screenshot-strip")
  }

  get utMockTests() {
    return this.page.getByTestId("ai-ut-mock-tests")
  }

  get utDocument() {
    return this.page.getByTestId("ai-ut-document")
  }

  get utDocumentTitle() {
    return this.page.getByTestId("ai-ut-document-title")
  }

  get utTable() {
    return this.page.getByTestId("ai-ut-table").first()
  }

  get utTableHeaders() {
    return this.utTable.locator("th")
  }

  get utGroups() {
    return this.page.locator('[data-testid="ai-ut-group"]')
  }

  get utRows() {
    return this.page.locator('[data-testid="ai-ut-row"]')
  }

  get utRowFunctions() {
    return this.page.locator('[data-testid="ai-ut-row-function"]')
  }

  get utRowResults() {
    return this.page.locator('[data-testid="ai-ut-row-result"]')
  }

  get utCopyTableButton() {
    return this.page.getByTestId("ai-ut-copy-table")
  }

  get utApplyError() {
    return this.page.getByTestId("ai-ut-apply-error")
  }

  get utApplyRetryButton() {
    return this.page.getByTestId("ai-ut-apply-retry")
  }

  get utEmptyState() {
    return this.page.getByTestId("ai-ut-empty")
  }

  get utBlockedNotice() {
    return this.page.getByTestId("ai-ut-blocked-notice")
  }

  get utError() {
    return this.page.getByTestId("ai-ut-error")
  }

  get utRetryButton() {
    return this.page.getByTestId("ai-ut-retry")
  }

  get utCancelButton() {
    return this.page.getByTestId("ai-ut-cancel")
  }

  get sopGenerateButton() {
    return this.page.getByTestId("detail-generate-sop")
  }

  get sopPanelHeading() {
    return this.page.getByTestId("ai-sop-heading")
  }

  get sopRoot() {
    return this.page.getByTestId("ai-sop-root")
  }

  get sopIdleState() {
    return this.page.getByTestId("ai-sop-idle")
  }

  get sopLoading() {
    return this.page.getByTestId("ai-sop-loading")
  }

  get sopError() {
    return this.page.getByTestId("ai-sop-error")
  }

  get sopRetryButton() {
    return this.page.getByTestId("ai-sop-retry")
  }

  get sopResult() {
    return this.page.getByTestId("ai-sop-result")
  }

  get sopIssueKey() {
    return this.page.getByTestId("ai-sop-issue-key")
  }

  get sopWarnings() {
    return this.page.getByTestId("ai-sop-warnings")
  }

  get sopDeployTitle() {
    return this.page.getByTestId("ai-sop-deploy-title")
  }

  get sopDeployExists() {
    return this.page.getByTestId("ai-sop-deploy-exists")
  }

  get sopDeployPreview() {
    return this.page.getByTestId("ai-sop-deploy-preview")
  }

  get sopCopyDeployButton() {
    return this.page.getByTestId("ai-sop-copy-deploy")
  }

  get sopOperasionalTitle() {
    return this.page.getByTestId("ai-sop-operasional-title")
  }

  get sopChildren() {
    return this.page.locator('[data-testid="ai-sop-child"]')
  }

  get sopChildStates() {
    return this.page.locator('[data-testid="ai-sop-child-exists"]')
  }

  get sopCopyOperasionalButton() {
    return this.page.getByTestId("ai-sop-copy-operasional")
  }

  get sopRepositories() {
    return this.page.locator('[data-testid="ai-sop-repository"]')
  }

  get sopRepositoryNames() {
    return this.page.locator('[data-testid="ai-sop-repository-name"]')
  }

  get sopApplyButton() {
    return this.page.getByTestId("ai-sop-apply")
  }

  get sopApplied() {
    return this.page.getByTestId("ai-sop-applied")
  }

  get sopApplyError() {
    return this.page.getByTestId("ai-sop-apply-error")
  }

  get sopApplyRetryButton() {
    return this.page.getByTestId("ai-sop-apply-retry")
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
