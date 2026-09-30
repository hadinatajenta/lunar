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
    await authPage.loginWith("developer@lunar.dev", "12345678")
    await authPage.assertLandedOnDashboard()
    await expect(page.getByRole("heading", { level: 1 })).toContainText("Operations Overview")
    await page.screenshot({ path: path.join(EVIDENCE_DIR, "04_login_success_dashboard.png"), fullPage: true })
  })

  test("user can log out from sidebar and be redirected to login page", async ({ page }) => {
    const authPage = new AuthPage(page)
    await authPage.navigateTo()
    await authPage.loginWith("developer@lunar.dev", "12345678")
    await authPage.assertLandedOnDashboard()

    const signOutBtn = page.getByRole("button", { name: "Sign out" })
    await expect(signOutBtn).toBeVisible()
    await signOutBtn.click()

    await expect(page).toHaveURL(/\/login/)
    await expect(page.getByRole("heading", { level: 1 })).toHaveText("Work closer to the signal.")
  })
})

test.describe("Jira Verified Onboarding & Registration", () => {
  test.use({ storageState: { cookies: [], origins: [] } })

  test("user can navigate to register page from login form link", async ({ page }) => {
    await page.goto("/login")
    await expect(page.getByRole("link", { name: "Create one" })).toBeVisible()
    await page.getByRole("link", { name: "Create one" }).click()
    await expect(page).toHaveURL(/\/register/)
    await expect(page.getByRole("heading", { name: "Create your workspace account" })).toBeVisible()
    await expect(page.getByRole("heading", { level: 1 })).toHaveText("Work closer to the signal.")
  })

  test("user can navigate back to login page from register form link", async ({ page }) => {
    await page.goto("/register")
    await expect(page.getByRole("link", { name: "Sign in" })).toBeVisible()
    await page.getByRole("link", { name: "Sign in" }).click()
    await expect(page).toHaveURL(/\/login/)
    await expect(page.getByRole("heading", { name: "Welcome back" })).toBeVisible()
  })

  test("user sees validation error when clicking verify with empty Jira PAT", async ({ page }) => {
    await page.goto("/register")
    await page.getByRole("button", { name: "Verify Identity" }).click()
    await expect(page.getByText("Personal Access Token is required")).toBeVisible()
    await expect(page.locator(".verified-badge")).not.toBeVisible()
  })

  test("user sees error banner when Jira PAT is invalid (401 Unauthorized)", async ({ page }) => {
    await page.route(/\/api\/auth\/verify-jira-pat/, (route) =>
      route.fulfill({ status: 401, json: { error: "invalid or expired Jira PAT" } })
    )
    await page.goto("/register")
    await page.getByLabel("Jira Personal Access Token").fill("invalid-pat-token")
    await page.getByRole("button", { name: "Verify Identity" }).click()
    await expect(page.locator(".error-banner")).toBeVisible()
    await expect(page.locator(".error-banner")).toContainText("Invalid or expired Jira Personal Access Token")
    await expect(page.locator(".verified-badge")).not.toBeVisible()
  })

  test("user sees VPN warning banner when Jira BRI is unreachable (502 Bad Gateway)", async ({ page }) => {
    await page.route(/\/api\/auth\/verify-jira-pat/, (route) =>
      route.fulfill({ status: 502, json: { error: "unable to reach Jira BRI API. Please check your VPN connection." } })
    )
    await page.goto("/register")
    await page.getByLabel("Jira Personal Access Token").fill("some-token")
    await page.getByRole("button", { name: "Verify Identity" }).click()
    await expect(page.locator(".error-banner")).toBeVisible()
    await expect(page.locator(".error-banner")).toContainText("Please verify your VPN or network connection")
    await expect(page.locator(".verified-badge")).not.toBeVisible()
  })

  test("user can verify valid Jira PAT and see verified badge", async ({ page }) => {
    await page.route(/\/api\/auth\/verify-jira-pat/, (route) =>
      route.fulfill({
        status: 200,
        json: {
          display_name: "Budi Santoso",
          email: "budi.santoso@bri.co.id",
          username: "70001234"
        }
      })
    )
    await page.goto("/register")
    await page.getByLabel("Jira Personal Access Token").fill("valid-jira-pat")
    await page.getByRole("button", { name: "Verify Identity" }).click()

    await expect(page.locator(".verified-badge")).toBeVisible()
    await expect(page.locator(".badge-name")).toHaveText("Budi Santoso")
    await expect(page.locator(".badge-email")).toHaveText("budi.santoso@bri.co.id")
    await expect(page.locator(".badge-username")).toHaveText("70001234")
    await expect(page.getByLabel("Create Password")).toBeVisible()
    await expect(page.getByLabel("Confirm Password")).toBeVisible()
  })

  test("user can click Change button on verified badge to reset token", async ({ page }) => {
    await page.route(/\/api\/auth\/verify-jira-pat/, (route) =>
      route.fulfill({
        status: 200,
        json: {
          display_name: "Budi Santoso",
          email: "budi.santoso@bri.co.id",
          username: "70001234"
        }
      })
    )
    await page.goto("/register")
    await page.getByLabel("Jira Personal Access Token").fill("valid-jira-pat")
    await page.getByRole("button", { name: "Verify Identity" }).click()
    await expect(page.locator(".verified-badge")).toBeVisible()

    await page.getByRole("button", { name: "Change token" }).click()
    await expect(page.locator(".verified-badge")).not.toBeVisible()
    await expect(page.getByLabel("Jira Personal Access Token")).toBeVisible()
  })

  test("user sees validation error when password is less than 6 characters", async ({ page }) => {
    await page.route(/\/api\/auth\/verify-jira-pat/, (route) =>
      route.fulfill({
        status: 200,
        json: {
          display_name: "Budi Santoso",
          email: "budi.santoso@bri.co.id",
          username: "70001234"
        }
      })
    )
    await page.goto("/register")
    await page.getByLabel("Jira Personal Access Token").fill("valid-jira-pat")
    await page.getByRole("button", { name: "Verify Identity" }).click()

    await page.getByLabel("Create Password").fill("12345")
    await page.getByLabel("Confirm Password").fill("12345")
    await page.getByRole("button", { name: "Create Workspace Account" }).click()

    await expect(page.getByText("Password must be at least 6 characters")).toBeVisible()
  })

  test("user sees validation error when passwords do not match", async ({ page }) => {
    await page.route(/\/api\/auth\/verify-jira-pat/, (route) =>
      route.fulfill({
        status: 200,
        json: {
          display_name: "Budi Santoso",
          email: "budi.santoso@bri.co.id",
          username: "70001234"
        }
      })
    )
    await page.goto("/register")
    await page.getByLabel("Jira Personal Access Token").fill("valid-jira-pat")
    await page.getByRole("button", { name: "Verify Identity" }).click()

    await page.getByLabel("Create Password").fill("secure-password-1")
    await page.getByLabel("Confirm Password").fill("different-password-2")
    await page.getByRole("button", { name: "Create Workspace Account" }).click()

    await expect(page.getByText("Passwords do not match")).toBeVisible()
  })

  test("user sees conflict error banner when registering an already registered Jira identity (409)", async ({ page }) => {
    await page.route(/\/api\/auth\/verify-jira-pat/, (route) =>
      route.fulfill({
        status: 200,
        json: {
          display_name: "Budi Santoso",
          email: "budi.santoso@bri.co.id",
          username: "70001234"
        }
      })
    )
    await page.route(/\/api\/auth\/register-with-jira/, (route) =>
      route.fulfill({ status: 409, json: { error: "an account with this BRI email already exists" } })
    )

    await page.goto("/register")
    await page.getByLabel("Jira Personal Access Token").fill("valid-jira-pat")
    await page.getByRole("button", { name: "Verify Identity" }).click()

    await page.getByLabel("Create Password").fill("secure-password")
    await page.getByLabel("Confirm Password").fill("secure-password")
    await page.getByRole("button", { name: "Create Workspace Account" }).click()

    await expect(page.locator(".error-banner")).toBeVisible()
    await expect(page.locator(".error-banner")).toContainText("already exists")
  })

  test("user can complete registration with valid Jira PAT and lands on dashboard", async ({ page }) => {
    await page.route(/\/api\/auth\/verify-jira-pat/, (route) =>
      route.fulfill({
        status: 200,
        json: {
          display_name: "Budi Santoso",
          email: "budi.santoso@bri.co.id",
          username: "70001234"
        }
      })
    )
    await page.route(/\/api\/auth\/register-with-jira/, (route) =>
      route.fulfill({
        status: 201,
        json: {
          token: "mock-jwt-token",
          user: {
            id: "user-budi-id",
            email: "budi.santoso@bri.co.id",
            full_name: "Budi Santoso"
          }
        }
      })
    )

    await page.goto("/register")
    await page.getByLabel("Jira Personal Access Token").fill("valid-jira-pat")
    await page.getByRole("button", { name: "Verify Identity" }).click()

    await page.getByLabel("Create Password").fill("secure-password")
    await page.getByLabel("Confirm Password").fill("secure-password")
    await page.getByRole("button", { name: "Create Workspace Account" }).click()

    await expect(page).toHaveURL(/\/dashboard/)
    await expect(page.getByRole("heading", { level: 1 })).toContainText("Operations Overview")
  })
})
