import { test, expect } from "../../../fixtures/authenticated.fixture"
import { captureEvidence } from "../../../helpers/evidence"
import { routeStandardPushes, routeStandardPRs, routeSecrets } from "../../../helpers/route-mock"

test.describe("Bitbucket BRI - PR Review Positive Flows", () => {
  test.beforeEach(async ({ page }) => {
    await routeStandardPushes(page)
    await routeStandardPRs(page)
  })

  test(
    "BB-REV-001 engineer can see model selector in PR review modal populated with configured models",
    {
      tag: ["@bitbucket", "@positive", "@review"]
    },
    async ({ page, bitbucketPage }) => {
      await routeSecrets(page, { hasAiKeys: true, configuredAiProviders: ["deepseek"] })

      await bitbucketPage.navigateTo()
      await bitbucketPage.reviewButton(184).click()
      await expect(bitbucketPage.reviewModal).toBeVisible()

      await expect(bitbucketPage.prReviewModelSelect).toBeVisible()
      await expect(bitbucketPage.prReviewModelSelect).toContainText("DeepSeek-V4 Pro (Thinking)")
      await expect(bitbucketPage.prReviewModelSelect).toContainText("DeepSeek Flash")

      await bitbucketPage.prReviewModelSelect.selectOption("DeepSeek Flash")
      await expect(bitbucketPage.prReviewModelSelect).toHaveValue("DeepSeek Flash")
    }
  )

  test(
    "BB-REV-002 engineer sees warning when no AI provider is configured in PR review modal",
    {
      tag: ["@bitbucket", "@positive", "@edge"]
    },
    async ({ page, bitbucketPage }) => {
      await routeSecrets(page, { hasAiKeys: false, configuredAiProviders: [] })

      await bitbucketPage.navigateTo()
      await bitbucketPage.reviewButton(184).click()
      await expect(bitbucketPage.reviewModal).toBeVisible()

      await expect(bitbucketPage.prReviewNoAiWarning).toBeVisible()
      await expect(bitbucketPage.prReviewNoAiWarning).toContainText("AI Key Required for Automated Code Review")
      await expect(bitbucketPage.generateAiReviewButton).toBeDisabled()
    }
  )

  test(
    "BB-REV-003 engineer can open PR review modal and inspect diff then generate AI review",
    {
      tag: ["@bitbucket", "@positive", "@review", "@p0"]
    },
    async ({ page, bitbucketPage }) => {
      await routeSecrets(page, { hasAiKeys: true, configuredAiProviders: ["deepseek"] })

      await bitbucketPage.navigateTo()
      await bitbucketPage.reviewButton(184).click()
      await expect(bitbucketPage.reviewModal).toBeVisible()
      await expect(bitbucketPage.reviewModalTitle).toContainText("feat(auth): LUN-421")

      await expect(bitbucketPage.diffContainer).toBeVisible()
      await expect(bitbucketPage.diffContainer).toContainText("src/auth/session.ts")
      await captureEvidence(page, "bitbucket", "06_pr_review_modal_empty.png")

      await bitbucketPage.generateAiReviewButton.click()
      await expect(bitbucketPage.aiReviewFindings).toBeVisible()
      await expect(bitbucketPage.aiReviewFindings).toContainText("Automated analysis completed")
      await expect(bitbucketPage.aiReviewFindings).toContainText("session token lifecycle complies with security standards")

      await expect(bitbucketPage.reviewCommentTextarea).toHaveValue(
        "Lunar Copilot review: Automated analysis completed. Verified changed files and scope against standard lint and test coverage guidelines."
      )
      await captureEvidence(page, "bitbucket", "07_pr_review_ai_generated.png")
    }
  )

  test(
    "BB-REV-006 engineer can approve a pull request and see review status toast",
    {
      tag: ["@bitbucket", "@positive", "@review"]
    },
    async ({ page, bitbucketPage }) => {
      await routeSecrets(page, { hasAiKeys: true })

      await bitbucketPage.navigateTo()
      await bitbucketPage.reviewButton(188).click()
      await expect(bitbucketPage.reviewModal).toBeVisible()

      await bitbucketPage.approvePrButton.click()
      await expect(bitbucketPage.globalToast).toBeVisible()
      await expect(bitbucketPage.globalToast).toContainText("Pull request approved")
      await captureEvidence(page, "bitbucket", "08_pr_review_status_action.png")
    }
  )

  test(
    "BB-REV-007 engineer can post a comment to a pull request and see confirmation toast",
    {
      tag: ["@bitbucket", "@positive", "@review"]
    },
    async ({ page, bitbucketPage }) => {
      await routeSecrets(page, { hasAiKeys: true })

      await bitbucketPage.navigateTo()
      await bitbucketPage.reviewButton(191).click()
      await expect(bitbucketPage.reviewModal).toBeVisible()

      await bitbucketPage.reviewCommentTextarea.fill("Verified SQLite FTS5 tokenization looks optimal.")
      await bitbucketPage.postCommentButton.click()
      await expect(bitbucketPage.globalToast).toBeVisible()
      await expect(bitbucketPage.globalToast).toContainText("Review comment posted")
      await captureEvidence(page, "bitbucket", "09_pr_review_comment_posted.png")
    }
  )
})
