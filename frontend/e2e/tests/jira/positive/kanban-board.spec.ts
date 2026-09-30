import { test, expect } from "../../../fixtures/authenticated.fixture"
import { captureEvidence } from "../../../helpers/evidence"
import { routeSecrets, routeStandardJira } from "../../../helpers/route-mock"
import { mockAssignedIssues, mockBacklogResponse } from "../../../data/jira.data"

test.describe("Jira BRI - Kanban Board Positive Flows", () => {
  test(
    "JIRA-BOARD-001 user can view assigned issues Kanban board with default 5 items per column",
    {
      tag: ["@jira", "@positive", "@board", "@p0"]
    },
    async ({ page, jiraPage }) => {
      await routeSecrets(page, { hasJiraPat: true })
      await routeStandardJira(page)
      await jiraPage.goto()

      await expect(jiraPage.kanbanCol("open")).toBeVisible()
      await expect(jiraPage.colCount("open")).toHaveText("7")
      await expect(jiraPage.kanbanCards("open")).toHaveCount(5)
      await expect(jiraPage.colCount("progress")).toHaveText("2")
      await expect(jiraPage.kanbanCards("progress")).toHaveCount(2)
      await expect(jiraPage.colCount("done")).toHaveText("1")
      await expect(jiraPage.kanbanCards("done")).toHaveCount(1)
      await captureEvidence(page, "jira", "01_jira_board_overview.png", true)
    }
  )

  test(
    "JIRA-BOARD-002 user can click load more button to expand column when more than 5 cards exist",
    {
      tag: ["@jira", "@positive", "@pagination"]
    },
    async ({ page, jiraPage }) => {
      await routeSecrets(page, { hasJiraPat: true })
      await routeStandardJira(page)
      await jiraPage.goto()

      await expect(jiraPage.kanbanCards("open")).toHaveCount(5)
      await expect(jiraPage.loadMoreButton("open")).toBeVisible()
      await expect(jiraPage.loadMoreButton("open")).toContainText("Load more (2 more)")

      await jiraPage.loadMoreButton("open").click()
      await expect(jiraPage.kanbanCards("open")).toHaveCount(7)
      await expect(jiraPage.loadMoreButton("open")).not.toBeVisible()
    }
  )

  test(
    "JIRA-SKELETON-001 user sees skeleton loader instead of dummy data while Jira data is loading",
    {
      tag: ["@jira", "@positive", "@loading"]
    },
    async ({ page, jiraPage }) => {
      await routeSecrets(page, { hasJiraPat: true })
      let fulfillIssues: () => void = () => {}
      const issuesPromise = new Promise<void>((resolve) => {
        fulfillIssues = resolve
      })

      await page.route(/\/api\/jira\/issues/, async (route) => {
        await issuesPromise
        await route.fulfill({ status: 200, json: mockAssignedIssues })
      })
      await page.route(/\/api\/jira\/backlog/, async (route) => {
        await issuesPromise
        await route.fulfill({ status: 200, json: mockBacklogResponse })
      })

      await jiraPage.goto()

      await expect(jiraPage.kanbanSkeleton).toBeVisible()
      await expect(jiraPage.kanbanCards()).toHaveCount(0)

      fulfillIssues()

      await expect(jiraPage.kanbanSkeleton).not.toBeVisible()
      await expect(jiraPage.kanbanCol("open")).toBeVisible()
    }
  )

  test(
    "JIRA-CACHE-001 returning to jira keeps the board without skeleton",
    {
      tag: ["@jira", "@positive", "@cache"]
    },
    async ({ page, jiraPage }) => {
      await routeSecrets(page, { hasJiraPat: true })
      await routeStandardJira(page)
      await jiraPage.goto()

      await expect(jiraPage.kanbanCards("open")).toHaveCount(5)
      await expect(jiraPage.kanbanCards("progress")).toHaveCount(2)
      await expect(jiraPage.kanbanCards("done")).toHaveCount(1)

      const mainNavigation = page.getByRole("navigation", {
        name: "Main Navigation"
      })
      await mainNavigation.getByRole("link", { name: "Settings" }).click()
      await expect(page).toHaveURL(/\/settings/)
      await mainNavigation.getByRole("link", { name: "Jira BRI" }).click()
      await expect(page).toHaveURL(/\/jira/)

      await expect(jiraPage.kanbanCol("open")).toBeVisible()
      await expect(jiraPage.kanbanSkeleton).not.toBeVisible()
      await expect(jiraPage.colCount("open")).toHaveText("7")
      await captureEvidence(page, "jira", "07_jira_cached_revisit.png", true)
    }
  )
})
