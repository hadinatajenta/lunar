import { test, expect } from "../../../fixtures/authenticated.fixture"
import { captureEvidence } from "../../../helpers/evidence"
import { routeSecrets, routeSystemConfig } from "../../../helpers/route-mock"
import { mockChatSuccessResponse } from "../../../data/copilot.data"

test.describe("AI Copilot - Chat & Conversation", () => {
  test.beforeEach(async ({ page }) => {
    await routeSecrets(page, { hasAiKeys: true, configuredAiProviders: ["deepseek"] })
    await routeSystemConfig(page, [])
  })

  test(
    "CP-VIEW-001 engineer can see empty chat state with greeting when configured",
    {
      tag: ["@copilot", "@positive", "@view"]
    },
    async ({ page, copilotPage }) => {
      await copilotPage.navigateTo()
      await expect(copilotPage.emptyHistoryNotice).toBeVisible()
      await expect(copilotPage.emptyTitle).toHaveText("How can I help you today?")
      await captureEvidence(page, "copilot", "05_copilot_initial.png", true)
    }
  )

  test(
    "CP-CONV-001 engineer can send a message and receive AI response with thought process and sources",
    {
      tag: ["@copilot", "@positive", "@conversation", "@p0"]
    },
    async ({ page, copilotPage }) => {
      let capturedPayload: { prompt?: string; model?: string } = {}
      await page.route(/\/api\/copilot\/chat/, async (route) => {
        capturedPayload = route.request().postDataJSON() || {}
        await route.fulfill({
          status: 200,
          contentType: "application/json",
          body: JSON.stringify(mockChatSuccessResponse)
        })
      })

      await copilotPage.navigateTo()
      await copilotPage.messageTextarea.fill("Analyze pull request #184")
      await copilotPage.sendButton.click()

      await expect(copilotPage.thoughtSection).toBeVisible({ timeout: 12000 })
      await expect(copilotPage.thoughtContent).toContainText("Deep reasoning pass initiated")
      await expect(copilotPage.lastAssistantMessage).toContainText("Analyzed pull request #184")
      await expect(copilotPage.firstSourceChip).toBeVisible()

      expect(capturedPayload.prompt).toBe("Analyze pull request #184")
      await captureEvidence(page, "copilot", "06_copilot_response_received.png", true)
    }
  )

  test(
    "CP-CONV-002 engineer can toggle thought process section visibility",
    {
      tag: ["@copilot", "@positive", "@ui"]
    },
    async ({ page, copilotPage }) => {
      await page.route(/\/api\/copilot\/chat/, async (route) => {
        await route.fulfill({
          status: 200,
          contentType: "application/json",
          body: JSON.stringify(mockChatSuccessResponse)
        })
      })

      await copilotPage.navigateTo()
      await copilotPage.messageTextarea.fill("Analyze pending pull requests")
      await copilotPage.sendButton.click()

      await expect(copilotPage.thoughtSection).toBeVisible({ timeout: 12000 })
      await copilotPage.thoughtToggle.click()
      await expect(copilotPage.thoughtContent).not.toBeVisible()

      await copilotPage.thoughtToggle.click()
      await expect(copilotPage.thoughtContent).toBeVisible()
    }
  )
})
