<script setup lang="ts">
import { ref, watch, onMounted, onUnmounted } from "vue"
import type { PushBranch, CreatePRPayload } from "../types"

const props = defineProps<{
  push: PushBranch | null
  loading: boolean
}>()

const emit = defineEmits<{
  (e: "close"): void
  (e: "submit", payload: CreatePRPayload): void
}>()

const targetBranch = ref("main")
const title = ref("")
const description = ref("")

const defaultDescription = (source: string, target: string) =>
  `## Summary\n\nDescribe the change introduced by \`${source}\`.\n\n## Target\n\nThis PR targets \`${target}\`.\n\n## Checklist\n\n- [ ] Tests added or updated\n- [ ] Docs updated if needed\n- [ ] No breaking changes without a migration note\n`

watch(
  () => props.push,
  (newPush) => {
    if (newPush) {
      targetBranch.value = "main"
      title.value = newPush.suggested_title || newPush.branch
      description.value = defaultDescription(newPush.branch, "main")
    }
  },
  { immediate: true }
)

const handleTargetChange = () => {
  if (props.push) {
    description.value = defaultDescription(props.push.branch, targetBranch.value)
  }
}

const handleSubmit = () => {
  if (!props.push || !title.value.trim()) return
  emit("submit", {
    repo: props.push.repo,
    source_branch: props.push.branch,
    target_branch: targetBranch.value,
    title: title.value.trim(),
    description: description.value.trim(),
  })
}

const handleKeydown = (event: KeyboardEvent) => {
  if (event.key === "Escape") {
    emit("close")
  }
}

onMounted(() => {
  window.addEventListener("keydown", handleKeydown)
})

onUnmounted(() => {
  window.removeEventListener("keydown", handleKeydown)
})
</script>

