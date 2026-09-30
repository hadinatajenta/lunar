import { test, expect } from "../../../fixtures/authenticated.fixture"
import { captureEvidence } from "../../../helpers/evidence"
import { routeSecrets } from "../../../helpers/route-mock"

test.describe("Bitbucket BRI - Empty States", () => {
  test(
    "BB-VIEW-002 engineer can see empty state message when no pushes or pull requests exist",
    {
      tag: ["@bitbucket", "@edge", "@empty-state"]
    },
    async ({ page, bitbucketPage }) => {
      await routeSecrets(page, { hasBitbucketPat: true })
      await page.route(/\/api\/bitbucket\/pushes/, (route) => route.fulfill({ json: [] }))
      await page.route(/\/api\/bitbucket\/prs/, (route) => route.fulfill({ json: [] }))

      await bitbucketPage.navigateTo()
      await expect(bitbucketPage.emptyStates.first()).toContainText("No pushed branches match this filter.")
      await expect(bitbucketPage.emptyStates.nth(1)).toContainText("No pull requests match this filter.")
      await captureEvidence(page, "bitbucket", "10_bitbucket_empty_state.png")
    }
  )
})
