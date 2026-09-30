import { test, expect } from "../../../fixtures/authenticated.fixture"
import { captureEvidence } from "../../../helpers/evidence"
import { routeSecrets, routeStandardJira } from "../../../helpers/route-mock"

test.describe("Jira BRI - Backlog & Sprints Positive Flows", () => {
  test.beforeEach(async ({ page }) => {
    await routeSecrets(page, { hasJiraPat: true })
    await routeStandardJira(page)
  })

  test(
    "JIRA-BACKLOG-001 user can switch to backlog tab and view active sprint, future sprint, and backlog issues",
    {
      tag: ["@jira", "@positive", "@backlog", "@p0"]
    },
    async ({ page, jiraPage }) => {
      await jiraPage.goto()
      await jiraPage.tabBacklog.click()
      await expect(jiraPage.tabBacklog).toHaveClass(/is-active/)
      await expect(jiraPage.backlogContainer).toBeVisible()
      await expect(jiraPage.backlogSprints).toHaveCount(2)
      await expect(jiraPage.backlogContainer).toContainText("ACTIVE")
      await expect(jiraPage.backlogIssues).toHaveCount(2)
      await captureEvidence(page, "jira", "03_jira_backlog_view.png", true)
    }
  )

  test(
    "JIRA-BACKLOG-002 user can click backlog issue to open issue detail modal",
    {
      tag: ["@jira", "@positive", "@backlog"]
    },
    async ({ jiraPage }) => {
      await jiraPage.goto()
      await jiraPage.tabBacklog.click()
      await expect(jiraPage.backlogContainer).toBeVisible()
      await jiraPage.backlogIssues.first().click()
      await expect(jiraPage.issueModal).toBeVisible()
      await expect(jiraPage.modalKey).toHaveText("CRMMS-77911")
      await expect(jiraPage.modalTitle).toContainText("Sebagai Administrator")
      await jiraPage.btnCloseModal.click()
      await expect(jiraPage.issueModal).not.toBeVisible()
    }
  )
})