<template>
  <div
    v-if="push"
    class="modal-overlay"
    data-testid="create-pr-modal-overlay"
    @click.self="emit('close')"
  >
    <div
      class="modal modal-sm"
      role="dialog"
      aria-modal="true"
      aria-labelledby="create-pr-title"
      data-testid="create-pr-modal"
    >
      <div class="modal-header">
        <div style="min-width: 0">
          <div class="modal-eyebrow" data-testid="create-pr-repo-label">{{ push.repo }}</div>
          <h3 id="create-pr-title" class="modal-title">Create pull request</h3>
        </div>
        <button
          class="modal-close"
          type="button"
          aria-label="Close"
          data-testid="btn-close-create-pr"
          @click="emit('close')"
        >
          <svg viewBox="0 0 24 24" aria-hidden="true">
            <path d="M6 6l12 12"></path>
            <path d="M18 6 6 18"></path>
          </svg>
        </button>
      </div>

      <div class="modal-body">
        <div class="branch-flow">
          <div class="branch-flow-node">
            <div class="branch-flow-icon">
              <svg viewBox="0 0 24 24" aria-hidden="true">
                <circle cx="6" cy="6" r="2.5"></circle>
                <circle cx="6" cy="18" r="2.5"></circle>
                <circle cx="18" cy="12" r="2.5"></circle>
                <path d="M6 8.5v7"></path>
                <path d="M8.5 6h4a3 3 0 0 1 3 3v1"></path>
              </svg>
            </div>
            <div style="min-width: 0">
              <div class="branch-flow-label">From</div>
              <div class="branch-flow-name" data-testid="create-pr-source">{{ push.branch }}</div>
            </div>
          </div>

          <div class="branch-flow-arrow" aria-hidden="true">
            <svg viewBox="0 0 24 24"><path d="M5 12h14"></path><path d="m13 6 6 6-6 6"></path></svg>
          </div>

          <div class="branch-flow-node">
            <div class="branch-flow-icon">
              <svg viewBox="0 0 24 24" aria-hidden="true">
                <circle cx="6" cy="6" r="2.5"></circle>
                <circle cx="6" cy="18" r="2.5"></circle>
                <circle cx="18" cy="12" r="2.5"></circle>
                <path d="M6 8.5v7"></path>
                <path d="M8.5 6h4a3 3 0 0 1 3 3v1"></path>
              </svg>
            </div>
            <div style="min-width: 0">
              <div class="branch-flow-label">Into</div>
              <div class="branch-flow-name" data-testid="create-pr-target-label">{{ targetBranch }}</div>
            </div>
          </div>
        </div>

        <div class="form-row">
          <label class="form-label" for="create-pr-target">Target branch</label>
          <select
            id="create-pr-target"
            v-model="targetBranch"
            class="form-select"
            data-testid="select-pr-target"
            @change="handleTargetChange"
          >
            <option value="main">main</option>
            <option value="develop">develop</option>
            <option value="release/2.4">release/2.4</option>
            <option value="release/2.5">release/2.5</option>
            <option value="staging">staging</option>
          </select>
          <div class="field-hint">The branch your changes will be merged into once approved.</div>
        </div>

        <div class="form-row">
          <label class="form-label" for="create-pr-title-input">Title</label>
          <input
            id="create-pr-title-input"
            v-model="title"
            class="form-input"
            type="text"
            placeholder="Short, descriptive title"
            data-testid="input-pr-title"
          />
        </div>

        <div class="form-row">
          <label class="form-label" for="create-pr-description">Description</label>
          <textarea
            id="create-pr-description"
            v-model="description"
            class="form-textarea"
            placeholder="Describe what changed, why, and anything reviewers should know."
            data-testid="input-pr-description"
          ></textarea>
        </div>
      </div>

      <div class="modal-footer">
        <button
          class="btn btn-ghost"
          type="button"
          data-testid="btn-cancel-pr"
          @click="emit('close')"
        >
          Cancel
        </button>
        <div class="spacer"></div>
        <button
          class="btn btn-primary"
          type="button"
          :disabled="loading || !title.trim()"
          data-testid="btn-submit-pr"
          @click="handleSubmit"
        >
          <svg viewBox="0 0 24 24" aria-hidden="true"><path d="M12 5v14"></path><path d="M5 12h14"></path></svg>
          {{ loading ? "Creating…" : "Create pull request" }}
        </button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.modal-overlay {
  position: fixed;
  inset: 0;
  z-index: 40;
  display: grid;
  place-items: center;
  padding: 24px;
  background: rgba(5, 6, 8, 0.72);
  backdrop-filter: blur(6px);
  -webkit-backdrop-filter: blur(6px);
}

.modal {
  width: 100%;
  max-width: 680px;
  max-height: calc(100vh - 48px);
  display: flex;
  flex-direction: column;
  border: 1px solid var(--border-strong);
  border-radius: 14px;
  background: #101217;
  box-shadow: 0 30px 80px rgba(0, 0, 0, 0.55);
  overflow: hidden;
}

.modal.modal-sm {
  max-width: 560px;
}

.modal-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  padding: 18px 20px;
  border-bottom: 1px solid var(--border);
  flex: 0 0 auto;
}

.modal-eyebrow {
  margin-bottom: 6px;
  color: var(--subtle);
  font-size: 10px;
  font-weight: 600;
  letter-spacing: 0.09em;
  text-transform: uppercase;
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
}

.modal-title {
  margin: 0;
  color: #eef1f4;
  font-size: 15px;
  font-weight: 600;
  letter-spacing: -0.02em;
  line-height: 1.35;
}

.modal-close {
  display: grid;
  place-items: center;
  width: 30px;
  height: 30px;
  flex: 0 0 auto;
  border: 1px solid var(--border);
  border-radius: 8px;
  background: transparent;
  color: var(--muted);
  cursor: pointer;
  transition: background-color 150ms ease, color 150ms ease, border-color 150ms ease;
}

.modal-close:hover {
  background: rgba(255, 255, 255, 0.05);
  border-color: var(--border-strong);
  color: var(--text);
}

.modal-close svg {
  width: 14px;
  height: 14px;
  stroke: currentColor;
  stroke-width: 1.8;
  fill: none;
  stroke-linecap: round;
  stroke-linejoin: round;
}

