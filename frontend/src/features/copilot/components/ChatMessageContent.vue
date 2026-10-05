<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from "vue"
import { renderMarkdown } from "../../../lib/markdown"
import { buildCodeBlockView } from "../../../lib/code-block"
import { parseMarkdownSegments } from "../../../lib/diagram"
import ChatDiagram from "./ChatDiagram.vue"

const props = withDefaults(defineProps<{ content: string; isStreaming?: boolean }>(), {
  isStreaming: false
})

const MIN_RENDER_INTERVAL_MS = 32
const COPY_FEEDBACK_MS = 2000
const SVG_NAMESPACE = "http://www.w3.org/2000/svg"
const COPY_ICON_PATH =
  "M9 4h6v3H9zM8 6h8a2 2 0 0 1 2 2v11a2 2 0 0 1-2 2H8a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2Z"
const COPIED_ICON_PATH =
  "M9 4h6v3H9zM8 6h8a2 2 0 0 1 2 2v11a2 2 0 0 1-2 2H8a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2ZM9 12.5l2 2 4-4"

const markdownContainers = ref<HTMLDivElement[]>([])
const diagramElements = ref<HTMLDivElement[]>([])

const registerMarkdownContainer = (element: unknown, index: number): void => {
  if (element instanceof HTMLDivElement) markdownContainers.value[index] = element
}

const registerDiagramElement = (element: unknown, index: number): void => {
  if (element instanceof HTMLDivElement) diagramElements.value[index] = element
}

const scrollToDiagram = (index: number): void => {
  const element = diagramElements.value[index]
  if (element) element.scrollIntoView({ behavior: "smooth", block: "center" })
}
const renderedSource = ref(props.content)
const copyError = ref("")
const copyTimers = new Map<HTMLButtonElement, ReturnType<typeof setTimeout>>()
let scheduledFrame: number | null = null
let scheduledTimer: ReturnType<typeof setTimeout> | null = null
let lastRenderAt = 0

const cancelScheduledRender = () => {
  if (scheduledFrame !== null) cancelAnimationFrame(scheduledFrame)
  if (scheduledTimer !== null) clearTimeout(scheduledTimer)
  scheduledFrame = null
  scheduledTimer = null
}

const applyNow = (value: string) => {
  cancelScheduledRender()
  lastRenderAt = Date.now()
  renderedSource.value = value
}

const scheduleRender = (value: string) => {
  cancelScheduledRender()
  const commit = () => {
    scheduledFrame = null
    scheduledTimer = null
    applyNow(value)
  }
  if (typeof requestAnimationFrame === "function") {
    scheduledFrame = requestAnimationFrame(commit)
    return
  }
  scheduledTimer = setTimeout(commit, MIN_RENDER_INTERVAL_MS)
}

watch(() => props.content, (value) => {
  if (!props.isStreaming || Date.now() - lastRenderAt >= MIN_RENDER_INTERVAL_MS) {
    applyNow(value)
    return
  }
  scheduleRender(value)
})

watch(() => props.isStreaming, (isNowStreaming) => {
  if (isNowStreaming) return
  applyNow(props.content)
})

const segments = computed(() => parseMarkdownSegments(renderedSource.value))

const diagramCount = computed(() => segments.value.filter((segment) => segment.type === "diagram").length)

let diagramCursor = 0
const renderedSegments = computed(() => {
  diagramCursor = 0
  return segments.value.map((segment) => {
    if (segment.type === "diagram") {
      const index = diagramCursor
      diagramCursor += 1
      return {
        type: "diagram" as const,
        source: segment.content,
        language: segment.language ?? "mermaid",
        index
      }
    }
    return { type: "markdown" as const, html: renderMarkdown(segment.content) }
  })
})

const buildElement = <K extends keyof HTMLElementTagNameMap>(
  tag: K,
  className: string,
  textContent?: string
): HTMLElementTagNameMap[K] => {
  const element = document.createElement(tag)
  element.className = className
  if (textContent !== undefined) element.textContent = textContent
  return element
}

const buildIcon = (pathData: string): SVGSVGElement => {
  const icon = document.createElementNS(SVG_NAMESPACE, "svg")
  icon.setAttribute("viewBox", "0 0 24 24")
  icon.setAttribute("fill", "none")
  icon.setAttribute("stroke", "currentColor")
  icon.setAttribute("aria-hidden", "true")
  const path = document.createElementNS(SVG_NAMESPACE, "path")
  path.setAttribute("d", pathData)
  icon.appendChild(path)
  return icon
}

