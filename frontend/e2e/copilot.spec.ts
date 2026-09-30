import { test, expect } from "@playwright/test"
import path from "path"
import { CopilotPage } from "./pages/CopilotPage"

const EVIDENCE_DIR = "/Users/erendt/code/lunar/docs/copilot/evidence"

const mockConfiguredSecrets = {
  user_id: "test-user-id",
  has_jira_pat: true,
  jira_username: "developer",
  has_bitbucket_pat: true,
  bitbucket_username: "developer",
  has_confluence_pat: true,
  has_ai_keys: true,
  configured_ai_providers: ["deepseek"],
  updated_at: "Just now"
}

const mockUnconfiguredSecrets = {
  user_id: "test-user-id",
  has_jira_pat: true,
  jira_username: "developer",
  has_bitbucket_pat: true,
  bitbucket_username: "developer",
  has_confluence_pat: true,
  has_ai_keys: false,
  configured_ai_providers: [],
  updated_at: "Just now"
}

const mockChatSuccessResponse = {
  session_id: "session-test-1",
  message: {
    id: "msg-ast-test-1",
    chat_id: "session-test-1",
    role: "assistant",
    content: "Analyzed pull request #184 and linked Jira issue LUN-421. Architecture boundaries and lint constraints are verified cleanly.",
    reasoning: "Deep reasoning pass initiated. Evaluated session renewal flow and verified zero comments rule.",
    thinking_duration_ms: 1240,
    sources: [
      { type: "jira", label: "Jira BRI · Active Issues" },
      { type: "bitbucket", label: "Bitbucket BRI · PRs & Diffs" }
    ],
    created_at: "Just now"
  }
}

function routeCopilotChatSuccess(page: import("@playwright/test").Page) {
  return page.route(/\/api\/copilot\/chat/, async (route) => {
    await route.fulfill({
      status: 200,
      json: mockChatSuccessResponse
    })
  })
}

function routeSecretsConfigured(page: import("@playwright/test").Page, providers = ["deepseek"]) {
  return page.route(/\/api\/auth\/secrets/, async (route) => {
    if (route.request().method() === "GET") {
      await route.fulfill({
        status: 200,
        json: {
          ...mockConfiguredSecrets,
          configured_ai_providers: providers,
          has_ai_keys: providers.length > 0
        }
      })
      return
    }
    await route.continue()
  })
}

function routeSecretsUnconfigured(page: import("@playwright/test").Page) {
  return page.route(/\/api\/auth\/secrets/, async (route) => {
    if (route.request().method() === "GET") {
      await route.fulfill({
        status: 200,
        json: mockUnconfiguredSecrets
      })
      return
    }
    await route.continue()
  })
}

function routeConfigNoSystemAI(page: import("@playwright/test").Page) {
  return page.route(/\/api\/config/, async (route) => {
    await route.fulfill({
      status: 200,
      json: {
        jira_base_url: "https://jira.bri.co.id",
        bitbucket_base_url: "https://bitbucket.bri.co.id",
        confluence_base_url: "https://confluence.bri.co.id",
        system_ai_providers: []
      }
    })
  })
}

