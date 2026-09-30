<script setup lang="ts">
import { ref } from "vue"
import { useToast } from "../../../composables/useToast"

const { showToast } = useToast()

const commentDraft = ref("")
const commentInput = ref<HTMLTextAreaElement | null>(null)

const handlePost = () => {
  if (!commentDraft.value.trim()) {
    commentInput.value?.focus()
    return
  }
  showToast("Commenting is ready for backend wiring.")
}
</script>

<template>
  <div class="detail-section">
    <div class="detail-comments">
      <div class="detail-comments-head">
        <span class="detail-comments-title">Comments</span>
        <span class="detail-comments-count">0 comments</span>
      </div>

      <p class="comments-empty">No comments yet.</p>

      <div class="comment-composer">
        <textarea
          ref="commentInput"
          v-model="commentDraft"
          data-testid="detail-comment-input"
          placeholder="Add a comment…"
        ></textarea>
        <button
          class="btn btn-primary"
          type="button"
          data-testid="detail-comment-post"
          @click="handlePost"
        >
          Post
        </button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.detail-section {
  margin-bottom: 24px;
}

.detail-comments {
  padding: 18px;
  border: 1px solid var(--border);
  border-radius: 12px;
  background: var(--surface);
}

.detail-comments-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 14px;
}

.detail-comments-title {
  color: var(--text);
  font-size: 12px;
  font-weight: 600;
}

.detail-comments-count {
  color: var(--subtle);
  font-size: 10px;
  font-variant-numeric: tabular-nums;
}

.comments-empty {
  margin: 0;
  padding: 8px 0 4px;
  color: var(--subtle);
  font-size: 12px;
  line-height: 1.6;
}

.comment-composer {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  margin-top: 12px;
  padding-top: 14px;
  border-top: 1px solid var(--border);
}

.comment-composer textarea {
  flex: 1;
  min-height: 44px;
  max-height: 160px;
  resize: vertical;
  padding: 10px 12px;
  border: 1px solid var(--border);
  border-radius: 8px;
  outline: 0;
  background: var(--surface-raised);
  color: var(--text);
  font-size: 12px;
  line-height: 1.55;
  font-family: inherit;
  transition:
    border-color 160ms ease,
    box-shadow 160ms ease;
}

.comment-composer textarea::placeholder {
  color: var(--subtle);
}

.comment-composer textarea:focus {
  border-color: var(--border-strong);
  box-shadow: 0 0 0 3px color-mix(in srgb, var(--accent) 15%, transparent);
}

.btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  min-height: 34px;
  padding: 0 12px;
  border: 1px solid transparent;
  border-radius: 8px;
  font-size: 11.5px;
  font-weight: 500;
  white-space: nowrap;
  cursor: pointer;
  transition:
    background-color 160ms ease,
    opacity 160ms ease;
}

.btn-primary {
  background: var(--accent);
  color: var(--accent-contrast);
}

.btn-primary:hover {
  opacity: 0.9;
}
</style>
