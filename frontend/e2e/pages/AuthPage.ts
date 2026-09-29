import { type Page, expect } from "@playwright/test"

export class AuthPage {
  readonly page: Page

  constructor(page: Page) {
    this.page = page
  }

  get emailInput() {
    return this.page.locator("#email")
  }

  get passwordInput() {
    return this.page.locator("#password")
  }

  get rememberMeCheckbox() {
    return this.page.locator("#remember")
  }

  get submitButton() {
    return this.page.getByRole("button", { name: "Sign in" })
  }

  get validationError() {
    return this.page.locator(".error-text").first()
  }

  get pageHeading() {
    return this.page.getByRole("heading", { level: 1 })
  }

  async navigateTo() {
    await this.page.goto("/login")
  }

  async loginWith(email: string, password: string) {
    await this.emailInput.fill(email)
    await this.passwordInput.fill(password)
    await this.submitButton.click()
  }

  async submitWithEmptyFields() {
    await this.submitButton.click()
  }

  async assertLandedOnDashboard() {
    await expect(this.page).toHaveURL(/\/dashboard/)
  }
}
