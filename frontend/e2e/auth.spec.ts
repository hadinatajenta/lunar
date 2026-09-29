import { test, expect } from "@playwright/test"
import path from "path"
import { AuthPage } from "./pages/AuthPage"

const EVIDENCE_DIR = "/Users/erendt/code/lunar/docs/auth/evidence"

test.describe("Authentication", () => {
  test.use({ storageState: { cookies: [], origins: [] } })

  test("user can see login page heading and hero copy", async ({ page }) => {
    const authPage = new AuthPage(page)
    await authPage.navigateTo()
    await expect(authPage.pageHeading).toHaveText("Work closer to the signal.")
    await page.screenshot({ path: path.join(EVIDENCE_DIR, "01_login_page_initial.png"), fullPage: true })
  })

  test("user cannot submit login form when credentials are empty", async ({ page }) => {
    const authPage = new AuthPage(page)
    await authPage.navigateTo()
    await authPage.submitWithEmptyFields()
    await expect(authPage.validationError).toBeVisible()
    await page.screenshot({ path: path.join(EVIDENCE_DIR, "02_login_validation_error.png") })
  })

  test("user can log in with valid credentials and land on dashboard", async ({ page }) => {
    const authPage = new AuthPage(page)
    await authPage.navigateTo()
    await authPage.rememberMeCheckbox.check()
    await page.screenshot({ path: path.join(EVIDENCE_DIR, "03_login_filled.png") })
    await authPage.loginWith("hafinata19@gmail.com", "12345678")
    await authPage.assertLandedOnDashboard()
    await expect(page.getByRole("heading", { level: 1 })).toContainText("Operations Overview")
    await page.screenshot({ path: path.join(EVIDENCE_DIR, "04_login_success_dashboard.png"), fullPage: true })
  })
})
