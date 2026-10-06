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
    "JIRA-MODAL-003 user can open the full description on Jira in a new tab",
    {
      tag: ["@jira", "@positive", "@modal", "@external-link"]
    },
    async ({ page, jiraPage }) => {
      await jiraPage.goto()
      await page.getByTestId("kanban-card-CRMMS-77911").click()
      await expect(jiraPage.issueModal).toBeVisible()
      await expect(jiraPage.btnReadMore).toBeVisible()
      await expect(jiraPage.btnReadMore).toHaveText("Open full description in Jira")

      await page.context().route("https://jira.bri.co.id/**", async (route) => {
        await route.fulfill({
          status: 200,
          contentType: "text/html",
          body: "<!doctype html><title>CRMMS-77911 - Jira</title>"
        })
      })

      const newTabPromise = page.context().waitForEvent("page")
      await jiraPage.btnReadMore.click()
      const newTab = await newTabPromise

      await expect(newTab).toHaveURL("https://jira.bri.co.id/browse/CRMMS-77911")
      await expect(page).toHaveURL(/\/jira$/)
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

  test(
    "JIRA-MODAL-007 user can click Confluence wiki link to navigate to document detail",
    {
      tag: ["@jira", "@positive", "@modal", "@confluence-link"]
    },
    async ({ page, jiraPage }) => {
      await page.route(/\/api\/jira\/issues\/remotelinks/, async (route) => {
        await route.fulfill({
          status: 200,
          json: [
            {
              id: "link-1",
              relationship: "wiki",
              object: {
                url: "https://confluence.bri.co.id/pages/101",
                title: "Authentication Architecture Spec"
              }
            }
          ]
        })
      })

      await jiraPage.goto()
      await page.getByTestId("kanban-card-CRMMS-77911").click()
      await expect(jiraPage.issueModal).toBeVisible()

      const wikiSection = page.getByTestId("wiki-links-section")
      await expect(wikiSection).toBeVisible()
      await expect(wikiSection).toContainText("Authentication Architecture Spec")

      await wikiSection.getByTestId("wiki-link-item").click()
      await expect(page).toHaveURL("/confluence/101")
      await expect(jiraPage.issueModal).not.toBeVisible()
    }
  )

  test(
    "JIRA-MODAL-008 modal footer keeps Open in Jira and Close aligned on one row",
    {
      tag: ["@jira", "@positive", "@modal", "@layout"]
    },
    async ({ page, jiraPage }) => {
      await jiraPage.goto()
      await page.getByTestId("kanban-card-CRMMS-77911").click()
      await expect(jiraPage.issueModal).toBeVisible()

      const modalFooter = page.locator(".modal-footer")
      await expect(modalFooter).toHaveCount(1)
      await expect(page.locator(".modal-body .modal-footer")).toHaveCount(0)
      const footerDisplay = await modalFooter.evaluate(
        (element): string => getComputedStyle(element).display
      )
      expect(footerDisplay).toBe("flex")

      const openInJiraButton = page.getByTestId("btn-open-jira")
      const footerCloseButton = page.getByTestId("btn-footer-close")
      await expect(openInJiraButton).toBeVisible()
      await expect(footerCloseButton).toBeVisible()

      const openInJiraBox = await openInJiraButton.boundingBox()
      const footerCloseBox = await footerCloseButton.boundingBox()
      expect(openInJiraBox).not.toBeNull()
      expect(footerCloseBox).not.toBeNull()
      if (!openInJiraBox || !footerCloseBox) {
        throw new Error("Modal footer buttons did not render measurable boxes")
      }

      expect(Math.abs(openInJiraBox.y - footerCloseBox.y)).toBeLessThanOrEqual(2)
      expect(Math.abs(openInJiraBox.height - footerCloseBox.height)).toBeLessThanOrEqual(2)
      expect(footerCloseBox.x).toBeGreaterThan(openInJiraBox.x + openInJiraBox.width)

      const footerBox = await modalFooter.boundingBox()
      await page.locator(".modal-body").evaluate((element): void => {
        const scrollableBody = element as HTMLElement
        scrollableBody.scrollTop = scrollableBody.scrollHeight
      })
      const scrolledFooterBox = await modalFooter.boundingBox()
      expect(footerBox).not.toBeNull()
      expect(scrolledFooterBox).not.toBeNull()
      if (!footerBox || !scrolledFooterBox) {
        throw new Error("Modal footer did not render a measurable box")
      }
      expect(Math.abs(scrolledFooterBox.y - footerBox.y)).toBeLessThanOrEqual(1)
    }
  )
})
