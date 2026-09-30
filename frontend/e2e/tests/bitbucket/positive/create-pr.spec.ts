import { test, expect } from "../../../fixtures/authenticated.fixture"
import { captureEvidence } from "../../../helpers/evidence"
import { routeStandardPushes, routeStandardPRs } from "../../../helpers/route-mock"

test.describe("Bitbucket BRI - Create Pull Request", () => {
  test(
    "BB-PR-004 engineer can open create PR modal and submit a new pull request with toast confirmation",
    {
      tag: ["@bitbucket", "@positive", "@create-pr", "@p0"]
    },
    async ({ page, bitbucketPage }) => {
      await routeStandardPushes(page)
      await routeStandardPRs(page)

      await bitbucketPage.navigateTo()
      await bitbucketPage.createPrButton("push-1").click()
      await expect(bitbucketPage.createPrModal).toBeVisible()
      await expect(bitbucketPage.createPrSourceBranch).toContainText("feat/auth-refresh")
      await captureEvidence(page, "bitbucket", "04_create_pr_modal.png")

      await bitbucketPage.submitPrButton.click()
      await expect(bitbucketPage.globalToast).toBeVisible()
      await expect(bitbucketPage.globalToast).toContainText("Pull request created: feat/auth-refresh → main")
      await captureEvidence(page, "bitbucket", "05_create_pr_toast.png")
      await expect(bitbucketPage.createPrModal).not.toBeVisible()
    }
  )
})
