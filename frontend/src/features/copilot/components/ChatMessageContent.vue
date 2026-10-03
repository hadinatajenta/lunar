<script setup lang="ts">
import { computed } from "vue"
import { renderMarkdown } from "../../../lib/markdown"

const props = defineProps<{
  content: string
}>()

const renderedContent = computed(() => renderMarkdown(props.content))
</script>

<template>
  <div class="message-markdown" v-html="renderedContent"></div>
</template>

<style scoped>
.message-markdown {
  font-size: 14px;
  line-height: 1.65;
  color: inherit;
  overflow-wrap: anywhere;
}

.message-markdown :deep(> :first-child) {
  margin-top: 0;
}

.message-markdown :deep(> :last-child) {
  margin-bottom: 0;
}

.message-markdown :deep(p) {
  margin: 0 0 10px;
}

.message-markdown :deep(h1),
.message-markdown :deep(h2),
.message-markdown :deep(h3),
.message-markdown :deep(h4) {
  margin: 18px 0 8px;
  font-weight: 650;
  line-height: 1.3;
  color: var(--text);
}

.message-markdown :deep(h1) { font-size: 19px; }
.message-markdown :deep(h2) { font-size: 17px; }
.message-markdown :deep(h3) { font-size: 15px; }
.message-markdown :deep(h4) { font-size: 14px; }

.message-markdown :deep(ul),
.message-markdown :deep(ol) {
  margin: 0 0 10px;
  padding-left: 22px;
}

.message-markdown :deep(li) {
  margin: 3px 0;
}

.message-markdown :deep(code) {
  padding: 2px 5px;
  border-radius: 4px;
  background: var(--surface-raised);
  border: 1px solid var(--border);
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 12.5px;
}

.message-markdown :deep(pre) {
  margin: 0 0 10px;
  padding: 12px;
  border-radius: 8px;
  background: var(--surface-raised);
  border: 1px solid var(--border);
  overflow-x: auto;
}

.message-markdown :deep(pre code) {
  padding: 0;
  border: 0;
  background: transparent;
  font-size: 12.5px;
  line-height: 1.55;
}

.message-markdown :deep(table) {
  width: 100%;
  margin: 0 0 12px;
  border-collapse: collapse;
  font-size: 13px;
  display: block;
  overflow-x: auto;
}

.message-markdown :deep(th),
.message-markdown :deep(td) {
  padding: 7px 10px;
  border: 1px solid var(--border);
  text-align: left;
  vertical-align: top;
}

.message-markdown :deep(th) {
  background: var(--surface-raised);
  font-weight: 620;
  white-space: nowrap;
}

.message-markdown :deep(blockquote) {
  margin: 0 0 10px;
  padding: 6px 12px;
  border-left: 3px solid var(--border-strong);
  color: var(--text-muted);
}

.message-markdown :deep(hr) {
  margin: 14px 0;
  border: 0;
  border-top: 1px solid var(--border);
}

.message-markdown :deep(a) {
  color: var(--accent);
  text-decoration: underline;
}
</style>
