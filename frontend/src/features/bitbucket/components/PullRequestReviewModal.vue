<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from "vue"
import type { PullRequest, PRDiff, AICodeReview, ReviewFinding, ReviewSeverity } from "../types"
import { useCopilot } from "../../copilot/composables/useCopilot"
import { getDefaultModel, isModelConfigured } from "../../copilot/utils/models"
import { renderMarkdown } from "../../../lib/markdown"
import { useFocusTrap } from "../../../composables/useFocusTrap"
import ConfirmDialog from "../../../components/ui/ConfirmDialog.vue"
import ModelPicker from "../../copilot/components/ModelPicker.vue"
import PullRequestReviewDiff from "./PullRequestReviewDiff.vue"
import ReviewFindingGroup from "./ReviewFindingGroup.vue"

const props = defineProps<{
  pr: PullRequest | null
  diff: PRDiff | null
  aiReview: AICodeReview | null
  loadingDiff: boolean
  loadingAI: boolean
  postingComment: boolean
  reviewError: string | null
}>()

const emit = defineEmits<{
  (e: "close"): void
  (e: "trigger-ai", payload: { pr: PullRequest; model: string }): void
  (e: "send-comment", payload: { pr: PullRequest; comment: string }): void
  (e: "review-action", payload: { pr: PullRequest; action: string }): void
  (e: "retry-ai", payload: { pr: PullRequest; model: string }): void
}>()

const SEVERITY_GROUPS: { severity: ReviewSeverity; title: string; hint: string; testId: string }[] = [
  { severity: "blocker", title: "Blocker", hint: "Must be fixed before merging.", testId: "ai-group-blocker" },
  { severity: "suggestion", title: "Suggestion", hint: "Should be fixed.", testId: "ai-group-suggestion" },
  { severity: "nitpick", title: "Nitpick", hint: "Optional, or a style preference.", testId: "ai-group-nitpick" }
]

const { configuredModels, configuredProviders, hasConfiguredAI, refreshConfiguredProviders } = useCopilot()

const modalElement = ref<HTMLElement | null>(null)
const reviewModel = ref(getDefaultModel([]))
const commentText = ref("")
const activeTab = ref<"findings" | "comment">("findings")
const isPreviewingComment = ref(false)
const isDiffVisible = ref(true)
const isConfirmingDecline = ref(false)

const isOpen = computed(() => props.pr !== null)

useFocusTrap(modalElement, isOpen)

const findings = computed<ReviewFinding[]>(() => props.aiReview?.findings ?? [])

const groupFindings = (severity: ReviewSeverity): ReviewFinding[] =>
  findings.value.filter((finding) => finding.severity === severity)

const groupOffset = (severity: ReviewSeverity): number => {
  const order: ReviewSeverity[] = ["blocker", "suggestion", "nitpick"]
  return order
    .slice(0, order.indexOf(severity))
    .reduce((total, current) => total + groupFindings(current).length, 0)
}

const isTruncated = computed(() => props.aiReview?.truncated === true)

const verdictLabel = computed(() => {
  if (props.aiReview === null) return ""
  return props.aiReview.recommendation === "request_changes" ? "Request changes" : "Approve"
})

const changedFilesCount = computed(() => props.diff?.files.length ?? props.pr?.files_count ?? 0)

const linesAdded = computed(() => props.diff?.total_added ?? props.pr?.lines_added ?? 0)

const linesDeleted = computed(() => props.diff?.total_deleted ?? props.pr?.lines_deleted ?? 0)

const commentPreviewHtml = computed(() => renderMarkdown(commentText.value))

const commentCharacterCount = computed(() => commentText.value.trim().length)

const canSendComment = computed(() => commentCharacterCount.value > 0 && !props.postingComment)

const isReviewing = computed(() => props.loadingAI)

const showFindingsEmptyState = computed(
  () => !props.loadingAI && props.reviewError === null && props.aiReview === null
)

watch(
  () => props.pr,
  (pullRequest) => {
    if (pullRequest === null) return
    commentText.value = ""
    activeTab.value = "findings"
    isPreviewingComment.value = false
    isDiffVisible.value = true
    isConfirmingDecline.value = false
  },
  { immediate: true }
)

