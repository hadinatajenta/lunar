import { type Page } from "@playwright/test"

export class RegisterPage {
  readonly page: Page

  constructor(page: Page) {
    this.page = page
  }

  get jiraPatInput() {
    return this.page.locator("#jira-pat")
  }

  get verifyIdentityButton() {
    return this.page.getByRole("button", { name: "Verify Identity" })
  }

  get verifiedBadge() {
    return this.page.locator(".verified-badge")
  }

  get badgeEmail() {
    return this.page.locator(".badge-email")
  }

  get badgeChangeButton() {
    return this.page.locator(".change-button")
  }

  get passwordInput() {
    return this.page.locator("#password")
  }

  get confirmPasswordInput() {
    return this.page.locator("#confirm-password")
  }

  get completeRegistrationButton() {
    return this.page.getByRole("button", { name: "Create Workspace Account" })
  }

  get errorBanner() {
    return this.page.locator(".error-banner")
  }

  get fieldValidationError() {
    return this.page.locator(".error-text").first()
  }

  async navigateTo() {
    await this.page.goto("/register")
  }

  async fillPat(pat: string) {
    await this.jiraPatInput.fill(pat)
  }

  async clickVerify() {
    await this.verifyIdentityButton.click()
  }

  async fillPasswords(password: string, confirmPassword: string) {
    await this.passwordInput.fill(password)
    await this.confirmPasswordInput.fill(confirmPassword)
  }

  async submitRegistration() {
    await this.completeRegistrationButton.click()
  }
}