.modal-body {
  flex: 1;
  overflow-y: auto;
  padding: 18px 20px;
}

.branch-flow {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 12px 14px;
  margin-bottom: 14px;
  border: 1px solid var(--border);
  border-radius: 9px;
  background: rgba(255, 255, 255, 0.018);
}

.branch-flow-node {
  display: flex;
  align-items: center;
  gap: 7px;
  min-width: 0;
  flex: 1;
}

.branch-flow-icon {
  display: grid;
  place-items: center;
  width: 26px;
  height: 26px;
  flex: 0 0 auto;
  border: 1px solid var(--border-strong);
  border-radius: 7px;
  background: var(--surface-raised);
  color: #cfd6de;
}

.branch-flow-icon svg {
  width: 12px;
  height: 12px;
  stroke: currentColor;
  stroke-width: 2;
  fill: none;
  stroke-linecap: round;
  stroke-linejoin: round;
}

.branch-flow-label {
  color: var(--subtle);
  font-size: 9px;
  font-weight: 600;
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

.branch-flow-name {
  margin-top: 2px;
  color: #e3e7ec;
  font-size: 11.5px;
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.branch-flow-arrow {
  color: var(--subtle);
  flex: 0 0 auto;
}

.branch-flow-arrow svg {
  width: 16px;
  height: 16px;
  stroke: currentColor;
  stroke-width: 1.8;
  fill: none;
  stroke-linecap: round;
  stroke-linejoin: round;
}

.form-row {
  margin-bottom: 14px;
}

.form-row:last-child {
  margin-bottom: 0;
}

.form-label {
  display: block;
  margin-bottom: 7px;
  color: var(--subtle);
  font-size: 10px;
  font-weight: 600;
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

.form-input,
.form-select,
.form-textarea {
  width: 100%;
  padding: 0 12px;
  border: 1px solid var(--border);
  border-radius: 8px;
  outline: 0;
  background: var(--surface-raised);
  color: #dfe4e9;
  font-size: 12px;
  transition: border-color 160ms ease, box-shadow 160ms ease;
}

.form-input,
.form-select {
  height: 38px;
}

.form-textarea {
  min-height: 110px;
  padding: 10px 12px;
  resize: vertical;
  line-height: 1.6;
  font-family: inherit;
}

.form-input:focus,
.form-select:focus,
.form-textarea:focus {
  border-color: var(--border-strong);
  box-shadow: 0 0 0 3px rgba(255, 255, 255, 0.03);
}

.form-input::placeholder,
.form-textarea::placeholder {
  color: #5d6570;
}

.field-hint {
  margin-top: 6px;
  color: var(--subtle);
  font-size: 10px;
  line-height: 1.5;
}

.modal-footer {
  display: flex;
  align-items: center;
  gap: 7px;
  flex-wrap: wrap;
  padding: 14px 20px;
  border-top: 1px solid var(--border);
  background: rgba(255, 255, 255, 0.014);
  flex: 0 0 auto;
}

.modal-footer .spacer {
  flex: 1;
}

.btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  height: 32px;
  padding: 0 13px;
  border-radius: 8px;
  font-size: 11.5px;
  font-weight: 500;
  cursor: pointer;
  transition: all 160ms ease;
  white-space: nowrap;
}

.btn-ghost {
  border: 1px solid var(--border);
  background: transparent;
  color: #c8ced5;
}

.btn-ghost:hover {
  background: rgba(255, 255, 255, 0.04);
  color: var(--text);
}

.btn-primary {
  border: 1px solid rgba(255, 255, 255, 0.16);
  background: #f0f3f6;
  color: #0b0c0f;
}

.btn-primary:hover:not(:disabled) {
  background: #ffffff;
}

.btn-primary:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.btn-primary svg {
  width: 13px;
  height: 13px;
  stroke: currentColor;
  stroke-width: 2.2;
  fill: none;
  stroke-linecap: round;
  stroke-linejoin: round;
}
</style>
