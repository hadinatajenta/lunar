import { test, expect } from "../../../fixtures/authenticated.fixture"
import { routeSecrets, routeStandardJira } from "../../../helpers/route-mock"
import { mockAssignedIssues } from "../../../data/jira.data"

test.describe("Jira BRI - Filters & Squad Views", () => {
  test(
    "JIRA-FILTER-001 user can filter by category",
    {
      tag: ["@jira", "@positive", "@filter"]
    },
    async ({ page, jiraPage }) => {
      await routeSecrets(page, { hasJiraPat: true })
      await routeStandardJira(page)
      await jiraPage.goto()

      await jiraPage.filterBugs.click()
      await expect(jiraPage.filterBugs).toHaveClass(/is-active/)
      await expect(jiraPage.kanbanCards("open")).toHaveCount(1)
      await expect(jiraPage.kanbanCards("progress")).toHaveCount(1)
      await expect(jiraPage.emptyColumnState("done")).toBeVisible()

      await jiraPage.filterSubtasks.click()
      await expect(jiraPage.filterSubtasks).toHaveClass(/is-active/)
      await expect(jiraPage.kanbanCards("open")).toHaveCount(1)
      await expect(jiraPage.emptyColumnState("progress")).toBeVisible()
      await expect(jiraPage.kanbanCards("done")).toHaveCount(1)

      await jiraPage.filterDocs.click()
      await expect(jiraPage.filterDocs).toHaveClass(/is-active/)
      await expect(jiraPage.kanbanCards("open")).toHaveCount(5)
    }
  )

  test(
    "JIRA-FILTER-002 user can filter dev documents by sub-type",
    {
      tag: ["@jira", "@positive", "@filter"]
    },
    async ({ page, jiraPage }) => {
      await routeSecrets(page, { hasJiraPat: true })
      await routeStandardJira(page)
      await jiraPage.goto()

      await expect(jiraPage.docFilterAll).toHaveClass(/is-active/)
      await jiraPage.docFilterUt.click()
      await expect(jiraPage.docFilterUt).toHaveClass(/is-active/)
      await expect(jiraPage.kanbanCards("open")).toHaveCount(3)
      await expect(jiraPage.kanbanCards("progress")).toHaveCount(1)
      await expect(jiraPage.emptyColumnState("done")).toBeVisible()

      await jiraPage.docFilterQuery.click()
      await expect(jiraPage.docFilterQuery).toHaveClass(/is-active/)
      await expect(jiraPage.kanbanCards("open")).toHaveCount(2)
      await expect(jiraPage.kanbanCards("progress")).toHaveCount(1)
      await expect(jiraPage.emptyColumnState("done")).toBeVisible()

      await jiraPage.docFilterSop.click()
      await expect(jiraPage.docFilterSop).toHaveClass(/is-active/)
      await expect(jiraPage.kanbanCards("open")).toHaveCount(2)
      await expect(jiraPage.emptyColumnState("progress")).toBeVisible()
      await expect(jiraPage.kanbanCards("done")).toHaveCount(1)

      await jiraPage.docFilterAll.click()
      await expect(jiraPage.docFilterAll).toHaveClass(/is-active/)
      await expect(jiraPage.kanbanCards("open")).toHaveCount(5)
    }
  )

  test(
    "JIRA-FILTER-003 user can switch to all filter and view all assigned issues",
    {
      tag: ["@jira", "@positive", "@filter"]
    },
    async ({ page, jiraPage }) => {
      await routeSecrets(page, { hasJiraPat: true })
      await routeStandardJira(page)
      await jiraPage.goto()

      await jiraPage.filterAll.click()
      await expect(jiraPage.filterAll).toHaveClass(/is-active/)
      await expect(jiraPage.kanbanCol("open")).toBeVisible()
      await expect(jiraPage.colCount("open")).toHaveText("9")
    }
  )

  test(
    "JIRA-SQUAD-001 user sees squad filter pills and can filter sprints by squad",
    {
      tag: ["@jira", "@positive", "@squad"]
    },
    async ({ page, jiraPage }) => {
      await routeSecrets(page, { hasJiraPat: true })
      const multiSquadBacklog = {
        sprints: [
          {
            id: "sprint-44",
            name: "Sprint 44 - Fortune Squad",
            is_active: true,
            dates: "Sep 15 - Sep 29",
            issues: [
              {
                id: "CRMMS-77911",
                key: "CRMMS-77911",
                kind: "story",
                title: "Fortune story",
                status: "progress",
                priority: "high",
                points: 5
              }
            ]
          }
        ],
        total_issues: 1,
        detected_squad: "Fortune Squad",
        available_squads: ["Fortune Squad", "Azzuri"]
      }

      await page.route(/\/api\/jira\/issues/, async (route) => {
        await route.fulfill({ status: 200, json: mockAssignedIssues })
      })
      await page.route(/\/api\/jira\/backlog/, async (route) => {
        await route.fulfill({ status: 200, json: multiSquadBacklog })
      })

      await jiraPage.goto()
      await jiraPage.tabBacklog.click()

      await expect(jiraPage.squadFilterBar).toBeVisible()
      await expect(page.getByTestId("squad-pill-fortune-squad")).toHaveClass(/is-active/)
      await expect(page.getByTestId("squad-pill-fortune-squad")).toContainText("Yours")
      await expect(page.getByTestId("squad-pill-azzuri")).toBeVisible()
    }
  )
})
