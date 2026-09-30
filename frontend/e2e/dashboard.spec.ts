import { test, expect, type Page } from "@playwright/test"
import fs from "fs"
import path from "path"
import { fileURLToPath } from "url"
import { DashboardPage } from "./pages/DashboardPage"
import type { DashboardSummary } from "../src/features/dashboard/types"

const currentDir = path.dirname(fileURLToPath(import.meta.url))
const EVIDENCE_DIR = path.join(currentDir, "../../docs/dashboard/evidence")
const SUMMARY_URL_PATTERN = "**/api/dashboard/summary"
const PLACEHOLDER_VALUE = "—"

fs.mkdirSync(EVIDENCE_DIR, { recursive: true })

const quickActionTargets = ["/copilot", "/jira", "/bitbucket", "/settings"]

function expectedMetricValues(summary: DashboardSummary) {
  const isJiraUnavailable = summary.jira.status === "error"
  const isBitbucketUnavailable = summary.bitbucket.status === "error"
  return {
    jiraTickets: isJiraUnavailable ? PLACEHOLDER_VALUE : String(summary.jira.assigned_tickets),
    bitbucketPullRequests: isBitbucketUnavailable
      ? PLACEHOLDER_VALUE
      : String(summary.bitbucket.open_pull_requests),
    documents: isJiraUnavailable ? PLACEHOLDER_VALUE : String(summary.jira.technical_documents),
    copilotTools:
      summary.copilot.status === "error"
        ? PLACEHOLDER_VALUE
        : `${summary.copilot.active_tools}/${summary.copilot.total_tools}`
  }
}

function formatSyncedTime(syncedAt: string): string {
  return new Date(syncedAt).toLocaleTimeString("en-GB", {
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit"
  })
}

async function fetchDashboardSummary(page: Page): Promise<DashboardSummary> {
  return page.evaluate(async () => {
    const token = localStorage.getItem("lunar_auth_token")
    const response = await fetch("/api/dashboard/summary", {
      headers: { Authorization: `Bearer ${token}` }
    })
    return response.json()
  })
}

test.describe("Dashboard unauthenticated access", () => {
  test.use({ storageState: { cookies: [], origins: [] } })

  test("E2E-N01 unauthenticated visitor is redirected to login and never sees metric cards", async ({ page }) => {
    const dashboardPage = new DashboardPage(page)
    await page.goto("/dashboard")
    await expect(page).toHaveURL(/\/login/)
    await expect(dashboardPage.metricCards).toHaveCount(0)
    await expect(dashboardPage.jiraTicketsMetric).toHaveCount(0)
    await expect(dashboardPage.bitbucketPullRequestsMetric).toHaveCount(0)
    await expect(dashboardPage.documentsMetric).toHaveCount(0)
    await expect(dashboardPage.copilotToolsMetric).toHaveCount(0)
    await page.screenshot({ path: path.join(EVIDENCE_DIR, "04_unauthenticated_redirect.png"), fullPage: true })
  })
})

