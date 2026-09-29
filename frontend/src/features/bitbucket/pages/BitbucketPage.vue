<script setup lang="ts">
import { ref, onMounted } from "vue"
import PageLayout from "../../../components/layout/PageLayout.vue"
import { useSettings } from "../../settings/composables/useSettings"
import { useToast } from "../../../composables/useToast"
import { useBitbucket } from "../composables/useBitbucket"
import type { PushBranch, PullRequest, CreatePRPayload } from "../types"
import YourWorkList from "../components/YourWorkList.vue"
import PullRequestTable from "../components/PullRequestTable.vue"
import PullRequestReviewModal from "../components/PullRequestReviewModal.vue"
import CreatePrModal from "../components/CreatePrModal.vue"

const { secrets, fetchSettings } = useSettings()
const { showToast } = useToast()
const {
  pushes,
  pullRequests,
  selectedDiff,
  activeAIReview,
  pushFilter,
  prFilter,
  isLoadingPushes,
  isLoadingPRs,
  isLoadingDiff,
  isGeneratingAI,
  isSubmittingPR,
  isPostingComment,
  error,
  loadPushes,
  loadPullRequests,
  loadDiff,
  requestAIReview,
  submitPullRequest,
  sendPRComment,
  executeReviewAction,
} = useBitbucket()

const selectedPushForPr = ref<PushBranch | null>(null)
const selectedPrForReview = ref<PullRequest | null>(null)

onMounted(async () => {
  await Promise.all([
    fetchSettings(),
    loadPushes("all"),
    loadPullRequests("all"),
  ])
})

const handleFilterPushes = async (filter: string) => {
  await loadPushes(filter)
}

const handleFilterPRs = async (filter: string) => {
  await loadPullRequests(filter)
}

const handleOpenCreatePr = (push: PushBranch) => {
  selectedPushForPr.value = push
}

const handleCloseCreatePr = () => {
  selectedPushForPr.value = null
}

const handleSubmitPr = async (payload: CreatePRPayload) => {
  try {
    await submitPullRequest(payload)
    showToast(`Pull request created: ${payload.source_branch} → ${payload.target_branch}`)
    selectedPushForPr.value = null
  } catch (err) {
    showToast(err instanceof Error ? err.message : "Failed to create pull request")
  }
}

const handleOpenReview = async (pr: PullRequest) => {
  selectedPrForReview.value = pr
  activeAIReview.value = null
  await loadDiff(pr.id, pr.repo)
}

const handleCloseReview = () => {
  selectedPrForReview.value = null
}

const handleTriggerAI = async (pr: PullRequest) => {
  const result = await requestAIReview(pr.id, pr.repo)
  if (result) {
    showToast("AI review generated. Edit and send it as a comment.")
  } else {
    showToast("Failed to generate AI review")
  }
}

const handleSendComment = async (payload: { pr: PullRequest; comment: string }) => {
  try {
    await sendPRComment(payload.pr.id, payload.pr.repo, payload.comment)
    showToast("Review posted to the pull request as a comment.")
    selectedPrForReview.value = null
  } catch (err) {
    showToast(err instanceof Error ? err.message : "Failed to post comment")
  }
}

const handleReviewAction = async (payload: { pr: PullRequest; action: string }) => {
  try {
    await executeReviewAction(payload.pr.id, payload.pr.repo, payload.action)
    showToast(`Review marked as ${payload.action.replace("-", " ")}.`)
    selectedPrForReview.value = null
  } catch (err) {
    showToast(err instanceof Error ? err.message : "Failed to update status")
  }
}
</script>

<template>
  <PageLayout>
    <div class="content" data-testid="bitbucket-page">
      <div class="page-header">
        <p class="eyebrow">Code review workspace</p>
        <h1>Bitbucket</h1>
        <p class="page-copy">
          Your recent pushes, ready to turn into pull requests — plus open PRs to review with AI and post comments back.
        </p>
      </div>

      <div v-if="!secrets?.has_bitbucket_pat" class="warning-banner" data-testid="banner-no-pat">
        <span>Bitbucket Personal Access Token (PAT) is not configured yet.</span>
        <RouterLink to="/settings" class="banner-link">Configure in Settings →</RouterLink>
      </div>

      <div v-else-if="error" class="error-banner" data-testid="banner-bitbucket-error">
        <span>{{ error }}</span>
        <RouterLink to="/settings" class="banner-link">Update in Settings →</RouterLink>
      </div>

      <YourWorkList
        :pushes="pushes"
        :active-filter="pushFilter"
        :loading="isLoadingPushes"
        @filter="handleFilterPushes"
        @create-pr="handleOpenCreatePr"
      />

      <PullRequestTable
        :pull-requests="pullRequests"
        :active-filter="prFilter"
        :loading="isLoadingPRs"
        @filter="handleFilterPRs"
        @review="handleOpenReview"
      />

      <CreatePrModal
        :push="selectedPushForPr"
        :loading="isSubmittingPR"
        @close="handleCloseCreatePr"
        @submit="handleSubmitPr"
      />

      <PullRequestReviewModal
        :pr="selectedPrForReview"
        :diff="selectedDiff"
        :ai-review="activeAIReview"
        :loading-diff="isLoadingDiff"
        :loading-a-i="isGeneratingAI"
        :posting-comment="isPostingComment"
        @close="handleCloseReview"
        @trigger-ai="handleTriggerAI"
        @send-comment="handleSendComment"
        @review-action="handleReviewAction"
      />
    </div>
  </PageLayout>
</template>

<style scoped>
.content {
  max-width: 1200px;
  margin: 0 auto;
  padding: 34px 30px 48px;
}

.page-header {
  margin-bottom: 28px;
}

.eyebrow {
  margin: 0 0 9px;
  color: var(--muted);
  font-size: 11px;
  font-weight: 600;
  letter-spacing: 0.1em;
  text-transform: uppercase;
}

h1 {
  margin: 0;
  color: var(--text);
  font-size: 32px;
  font-weight: 600;
  letter-spacing: -0.03em;
}

.page-copy {
  margin: 8px 0 0;
  color: var(--muted);
  font-size: 13px;
  line-height: 1.5;
}

.warning-banner {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 18px;
  border: 1px solid rgba(187, 169, 132, 0.3);
  border-radius: 9px;
  background: rgba(187, 169, 132, 0.1);
  color: var(--warning);
  font-size: 13px;
  margin-bottom: 24px;
}

.error-banner {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 18px;
  border: 1px solid rgba(239, 68, 68, 0.3);
  border-radius: 9px;
  background: rgba(239, 68, 68, 0.08);
  color: #ef4444;
  font-size: 13px;
  margin-bottom: 24px;
}

.banner-link {
  color: var(--text);
  font-weight: 600;
  text-decoration: underline;
}
</style>
