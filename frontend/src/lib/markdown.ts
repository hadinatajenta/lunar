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

const ALLOWED_ATTRIBUTES = ["href", "title", "target", "rel"]

const FORBIDDEN_TAGS = ["script", "style", "iframe", "object", "embed", "form", "input", "link", "meta"]

marked.setOptions({
  gfm: true,
  breaks: true
})

export const renderMarkdown = (source: string): string => {
  if (source.trim().length === 0) {
    return ""
  }

  const parsed = marked.parse(source, { async: false })
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
