import DOMPurify from "dompurify"
import { marked } from "marked"

const ALLOWED_TAGS = [
  "p",
  "br",
  "strong",
  "em",
  "del",
  "code",
  "pre",
  "h1",
  "h2",
  "h3",
  "h4",
  "h5",
  "h6",
  "ul",
  "ol",
  "li",
  "blockquote",
  "hr",
  "table",
  "thead",
  "tbody",
  "tr",
  "th",
  "td",
  "a"
]

const ALLOWED_ATTRIBUTES = ["href", "title", "target", "rel", "data-language", "data-highlight"]

const FORBIDDEN_TAGS = ["script", "style", "iframe", "object", "embed", "form", "input", "link", "meta"]

marked.setOptions({
  gfm: true,
  breaks: true
})

const HIGHLIGHT_SPEC_PATTERN = /^\{([0-9,\s-]+)\}$/

const escapeHtml = (value: string): string =>
  value.replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;")

const parseFenceInfo = (info: string): { language: string; highlight: string } => {
  const parts = info.trim().split(/\s+/)
  const language = (parts[0] ?? "").replace(/[^A-Za-z0-9+#-]/g, "").toLowerCase()
  const spec = parts.slice(1).join(" ").trim()
  const match = spec.match(HIGHLIGHT_SPEC_PATTERN)
  return { language, highlight: match === null ? "" : match[1].replace(/\s+/g, "") }
}

const renderer = new marked.Renderer()

renderer.code = ({ text, lang }) => {
  const { language, highlight } = parseFenceInfo(lang ?? "")
  const languageClass = language.length > 0 ? ` class="language-${language}"` : ""
  const highlightAttribute = highlight.length > 0 ? ` data-highlight="${highlight}"` : ""
  return `<pre data-language="${language}"${highlightAttribute}><code${languageClass}>${escapeHtml(text)}</code></pre>\n`
}

export const renderMarkdown = (source: string): string => {
  if (source.trim().length === 0) {
    return ""
  }

  const parsed = marked.parse(source, { async: false, renderer })
  if (typeof parsed !== "string") {
    return ""
  }

  return DOMPurify.sanitize(parsed, {
    ALLOWED_TAGS,
    ALLOWED_ATTR: ALLOWED_ATTRIBUTES,
    FORBID_TAGS: FORBIDDEN_TAGS,
    ALLOW_DATA_ATTR: false
  })
}
