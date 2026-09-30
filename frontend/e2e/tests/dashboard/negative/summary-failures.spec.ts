import { test, expect } from "../../../fixtures/authenticated.fixture"
import { captureEvidence } from "../../../helpers/evidence"

const PLACEHOLDER_VALUE = "—"
const SUMMARY_URL_PATTERN = /\/api\/dashboard\/summary/

test.describe("Dashboard - Summary Error Handling & Recovery", () => {
  test(
    "DASH-SUMMARY-001 E2E-N02 summary endpoint returning 500 shows error banner with placeholders and retry recovers",
    {
      tag: ["@dashboard", "@negative", "@recovery", "@p0"]
    },
    async ({ page, dashboardPage }) => {
      let shouldFail = true
      await page.route(SUMMARY_URL_PATTERN, (route) => {
        if (shouldFail) {
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
      await captureEvidence(page, "dashboard", "05_summary_error_banner.png", true)

      shouldFail = false
      await dashboardPage.retryButton.click()
      await expect(dashboardPage.errorBanner).toBeHidden()
      await expect(dashboardPage.jiraTicketsMetric).not.toHaveText(PLACEHOLDER_VALUE)
    }
  )

  test(
    "DASH-SUMMARY-002 E2E-N03 aborted summary request surfaces the error banner and placeholders",
    {
      tag: ["@dashboard", "@negative", "@network"]
    },
    async ({ page, dashboardPage }) => {
      await page.route(SUMMARY_URL_PATTERN, (route) => route.abort("failed"))
      await dashboardPage.navigateTo()
      await expect(dashboardPage.errorBanner).toBeVisible()
      await expect(dashboardPage.retryButton).toBeVisible()
      await expect(dashboardPage.jiraTicketsMetric).toHaveText(PLACEHOLDER_VALUE)
      await expect(dashboardPage.bitbucketPullRequestsMetric).toHaveText(PLACEHOLDER_VALUE)
      await expect(dashboardPage.documentsMetric).toHaveText(PLACEHOLDER_VALUE)
      await expect(dashboardPage.copilotToolsMetric).toHaveText(PLACEHOLDER_VALUE)
      await captureEvidence(page, "dashboard", "06_summary_network_error.png", true)
    }
  )

  test(
    "DASH-SYNC-002 E2E-N04 failed sync restores the button and surfaces a failure toast",
    {
      tag: ["@dashboard", "@negative", "@sync"]
    },
    async ({ page, dashboardPage }) => {
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
      await captureEvidence(page, "dashboard", "07_sync_failure_toast.png", true)
    }
  )
})
