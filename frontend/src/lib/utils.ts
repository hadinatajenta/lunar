export function formatErrorMessage(error: unknown, fallbackMessage = "An error occurred"): string {
  if (error instanceof Error) {
    return error.message
  }
  if (typeof error === "string") {
    return error
  }
  return fallbackMessage
}

export function formatRelativeTime(isoDate: string): string {
  if (!isoDate) {
    return "-"
  }
  const timestamp = new Date(isoDate).getTime()
  if (Number.isNaN(timestamp)) {
    return "-"
  }
  const elapsedMinutes = Math.floor((Date.now() - timestamp) / 60000)
  if (elapsedMinutes < 1) {
    return "just now"
  }
  if (elapsedMinutes < 60) {
    return `${elapsedMinutes}m ago`
  }
  const elapsedHours = Math.floor(elapsedMinutes / 60)
  if (elapsedHours < 24) {
    return `${elapsedHours}h ago`
  }
  const elapsedDays = Math.floor(elapsedHours / 24)
  if (elapsedDays < 7) {
    return elapsedDays === 1 ? "1 day ago" : `${elapsedDays} days ago`
  }
  const elapsedWeeks = Math.floor(elapsedDays / 7)
  if (elapsedWeeks < 5) {
    return elapsedWeeks === 1 ? "1 week ago" : `${elapsedWeeks} weeks ago`
  }
  const elapsedMonths = Math.floor(elapsedDays / 30)
  if (elapsedMonths < 12) {
    return elapsedMonths === 1 ? "1 month ago" : `${elapsedMonths} months ago`
  }
  const elapsedYears = Math.floor(elapsedDays / 365)
  return elapsedYears === 1 ? "1 year ago" : `${elapsedYears} years ago`
}
