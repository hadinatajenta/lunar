import { test, expect } from "../../../fixtures/unauthenticated.fixture"
import { mockVerifiedJiraProfile } from "../../../data/auth.data"

test.describe("Authentication - Registration Positive Flows", () => {
  test(
    "AUTH-NAV-001 user can navigate to register page from login form link",
    {
      tag: ["@auth", "@positive", "@navigation"]
    },
    async ({ page }) => {
      await page.goto("/login")
      await expect(page.getByRole("link", { name: "Create one" })).toBeVisible()
      await page.getByRole("link", { name: "Create one" }).click()
      await expect(page).toHaveURL(/\/register/)
      await expect(page.getByRole("heading", { name: "Create your workspace account" })).toBeVisible()
      await expect(page.getByRole("heading", { level: 1 })).toHaveText("Work closer to the signal.")
    }
  )

  test(
    "AUTH-NAV-002 user can navigate back to login page from register form link",
    {
      tag: ["@auth", "@positive", "@navigation"]
    },
    async ({ page }) => {
      await page.goto("/register")
      await expect(page.getByRole("link", { name: "Sign in" })).toBeVisible()
      await page.getByRole("link", { name: "Sign in" }).click()
      await expect(page).toHaveURL(/\/login/)
      await expect(page.getByRole("heading", { name: "Welcome back" })).toBeVisible()
    }
  )

  test(
    "AUTH-REG-004 user can verify valid Jira PAT and see verified badge",
    {
      tag: ["@auth", "@positive", "@integration"]
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
      await expect(page.getByText(mockVerifiedJiraProfile.display_name)).toBeVisible()
      await expect(registerPage.badgeEmail).toHaveText(mockVerifiedJiraProfile.email)
      await expect(registerPage.passwordInput).toBeVisible()
      await expect(registerPage.confirmPasswordInput).toBeVisible()
      await expect(registerPage.completeRegistrationButton).toBeVisible()
    }
  )

  test(
    "AUTH-REG-005 user can click Change button on verified badge to reset token",
    {
      tag: ["@auth", "@positive", "@edge"]
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
      await registerPage.badgeChangeButton.click()

      await expect(registerPage.verifiedBadge).not.toBeVisible()
      await expect(registerPage.jiraPatInput).toBeVisible()
      await expect(registerPage.verifyIdentityButton).toBeVisible()
    }
  )

  test(
    "AUTH-REG-009 user can complete registration with valid Jira PAT and lands on dashboard",
    {
      tag: ["@auth", "@positive", "@integration", "@p0"]
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
          status: 201,
          contentType: "application/json",
          body: JSON.stringify({
            token: "mock-jwt-token-newly-registered",
            user: {
              id: "usr-newly-registered-id",
              email: mockVerifiedJiraProfile.email,
              full_name: mockVerifiedJiraProfile.display_name,
              created_at: "Just now"
            }
          })
        })
      })

      await registerPage.navigateTo()
      await registerPage.fillPat("valid-jira-pat-token-value")
      await registerPage.clickVerify()

      await expect(registerPage.verifiedBadge).toBeVisible()
      await registerPage.fillPasswords("Secret123!", "Secret123!")
      await registerPage.submitRegistration()

      await expect(page).toHaveURL(/\/dashboard/)
      await expect(page.getByRole("heading", { level: 1 })).toContainText("Operations Overview")
    }
  )
})