test.describe("Dashboard summary failure handling", () => {
  test("E2E-N02 summary endpoint returning 500 shows error banner with placeholders and retry recovers", async ({
    page
  }) => {
    const dashboardPage = new DashboardPage(page)
    let shouldFailSummaryRequest = true
    await page.route(SUMMARY_URL_PATTERN, (route) => {
      if (shouldFailSummaryRequest) {
        return route.fulfill({ status: 500, json: { message: "Internal server error" } })
      }
      return route.continue()
    })
    await dashboardPage.navigateTo()
    await expect(dashboardPage.errorBanner).toBeVisible()
    await expect(dashboardPage.jiraTicketsMetric).toHaveText(PLACEHOLDER_VALUE)
    await expect(dashboardPage.bitbucketPullRequestsMetric).toHaveText(PLACEHOLDER_VALUE)
    await expect(dashboardPage.documentsMetric).toHaveText(PLACEHOLDER_VALUE)
    await expect(dashboardPage.copilotToolsMetric).toHaveText(PLACEHOLDER_VALUE)
    await page.screenshot({ path: path.join(EVIDENCE_DIR, "05_summary_error_banner.png"), fullPage: true })

    shouldFailSummaryRequest = false
    await dashboardPage.retryButton.click()
    await expect(dashboardPage.errorBanner).toBeHidden()
    const summary = await fetchDashboardSummary(page)
    const expectedValues = expectedMetricValues(summary)
    await expect(dashboardPage.jiraTicketsMetric).toHaveText(expectedValues.jiraTickets)
    await expect(dashboardPage.bitbucketPullRequestsMetric).toHaveText(expectedValues.bitbucketPullRequests)
    await expect(dashboardPage.documentsMetric).toHaveText(expectedValues.documents)
    await expect(dashboardPage.copilotToolsMetric).toHaveText(expectedValues.copilotTools)
  })

  test("E2E-N03 aborted summary request surfaces the error banner and placeholders", async ({ page }) => {
    const dashboardPage = new DashboardPage(page)
    await page.route(SUMMARY_URL_PATTERN, (route) => route.abort("failed"))
    await dashboardPage.navigateTo()
    await expect(dashboardPage.errorBanner).toBeVisible()
    await expect(dashboardPage.retryButton).toBeVisible()
    await expect(dashboardPage.jiraTicketsMetric).toHaveText(PLACEHOLDER_VALUE)
    await expect(dashboardPage.bitbucketPullRequestsMetric).toHaveText(PLACEHOLDER_VALUE)
    await expect(dashboardPage.documentsMetric).toHaveText(PLACEHOLDER_VALUE)
    await expect(dashboardPage.copilotToolsMetric).toHaveText(PLACEHOLDER_VALUE)
    await page.screenshot({ path: path.join(EVIDENCE_DIR, "06_summary_network_error.png"), fullPage: true })
  })

  test("E2E-N04 failed sync restores the button and surfaces a failure toast", async ({ page }) => {
    const dashboardPage = new DashboardPage(page)
    await dashboardPage.navigateTo()
    await expect(dashboardPage.errorBanner).toBeHidden()
    await expect(dashboardPage.jiraTicketsMetric).not.toHaveText(PLACEHOLDER_VALUE)
    await page.route(SUMMARY_URL_PATTERN, (route) =>
      route.fulfill({ status: 500, json: { message: "Internal server error" } })
    )
    const syncResponsePromise = page.waitForResponse((response) =>
      response.url().includes("/api/dashboard/summary")
    )
    await dashboardPage.syncWorkspaceButton.click()
    const syncResponse = await syncResponsePromise
    expect(syncResponse.status()).toBe(500)
    await expect(dashboardPage.syncWorkspaceButton).toBeEnabled()
    await expect(dashboardPage.syncWorkspaceButton).toContainText("Sync workspace")
    await expect(dashboardPage.syncWorkspaceButton).not.toContainText("Syncing...")
    await expect(dashboardPage.errorBanner).toBeVisible()
    await expect(page.getByTestId("global-toast")).toBeVisible()
    await page.screenshot({ path: path.join(EVIDENCE_DIR, "07_sync_failure_toast.png"), fullPage: true })
  })
})

test.describe("Dashboard summary API contract negatives", () => {
  test("API-N01 summary request without Authorization header returns 401", async ({ request }) => {
    const response = await request.get("/api/dashboard/summary")
    expect(response.status()).toBe(401)
    const errorBody = await response.json()
    expect(typeof errorBody.error).toBe("string")
  })

  test("API-N02 summary request with invalid bearer token returns 401", async ({ request }) => {
    const response = await request.get("/api/dashboard/summary", {
      headers: { Authorization: "Bearer invalid-token-value" }
    })
    expect(response.status()).toBe(401)
    const errorBody = await response.json()
    expect(typeof errorBody.error).toBe("string")
  })
})

test.describe("Dashboard summary API contract", () => {
  test("API-P01 summary response matches the contract shape with the session token", async ({ page, request }) => {
    await page.goto("/dashboard")
    await expect(page).toHaveURL(/\/dashboard/)
    const sessionToken = await page.evaluate(() => localStorage.getItem("lunar_auth_token"))
    expect(Boolean(sessionToken)).toBe(true)
    const response = await request.get("/api/dashboard/summary", {
      headers: { Authorization: `Bearer ${sessionToken}` }
    })
    expect(response.status()).toBe(200)
    const summary: DashboardSummary = await response.json()
    expect(summary.synced_at).toMatch(/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(\.\d+)?(Z|[+-]\d{2}:\d{2})$/)
    expect(["ok", "unconfigured", "error"]).toContain(summary.jira.status)
    expect(["ok", "unconfigured", "error"]).toContain(summary.bitbucket.status)
    expect(["ok", "error"]).toContain(summary.copilot.status)
    expect(typeof summary.jira.is_configured).toBe("boolean")
    expect(typeof summary.jira.assigned_tickets).toBe("number")
    expect(typeof summary.jira.technical_documents).toBe("number")
    expect(summary.jira.active_sprint === null || typeof summary.jira.active_sprint === "string").toBe(true)
    expect(typeof summary.bitbucket.is_configured).toBe("boolean")
    expect(typeof summary.bitbucket.open_pull_requests).toBe("number")
    expect(typeof summary.bitbucket.review_requested).toBe("number")
    expect(typeof summary.copilot.active_tools).toBe("number")
    expect(typeof summary.copilot.total_tools).toBe("number")
    expect(Array.isArray(summary.copilot.configured_providers)).toBe(true)
    for (const providerName of summary.copilot.configured_providers) {
      expect(typeof providerName).toBe("string")
    }
  })
})

