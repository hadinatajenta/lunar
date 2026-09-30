import { test, expect } from "../../../fixtures/authenticated.fixture"
import { captureEvidence } from "../../../helpers/evidence"
import { routeSecrets, routeStandardJira } from "../../../helpers/route-mock"

test.describe("Jira BRI - Issue Detail Modal Positive Flows", () => {
  test.beforeEach(async ({ page }) => {
    await routeSecrets(page, { hasJiraPat: true })
    await routeStandardJira(page)
  })

  test(
    "JIRA-MODAL-001 user can click card to open centered modal dialog",
    {
      tag: ["@jira", "@positive", "@modal", "@p0"]
    },
    async ({ page, jiraPage }) => {
      await jiraPage.goto()
      await jiraPage.kanbanCards("open").first().click()
      await expect(jiraPage.issueModal).toBeVisible()
      await expect(jiraPage.issueModal).toHaveAttribute("role", "dialog")
      await expect(jiraPage.issueModal).toHaveAttribute("aria-modal", "true")
      await expect(jiraPage.modalScrim).toBeVisible()
      await captureEvidence(page, "jira", "02_jira_modal_open.png")
    }
  )

  test(
    "JIRA-MODAL-002 user can verify modal contains Jira code, user story title, status, and priority",
    {
      tag: ["@jira", "@positive", "@modal"]
    },
    async ({ page, jiraPage }) => {
      await jiraPage.goto()
      await page.getByTestId("kanban-card-CRMMS-77911").click()
      await expect(jiraPage.issueModal).toBeVisible()
      await expect(jiraPage.modalKey).toHaveText("CRMMS-77911")
      await expect(jiraPage.modalTitle).toContainText("Sebagai Administrator")
      await expect(page.getByTestId("modal-issue-status")).toContainText("In Progress")
      await expect(page.getByTestId("modal-issue-priority")).toContainText("High")
      await expect(page.getByTestId("modal-issue-points")).toContainText("5 pts")
    }
  )

  test(
    "JIRA-MODAL-003 user can toggle read more and read less description button",
    {
      tag: ["@jira", "@positive", "@modal"]
    },
    async ({ page, jiraPage }) => {
      await jiraPage.goto()
      await page.getByTestId("kanban-card-CRMMS-77911").click()
      await expect(jiraPage.issueModal).toBeVisible()
      await expect(jiraPage.btnReadMore).toBeVisible()
      await expect(jiraPage.btnReadMore).toHaveText("Read more")
      await jiraPage.btnReadMore.click()
      await expect(jiraPage.btnReadMore).toHaveText("Read less")
      await jiraPage.btnReadMore.click()
      await expect(jiraPage.btnReadMore).toHaveText("Read more")
    }
  )

  test(
    "JIRA-MODAL-004 user can close modal via close button",
    {
      tag: ["@jira", "@positive", "@modal"]
    },
    async ({ jiraPage }) => {
      await jiraPage.goto()
      await jiraPage.kanbanCards("open").first().click()
      await expect(jiraPage.issueModal).toBeVisible()
      await jiraPage.btnCloseModal.click()
      await expect(jiraPage.issueModal).not.toBeVisible()
    }
  )

  test(
    "JIRA-MODAL-005 user can close modal via backdrop click",
    {
      tag: ["@jira", "@positive", "@modal"]
    },
    async ({ jiraPage }) => {
      await jiraPage.goto()
      await jiraPage.kanbanCards("open").first().click()
      await expect(jiraPage.issueModal).toBeVisible()
      await jiraPage.modalScrim.click({ position: { x: 10, y: 10 } })
      await expect(jiraPage.issueModal).not.toBeVisible()
    }
  )

  test(
    "JIRA-MODAL-006 user can close modal via escape key",
    {
      tag: ["@jira", "@positive", "@modal"]
    },
    async ({ page, jiraPage }) => {
      await jiraPage.goto()
      await jiraPage.kanbanCards("open").first().click()
      await expect(jiraPage.issueModal).toBeVisible()
      await page.keyboard.press("Escape")
      await expect(jiraPage.issueModal).not.toBeVisible()
    }
  )
})
