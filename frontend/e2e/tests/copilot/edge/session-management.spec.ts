import { test, expect } from "../../../fixtures/authenticated.fixture"
import { captureEvidence } from "../../../helpers/evidence"
import { routeSecrets, routeSystemConfig } from "../../../helpers/route-mock"
import { mockChatSuccessResponse } from "../../../data/copilot.data"

test.describe("AI Copilot - Session Management", () => {
  test(
    "CP-SESSION-001 engineer can delete a chat session and return to empty state",
    {
      tag: ["@copilot", "@edge", "@session"]
    },
    async ({ page, copilotPage }) => {
      await routeSecrets(page, { hasAiKeys: true, configuredAiProviders: ["deepseek"] })
      await routeSystemConfig(page, [])
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

      await expect(copilotPage.firstHistoryItem).toBeVisible({ timeout: 12000 })
      await copilotPage.firstHistoryItem.hover()

      const deleteBtn = copilotPage.historyDeleteButton(copilotPage.firstHistoryItem)
      await expect(deleteBtn).toBeVisible()
      await deleteBtn.click()

      await expect(copilotPage.emptyHistoryNotice).toBeVisible()
      await captureEvidence(page, "copilot", "07_copilot_session_deleted.png", true)
    }
  )
})
