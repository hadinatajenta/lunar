import { test, expect } from "@playwright/test"
import path from "path"
import { CopilotPage } from "./pages/CopilotPage"

const EVIDENCE_DIR = "/Users/erendt/code/lunar/docs/copilot/evidence"

test.describe("AI Copilot & Active Tools", () => {
  test("engineer can see empty chat state with greeting on first visit", async ({ page }) => {
    const copilotPage = new CopilotPage(page)
    await copilotPage.navigateTo()
    await expect(copilotPage.emptyHistoryNotice).toBeVisible()
    await expect(copilotPage.emptyTitle).toHaveText("How can I help you today?")
    await page.screenshot({ path: path.join(EVIDENCE_DIR, "01_copilot_initial.png"), fullPage: true })
  })

  test("engineer can open Active Copilot Tools modal and toggle a tool", async ({ page }) => {
    const copilotPage = new CopilotPage(page)
    await copilotPage.navigateTo()
    await copilotPage.settingsGearButton.click()
    await expect(copilotPage.activeToolsModalTitle).toHaveText("Active Copilot Tools")
    await page.screenshot({ path: path.join(EVIDENCE_DIR, "02_copilot_tool_settings_modal.png") })
    await copilotPage.domainRow("Query Review Assistant").click()
    await page.screenshot({ path: path.join(EVIDENCE_DIR, "03_copilot_tool_settings_toggled.png") })
    await copilotPage.doneButton.click()
    await expect(copilotPage.activeToolsModalTitle).not.toBeVisible()
  })

  test("engineer can see Thinking ON toggle by default", async ({ page }) => {
    const copilotPage = new CopilotPage(page)
    await copilotPage.navigateTo()
    await expect(copilotPage.thinkingToggle).toBeVisible()
    await expect(copilotPage.thinkingToggle).toContainText("Thinking ON")
  })

  test("engineer can change reasoning effort level via dropdown", async ({ page }) => {
    const copilotPage = new CopilotPage(page)
    await copilotPage.navigateTo()
    await copilotPage.effortTrigger.click()
    await expect(copilotPage.effortMenu).toBeVisible()
    await copilotPage.effortOption("High effort").click()
    await expect(copilotPage.effortTrigger).toContainText("Effort: high")
  })

  test("engineer can select a different AI model from the model picker", async ({ page }) => {
    const copilotPage = new CopilotPage(page)
    await copilotPage.navigateTo()
    await copilotPage.modelTrigger.click()
    await expect(copilotPage.modelMenu).toBeVisible()
    await copilotPage.modelOption("DeepSeek-V4 Pro (Thinking)").click()
    await expect(copilotPage.modelTriggerLabel).toHaveText("DeepSeek-V4 Pro (Thinking)")
    await page.screenshot({ path: path.join(EVIDENCE_DIR, "04_copilot_model_selected.png") })
  })

  test("engineer can send a message and receive AI response with thought process", async ({ page }) => {
    const copilotPage = new CopilotPage(page)
    await copilotPage.navigateTo()
    await copilotPage.messageTextarea.fill(
      "Analyze pending Bitbucket pull requests and linked Jira issues for MMS Sprint 4"
    )
    await page.screenshot({ path: path.join(EVIDENCE_DIR, "05_copilot_message_typed.png") })
    await copilotPage.sendButton.click()
    await expect(copilotPage.thoughtSection).toBeVisible({ timeout: 12000 })
    await expect(copilotPage.thoughtTitle).toHaveText("Thought process")
    await expect(copilotPage.thoughtContent).not.toBeEmpty()
    await expect(copilotPage.lastAssistantMessage).toBeVisible()
    await expect(copilotPage.firstSourceChip).toBeVisible()
    await page.screenshot({ path: path.join(EVIDENCE_DIR, "06_copilot_response_received.png"), fullPage: true })
  })

  test("engineer can toggle thought process section visibility", async ({ page }) => {
    const copilotPage = new CopilotPage(page)
    await copilotPage.navigateTo()
    await copilotPage.messageTextarea.fill("Analyze pending pull requests")
    await copilotPage.sendButton.click()
    await expect(copilotPage.thoughtSection).toBeVisible({ timeout: 12000 })
    await copilotPage.thoughtToggle.click()
    await expect(copilotPage.thoughtContent).not.toBeVisible()
    await copilotPage.thoughtToggle.click()
    await expect(copilotPage.thoughtContent).toBeVisible()
  })

  test("engineer can delete a chat session and return to empty state", async ({ page }) => {
    const copilotPage = new CopilotPage(page)
    await copilotPage.navigateTo()
    await copilotPage.messageTextarea.fill("Analyze pending pull requests")
    await copilotPage.sendButton.click()
    await expect(copilotPage.firstHistoryItem).toBeVisible({ timeout: 12000 })
    await copilotPage.firstHistoryItem.hover()
    const deleteBtn = copilotPage.historyDeleteButton(copilotPage.firstHistoryItem)
    await expect(deleteBtn).toBeVisible()
    await page.screenshot({ path: path.join(EVIDENCE_DIR, "07_copilot_history_hover_delete.png") })
    await deleteBtn.click()
    await expect(copilotPage.emptyHistoryNotice).toBeVisible()
    await page.screenshot({ path: path.join(EVIDENCE_DIR, "08_copilot_session_deleted.png"), fullPage: true })
  })
})
