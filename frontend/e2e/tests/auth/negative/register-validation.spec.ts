import { test, expect } from "../../../fixtures/unauthenticated.fixture"
import { mockVerifiedJiraProfile } from "../../../data/auth.data"

test.describe("Authentication - Registration Validation & Errors", () => {
  test(
    "AUTH-REG-001 user sees validation error when clicking verify with empty Jira PAT",
    {
      tag: ["@auth", "@negative", "@validation"]
    },
    async ({ registerPage }) => {
      await registerPage.navigateTo()
      await registerPage.clickVerify()
      await expect(registerPage.fieldValidationError).toBeVisible()
    }
  )

  test(
    "AUTH-REG-002 user sees error banner when Jira PAT is invalid (401 Unauthorized)",
    {
      tag: ["@auth", "@negative", "@authorization"]
    },
    async ({ page, registerPage }) => {
      await page.route(/\/api\/auth\/verify-jira-pat/, async (route) => {
        await route.fulfill({
          status: 401,
          contentType: "application/json",
          body: JSON.stringify({ error: "invalid or expired Jira personal access token" })
        })
      })

      await registerPage.navigateTo()
      await registerPage.fillPat("invalid-pat-token")
      await registerPage.clickVerify()

      await expect(registerPage.errorBanner).toBeVisible()
      await expect(registerPage.errorBanner).toContainText("Invalid or expired Jira Personal Access Token")
    }
  )

  test(
    "AUTH-REG-003 user sees VPN warning banner when Jira BRI is unreachable (502 Bad Gateway)",
    {
      tag: ["@auth", "@negative", "@network"]
    },
    async ({ page, registerPage }) => {
      await page.route(/\/api\/auth\/verify-jira-pat/, async (route) => {
        await route.fulfill({
          status: 502,
          contentType: "application/json",
          body: JSON.stringify({ error: "failed to connect to Jira BRI: ensure corporate VPN is connected" })
        })
      })

      await registerPage.navigateTo()
      await registerPage.fillPat("any-token-vpn-down")
      await registerPage.clickVerify()

      await expect(registerPage.errorBanner).toBeVisible()
      await expect(registerPage.errorBanner).toContainText("verify your VPN or network connection")
    }
  )

  test(
    "AUTH-REG-006 user sees validation error when password is less than 6 characters",
    {
      tag: ["@auth", "@negative", "@validation"]
    },
    async ({ page, registerPage }) => {
      await page.route(/\/api\/auth\/verify-jira-pat/, async (route) => {
        await route.fulfill({
          status: 200,
          contentType: "application/json",
          body: JSON.stringify(mockVerifiedJiraProfile)
        })
      })

      await registerPage.navigateTo()
      await registerPage.fillPat("valid-jira-pat-token-value")
      await registerPage.clickVerify()

      await expect(registerPage.verifiedBadge).toBeVisible()
      await registerPage.fillPasswords("123", "123")
      await registerPage.submitRegistration()

      await expect(registerPage.fieldValidationError).toBeVisible()
    }
  )

  test(
    "AUTH-REG-007 user sees validation error when passwords do not match",
    {
      tag: ["@auth", "@negative", "@validation"]
    },
    async ({ page, registerPage }) => {
      await page.route(/\/api\/auth\/verify-jira-pat/, async (route) => {
        await route.fulfill({
          status: 200,
          contentType: "application/json",
          body: JSON.stringify(mockVerifiedJiraProfile)
        })
      })

      await registerPage.navigateTo()
      await registerPage.fillPat("valid-jira-pat-token-value")
      await registerPage.clickVerify()

      await expect(registerPage.verifiedBadge).toBeVisible()
      await registerPage.fillPasswords("Secret123!", "Mismatch456!")
      await registerPage.submitRegistration()

      await expect(registerPage.fieldValidationError).toBeVisible()
    }
  )

  test(
    "AUTH-REG-008 user sees conflict error banner when registering an already registered Jira identity (409)",
    {
      tag: ["@auth", "@negative", "@conflict"]
    },
    async ({ page, registerPage }) => {
      await page.route(/\/api\/auth\/verify-jira-pat/, async (route) => {
        await route.fulfill({
          status: 200,
          contentType: "application/json",
          body: JSON.stringify(mockVerifiedJiraProfile)
        })
      })

      await page.route(/\/api\/auth\/register-with-jira/, async (route) => {
        await route.fulfill({
          status: 409,
          contentType: "application/json",
          body: JSON.stringify({ error: "an account with this BRI email already exists" })
        })
      })

      await registerPage.navigateTo()
      await registerPage.fillPat("valid-jira-pat-token-value")
      await registerPage.clickVerify()

      await expect(registerPage.verifiedBadge).toBeVisible()
      await registerPage.fillPasswords("Secret123!", "Secret123!")
      await registerPage.submitRegistration()

      await expect(registerPage.errorBanner).toBeVisible()
      await expect(registerPage.errorBanner).toContainText("already exists")
    }
  )
})
