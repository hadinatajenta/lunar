const ISSUE_KEY_PATTERN = /\b([A-Z][A-Z0-9]{1,9}-\d{1,6})\b/

export const readIssueKey = (title: string): string | null => {
  const match = title.match(ISSUE_KEY_PATTERN)
  return match === null ? null : match[1]
}

export const hasIssueKey = (title: string): boolean => readIssueKey(title) !== null
