import { ApiError } from "../../../lib/http"

export const describeRequestError = (error: unknown, fallback: string): string => {
  if (error instanceof ApiError) {
    return error.message.length > 0 ? error.message : fallback
  }
  if (error instanceof Error) {
    return error.message.length > 0 ? error.message : fallback
  }
  return fallback
}
