<script setup lang="ts">
defineProps<{
  isChecking: boolean
}>()

const emit = defineEmits<{
  (event: "check-again"): void
}>()
</script>

<template>
  <div class="helper-guide">
    <div class="helper-guide-title">The helper is not running yet</div>
    <p class="helper-guide-copy">
      The helper is a small program that reads git metadata on this machine. Nothing leaves your computer.
      Run it once and leave the window open.
    </p>

    <ol class="helper-guide-steps">
      <li>
        <span class="helper-guide-text">
          Download the helper for your operating system from the Lunar release page.
        </span>
      </li>
      <li>
        <span class="helper-guide-text">Start it in a terminal:</span>
        <code class="helper-guide-code">lunar-helper</code>
      </li>
      <li>
        <span class="helper-guide-text">Copy the token it prints, or read it from this file:</span>
        <code class="helper-guide-code">~/.lunar/helper-token</code>
      </li>
      <li>
        <span class="helper-guide-text">
          Paste the token above, select Test connection, then set your repository root folder.
        </span>
      </li>
    </ol>

    <button class="field-btn helper-guide-action" type="button" :disabled="isChecking" @click="emit('check-again')">
      {{ isChecking ? "Checking…" : "Check again" }}
    </button>
  </div>
</template>

<style scoped>
.helper-guide {
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: 16px;
  border: 1px solid color-mix(in srgb, var(--warning) 30%, transparent);
  border-radius: 8px;
  background: color-mix(in srgb, var(--warning) 8%, transparent);
}

.helper-guide-title {
  color: var(--warning);
  font-size: 13px;
  font-weight: 600;
}

.helper-guide-copy {
  margin: 0;
  color: var(--muted);
  font-size: 12px;
  line-height: 1.55;
}

.helper-guide-steps {
  display: flex;
  flex-direction: column;
  gap: 10px;
  margin: 0;
  padding-left: 18px;
  color: var(--muted);
  font-size: 12px;
  line-height: 1.55;
}

.helper-guide-steps li {
  display: flex;
  flex-direction: column;
  gap: 5px;
}

.helper-guide-text {
  color: var(--muted);
}

.helper-guide-code {
  align-self: flex-start;
  padding: 5px 9px;
  border: 1px solid var(--border);
  border-radius: 6px;
  background: var(--surface-raised);
  color: var(--text);
  font-family: "JetBrains Mono", ui-monospace, Menlo, monospace;
  font-size: 11.5px;
}

.helper-guide-action {
  align-self: flex-start;
}

@media (max-width: 768px) {
  .helper-guide-action {
    align-self: stretch;
  }
}
</style>
