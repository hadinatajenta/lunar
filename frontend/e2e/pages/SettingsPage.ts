import { type Page, expect } from "@playwright/test"

export class SettingsPage {
  readonly page: Page

  constructor(page: Page) {
    this.page = page
  }

  get pageHeading() {
    return this.page.getByRole("heading", { level: 1 })
  }

  get jiraPatInput() {
    return this.page.locator("#jira-pat")
  }

  get bitbucketUsernameInput() {
    return this.page.locator("#bb-user")
  }

  get bitbucketPatInput() {
    return this.page.locator("#bb-pat")
  }

  get confluencePatInput() {
    return this.page.locator("#conf-pat")
  }

  get saveIntegrationsButton() {
    return this.page.locator(".save-button")
  }

  get successToast() {
    return this.page.locator(".toast-banner.success")
  }

  get securityCardTitle() {
    return this.page.locator(".settings-card-title")
  }

  aiProvidersTabButton() {
    return this.page.getByRole("button", { name: "AI providers" })
  }

  securityTabButton() {
    return this.page.getByRole("button", { name: "Security" })
  }

  providerRow(providerName: string) {
    return this.page.locator(`.provider-row:has-text("${providerName}")`)
  }

  providerSubtitle(providerName: string) {
    return this.providerRow(providerName).locator(".provider-subtitle")
  }

  providerSaveButton(providerName: string) {
    return this.providerRow(providerName).locator(".action-btn")
  }

  get geminiKeyInput() {
    return this.page.locator("#gemini-key")
  }

  get deepseekKeyInput() {
    return this.page.locator("#deepseek-key")
  }

  async navigateTo() {
    await this.page.goto("/settings")
    await expect(this.page).toHaveURL(/\/settings/)
  }

  async fillAtlassianCredentials(credentials: {
    jiraPat?: string
    bitbucketUsername?: string
    bitbucketPat?: string
    confluencePat?: string
  }) {
    if (credentials.jiraPat) await this.jiraPatInput.fill(credentials.jiraPat)
    if (credentials.bitbucketUsername) await this.bitbucketUsernameInput.fill(credentials.bitbucketUsername)
    if (credentials.bitbucketPat) await this.bitbucketPatInput.fill(credentials.bitbucketPat)
    if (credentials.confluencePat) await this.confluencePatInput.fill(credentials.confluencePat)
  }
}