test.describe("AI Copilot & Active Tools", () => {
  test("engineer cannot access Copilot chat when no AI provider API key is configured", async ({ page }) => {
    await routeSecretsUnconfigured(page)
    await routeConfigNoSystemAI(page)

    const copilotPage = new CopilotPage(page)
    await copilotPage.navigateTo()

    await expect(copilotPage.noAiKeysGate).toBeVisible()
    await expect(copilotPage.noAiKeysGate).toContainText("AI Provider API Key Required")
    await expect(copilotPage.goToSettingsButton).toBeVisible()
    await page.screenshot({ path: path.join(EVIDENCE_DIR, "01_copilot_gate_unconfigured.png") })

    await copilotPage.goToSettingsButton.click()
    await expect(page).toHaveURL(/\/settings/)
  })

  test("engineer only sees and can select configured models in model picker", async ({ page }) => {
    await routeSecretsConfigured(page, ["deepseek"])
    await routeConfigNoSystemAI(page)

    const copilotPage = new CopilotPage(page)
    await copilotPage.navigateTo()

    await expect(copilotPage.modelTrigger).toBeVisible()
    await copilotPage.modelTrigger.click()
    await expect(copilotPage.modelMenu).toBeVisible()

    await expect(copilotPage.modelOption("DeepSeek-V4 Pro (Thinking)")).toBeVisible()
    await expect(copilotPage.modelOption("DeepSeek Flash")).toBeVisible()

    await expect(copilotPage.disabledModelOptions).toHaveCount(8)
    await page.screenshot({ path: path.join(EVIDENCE_DIR, "02_copilot_model_picker_filtered.png") })

    await copilotPage.modelOption("DeepSeek Flash").click()
    await expect(copilotPage.modelTriggerLabel).toHaveText("DeepSeek Flash")
  })

  test("engineer receives clear error message when upstream AI provider returns 401 Unauthorized", async ({
    page
  }) => {
    await routeSecretsConfigured(page, ["deepseek"])
    await routeConfigNoSystemAI(page)
    await page.route(/\/api\/copilot\/chat/, async (route) => {
      await route.fulfill({
        status: 401,
        json: { error: "unauthorized: invalid or expired deepseek API key" }
      })
    })

    const copilotPage = new CopilotPage(page)
    await copilotPage.navigateTo()

    await copilotPage.messageTextarea.fill("Review database migration indexes")
    await copilotPage.sendButton.click()

    await expect(copilotPage.lastAssistantMessage).toBeVisible({ timeout: 8000 })
    await expect(copilotPage.lastAssistantMessage).toContainText("invalid or expired deepseek API key")
    await page.screenshot({ path: path.join(EVIDENCE_DIR, "03_copilot_error_401_invalid_key.png") })
  })

  test("engineer receives clear error message when upstream AI provider returns 502 Bad Gateway", async ({
    page
  }) => {
    await routeSecretsConfigured(page, ["deepseek"])
    await routeConfigNoSystemAI(page)
    await page.route(/\/api\/copilot\/chat/, async (route) => {
      await route.fulfill({
        status: 502,
        json: { error: "failed to contact AI provider (deepseek): connection timeout" }
      })
    })

    const copilotPage = new CopilotPage(page)
    await copilotPage.navigateTo()

    await copilotPage.messageTextarea.fill("Analyze service dependencies")
    await copilotPage.sendButton.click()

    await expect(copilotPage.lastAssistantMessage).toBeVisible({ timeout: 8000 })
    await expect(copilotPage.lastAssistantMessage).toContainText("connection timeout")
    await page.screenshot({ path: path.join(EVIDENCE_DIR, "04_copilot_error_502_timeout.png") })
  })

  test("engineer can see empty chat state with greeting when configured", async ({ page }) => {
    await routeSecretsConfigured(page)
    await routeConfigNoSystemAI(page)

    const copilotPage = new CopilotPage(page)
    await copilotPage.navigateTo()

    await expect(copilotPage.emptyHistoryNotice).toBeVisible()
    await expect(copilotPage.emptyTitle).toHaveText("How can I help you today?")
    await page.screenshot({ path: path.join(EVIDENCE_DIR, "05_copilot_initial.png"), fullPage: true })
  })

  test("engineer can open Active Copilot Tools modal and toggle a tool", async ({ page }) => {
    await routeSecretsConfigured(page)
    await routeConfigNoSystemAI(page)

    const copilotPage = new CopilotPage(page)
    await copilotPage.navigateTo()

    await copilotPage.settingsGearButton.click()
    await expect(copilotPage.activeToolsModalTitle).toHaveText("Active Copilot Tools")

    await copilotPage.domainRow("Query Review Assistant").click()
    await copilotPage.doneButton.click()
    await expect(copilotPage.activeToolsModalTitle).not.toBeVisible()
  })

  test("engineer can see Thinking ON toggle by default and toggle it off", async ({ page }) => {
    await routeSecretsConfigured(page)
    await routeConfigNoSystemAI(page)

    const copilotPage = new CopilotPage(page)
    await copilotPage.navigateTo()

    await expect(copilotPage.thinkingToggle).toBeVisible()
    await expect(copilotPage.thinkingToggle).toContainText("Thinking ON")

    await copilotPage.thinkingToggle.click()
    await expect(copilotPage.thinkingToggle).toContainText("Thinking OFF")

    await copilotPage.thinkingToggle.click()
    await expect(copilotPage.thinkingToggle).toContainText("Thinking ON")
  })

  test("engineer can change reasoning effort level via dropdown", async ({ page }) => {
    await routeSecretsConfigured(page)
    await routeConfigNoSystemAI(page)

    const copilotPage = new CopilotPage(page)
    await copilotPage.navigateTo()

    await copilotPage.effortTrigger.click()
    await expect(copilotPage.effortMenu).toBeVisible()

    await copilotPage.effortOption("High effort").click()
    await expect(copilotPage.effortTrigger).toContainText("Effort: high")
  })

  test("engineer can send a message and receive AI response with thought process and sources", async ({
    page
  }) => {
    await routeSecretsConfigured(page)
    await routeConfigNoSystemAI(page)
    await routeCopilotChatSuccess(page)

    const copilotPage = new CopilotPage(page)
    await copilotPage.navigateTo()

    await copilotPage.messageTextarea.fill(
      "Analyze pending Bitbucket pull requests and linked Jira issues for MMS Sprint 4"
    )
    await copilotPage.sendButton.click()

    await expect(copilotPage.thoughtSection).toBeVisible({ timeout: 12000 })
    await expect(copilotPage.thoughtTitle).toHaveText("Thought process")
    await expect(copilotPage.thoughtContent).not.toBeEmpty()

    await expect(copilotPage.lastAssistantMessage).toBeVisible()
    await expect(copilotPage.lastAssistantMessage).toContainText("Analyzed pull request #184")
    await expect(copilotPage.firstSourceChip).toBeVisible()
    await page.screenshot({ path: path.join(EVIDENCE_DIR, "06_copilot_response_received.png"), fullPage: true })
  })

  test("engineer can toggle thought process section visibility", async ({ page }) => {
    await routeSecretsConfigured(page)
    await routeConfigNoSystemAI(page)
    await routeCopilotChatSuccess(page)

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
    await routeSecretsConfigured(page)
    await routeConfigNoSystemAI(page)
    await routeCopilotChatSuccess(page)

    const copilotPage = new CopilotPage(page)
    await copilotPage.navigateTo()

    await copilotPage.messageTextarea.fill("Analyze pending pull requests")
    await copilotPage.sendButton.click()

    await expect(copilotPage.firstHistoryItem).toBeVisible({ timeout: 12000 })
    await copilotPage.firstHistoryItem.hover()

    const deleteBtn = copilotPage.historyDeleteButton(copilotPage.firstHistoryItem)
    await expect(deleteBtn).toBeVisible()
    await deleteBtn.click()

    await expect(copilotPage.emptyHistoryNotice).toBeVisible()
    await page.screenshot({ path: path.join(EVIDENCE_DIR, "07_copilot_session_deleted.png"), fullPage: true })
  })
})
