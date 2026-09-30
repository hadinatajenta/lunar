import { test, expect } from "../../../fixtures/unauthenticated.fixture"
import { captureEvidence } from "../../../helpers/evidence"

test.describe("Authentication - Login Validation", () => {
  test(
    "AUTH-LOGIN-002 user cannot submit login form when credentials are empty",
    {
      tag: ["@auth", "@negative", "@validation"]
    },
    async ({ page, authPage }) => {
      await authPage.navigateTo()
      await authPage.submitWithEmptyFields()
      await expect(authPage.validationError).toBeVisible()
      await captureEvidence(page, "auth", "02_login_validation_error.png")
    }
  )
})
