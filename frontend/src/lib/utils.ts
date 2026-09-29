export function formatErrorMessage(error: unknown, fallbackMessage = "An error occurred"): string {
  if (error instanceof Error) {
    return error.message
  }
  if (typeof error === "string") {
    return error
  }
  return fallbackMessage
}