test.describe("Dashboard Overview", () => {
  test("E2E-P01 four metric cards exactly match the live summary response", async ({ page }) => {
    const dashboardPage = new DashboardPage(page)
    const summaryResponsePromise = page.waitForResponse((response) =>
      response.url().includes("/api/dashboard/summary")
    )
    await dashboardPage.navigateTo()
    const summaryResponse = await summaryResponsePromise
    expect(summaryResponse.status()).toBe(200)
    const authorizationHeader = summaryResponse.request().headers()["authorization"]
    expect(Boolean(authorizationHeader && authorizationHeader.startsWith("Bearer "))).toBe(true)
    const summary: DashboardSummary = await summaryResponse.json()
    const expectedValues = expectedMetricValues(summary)
    await expect(dashboardPage.metricCards).toHaveCount(4)
    await expect(dashboardPage.jiraTicketsMetric).toHaveText(expectedValues.jiraTickets)
    await expect(dashboardPage.bitbucketPullRequestsMetric).toHaveText(expectedValues.bitbucketPullRequests)
    await expect(dashboardPage.documentsMetric).toHaveText(expectedValues.documents)
    await expect(dashboardPage.copilotToolsMetric).toHaveText(expectedValues.copilotTools)
    if (summary.jira.status === "unconfigured") {
      const jiraMetricCard = dashboardPage.metricCards.filter({ has: dashboardPage.jiraTicketsMetric })
      await expect(jiraMetricCard.locator(".metric-sub")).toContainText("PAT not configured")
    }
    await page.screenshot({ path: path.join(EVIDENCE_DIR, "01_dashboard_metrics.png"), fullPage: true })
  })

  test("E2E-P02 sync click enters syncing state, restores, and updates last-synced", async ({ page }) => {
    const dashboardPage = new DashboardPage(page)
    await dashboardPage.navigateTo()
    await expect(dashboardPage.lastSyncedLabel).toBeVisible()
    await page.route(SUMMARY_URL_PATTERN, async (route) => {
      await new Promise((resolve) => setTimeout(resolve, 1000))
      await route.continue()
    })
    const syncResponsePromise = page.waitForResponse((response) =>
      response.url().includes("/api/dashboard/summary")
    )
    await dashboardPage.syncWorkspaceButton.click()
    await expect(dashboardPage.syncWorkspaceButton).toBeDisabled()
    await expect(dashboardPage.syncWorkspaceButton).toContainText("Syncing...")
    const syncResponse = await syncResponsePromise
    expect(syncResponse.status()).toBe(200)
    await expect(dashboardPage.syncWorkspaceButton).toBeEnabled()
    await expect(dashboardPage.syncWorkspaceButton).toContainText("Sync workspace")
    const syncedSummary: DashboardSummary = await syncResponse.json()
    await expect(dashboardPage.lastSyncedLabel).toHaveText(`Synced ${formatSyncedTime(syncedSummary.synced_at)}`)
    await page.screenshot({ path: path.join(EVIDENCE_DIR, "02_dashboard_synced.png"), fullPage: true })
  })

  test("E2E-P03 four quick action tiles navigate to their target routes", async ({ page }) => {
    const dashboardPage = new DashboardPage(page)
    await dashboardPage.navigateTo()
    await expect(dashboardPage.quickActionTiles).toHaveCount(4)
    await page.screenshot({ path: path.join(EVIDENCE_DIR, "03_dashboard_quick_actions.png"), fullPage: true })
    for (const targetPath of quickActionTargets) {
      await page.locator(`.action-tile[href="${targetPath}"]`).click()
      await expect(page).toHaveURL(new RegExp(`${targetPath}$`))
      await dashboardPage.navigateTo()
    }
  })

  test("E2E-P04 integration credential badges show configured or needs PAT state", async ({ page }) => {
    const dashboardPage = new DashboardPage(page)
    await dashboardPage.navigateTo()
    await expect(dashboardPage.credentialPanel.locator(".tool-name")).toHaveText([
      "Jira BRI",
      "Bitbucket BRI",
      "Confluence BRI"
    ])
    const credentialBadges = dashboardPage.credentialPanel.locator(".status-badge")
    await expect(credentialBadges).toHaveCount(3)
    for (const credentialBadge of await credentialBadges.all()) {
      await expect(credentialBadge).toHaveText(/^(Configured|Needs PAT)$/)
    }
    await page.screenshot({ path: path.join(EVIDENCE_DIR, "08_credential_badges.png"), fullPage: true })
  })
})
