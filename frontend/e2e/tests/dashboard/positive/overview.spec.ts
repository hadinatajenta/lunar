import { test, expect } from "../../../fixtures/authenticated.fixture"
import { captureEvidence } from "../../../helpers/evidence"

type DashboardSummary = {
  synced_at: string
  jira: {
    status: string
    is_configured: boolean
    assigned_tickets: number
    technical_documents: number
    active_sprint: string | null
  }
  bitbucket: {
    status: string
    is_configured: boolean
    open_pull_requests: number
    review_requested: number
  }
  copilot: {
    status: string
    active_tools: number
    total_tools: number
    configured_providers: string[]
  }
}

function expectedMetricValues(summary: DashboardSummary) {
  return {
    jiraTickets: String(summary.jira.assigned_tickets),
    bitbucketPullRequests: String(summary.bitbucket.open_pull_requests),
    documents: String(summary.jira.technical_documents),
    copilotTools: `${summary.copilot.active_tools}/${summary.copilot.total_tools}`
  }
}

function formatSyncedTime(isoString: string): string {
  const date = new Date(isoString)
  return date.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" })
}

const quickActionTargets = ["/copilot", "/jira", "/bitbucket", "/settings"]

test.describe("Dashboard - Positive Flows & API Contract", () => {
  test(
    "DASH-API-001 API-P01 summary response matches the contract shape with the session token",
    {
      tag: ["@dashboard", "@positive", "@contract"]
    },
    async ({ page, request }) => {
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
    }
  )

  test(
    "DASH-VIEW-001 E2E-P01 four metric cards exactly match the live summary response",
    {
      tag: ["@dashboard", "@positive", "@p0"]
    },
    async ({ page, dashboardPage }) => {
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
      await captureEvidence(page, "dashboard", "01_dashboard_metrics.png", true)
    }
  )

  test(
    "DASH-SYNC-001 E2E-P02 sync click enters syncing state, restores, and updates last-synced",
    {
      tag: ["@dashboard", "@positive", "@sync"]
    },
    async ({ page, dashboardPage }) => {
      await dashboardPage.navigateTo()
      await expect(dashboardPage.lastSyncedLabel).toBeVisible()
      await page.route(/\/api\/dashboard\/summary/, async (route) => {
        await new Promise((resolve) => setTimeout(resolve, 800))
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
      await captureEvidence(page, "dashboard", "02_dashboard_synced.png", true)
    }
  )

  test(
    "DASH-NAV-001 E2E-P03 four quick action tiles navigate to their target routes",
    {
      tag: ["@dashboard", "@positive", "@navigation"]
    },
    async ({ page, dashboardPage }) => {
      await dashboardPage.navigateTo()
      await expect(dashboardPage.quickActionTiles).toHaveCount(4)
      await captureEvidence(page, "dashboard", "03_dashboard_quick_actions.png", true)
      for (const targetPath of quickActionTargets) {
        await page.locator(`.action-tile[href="${targetPath}"]`).click()
        await expect(page).toHaveURL(new RegExp(`${targetPath}$`))
        await dashboardPage.navigateTo()
      }
    }
  )

  test(
    "DASH-CRED-001 E2E-P04 integration credential badges show configured or needs PAT state",
    {
      tag: ["@dashboard", "@positive", "@credentials"]
    },
    async ({ page, dashboardPage }) => {
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
      await captureEvidence(page, "dashboard", "08_credential_badges.png", true)
    }
  )
})
