import { test, expect } from "../../../fixtures/authenticated.fixture"
import { captureEvidence } from "../../../helpers/evidence"
import { routeStandardPushes, routeStandardPRs, routeSecrets } from "../../../helpers/route-mock"

test.describe("Bitbucket BRI - Errors & Authorization", () => {
  test(
    "BB-AUTH-001 engineer cannot access Bitbucket data when PAT is invalid or expired (401 Unauthorized)",
    {
      tag: ["@bitbucket", "@negative", "@authorization", "@p0"]
    },
    async ({ page, bitbucketPage }) => {
      await routeSecrets(page, { hasBitbucketPat: true })
      await page.route(/\/api\/bitbucket\/pushes/, (route) =>
        route.fulfill({ status: 401, json: { error: "unauthorized access: invalid or expired Bitbucket PAT" } })
      )
      await page.route(/\/api\/bitbucket\/prs/, (route) =>
        route.fulfill({ status: 401, json: { error: "unauthorized access: invalid or expired Bitbucket PAT" } })
      )

      await bitbucketPage.navigateTo()
      await bitbucketPage.assertErrorBannerContains("invalid or expired Bitbucket PAT")
      await expect(bitbucketPage.errorBanner.locator("a")).toContainText("Update in Settings →")
      await captureEvidence(page, "bitbucket", "11_bitbucket_invalid_pat_negative.png")
    }
  )

  test(
    "BB-NET-001 engineer sees VPN guidance when server cannot reach Bitbucket due to network failure (502)",
    {
      tag: ["@bitbucket", "@negative", "@network"]
    },
    async ({ page, bitbucketPage }) => {
      await routeSecrets(page, { hasBitbucketPat: true })
      const vpnError =
        "failed to contact Bitbucket Server (https://bitbucket.bri.co.id): please verify your BRI VPN connection"
      await page.route(/\/api\/bitbucket\/pushes/, (route) =>
        route.fulfill({ status: 502, json: { error: vpnError } })
      )
      await page.route(/\/api\/bitbucket\/prs/, (route) =>
        route.fulfill({ status: 502, json: { error: vpnError } })
      )

      await bitbucketPage.navigateTo()
      await bitbucketPage.assertErrorBannerContains("verify your BRI VPN connection")
      await captureEvidence(page, "bitbucket", "12_bitbucket_vpn_disconnect_negative.png")
    }
  )

  test(
    "BB-AUTH-002 engineer cannot access Bitbucket data when PAT lacks repository permissions (403 Forbidden)",
    {
      tag: ["@bitbucket", "@negative", "@authorization"]
    },
    async ({ page, bitbucketPage }) => {
      await routeSecrets(page, { hasBitbucketPat: true })
      await page.route(/\/api\/bitbucket\/pushes/, (route) =>
        route.fulfill({
          status: 403,
          json: { error: "access forbidden: your PAT does not have permission for this repository" }
        })
      )
      await page.route(/\/api\/bitbucket\/prs/, (route) =>
        route.fulfill({
          status: 403,
          json: { error: "access forbidden: your PAT does not have permission for this repository" }
        })
      )

      await bitbucketPage.navigateTo()
      await bitbucketPage.assertErrorBannerContains("access forbidden")
      await captureEvidence(page, "bitbucket", "13_bitbucket_access_forbidden_negative.png")
    }
  )

  test(
    "BB-AUTH-003 engineer sees configuration warning when Bitbucket PAT has not been set up yet",
    {
      tag: ["@bitbucket", "@negative", "@edge"]
    },
    async ({ page, bitbucketPage }) => {
      await routeSecrets(page, { hasBitbucketPat: false, hasJiraPat: false, hasConfluencePat: false })
      await page.route(/\/api\/bitbucket\/pushes/, (route) => route.fulfill({ json: [] }))
      await page.route(/\/api\/bitbucket\/prs/, (route) => route.fulfill({ json: [] }))

      await bitbucketPage.navigateTo()
      await expect(bitbucketPage.noPatWarningBanner).toBeVisible()
      await expect(bitbucketPage.noPatWarningBanner).toContainText(
        "Bitbucket Personal Access Token (PAT) is not configured yet"
      )
      await captureEvidence(page, "bitbucket", "14_bitbucket_no_pat_configured.png")
    }
  )

  test(
    "BB-REV-004 engineer receives clear error toast when AI review request fails with upstream 401 Unauthorized",
    {
      tag: ["@bitbucket", "@negative", "@review"]
    },
    async ({ page, bitbucketPage }) => {
      await routeStandardPushes(page)
      await routeStandardPRs(page)
      await routeSecrets(page, { hasAiKeys: true })
      await page.route(/\/api\/bitbucket\/prs\/184\/ai-review/, (route) =>
        route.fulfill({
          status: 401,
          json: { error: "failed to generate review: invalid or expired deepseek API key" }
        })
      )

      await bitbucketPage.navigateTo()
      await bitbucketPage.reviewButton(184).click()
      await expect(bitbucketPage.reviewModal).toBeVisible()

      await bitbucketPage.generateAiReviewButton.click()
      await expect(bitbucketPage.globalToast).toBeVisible()
      await expect(bitbucketPage.globalToast).toContainText("invalid or expired deepseek API key")
    }
  )

  test(
    "BB-REV-005 engineer receives clear error toast when AI review request fails with upstream 502 Bad Gateway",
    {
      tag: ["@bitbucket", "@negative", "@review"]
    },
    async ({ page, bitbucketPage }) => {
      await routeStandardPushes(page)
      await routeStandardPRs(page)
      await routeSecrets(page, { hasAiKeys: true })
      await page.route(/\/api\/bitbucket\/prs\/184\/ai-review/, (route) =>
        route.fulfill({
          status: 502,
          json: { error: "failed to contact AI provider (deepseek): connection timeout" }
        })
      )

      await bitbucketPage.navigateTo()
      await bitbucketPage.reviewButton(184).click()
      await expect(bitbucketPage.reviewModal).toBeVisible()

      await bitbucketPage.generateAiReviewButton.click()
      await expect(bitbucketPage.globalToast).toBeVisible()
      await expect(bitbucketPage.globalToast).toContainText("connection timeout")
    }
  )
})
