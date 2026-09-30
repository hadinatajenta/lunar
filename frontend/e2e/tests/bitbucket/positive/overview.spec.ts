import { test, expect } from "../../../fixtures/authenticated.fixture"
import { captureEvidence } from "../../../helpers/evidence"
import { routeStandardPushes, routeStandardPRs } from "../../../helpers/route-mock"

test.describe("Bitbucket BRI - Overview", () => {
  test(
    "BB-VIEW-001 engineer can view pushed branches and open pull requests on load",
    {
      tag: ["@bitbucket", "@positive", "@overview", "@p0"]
    },
    async ({ page, bitbucketPage }) => {
      await routeStandardPushes(page)
      await routeStandardPRs(page)

      await bitbucketPage.navigateTo()
      await expect(bitbucketPage.pushCards).toHaveCount(3)
      await expect(bitbucketPage.prTableRows).toHaveCount(4)
      await captureEvidence(page, "bitbucket", "01_bitbucket_overview.png", true)
    }
  )
})
