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
    return this.page.locator(".heading-action")
  }

  async navigateTo() {
    await this.page.goto("/dashboard")
    await expect(this.page).toHaveURL(/\/dashboard/)
  }

  async triggerSync() {
    await this.syncWorkspaceButton.click()
    await expect(this.syncWorkspaceButton).toContainText("Syncing...")
    await expect(this.syncWorkspaceButton).toContainText("Sync workspace", { timeout: 3000 })
  }
}