const buildCopyControl = (kind: "code" | "table"): HTMLButtonElement => {
  const accessibleName = kind === "table" ? "Copy table as Markdown" : "Copy code block"
  const control = buildElement("button", "copy-control")
  control.type = "button"
  control.dataset.copyKind = kind
  control.dataset.copyName = accessibleName
  control.setAttribute("aria-label", accessibleName)
  control.title = accessibleName
  control.append(buildIcon(COPY_ICON_PATH), buildElement("span", "copy-control-label", "Copy"))
  return control
}

const buildBlockShell = (block: HTMLElement, kind: "code" | "table", kindLabel: string) => {
  const parent = block.parentElement
  if (!parent) return
  const shell = buildElement("div", `block-shell block-shell-${kind}`)
  const toolbar = buildElement("div", "block-toolbar")
  toolbar.append(buildElement("span", "block-kind", kindLabel), buildCopyControl(kind))
  const scroll = buildElement("div", "block-scroll")
  shell.append(toolbar, scroll)
  parent.replaceChild(shell, block)
  scroll.appendChild(block)
}

const buildCodeMockup = (pre: HTMLElement) => {
  const parent = pre.parentElement
  if (!parent) return
  const view = buildCodeBlockView(pre)
  if (view === null) return

  const shell = buildElement("div", "block-shell block-shell-code")
  const bar = buildElement("div", "code-bar")
  const dots = buildElement("span", "code-dots")
  dots.setAttribute("aria-hidden", "true")
  bar.append(dots, buildElement("span", "code-language", view.language || "text"), buildCopyControl("code"))

  const body = buildElement("div", "code-body")
  view.lines.forEach((line, index) => {
    const lineNumber = index + 1
    const row = buildElement("div", "code-line")
    if (view.highlighted.has(lineNumber)) row.classList.add("is-highlighted")
    const gutter = buildElement("span", "code-gutter", String(lineNumber))
    gutter.setAttribute("aria-hidden", "true")
    const content = buildElement("span", "code-content")
    content.innerHTML = line
    row.append(gutter, content)
    body.appendChild(row)
  })

  shell.append(bar, body)
  parent.replaceChild(shell, pre)
}

const decorateBlocks = () => {
  markdownContainers.value.forEach((root) => {
    if (!root) return
    root.querySelectorAll("pre").forEach((pre) => buildCodeMockup(pre))
    root.querySelectorAll("table").forEach((table) => buildBlockShell(table, "table", "Table"))
  })
}

const readTableAsMarkdown = (shell: Element): string => {
  const rows = Array.from(shell.querySelectorAll("tr"))
  if (rows.length === 0) return ""
  const lines: string[] = []
  rows.forEach((row, rowIndex) => {
    const cells = Array.from(row.querySelectorAll("th, td"), (cell) =>
      (cell.textContent ?? "").trim().replace(/\s+/g, " ").replace(/\|/g, "\\|")
    )
    lines.push(`| ${cells.join(" | ")} |`)
    if (rowIndex === 0) lines.push(`| ${cells.map(() => "---").join(" | ")} |`)
  })
  return lines.join("\n")
}

const markCopied = (control: HTMLButtonElement) => {
  const accessibleName = control.dataset.copyName ?? "Copy"
  const label = control.querySelector(".copy-control-label")
  const path = control.querySelector("path")
  control.dataset.copied = "true"
  control.setAttribute("aria-label", "Copied")
  control.title = "Copied"
  if (label) label.textContent = "Copied"
  if (path) path.setAttribute("d", COPIED_ICON_PATH)
  const existingTimer = copyTimers.get(control)
  if (existingTimer) clearTimeout(existingTimer)
  copyTimers.set(control, setTimeout(() => {
    copyTimers.delete(control)
    control.dataset.copied = "false"
    control.setAttribute("aria-label", accessibleName)
    control.title = accessibleName
    if (label) label.textContent = "Copy"
    if (path) path.setAttribute("d", COPY_ICON_PATH)
  }, COPY_FEEDBACK_MS))
}

const handleContainerClick = async (event: MouseEvent) => {
  const target = event.target
  if (!(target instanceof Element)) return
  const control = target.closest<HTMLButtonElement>(".copy-control")
  const shell = control?.closest(".block-shell")
  if (!control || !shell) return
  event.preventDefault()
  const text =
    control.dataset.copyKind === "table" ? readTableAsMarkdown(shell) : (shell.querySelector("pre")?.textContent ?? "")
  if (text.length === 0) return
  copyError.value = ""
  if (!navigator.clipboard?.writeText) {
    copyError.value = "Copy failed: the clipboard is unavailable in this browser."
    return
  }
  try {
    await navigator.clipboard.writeText(text)
  } catch (error) {
    const detail =
      error instanceof Error && error.message
        ? error.message
        : "the browser rejected clipboard access"
    copyError.value = `Copy failed: ${detail}`
    return
  }
  markCopied(control)
}

onMounted(decorateBlocks)

