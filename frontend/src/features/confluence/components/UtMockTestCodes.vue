<script setup lang="ts">
import { computed, nextTick, onMounted, ref, watch } from "vue"
import type { UtDocsMockTest } from "../types"
import { buildCodeBlockView } from "../../../lib/code-block"
import { renderMarkdown } from "../../../lib/markdown"
import { buildUtMockTestMarkdown } from "../utils/ut-test-table"

const props = defineProps<{
  mockTests: UtDocsMockTest[]
}>()

const container = ref<HTMLDivElement | null>(null)

const renderedHtml = computed(() => {
  const markdown = buildUtMockTestMarkdown(props.mockTests)
  return markdown.length > 0 ? renderMarkdown(markdown) : ""
})

const highlightCodeBlocks = (): void => {
  const root = container.value
  if (root === null) {
    return
  }
  root.querySelectorAll("pre").forEach((pre) => {
    const code = pre.querySelector("code")
    if (code === null) {
      return
    }
    const view = buildCodeBlockView(pre)
    if (view === null) {
      return
    }
    code.innerHTML = view.lines.join("\n")
  })
}

onMounted(highlightCodeBlocks)

watch(renderedHtml, async () => {
  await nextTick()
  highlightCodeBlocks()
})
</script>

<template>
  <div
    v-if="renderedHtml"
    ref="container"
    class="ut-mock-tests"
    data-testid="ai-ut-mock-tests"
    v-html="renderedHtml"
  ></div>
</template>

<style scoped>
.ut-mock-tests {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.ut-mock-tests :deep(pre) {
  margin: 0;
  padding: 12px 14px;
  border: 1px solid var(--border);
  border-radius: 10px;
  background: var(--surface-raised);
  overflow-x: auto;
}

.ut-mock-tests :deep(pre code) {
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 11.5px;
  line-height: 1.6;
  color: var(--text);
  white-space: pre;
}

.ut-mock-tests :deep(.hljs-keyword),
.ut-mock-tests :deep(.hljs-built_in),
.ut-mock-tests :deep(.hljs-literal) {
  color: #c678dd;
}

.ut-mock-tests :deep(.hljs-string) {
  color: #98c379;
}

.ut-mock-tests :deep(.hljs-number) {
  color: #d19a66;
}

.ut-mock-tests :deep(.hljs-title),
.ut-mock-tests :deep(.hljs-function) {
  color: #61afef;
}

.ut-mock-tests :deep(.hljs-type),
.ut-mock-tests :deep(.hljs-attr) {
  color: #e5c07b;
}

.ut-mock-tests :deep(.hljs-comment) {
  color: var(--muted);
  font-style: italic;
}

:root[data-theme="light"] .ut-mock-tests :deep(.hljs-keyword),
:root[data-theme="light"] .ut-mock-tests :deep(.hljs-built_in),
:root[data-theme="light"] .ut-mock-tests :deep(.hljs-literal) {
  color: #8b3a94;
}

:root[data-theme="light"] .ut-mock-tests :deep(.hljs-string) {
  color: #2f6f34;
}

:root[data-theme="light"] .ut-mock-tests :deep(.hljs-number) {
  color: #8a5510;
}

:root[data-theme="light"] .ut-mock-tests :deep(.hljs-title),
:root[data-theme="light"] .ut-mock-tests :deep(.hljs-function) {
  color: #1f5bbf;
}

:root[data-theme="light"] .ut-mock-tests :deep(.hljs-type),
:root[data-theme="light"] .ut-mock-tests :deep(.hljs-attr) {
  color: #8a5c04;
}
</style>
