import { test, expect } from "../../../fixtures/authenticated.fixture"
import { captureEvidence } from "../../../helpers/evidence"
import { routeStandardPushes, routeStandardPRs } from "../../../helpers/route-mock"

test.describe("Bitbucket BRI - Filters", () => {
  test.beforeEach(async ({ page }) => {
    await routeStandardPushes(page)
    await routeStandardPRs(page)
  })

  test(
    "BB-PUSH-001 engineer can filter pushed branches by ready status",
    {
      tag: ["@bitbucket", "@positive", "@filter"]
    },
    async ({ page, bitbucketPage }) => {
      await bitbucketPage.navigateTo()
      await bitbucketPage.pushFilterButton("ready").click()
      await expect(bitbucketPage.pushCards).toHaveCount(2)
      await captureEvidence(page, "bitbucket", "02_pushes_filter_ready.png")
    }
  )

  test(
    "BB-PUSH-002 engineer can filter pushed branches by stale status",
    {
      tag: ["@bitbucket", "@positive", "@filter"]
    },
    async ({ bitbucketPage }) => {
      await bitbucketPage.navigateTo()
      await bitbucketPage.pushFilterButton("stale").click()
      await expect(bitbucketPage.pushCards).toHaveCount(1)
    }
  )

  test(
    "BB-PUSH-003 engineer can restore all pushed branches after applying a filter",
    {
      tag: ["@bitbucket", "@positive", "@edge"]
    },
    async ({ bitbucketPage }) => {
      await bitbucketPage.navigateTo()
      await bitbucketPage.pushFilterButton("ready").click()
      await expect(bitbucketPage.pushCards).toHaveCount(2)
      await bitbucketPage.pushFilterButton("all").click()
      await expect(bitbucketPage.pushCards).toHaveCount(3)
    }
  )

  test(
    "BB-PR-001 engineer can filter pull requests by AI flagged",
    {
      tag: ["@bitbucket", "@positive", "@filter"]
    },
    async ({ page, bitbucketPage }) => {
      await bitbucketPage.navigateTo()
      await bitbucketPage.prFilterButton("ai").click()
      await expect(bitbucketPage.prTableRows).toHaveCount(3)
      await captureEvidence(page, "bitbucket", "03_prs_filter_ai_flagged.png")
    }
  )

  test(
    "BB-PR-002 engineer can filter pull requests by assigned to me",
    {
      tag: ["@bitbucket", "@positive", "@filter"]
    },
    async ({ bitbucketPage }) => {
      await bitbucketPage.navigateTo()
      await bitbucketPage.prFilterButton("mine").click()
      await expect(bitbucketPage.prTableRows).toHaveCount(2)
    }
  )

  test(
    "BB-PR-003 engineer can restore all pull requests after applying a filter",
    {
      tag: ["@bitbucket", "@positive", "@edge"]
    },
    async ({ bitbucketPage }) => {
      await bitbucketPage.navigateTo()
      await bitbucketPage.prFilterButton("mine").click()
      await expect(bitbucketPage.prTableRows).toHaveCount(2)
      await bitbucketPage.prFilterButton("all").click()
      await expect(bitbucketPage.prTableRows).toHaveCount(4)
    }
  )
})
