import { test, expect } from "../../../fixtures/unauthenticated.fixture"
import { captureEvidence } from "../../../helpers/evidence"

test.describe("Dashboard - Unauthenticated Access & API Contracts", () => {
  test(
    "DASH-AUTH-001 E2E-N01 unauthenticated visitor is redirected to login and never sees metric cards",
    {
      tag: ["@dashboard", "@negative", "@authorization", "@p0"]
    },
    async ({ page, authPage, dashboardPage }) => {
      await page.goto("/dashboard")
      await expect(page).toHaveURL(/\/login/)
      await expect(authPage.pageHeading).toBeVisible()
      await expect(dashboardPage.metricCards).toHaveCount(0)
      await captureEvidence(page, "dashboard", "04_unauthenticated_redirect.png", true)
    }
  )

  test(
    "DASH-AUTH-002 unauthenticated access to protected routes redirects to login",
    {
      tag: ["@dashboard", "@negative", "@authorization"]
    },
    async ({ page, authPage }) => {
      const protectedRoutes = ["/copilot", "/jira", "/bitbucket", "/confluence", "/settings"]
      for (const route of protectedRoutes) {
        await page.goto(route)
        await expect(page).toHaveURL(/\/login/)
        await expect(authPage.pageHeading).toBeVisible()
      }
    }
  )

  test(
    "DASH-API-002 API-N01 summary request without Authorization header returns 401",
    {
      tag: ["@dashboard", "@negative", "@api-contract"]
    },
    async ({ request }) => {
      const response = await request.get("/api/dashboard/summary")
      expect(response.status()).toBe(401)
    }
  )

  test(
    "DASH-API-003 API-N02 summary request with invalid bearer token returns 401",
    {
      tag: ["@dashboard", "@negative", "@api-contract"]
    },
    async ({ request }) => {
      const response = await request.get("/api/dashboard/summary", {
        headers: { Authorization: "Bearer invalid.token.payload" }
      })
      expect(response.status()).toBe(401)
    }
  )
})