watch(renderedSegments, async () => {
  await nextTick()
  decorateBlocks()
})

onUnmounted(() => {
  cancelScheduledRender()
  copyTimers.forEach((timer) => clearTimeout(timer))
  copyTimers.clear()
})
</script>

<template>
  <div class="message-content">
    <template v-for="(segment, index) in renderedSegments" :key="index">
      <div
        v-if="segment.type === 'markdown'"
        :ref="(element) => registerMarkdownContainer(element, index)"
        class="message-markdown"
        v-html="segment.html"
        @click="handleContainerClick"
      ></div>

      <div v-else :ref="(element) => registerDiagramElement(element, segment.index)" class="message-diagram">
        <ChatDiagram
          :source="segment.source"
          :language="segment.language"
          :index="segment.index"
          :total="diagramCount"
          @navigate="scrollToDiagram(segment.index + $event)"
        />
      </div>
    </template>

    <p v-if="copyError" class="message-copy-error" role="alert">{{ copyError }}</p>
  </div>
</template>

<style scoped>
.message-content { min-width: 0; }

.message-markdown { font-size: 14px; line-height: 1.65; color: inherit; overflow-wrap: anywhere; }

.message-markdown :deep(> :first-child) { margin-top: 0; }
.message-markdown :deep(> :last-child) { margin-bottom: 0; }
.message-markdown :deep(p) { margin: 0 0 10px; }
.message-markdown :deep(li) { margin: 3px 0; }
.message-markdown :deep(h1),
.message-markdown :deep(h2),
.message-markdown :deep(h3),
.message-markdown :deep(h4) { margin: 18px 0 8px; font-weight: 650; line-height: 1.3; color: var(--text); }
.message-markdown :deep(h1) { font-size: 19px; }
.message-markdown :deep(h2) { font-size: 17px; }
.message-markdown :deep(h3) { font-size: 15px; }
.message-markdown :deep(h4) { font-size: 14px; }
.message-markdown :deep(ul),
.message-markdown :deep(ol) { margin: 0 0 10px; padding-left: 22px; }

.message-markdown :deep(code) {
  padding: 2px 5px;
  border-radius: 4px;
  background: var(--surface-raised);
  border: 1px solid var(--border);
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 12.5px;
}

.message-markdown :deep(pre) { margin: 0; padding: 12px; border-radius: 8px; background: var(--surface-raised); border: 1px solid var(--border); overflow-x: auto; }

.message-markdown :deep(pre code) { padding: 0; border: 0; background: transparent; font-size: 12.5px; line-height: 1.55; }

.message-markdown :deep(table) {
  width: 100%;
  margin: 0;
  border-collapse: collapse;
  display: block;
  overflow-x: auto;
  font-size: 13px;
}

.message-markdown :deep(th),
.message-markdown :deep(td) { padding: 7px 10px; border: 1px solid var(--border); text-align: left; vertical-align: top; }
.message-markdown :deep(th) { background: var(--surface-raised); font-weight: 620; white-space: nowrap; }
.message-markdown :deep(blockquote) { margin: 0 0 10px; padding: 6px 12px; border-left: 3px solid var(--border-strong); color: var(--muted); }
.message-markdown :deep(hr) { margin: 14px 0; border: 0; border-top: 1px solid var(--border); }
.message-markdown :deep(a) { color: var(--accent); text-decoration: underline; }

.message-markdown :deep(.block-shell) { margin: 0 0 10px; }
.message-markdown :deep(.block-shell-table) { margin-bottom: 12px; }
.message-markdown :deep(.block-scroll) { overflow-x: auto; }
.message-markdown :deep(.block-toolbar) { display: flex; align-items: center; justify-content: space-between; gap: 8px; min-height: 20px; margin-bottom: 4px; }
.message-markdown :deep(.block-kind) { color: var(--subtle); font-size: 10px; font-weight: 600; text-transform: uppercase; letter-spacing: 0.07em; }

.message-markdown :deep(.copy-control) {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  padding: 2px 8px;
  border: 1px solid var(--border);
  border-radius: 6px;
  background: transparent;
  color: var(--muted);
  font-family: inherit;
  font-size: 10.5px;
  font-weight: 500;
  line-height: 1.5;
  cursor: pointer;
  transition: color 160ms ease, border-color 160ms ease, background-color 160ms ease;
}

.message-markdown :deep(.copy-control svg) { width: 12px; height: 12px; flex-shrink: 0; fill: none; stroke: currentColor; stroke-width: 1.8; stroke-linecap: round; stroke-linejoin: round; }
.message-markdown :deep(.copy-control:hover) { border-color: var(--border-strong); background: var(--surface-hover); color: var(--text); }
.message-markdown :deep(.copy-control:focus-visible) { outline: 2px solid var(--accent); outline-offset: 1px; }
.message-markdown :deep(.copy-control[data-copied="true"]) { border-color: var(--positive); color: var(--positive); }

