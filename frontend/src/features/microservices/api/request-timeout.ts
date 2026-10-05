export const VM_REQUEST_TIMEOUT_MS = 15000
export const HELPER_REQUEST_TIMEOUT_MS = 8000
export const GIT_ACTION_TIMEOUT_MS = 120000
export const COLLECTION_EXPORT_TIMEOUT_MS = 300000

export class RequestTimeoutError extends Error {
  readonly timeoutMs: number

  constructor(timeoutMs: number) {
    super(`The request timed out after ${Math.round(timeoutMs / 1000)} seconds`)
    this.name = "RequestTimeoutError"
    this.timeoutMs = timeoutMs
  }
}

export async function withRequestSignalTimeout<T>(
  timeoutMs: number,
  externalSignal: AbortSignal | undefined,
  run: (signal: AbortSignal) => Promise<T>
): Promise<T> {
  const controller = new AbortController()
  let hasTimedOut = false
  const timeoutId = setTimeout(() => {
    hasTimedOut = true
    controller.abort()
  }, timeoutMs)
  const abortFromExternalSignal = () => controller.abort()
  if (externalSignal?.aborted === true) {
    controller.abort()
  }
  externalSignal?.addEventListener("abort", abortFromExternalSignal)

  try {
    return await run(controller.signal)
  } catch (thrownError) {
    if (controller.signal.aborted && hasTimedOut) {
      throw new RequestTimeoutError(timeoutMs)
    }
    throw thrownError
  } finally {
    clearTimeout(timeoutId)
    externalSignal?.removeEventListener("abort", abortFromExternalSignal)
  }
}

export async function withRequestTimeout<T>(
  timeoutMs: number,
  run: (signal: AbortSignal) => Promise<T>
): Promise<T> {
  return withRequestSignalTimeout(timeoutMs, undefined, run)
}
