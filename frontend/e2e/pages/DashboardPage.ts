import { type Page, expect } from "@playwright/test"

export class DashboardPage {
  readonly page: Page

  constructor(page: Page) {
    this.page = page
  }

  get pageHeading() {
    return this.page.getByRole("heading", { level: 1 })
  }

  get metricCards() {
    return this.page.locator(".metric-card")
  }

  get quickActionTiles() {
    return this.page.locator(".action-tile")
  }

  get syncWorkspaceButton() {
    return this.page.getByTestId("btn-sync-workspace")
  }

  get errorBanner() {
    return this.page.getByTestId("banner-dashboard-error")
  }

  get retryButton() {
    return this.page.getByTestId("btn-retry")
  }

  get lastSyncedLabel() {
    return this.page.getByTestId("last-synced")
  }

  get jiraTicketsMetric() {
    return this.page.getByTestId("metric-jira-tickets")
  }

  get bitbucketPullRequestsMetric() {
    return this.page.getByTestId("metric-bitbucket-prs")
  }

  get documentsMetric() {
    return this.page.getByTestId("metric-documents")
  }

  get copilotToolsMetric() {
    return this.page.getByTestId("metric-copilot-tools")
  }

  get credentialPanel() {
    return this.page.locator(".panel", { hasText: "Integration Credentials" })
  }

  async navigateTo() {
    await this.page.goto("/dashboard")
    await expect(this.page).toHaveURL(/\/dashboard/)
  }
}
