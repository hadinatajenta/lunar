import { test, expect } from "@playwright/test"
import path from "path"
import { DashboardPage } from "./pages/DashboardPage"

const EVIDENCE_DIR = "/Users/erendt/code/lunar/docs/dashboard/evidence"

test.describe("Dashboard Overview", () => {
  test("engineer can view four engineering metric cards on the dashboard", async ({ page }) => {
    const dashboardPage = new DashboardPage(page)
    await dashboardPage.navigateTo()
    await expect(dashboardPage.metricCards).toHaveCount(4)
    await page.screenshot({ path: path.join(EVIDENCE_DIR, "01_dashboard_metrics.png"), fullPage: true })
  })

  test("engineer can trigger workspace sync and see syncing state then completion", async ({ page }) => {
    const dashboardPage = new DashboardPage(page)
    await dashboardPage.navigateTo()
    await dashboardPage.syncWorkspaceButton.click()
    await expect(dashboardPage.syncWorkspaceButton).toContainText("Syncing...")
    await expect(dashboardPage.syncWorkspaceButton).toContainText("Sync workspace", { timeout: 3000 })
    await page.screenshot({ path: path.join(EVIDENCE_DIR, "02_dashboard_synced.png") })
  })

  test("engineer can see four quick action tiles on the dashboard", async ({ page }) => {
    const dashboardPage = new DashboardPage(page)
    await dashboardPage.navigateTo()
    await expect(dashboardPage.quickActionTiles).toHaveCount(4)
    await page.screenshot({ path: path.join(EVIDENCE_DIR, "03_dashboard_quick_actions.png") })
  })
})
