import { test, expect } from "../../../fixtures/unauthenticated.fixture"
import { captureEvidence } from "../../../helpers/evidence"

const testEmail = process.env.E2E_USER_EMAIL || process.env.SEED_USER_EMAIL || "developer@lunar.dev"
const testPassword = process.env.E2E_USER_PASSWORD || process.env.SEED_USER_PASSWORD || "12345678"

test.describe("Authentication - Positive Login", () => {
  test(
    "AUTH-LOGIN-001 user can see login page heading and hero copy",
    {
      tag: ["@auth", "@positive", "@p0"]
    },
    async ({ page, authPage }) => {
      await authPage.navigateTo()
      await expect(authPage.pageHeading).toHaveText("Work closer to the signal.")
      await captureEvidence(page, "auth", "01_login_page_initial.png", true)
    }
  )

  test(
    "AUTH-LOGIN-003 user can log in with valid credentials and land on dashboard",
    {
      tag: ["@auth", "@positive", "@p0"]
    },
    async ({ page, authPage }) => {
      await authPage.navigateTo()
      await authPage.rememberMeCheckbox.check()
      await captureEvidence(page, "auth", "03_login_filled.png")
      await authPage.loginWith(testEmail, testPassword)
      await authPage.assertLandedOnDashboard()
      await expect(page.getByRole("heading", { level: 1 })).toContainText("Operations Overview")
      await captureEvidence(page, "auth", "04_login_success_dashboard.png", true)
    }
  )

  test(
    "AUTH-SESS-001 user can log out from sidebar and be redirected to login page",
    {
      tag: ["@auth", "@positive", "@session"]
    },
    async ({ page, authPage }) => {
      await authPage.navigateTo()
      await authPage.loginWith(testEmail, testPassword)
      await authPage.assertLandedOnDashboard()

      const signOutBtn = page.getByRole("button", { name: "Sign out" })
      await expect(signOutBtn).toBeVisible()
      await signOutBtn.click()

      await expect(page).toHaveURL(/\/login/)
      await expect(page.getByRole("heading", { level: 1 })).toHaveText("Work closer to the signal.")
    }
  )
})
