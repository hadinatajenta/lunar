import { test, expect } from "@playwright/test"
import path from "path"
import { SettingsPage } from "./pages/SettingsPage"

const EVIDENCE_DIR = "/Users/erendt/code/lunar/docs/settings/evidence"

const mockSaveSecretsResponse = {
  user_id: "test-user",
  has_jira_pat: true,
  jira_username: "developer",
  has_bitbucket_pat: true,
  bitbucket_username: "developer",
  has_confluence_pat: true,
  has_ai_keys: true,
  configured_ai_providers: ["deepseek", "gemini"],
  updated_at: "Just now"
}

function interceptSecretsWrite(page: import("@playwright/test").Page) {
  return page.route(/\/api\/auth\/secrets/, async (route) => {
    if (route.request().method() === "PUT") {
      await route.fulfill({ status: 200, json: mockSaveSecretsResponse })
      return
    }
    await route.continue()
  })
}

test.describe("Settings & Credential Vault", () => {
  test("engineer can view settings page with integrations tab open by default", async ({ page }) => {
    const settingsPage = new SettingsPage(page)
    await settingsPage.navigateTo()
    await expect(settingsPage.pageHeading).toHaveText("Settings")
    await page.screenshot({ path: path.join(EVIDENCE_DIR, "01_settings_integrations_tab.png"), fullPage: true })
  })

  test("engineer can save Atlassian PATs and see success confirmation", async ({ page }) => {
    await interceptSecretsWrite(page)
    const settingsPage = new SettingsPage(page)
    await settingsPage.navigateTo()
    await settingsPage.fillAtlassianCredentials({
      jiraPat: "jira-pat-test-token-12345",
      bitbucketUsername: "developer",
      bitbucketPat: "bitbucket-pat-test-token-67890",
      confluencePat: "confluence-pat-test-token-abcde"
    })
    await page.screenshot({ path: path.join(EVIDENCE_DIR, "02_settings_integrations_filled.png") })
    await settingsPage.saveIntegrationsButton.click()
    await expect(settingsPage.successToast).toBeVisible()
    await page.screenshot({ path: path.join(EVIDENCE_DIR, "03_settings_integrations_saved.png") })
  })

  test("engineer can view AI providers tab with correct model labels for each provider", async ({ page }) => {
    const settingsPage = new SettingsPage(page)
    await settingsPage.navigateTo()
    await settingsPage.aiProvidersTabButton().click()
    await expect(page.locator("#gemini-key")).toBeVisible()

    await expect(settingsPage.providerSubtitle("OpenAI")).toContainText("GPT-6 Astra")
    await expect(settingsPage.providerSubtitle("Anthropic Claude")).toContainText("Claude Opus 5.5")
    await expect(settingsPage.providerSubtitle("Google Gemini")).toContainText("Gemini 3.8 Flash")
    await expect(settingsPage.providerSubtitle("DeepSeek")).toContainText("deepseek-v4-pro")
    await expect(settingsPage.providerSubtitle("Xiaomi MiMo")).toContainText("mimo-v2.5-pro")

    await page.screenshot({ path: path.join(EVIDENCE_DIR, "04_settings_ai_tab.png"), fullPage: true })
  })

  test("engineer can save a Gemini API key and see success toast", async ({ page }) => {
    await interceptSecretsWrite(page)
    const settingsPage = new SettingsPage(page)
    await settingsPage.navigateTo()
    await settingsPage.aiProvidersTabButton().click()
    await settingsPage.geminiKeyInput.fill("AIzaSyD-sample-gemini-key-998877")
    await settingsPage.providerSaveButton("Google Gemini").click()
    await expect(settingsPage.successToast).toBeVisible()
    await page.screenshot({ path: path.join(EVIDENCE_DIR, "05_settings_ai_saved.png") })
  })

  test("engineer can save a DeepSeek API key and see success toast", async ({ page }) => {
    await interceptSecretsWrite(page)
    const settingsPage = new SettingsPage(page)
    await settingsPage.navigateTo()
    await settingsPage.aiProvidersTabButton().click()
    await settingsPage.deepseekKeyInput.fill("sk-deepseek-sample-key-112233")
    await settingsPage.providerSaveButton("DeepSeek").click()
    await expect(settingsPage.successToast).toBeVisible()
  })

  test("engineer can navigate to the Security tab and see encryption status", async ({ page }) => {
    const settingsPage = new SettingsPage(page)
    await settingsPage.navigateTo()
    await settingsPage.securityTabButton().click()
    await expect(settingsPage.securityCardTitle).toContainText("Security & Encryption")
    await page.screenshot({ path: path.join(EVIDENCE_DIR, "06_settings_security_tab.png"), fullPage: true })
  })
})
