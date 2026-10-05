import hljs from "highlight.js/lib/core"
import bash from "highlight.js/lib/languages/bash"
import css from "highlight.js/lib/languages/css"
import go from "highlight.js/lib/languages/go"
import javascript from "highlight.js/lib/languages/javascript"
import json from "highlight.js/lib/languages/json"
import php from "highlight.js/lib/languages/php"
import python from "highlight.js/lib/languages/python"
import sql from "highlight.js/lib/languages/sql"
import typescript from "highlight.js/lib/languages/typescript"
import xml from "highlight.js/lib/languages/xml"
import yaml from "highlight.js/lib/languages/yaml"

const LANGUAGES: Record<string, unknown> = {
  bash,
  css,
  go,
  javascript,
  json,
  php,
  python,
  sql,
  typescript,
  xml,
  yaml
}

const LANGUAGE_ALIASES: Record<string, string> = {
  golang: "go",
  js: "javascript",
  jsx: "javascript",
  ts: "typescript",
  tsx: "typescript",
  html: "xml",
  sh: "bash",
  shell: "bash",
  yml: "yaml"
}

Object.entries(LANGUAGES).forEach(([name, definition]) => {
  hljs.registerLanguage(name, definition as never)
})

export const parseHighlightSpec = (spec: string): Set<number> => {
  const lines = new Set<number>()
  if (spec.trim().length === 0) return lines

  spec.split(",").forEach((part) => {
    const trimmed = part.trim()
    if (trimmed.length === 0) return
    const range = trimmed.match(/^(\d+)-(\d+)$/)
    if (range !== null) {
      const start = Number(range[1])
      const end = Number(range[2])
      for (let line = Math.min(start, end); line <= Math.max(start, end); line += 1) lines.add(line)
      return
    }
    const single = Number(trimmed)
    if (Number.isInteger(single) && single > 0) lines.add(single)
  })

  return lines
}

const resolveLanguage = (language: string): string | null => {
  const normalized = language.trim().toLowerCase()
  if (normalized.length === 0) return null
  const alias = LANGUAGE_ALIASES[normalized] ?? normalized
  return alias in LANGUAGES ? alias : null
}

const splitHighlightedLines = (html: string): string[] => {
  const lines: string[] = []
  const openTags: string[] = []
  let current = ""
  let index = 0

  while (index < html.length) {
    if (html[index] === "<") {
      const end = html.indexOf(">", index)
      if (end === -1) {
        current += html.slice(index)
        break
      }
      const tag = html.slice(index, end + 1)
      if (tag.startsWith("</")) {
        openTags.pop()
      } else if (!tag.endsWith("/>")) {
        openTags.push(tag)
      }
      current += tag
      index = end + 1
      continue
    }

    if (html[index] === "\n") {
      lines.push(current + "</span>".repeat(openTags.length))
      current = openTags.join("")
      index += 1
      continue
    }

    current += html[index]
    index += 1
  }

  lines.push(current)
  return lines
}

const highlightLines = (code: string, language: string | null): string[] => {
  if (language === null) {
    return code.split("\n").map((line) => escapeLine(line))
  }
  try {
    return splitHighlightedLines(hljs.highlight(code, { language, ignoreIllegals: true }).value)
  } catch {
    return code.split("\n").map((line) => escapeLine(line))
  }
}

const escapeLine = (line: string): string =>
  line.replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;")

export interface CodeBlockView {
  language: string
  lines: string[]
  highlighted: Set<number>
}

export const buildCodeBlockView = (pre: HTMLElement): CodeBlockView | null => {
  const code = pre.querySelector("code")
  if (code === null) return null

  const source = code.textContent ?? ""
  if (source.trim().length === 0) return null

  const language = resolveLanguage(pre.dataset.language ?? "") ?? resolveLanguage(
    (code.className.match(/language-([A-Za-z0-9+#-]+)/) ?? [])[1] ?? ""
  )

  return {
    language: language ?? pre.dataset.language ?? "",
    lines: highlightLines(source.replace(/\n$/, ""), language),
    highlighted: parseHighlightSpec(pre.dataset.highlight ?? "")
  }
}
