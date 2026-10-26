const STORAGE_TAG_PATTERN = /<[a-z!/][^>]*>/i

export const isMarkdownSopBlock = (block: string): boolean => {
  const trimmed = block.trim()
  return trimmed.length > 0 && !STORAGE_TAG_PATTERN.test(trimmed)
}

export const sopBlockToText = (block: string): string =>
  block
    .replace(/<br\s*\/?>/gi, "\n")
    .replace(/<\/(p|li|h[1-6]|tr|div|td|th)>/gi, "\n")
    .replace(/<[^>]+>/g, "")
    .replace(/&nbsp;/g, " ")
    .replace(/&lt;/g, "<")
    .replace(/&gt;/g, ">")
    .replace(/&quot;/g, '"')
    .replace(/&#39;/g, "'")
    .replace(/&amp;/g, "&")
    .replace(/[ \t]+\n/g, "\n")
    .replace(/\n{3,}/g, "\n\n")
    .trim()