.message-markdown :deep(.block-shell-code) {
  --code-keyword: #c678dd;
  --code-string: #98c379;
  --code-number: #d19a66;
  --code-function: #61afef;
  --code-type: #e5c07b;
  --code-tag: #e06c75;
  --code-comment: var(--muted);
  --code-line-highlight: color-mix(in srgb, var(--accent) 9%, transparent);
  border: 1px solid var(--border);
  border-radius: 10px;
  background: var(--surface-raised);
  overflow: hidden;
}

:root[data-theme="light"] .message-markdown :deep(.block-shell-code) {
  --code-keyword: #8b3a94;
  --code-string: #2f6f34;
  --code-number: #8a5510;
  --code-function: #1f5bbf;
  --code-type: #8a5c04;
  --code-tag: #b03024;
  --code-line-highlight: color-mix(in srgb, var(--accent) 7%, transparent);
}

.message-markdown :deep(.code-bar) {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 10px 8px 12px;
  border-bottom: 1px solid var(--border);
  background: color-mix(in srgb, var(--surface) 60%, transparent);
}

.message-markdown :deep(.code-dots) {
  width: 9px;
  height: 9px;
  flex-shrink: 0;
  border-radius: 50%;
  background: var(--muted);
  opacity: 0.4;
  box-shadow: 1.4em 0 0 var(--muted), 2.8em 0 0 var(--muted);
  margin-right: 3.4em;
}

.message-markdown :deep(.code-language) {
  color: var(--muted);
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 10px;
  font-weight: 600;
  letter-spacing: 0.07em;
  text-transform: uppercase;
}

.message-markdown :deep(.block-shell-code .block-toolbar) { display: none; }
.message-markdown :deep(.block-shell-code .copy-control) { margin-left: auto; }

.message-markdown :deep(.code-body) {
  padding: 10px 0;
  overflow-x: auto;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 12.5px;
  line-height: 1.6;
}

.message-markdown :deep(.code-line) {
  display: flex;
  gap: 12px;
  padding-right: 14px;
  border-left: 2px solid transparent;
}

.message-markdown :deep(.code-line.is-highlighted) {
  background: var(--code-line-highlight);
  border-left-color: var(--accent);
}

.message-markdown :deep(.code-gutter) {
  flex-shrink: 0;
  width: 40px;
  padding-right: 10px;
  text-align: right;
  color: var(--muted);
  opacity: 0.6;
  user-select: none;
  font-variant-numeric: tabular-nums;
}

.message-markdown :deep(.code-content) {
  white-space: pre;
  color: var(--text);
}

.message-markdown :deep(.code-content .hljs-keyword),
.message-markdown :deep(.code-content .hljs-selector-tag),
.message-markdown :deep(.code-content .hljs-literal),
.message-markdown :deep(.code-content .hljs-built_in) { color: var(--code-keyword); }

.message-markdown :deep(.code-content .hljs-string),
.message-markdown :deep(.code-content .hljs-regexp),
.message-markdown :deep(.code-content .hljs-addition) { color: var(--code-string); }

.message-markdown :deep(.code-content .hljs-number),
.message-markdown :deep(.code-content .hljs-symbol),
.message-markdown :deep(.code-content .hljs-bullet) { color: var(--code-number); }

.message-markdown :deep(.code-content .hljs-title),
.message-markdown :deep(.code-content .hljs-section) { color: var(--code-function); }

.message-markdown :deep(.code-content .hljs-type),
.message-markdown :deep(.code-content .hljs-attr),
.message-markdown :deep(.code-content .hljs-attribute),
.message-markdown :deep(.code-content .hljs-variable) { color: var(--code-type); }

.message-markdown :deep(.code-content .hljs-tag),
.message-markdown :deep(.code-content .hljs-name),
.message-markdown :deep(.code-content .hljs-selector-id),
.message-markdown :deep(.code-content .hljs-selector-class) { color: var(--code-tag); }

.message-markdown :deep(.code-content .hljs-comment),
.message-markdown :deep(.code-content .hljs-quote),
.message-markdown :deep(.code-content .hljs-meta) { color: var(--code-comment); font-style: italic; }

.message-markdown :deep(.code-content .hljs-deletion) { color: var(--danger); }
.message-markdown :deep(.code-content .hljs-emphasis) { font-style: italic; }
.message-markdown :deep(.code-content .hljs-strong) { font-weight: 700; }

.message-copy-error { margin: 6px 0 0; color: var(--danger); font-size: 11px; }

@media (prefers-reduced-motion: reduce) {
  .message-markdown :deep(.copy-control) { transition: none; }
}
</style>
