import { test, expect } from "../../../fixtures/authenticated.fixture"
import { captureEvidence } from "../../../helpers/evidence"
import { routeSecrets } from "../../../helpers/route-mock"
import { mockAssignedIssuesWithoutSop } from "../../../data/jira.data"

test.describe("Jira BRI - Negative & Edge Cases", () => {
  test(
    "JIRA-AUTH-001 user cannot see board when Jira PAT is unconfigured",
    {
      tag: ["@jira", "@negative", "@authorization", "@p0"]
    },
    async ({ page, jiraPage }) => {
      await routeSecrets(page, { hasJiraPat: false })
      await jiraPage.goto()
      await expect(jiraPage.noPatBanner).toBeVisible()
      await expect(jiraPage.noPatBanner).toContainText(
        "Jira Personal Access Token (PAT) is not configured yet"
      )
      await expect(jiraPage.kanbanCol("open")).not.toBeVisible()
      const settingsLink = jiraPage.noPatBanner.getByRole("link", {
        name: /Configure in Settings/
      })
      await expect(settingsLink).toBeVisible()
      await expect(settingsLink).toHaveAttribute("href", "/settings")
      await captureEvidence(page, "jira", "04_jira_negative_unconfigured.png")
    }
  )

  test(
    "JIRA-AUTH-002 user can see 401 error banner when Jira PAT is invalid or expired",
    {
      tag: ["@jira", "@negative", "@authorization"]
    },
    async ({ page, jiraPage }) => {
      await routeSecrets(page, { hasJiraPat: true })
      await page.route(/\/api\/jira\//, async (route) => {
        await route.fulfill({
          status: 401,
          json: { error: "unauthorized: invalid or expired token" }
        })
      })
      await jiraPage.goto()
      await expect(jiraPage.errorBanner).toBeVisible()
      await expect(jiraPage.errorBanner).toContainText(
        "Unauthorized access: invalid or expired Jira PAT"
      )
      const updateLink = jiraPage.errorBanner.getByRole("link", {
        name: /Update in Settings/
      })
      await expect(updateLink).toBeVisible()
      await expect(updateLink).toHaveAttribute("href", "/settings")
      await captureEvidence(page, "jira", "05_jira_negative_401.png")
    }
  )

  test(
    "JIRA-NET-001 user can see 502 VPN error banner when upstream is unreachable",
    {
      tag: ["@jira", "@negative", "@network"]
    },
    async ({ page, jiraPage }) => {
      await routeSecrets(page, { hasJiraPat: true })
      await page.route(/\/api\/jira\//, async (route) => {
        await route.fulfill({
          status: 502,
          json: { error: "bad gateway: upstream unreachable" }
        })
      })
      await jiraPage.goto()
      await expect(jiraPage.vpnBanner).toBeVisible()
      await expect(jiraPage.vpnBanner).toContainText(
        "Unable to reach Jira BRI API. Please ensure you are connected to the BRI VPN."
      )
      await expect(jiraPage.retryButton).toBeVisible()
      await captureEvidence(page, "jira", "06_jira_negative_502.png")
    }
  )

  test(
    "JIRA-EMPTY-001 user can see empty column state when no issues exist for a status",
    {
      tag: ["@jira", "@edge", "@empty-state"]
    },
    async ({ page, jiraPage }) => {
      await routeSecrets(page, { hasJiraPat: true })
      await page.route(/\/api\/jira\/issues/, async (route) => {
        await route.fulfill({ status: 200, json: [] })
      })
      await page.route(/\/api\/jira\/backlog/, async (route) => {
        await route.fulfill({ status: 200, json: { sprints: [] } })
      })
      await jiraPage.goto()
      await expect(jiraPage.emptyColumnState("open")).toBeVisible()
      await expect(jiraPage.emptyColumnState("open")).toHaveText("Nothing here")
      await expect(jiraPage.emptyColumnState("progress")).toBeVisible()
      await expect(jiraPage.emptyColumnState("progress")).toHaveText("Nothing here")
      await expect(jiraPage.emptyColumnState("done")).toBeVisible()
      await expect(jiraPage.emptyColumnState("done")).toHaveText("Nothing here")
    }
  )

  test(
    "JIRA-EMPTY-002 user can see empty state when subfilter yields zero matches",
    {
      tag: ["@jira", "@edge", "@empty-state"]
    },
    async ({ page, jiraPage }) => {
      await routeSecrets(page, { hasJiraPat: true })
      await page.route(/\/api\/jira\/issues/, async (route) => {
        await route.fulfill({ status: 200, json: mockAssignedIssuesWithoutSop })
      })
      await page.route(/\/api\/jira\/backlog/, async (route) => {
        await route.fulfill({ status: 200, json: { sprints: [] } })
      })
      await jiraPage.goto()
      await jiraPage.filterDocs.click()
      await jiraPage.docFilterSop.click()
      await expect(jiraPage.emptyColumnState("open")).toBeVisible()
      await expect(jiraPage.emptyColumnState("open")).toHaveText("Nothing here")
      await expect(jiraPage.emptyColumnState("progress")).toBeVisible()
      await expect(jiraPage.emptyColumnState("progress")).toHaveText("Nothing here")
      await expect(jiraPage.emptyColumnState("done")).toBeVisible()
      await expect(jiraPage.emptyColumnState("done")).toHaveText("Nothing here")
    }
  )
})
