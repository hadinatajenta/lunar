export const VM_REQUEST_TIMEOUT_MS = 15000
export const HELPER_REQUEST_TIMEOUT_MS = 8000

export class RequestTimeoutError extends Error {
  readonly timeoutMs: number

  constructor(timeoutMs: number) {
    super(`The request timed out after ${Math.round(timeoutMs / 1000)} seconds`)
    this.name = "RequestTimeoutError"
    this.timeoutMs = timeoutMs
  }
}

export async function withRequestTimeout<T>(
  timeoutMs: number,
  run: (signal: AbortSignal) => Promise<T>
): Promise<T> {
  const controller = new AbortController()
  const timeoutId = setTimeout(() => controller.abort(), timeoutMs)

  try {
    return await run(controller.signal)
  } catch (thrownError) {
    if (controller.signal.aborted) {
      throw new RequestTimeoutError(timeoutMs)
    }
    throw thrownError
  } finally {
    clearTimeout(timeoutId)
  }
}