watch(
  () => props.aiReview,
  (review) => {
    if (review === null) return
    commentText.value = review.generated_comment
    isPreviewingComment.value = true
  }
)

watch(
  configuredModels,
  (models) => {
    if (models.length === 0) return
    if (!isModelConfigured(reviewModel.value, configuredProviders.value)) {
      reviewModel.value = getDefaultModel(configuredProviders.value)
    }
  },
  { immediate: true }
)

const requestReview = (): void => {
  if (props.pr === null || !hasConfiguredAI.value) return
  emit("trigger-ai", { pr: props.pr, model: reviewModel.value })
}

const requestRetry = (): void => {
  if (props.pr === null) return
  emit("retry-ai", { pr: props.pr, model: reviewModel.value })
}

const sendComment = (): void => {
  if (props.pr === null || !canSendComment.value) return
  emit("send-comment", { pr: props.pr, comment: commentText.value.trim() })
}

const runAction = (action: string): void => {
  if (props.pr === null) return
  if (action === "decline") {
    isConfirmingDecline.value = true
    return
  }
  emit("review-action", { pr: props.pr, action })
}

const confirmDecline = (): void => {
  if (props.pr === null) return
  isConfirmingDecline.value = false
  emit("review-action", { pr: props.pr, action: "decline" })
}

const openCommentTab = (): void => {
  activeTab.value = "comment"
}

const toggleCommentPreview = (): void => {
  isPreviewingComment.value = !isPreviewingComment.value
}

const handleKeydown = (event: KeyboardEvent): void => {
  if (event.key !== "Escape") return
  if (isConfirmingDecline.value) {
    isConfirmingDecline.value = false
    return
  }
  emit("close")
}

onMounted(async () => {
  window.addEventListener("keydown", handleKeydown)
  await refreshConfiguredProviders()
})

onUnmounted(() => {
  window.removeEventListener("keydown", handleKeydown)
})
</script>

