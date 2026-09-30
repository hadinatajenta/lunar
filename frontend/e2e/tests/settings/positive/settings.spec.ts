import { test, expect } from "../../../fixtures/authenticated.fixture"
import { mockSettingsSecrets } from "../../../data/settings.data"
import { captureEvidence } from "../../../helpers/evidence"

function interceptSecretsWrite(page: import("@playwright/test").Page) {
  return page.route(/\/api\/auth\/secrets/, async (route) => {
    if (route.request().method() === "PUT") {
      await route.fulfill({ status: 200, json: mockSettingsSecrets })
      return
    }
    await route.continue()
  })
}

test.describe("Settings & Credential Vault Positive Flows", () => {
  test(
    "SETTINGS-001 engineer can view settings page with integrations tab open by default",
    {
      tag: ["@settings", "@positive", "@navigation", "@p0"]
    },
    async ({ page, settingsPage }) => {
      await settingsPage.navigateTo()
      await expect(settingsPage.pageHeading).toHaveText("Settings")
      await captureEvidence(page, "settings", "01_settings_integrations_tab.png", { fullPage: true })
    }
  )

  test(
    "SETTINGS-002 engineer can save Atlassian PATs and see success confirmation",
    {
      tag: ["@settings", "@positive", "@atlassian", "@p0"]
    },
    async ({ page, settingsPage }) => {
      await interceptSecretsWrite(page)
      await settingsPage.navigateTo()
      await settingsPage.fillAtlassianCredentials({
        jiraPat: "jira-pat-test-token-12345",
        bitbucketUsername: "developer",
        bitbucketPat: "bitbucket-pat-test-token-67890",
        confluencePat: "confluence-pat-test-token-abcde"
      })
      await captureEvidence(page, "settings", "02_settings_integrations_filled.png")
      await settingsPage.saveIntegrationsButton.click()
      await expect(settingsPage.successToast).toBeVisible()
      await captureEvidence(page, "settings", "03_settings_integrations_saved.png")
    }
  )

  test(
    "SETTINGS-003 engineer can view AI providers tab with correct model labels for each provider",
    {
      tag: ["@settings", "@positive", "@ai-providers", "@p1"]
    },
    async ({ page, settingsPage }) => {
      await settingsPage.navigateTo()
      await settingsPage.aiProvidersTabButton().click()
      await expect(page.locator("#gemini-key")).toBeVisible()

      await expect(settingsPage.providerSubtitle("OpenAI")).toContainText("GPT-6 Astra")
      await expect(settingsPage.providerSubtitle("Anthropic Claude")).toContainText("Claude Opus 5.5")
      await expect(settingsPage.providerSubtitle("Google Gemini")).toContainText("Gemini 3.8 Flash")
      await expect(settingsPage.providerSubtitle("DeepSeek")).toContainText("deepseek-v4-pro")
      await expect(settingsPage.providerSubtitle("Xiaomi MiMo")).toContainText("mimo-v2.5-pro")

      await captureEvidence(page, "settings", "04_settings_ai_tab.png", { fullPage: true })
    }
  )

  test(
    "SETTINGS-004 engineer can save a Gemini API key and see success toast",
    {
      tag: ["@settings", "@positive", "@ai-providers", "@gemini"]
    },
    async ({ page, settingsPage }) => {
      await interceptSecretsWrite(page)
      await settingsPage.navigateTo()
      await settingsPage.aiProvidersTabButton().click()
      await settingsPage.geminiKeyInput.fill("AIzaSyD-sample-gemini-key-998877")
      await settingsPage.providerSaveButton("Google Gemini").click()
      await expect(settingsPage.successToast).toBeVisible()
      await captureEvidence(page, "settings", "05_settings_ai_saved.png")
    }
  )

  test(
    "SETTINGS-005 engineer can save a DeepSeek API key and see success toast",
    {
      tag: ["@settings", "@positive", "@ai-providers", "@deepseek"]
    },
    async ({ page, settingsPage }) => {
      await interceptSecretsWrite(page)
      await settingsPage.navigateTo()
      await settingsPage.aiProvidersTabButton().click()
      await settingsPage.deepseekKeyInput.fill("sk-deepseek-sample-key-112233")
      await settingsPage.providerSaveButton("DeepSeek").click()
      await expect(settingsPage.successToast).toBeVisible()
    }
  )

  test(
    "SETTINGS-006 engineer can navigate to the Security tab and see encryption status",
    {
      tag: ["@settings", "@positive", "@security", "@p1"]
    },
    async ({ page, settingsPage }) => {
      await settingsPage.navigateTo()
      await settingsPage.securityTabButton().click()
      await expect(settingsPage.securityCardTitle).toContainText("Security & Encryption")
      await captureEvidence(page, "settings", "06_settings_security_tab.png", { fullPage: true })
    }
  )
})
