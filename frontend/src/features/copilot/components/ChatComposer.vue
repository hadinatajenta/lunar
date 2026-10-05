<script setup lang="ts">
import { computed, ref } from "vue"
import ModelPicker from "./ModelPicker.vue"
import AttachmentPicker from "./AttachmentPicker.vue"
import AttachmentChips from "./AttachmentChips.vue"
import { useCopilot } from "../composables/useCopilot"
import { useAttachments } from "../composables/useAttachments"
import { imageCapableModelNames, modelAcceptsImages } from "../utils/attachment-rules"

const props = defineProps<{
  isLoading: boolean
}>()

const emit = defineEmits<{
  (event: "send", text: string, attachments: string[]): void
}>()

const { selectedModel } = useCopilot()

const {
  drafts,
  uploadingCount,
  failedCount,
  hasImageDraft,
  readyAttachmentIds,
  addFiles,
  removeDraft,
  clearDrafts
} = useAttachments()

const inputPrompt = ref("")
const textareaEl = ref<HTMLTextAreaElement | null>(null)

const handleInput = (): void => {
  const el = textareaEl.value
  if (el === null) return
  el.style.height = "auto"
  el.style.height = `${Math.min(el.scrollHeight, window.innerHeight * 0.4)}px`
}

const visionModelNames = imageCapableModelNames().join(" or ")

const hasVisionConflict = computed(
  () => hasImageDraft.value && !modelAcceptsImages(selectedModel.value)
)

const uploadNotice = computed(() => {
  if (uploadingCount.value > 0) return "Waiting for uploads to finish"
  if (failedCount.value > 0) return "Remove the failed attachment before sending"
  return ""
})

const visionNotice = computed(() => {
  if (!hasVisionConflict.value) return ""
  return `${selectedModel.value} cannot read images. Images are read only by ${visionModelNames}. Remove the image or switch the model.`
})

const isSendDisabled = computed(() =>
  inputPrompt.value.trim().length === 0 ||
  props.isLoading ||
  uploadNotice.value.length > 0 ||
  visionNotice.value.length > 0
)

const handleSend = (): void => {
  const text = inputPrompt.value.trim()
  if (text.length === 0 || isSendDisabled.value) return
  inputPrompt.value = ""
  if (textareaEl.value !== null) textareaEl.value.style.height = "auto"
  emit("send", text, readyAttachmentIds.value)
}

const handleKeydown = (event: KeyboardEvent): void => {
  if (event.key !== "Enter" || event.shiftKey) return
  event.preventDefault()
  handleSend()
}

defineExpose({ clearDrafts })
</script>

<template>
  <form class="chat-composer" @submit.prevent="handleSend">
    <AttachmentChips :drafts="drafts" @remove="removeDraft" />

    <textarea
      v-model="inputPrompt"
      rows="1"
      data-testid="composer-input"
      aria-label="Message Lunar Copilot"
      placeholder="Message Lunar Copilot with tool integrations..."
      ref="textareaEl"
      @keydown="handleKeydown"
      @input="handleInput"
    ></textarea>

    <p v-if="uploadNotice" class="composer-notice" role="status" data-testid="composer-upload-notice">
      {{ uploadNotice }}
    </p>

    <p v-if="visionNotice" class="composer-warning" role="alert" data-testid="composer-vision-notice">
      {{ visionNotice }}
    </p>

    <div class="composer-actions">
      <AttachmentPicker @select="addFiles" />
      <ModelPicker />

      <button
        class="send-btn"
        type="submit"
        data-testid="composer-send"
        aria-label="Send"
        :disabled="isSendDisabled"
      >
        <svg aria-hidden="true" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <path d="M12 19V5"></path>
          <path d="m5 12 7-7 7 7"></path>
        </svg>
      </button>
    </div>
  </form>
</template>

<style scoped>
.chat-composer {
  max-width: 820px;
  margin: 0 auto;
  padding: 12px 14px;
  border: 1px solid var(--border);
  border-radius: 12px;
  background: var(--surface);
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.04);
  transition: border-color 180ms ease, box-shadow 180ms ease;
}

.chat-composer:focus-within {
  border-color: var(--border-strong);
  box-shadow: 0 0 0 3px color-mix(in srgb, var(--accent) 10%, transparent), 0 1px 3px rgba(0, 0, 0, 0.04);
}

.chat-composer textarea {
  width: 100%;
  border: 0;
  outline: none;
  background: transparent;
  color: var(--text);
  font-size: 14px;
  line-height: 1.5;
  resize: none;
  max-height: 40vh;
  overflow-y: auto;
}

.chat-composer textarea::placeholder {
  color: var(--subtle);
}

.composer-notice {
  margin: 8px 0 0;
  color: var(--muted);
  font-size: 11px;
  line-height: 1.5;
}

.composer-warning {
  margin: 8px 0 0;
  padding: 6px 8px;
  border: 1px solid color-mix(in srgb, var(--warning) 35%, transparent);
  border-radius: 8px;
  background: color-mix(in srgb, var(--warning) 8%, transparent);
  color: var(--text);
  font-size: 11px;
  line-height: 1.5;
}

.composer-actions {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 8px;
  margin-top: 10px;
}

.send-btn {
  display: grid;
  place-items: center;
  width: 32px;
  height: 32px;
  border: 0;
  border-radius: 8px;
  background: var(--accent);
  color: var(--accent-contrast);
  cursor: pointer;
  transition: opacity 180ms ease, transform 180ms ease;
}

.send-btn:hover:not(:disabled) {
  opacity: 0.9;
  transform: translateY(-1px);
}

.send-btn:focus-visible {
  outline: 2px solid var(--accent);
  outline-offset: 2px;
}

.send-btn:disabled {
  opacity: 0.3;
  cursor: not-allowed;
}

.send-btn svg {
  width: 14px;
  height: 14px;
}

@media (prefers-reduced-motion: reduce) {
  .chat-composer,
  .send-btn {
    transition: none;
  }

  .send-btn:hover:not(:disabled) {
    transform: none;
  }
}
</style>