<template>
  <div
    v-if="pr"
    class="review-overlay"
    data-testid="review-modal-overlay"
    @click.self="emit('close')"
  >
    <div
      ref="modalElement"
      class="review-dialog"
      role="dialog"
      aria-modal="true"
      aria-labelledby="review-modal-title"
      data-testid="review-modal"
    >
      <header class="review-header">
        <div class="review-heading">
          <div class="review-eyebrow" data-testid="review-modal-id">Pull request {{ pr.id }}</div>
          <h3 id="review-modal-title" class="review-title" data-testid="review-modal-title">{{ pr.title }}</h3>
          <div class="review-meta">
            <span class="meta-item">Repo <strong data-testid="review-modal-repo">{{ pr.repo }}</strong></span>
            <span class="meta-item">Author <strong data-testid="review-modal-author">{{ pr.author }}</strong></span>
            <span class="meta-item">
              Branch
              <strong data-testid="review-modal-branch">{{ pr.source_branch }} → {{ pr.target_branch }}</strong>
            </span>
            <span class="meta-item tabular" data-testid="review-modal-files">
              {{ changedFilesCount }} files · +{{ linesAdded }} −{{ linesDeleted }}
            </span>
          </div>
        </div>

        <button
          class="icon-button"
          type="button"
          aria-label="Close review"
          data-testid="btn-close-review"
          @click="emit('close')"
        >
          <svg viewBox="0 0 24 24" aria-hidden="true">
            <path d="M6 6l12 12"></path>
            <path d="M18 6 6 18"></path>
          </svg>
        </button>
      </header>

      <div class="review-body">
        <section class="diff-pane" :class="{ 'is-hidden': !isDiffVisible }" aria-label="Changed files">
          <div class="pane-head">
            <h4 class="pane-title">Changed files</h4>
            <button
              class="ghost-button diff-toggle"
              type="button"
              data-testid="btn-toggle-diff"
              :aria-expanded="isDiffVisible"
              @click="isDiffVisible = !isDiffVisible"
            >
              {{ isDiffVisible ? "Hide diff" : "Show diff" }}
            </button>
          </div>
          <PullRequestReviewDiff :diff="diff" :loading="loadingDiff" />
        </section>

        <section class="review-pane" aria-label="Review">
          <div class="tabs" role="tablist" aria-label="Review sections">
            <button
              class="tab"
              type="button"
              id="review-findings-tab"
              role="tab"
              :aria-selected="activeTab === 'findings'"
              aria-controls="review-findings-panel"
              data-testid="tab-review-findings"
              @click="activeTab = 'findings'"
            >
              Review findings
              <span v-if="findings.length > 0" class="tab-count">{{ findings.length }}</span>
            </button>
            <button
              class="tab"
              type="button"
              id="review-comment-tab"
              role="tab"
              :aria-selected="activeTab === 'comment'"
              aria-controls="review-comment-panel"
              data-testid="tab-review-comment"
              @click="openCommentTab"
            >
              Comment
              <span v-if="commentCharacterCount > 0" class="tab-dot" aria-hidden="true"></span>
            </button>
          </div>

          <div
            v-show="activeTab === 'findings'"
            id="review-findings-panel"
            class="tab-panel"
            role="tabpanel"
            aria-labelledby="review-findings-tab"
            data-testid="ai-panel"
          >
            <div v-if="!hasConfiguredAI" class="notice is-warning" data-testid="banner-no-ai-keys">
              <span>No AI provider is configured yet. Add an API key in Settings to use the AI review.</span>
              <RouterLink to="/settings" class="notice-link">Configure in Settings →</RouterLink>
            </div>

            <div v-else-if="loadingAI" class="notice" data-testid="ai-loading">
              <span class="spinner" aria-hidden="true"></span>
              <span>Reading the diff, the pull request description, and the test files it changes…</span>
            </div>

            <div v-else-if="reviewError" class="notice is-danger" data-testid="ai-error">
              <span>{{ reviewError }}</span>
              <button class="ghost-button" type="button" data-testid="btn-retry-ai-review" @click="requestRetry">
                Try again
              </button>
            </div>

            <div v-else-if="showFindingsEmptyState" class="notice" data-testid="ai-empty">
              Choose a model, then run <strong>Review with AI</strong>. Findings are grouped by severity and the
              comment below is filled in from them, ready to edit and post.
            </div>

            <template v-else-if="aiReview">
              <div class="verdict">
                <span
                  class="verdict-pill"
                  :class="aiReview.recommendation === 'request_changes' ? 'is-changes' : 'is-approve'"
                  data-testid="ai-recommendation"
                >
                  {{ verdictLabel }}
                </span>
                <span class="verdict-model">{{ aiReview.model }}</span>
              </div>

              <p class="review-summary" data-testid="ai-summary">{{ aiReview.summary }}</p>

              <p v-if="isTruncated" class="notice is-warning" data-testid="ai-truncated-notice">
                The diff was larger than one review pass, so some files were not read. They are named in the
                comment below.
              </p>

              <div class="finding-groups" data-testid="ai-findings">
                <ReviewFindingGroup
                  v-for="group in SEVERITY_GROUPS"
                  :key="group.severity"
                  :severity="group.severity"
                  :title="group.title"
                  :hint="group.hint"
                  :findings="groupFindings(group.severity)"
                  :index-offset="groupOffset(group.severity)"
                  :test-id="group.testId"
                />

                <p v-if="findings.length === 0" class="notice">
                  No blockers, suggestions, or nitpicks were found in the reviewed diff.
                </p>
              </div>

              <section v-if="aiReview.strengths.length > 0" class="text-block" data-testid="ai-strengths">
                <h4 class="block-title">What was done well</h4>
                <ul class="block-list">
                  <li v-for="strength in aiReview.strengths" :key="strength">{{ strength }}</li>
                </ul>
              </section>

              <section v-if="aiReview.assumptions.length > 0" class="text-block is-assumption" data-testid="ai-assumptions">
                <h4 class="block-title">Assumptions, not confirmed from the diff</h4>
                <ul class="block-list">
                  <li v-for="assumption in aiReview.assumptions" :key="assumption">{{ assumption }}</li>
                </ul>
              </section>

              <button class="ghost-button is-wide" type="button" @click="openCommentTab">
                Review the comment and post it
              </button>
            </template>
          </div>

          <div
            v-show="activeTab === 'comment'"
            id="review-comment-panel"
            class="tab-panel"
            role="tabpanel"
            aria-labelledby="review-comment-tab"
          >
            <div class="comment-head">
              <label class="comment-label" for="review-comment-input">Review comment</label>
              <div class="comment-tools">
                <span class="comment-count tabular">{{ commentCharacterCount }} characters</span>
                <button
                  class="ghost-button"
                  type="button"
                  :aria-pressed="isPreviewingComment"
                  data-testid="btn-toggle-comment-preview"
                  @click="toggleCommentPreview"
                >
                  {{ isPreviewingComment ? "Edit" : "Preview" }}
                </button>
              </div>
            </div>

            <div
              v-if="isPreviewingComment"
              class="comment-preview"
              data-testid="comment-preview"
              v-html="commentPreviewHtml"
            ></div>
            <textarea
              v-show="!isPreviewingComment"
              id="review-comment-input"
              v-model="commentText"
              class="comment-input"
              placeholder="Run Review with AI to fill this in, or write the review yourself. Markdown is supported."
              data-testid="input-review-comment"
            ></textarea>

            <p class="comment-hint">
              This text is posted to the pull request as a comment. Findings above are already summarised into it.
            </p>
          </div>
        </section>
      </div>

      <footer class="review-footer">
        <div class="footer-group">
          <button
            class="action-button is-positive"
            type="button"
            data-testid="btn-action-approve"
            @click="runAction('approve')"
          >
            <svg viewBox="0 0 24 24" aria-hidden="true"><path d="m5 13 4 4L19 7"></path></svg>
            Approve
          </button>
          <button
            class="action-button is-warning"
            type="button"
            data-testid="btn-action-needs-work"
            @click="runAction('needs-work')"
          >
            <svg viewBox="0 0 24 24" aria-hidden="true">
              <path d="M12 8v5"></path>
              <path d="M12 16h.01"></path>
              <circle cx="12" cy="12" r="9"></circle>
            </svg>
            Needs work
          </button>
          <button
            class="action-button is-danger"
            type="button"
            data-testid="btn-action-decline"
            @click="runAction('decline')"
          >
            <svg viewBox="0 0 24 24" aria-hidden="true">
              <path d="M6 6l12 12"></path>
              <path d="M18 6 6 18"></path>
            </svg>
            Decline
          </button>
        </div>

        <div class="footer-group">
          <ModelPicker
            v-if="hasConfiguredAI"
            v-model="reviewModel"
            :show-effort="false"
            trigger-test-id="pr-review-model-trigger"
          />
          <button
            class="ghost-button"
            type="button"
            :disabled="isReviewing || !hasConfiguredAI"
            data-testid="btn-trigger-ai-review"
            @click="requestReview"
          >
            <svg viewBox="0 0 24 24" aria-hidden="true">
              <path d="M12 3v3"></path>
              <path d="M12 18v3"></path>
              <path d="m5.6 5.6 2.1 2.1"></path>
              <path d="m16.3 16.3 2.1 2.1"></path>
              <path d="M3 12h3"></path>
              <path d="M18 12h3"></path>
              <path d="m5.6 18.4 2.1-2.1"></path>
              <path d="m16.3 7.7 2.1-2.1"></path>
              <circle cx="12" cy="12" r="3.5"></circle>
            </svg>
            {{ isReviewing ? "Reviewing…" : aiReview ? "Review again" : "Review with AI" }}
          </button>
          <button
            class="action-button is-primary"
            type="button"
            :disabled="!canSendComment"
            data-testid="btn-send-comment"
            @click="sendComment"
          >
            <svg viewBox="0 0 24 24" aria-hidden="true">
              <path d="M22 2 11 13"></path>
              <path d="M22 2 15 22l-4-9-9-4z"></path>
            </svg>
            {{ postingComment ? "Posting…" : "Send comment" }}
          </button>
        </div>
      </footer>
    </div>

    <ConfirmDialog
      :open="isConfirmingDecline"
      tone="danger"
      title="Decline this pull request?"
      message="The pull request is marked as declined on Bitbucket. This is visible to the author immediately."
      confirm-label="Decline pull request"
      cancel-label="Keep it open"
      @confirm="confirmDecline"
      @cancel="isConfirmingDecline = false"
    />
  </div>
</template>

<style scoped src="./pull-request-review-modal.css"></style>
