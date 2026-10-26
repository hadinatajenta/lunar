import { test, expect } from "../../../fixtures/authenticated.fixture"
import { mockAIReview } from "../../../data/bitbucket.data"
import { captureEvidence } from "../../../helpers/evidence"
import { routeStandardPushes, routeStandardPRs, routeSecrets, routeSystemConfig } from "../../../helpers/route-mock"

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

      await expect(bitbucketPage.prReviewModelTrigger).toBeVisible()
      await expect(bitbucketPage.prReviewModelTrigger).toContainText("DeepSeek-V4 Pro (Thinking)")

      await bitbucketPage.openReviewModelPicker()
      await expect(bitbucketPage.prReviewModelOption("DeepSeek-V4 Pro (Thinking)")).toBeVisible()
      await expect(bitbucketPage.prReviewModelOption("DeepSeek Flash")).toBeVisible()

      await bitbucketPage.prReviewModelOption("DeepSeek Flash").click()
      await expect(bitbucketPage.prReviewModelTrigger).toContainText("DeepSeek Flash")
    }
  )

  test(
    "BB-REV-002 engineer sees warning when no AI provider is configured in PR review modal",
    {
      tag: ["@bitbucket", "@positive", "@edge"]
    },
    async ({ page, bitbucketPage }) => {
      await routeSecrets(page, { hasAiKeys: false, configuredAiProviders: [] })
      await routeSystemConfig(page, [])

      await bitbucketPage.navigateTo()
      await bitbucketPage.reviewButton(184).click()
      await expect(bitbucketPage.reviewModal).toBeVisible()

      await expect(bitbucketPage.noAiKeysBanner).toBeVisible()
      await expect(bitbucketPage.noAiKeysBanner).toContainText("No AI provider is configured yet")
      await expect(bitbucketPage.triggerAiReviewButton).toBeDisabled()
    }
  )

  test(
    "BB-REV-003 engineer can open PR review modal and inspect diff then generate AI review",
    {
      tag: ["@bitbucket", "@positive", "@review", "@p0"]
    },
    async ({ page, bitbucketPage }) => {
      const [blockerFinding, assumptionFinding, nitpickFinding] = mockAIReview.findings

      await routeSecrets(page, { hasAiKeys: true, configuredAiProviders: ["deepseek"] })

      await bitbucketPage.navigateTo()
      await bitbucketPage.reviewButton(184).click()
      await expect(bitbucketPage.reviewModalOverlay).toBeVisible()
      await expect(bitbucketPage.reviewModal).toBeVisible()
      await expect(bitbucketPage.closeReviewButton).toBeVisible()
      await expect(bitbucketPage.reviewModalId).toContainText("#184")
      await expect(bitbucketPage.reviewModalTitle).toContainText("feat(auth): LUN-421")
      await expect(bitbucketPage.reviewModalRepo).toContainText("BRI/mms-frontend")
      await expect(bitbucketPage.reviewModalBranch).toContainText("feat/auth-refresh")
      await expect(bitbucketPage.reviewModalBranch).toContainText("main")
      await expect(bitbucketPage.reviewModalAuthor).toContainText("Lunar Developer")
      await expect(bitbucketPage.reviewModalFiles).toContainText("124")

      await expect(bitbucketPage.toggleDiffButton).toBeHidden()
      await expect(bitbucketPage.diffViewer).toBeVisible()
      await expect(bitbucketPage.diffFileList).toBeVisible()
      await expect(bitbucketPage.diffFile(0)).toContainText("src/auth/session.ts")
      await expect(bitbucketPage.diffFileToggle(0)).toBeVisible()
      await captureEvidence(page, "bitbucket", "06_pr_review_modal_empty.png")

      await expect(bitbucketPage.reviewFindingsTab).toBeVisible()
      await expect(bitbucketPage.aiEmptyState).toBeVisible()

      await bitbucketPage.triggerAiReviewButton.click()
      await expect(bitbucketPage.aiFindings).toBeVisible()
      await expect(bitbucketPage.aiLoading).toBeHidden()
      await expect(bitbucketPage.aiEmptyState).toBeHidden()
      await expect(bitbucketPage.aiSummary).toContainText(mockAIReview.summary)
      await expect(bitbucketPage.aiRecommendation).toContainText("Request changes")

      await expect(bitbucketPage.aiGroupBlocker).toBeVisible()
      await expect(bitbucketPage.aiGroupBlocker).toContainText(blockerFinding.title)
      await expect(bitbucketPage.aiGroupBlocker).toContainText(blockerFinding.file)
      await expect(bitbucketPage.aiGroupSuggestion).toBeVisible()
      await expect(bitbucketPage.aiGroupSuggestion).toContainText(assumptionFinding.title)
      await expect(bitbucketPage.aiGroupNitpick).toBeVisible()
      await expect(bitbucketPage.aiGroupNitpick).toContainText(nitpickFinding.title)
      await expect(bitbucketPage.aiFinding(0)).toContainText(blockerFinding.title)

      await expect(bitbucketPage.aiAssumptions).toContainText(mockAIReview.assumptions[0])
      await expect(bitbucketPage.aiStrengths).toContainText(mockAIReview.strengths[0])
      await expect(bitbucketPage.aiTruncatedNotice).toBeHidden()

      await bitbucketPage.reviewCommentTab.click()
      await expect(bitbucketPage.reviewCommentInput).toHaveValue(mockAIReview.generated_comment)
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

      await bitbucketPage.approveButton().click()
      await expect(bitbucketPage.globalToast).toBeVisible()
      await expect(bitbucketPage.globalToast).toContainText("Review marked as approve")
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

      await bitbucketPage.reviewCommentTab.click()
      await bitbucketPage.reviewCommentInput.fill("Verified SQLite FTS5 tokenization looks optimal.")
      await bitbucketPage.sendCommentButton.click()
      await expect(bitbucketPage.globalToast).toBeVisible()
      await expect(bitbucketPage.globalToast).toContainText("Review posted to the pull request as a comment")
      await captureEvidence(page, "bitbucket", "09_pr_review_comment_posted.png")
    }
  )

  test(
    "BB-REV-008 engineer can preview the generated review comment and post it",
    {
      tag: ["@bitbucket", "@positive", "@review"]
    },
    async ({ page, bitbucketPage }) => {
      await routeSecrets(page, { hasAiKeys: true, configuredAiProviders: ["deepseek"] })

      await bitbucketPage.navigateTo()
      await bitbucketPage.reviewButton(184).click()
      await expect(bitbucketPage.reviewModal).toBeVisible()

      await bitbucketPage.triggerAiReviewButton.click()
      await expect(bitbucketPage.aiFindings).toBeVisible()

      await bitbucketPage.reviewCommentTab.click()
      await expect(bitbucketPage.commentPreview).toBeVisible()
      await expect(bitbucketPage.commentPreview).toContainText("Code review: feat(auth): LUN-421")
      await expect(bitbucketPage.commentPreview).toContainText("Verdict:")
      await expect(bitbucketPage.commentPreview).toContainText("Request changes")

      await bitbucketPage.toggleCommentPreviewButton.click()
      await expect(bitbucketPage.commentPreview).toBeHidden()
      await expect(bitbucketPage.reviewCommentInput).toHaveValue(mockAIReview.generated_comment)

      await bitbucketPage.sendCommentButton.click()
      await expect(bitbucketPage.globalToast).toBeVisible()
      await expect(bitbucketPage.globalToast).toContainText("Review posted to the pull request as a comment")
      await captureEvidence(page, "bitbucket", "15_pr_review_generated_comment_posted.png")
    }
  )

  test(
    "BB-REV-009 engineer confirms the decline dialog before the review status is posted",
    {
      tag: ["@bitbucket", "@positive", "@review"]
    },
    async ({ page, bitbucketPage }) => {
      await routeSecrets(page, { hasAiKeys: true })

      await bitbucketPage.navigateTo()
      await bitbucketPage.reviewButton(188).click()
      await expect(bitbucketPage.reviewModal).toBeVisible()

      await bitbucketPage.declineButton().click()
      await expect(bitbucketPage.confirmAcceptButton).toBeVisible()
      await expect(bitbucketPage.globalToast).toBeHidden()

      await bitbucketPage.confirmAcceptButton.click()
      await expect(bitbucketPage.globalToast).toBeVisible()
      await expect(bitbucketPage.globalToast).toContainText("Review marked as decline")
      await captureEvidence(page, "bitbucket", "16_pr_review_decline_confirmed.png")
    }
  )
})
