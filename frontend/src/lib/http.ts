export class ApiError extends Error {
  readonly statusCode: number
  readonly details?: unknown

  constructor(message: string, statusCode: number, details?: unknown) {
    super(message)
    this.name = "ApiError"
    this.statusCode = statusCode
    this.details = details
  }
}

const TOKEN_STORAGE_KEY = "lunar_auth_token"

export function getStoredToken(): string | null {
  try {
    return localStorage.getItem(TOKEN_STORAGE_KEY)
  } catch {
    return null
  }
}

export function setStoredToken(token: string): void {
  try {
    localStorage.setItem(TOKEN_STORAGE_KEY, token)
  } catch {
    return
  }
}

export function clearStoredToken(): void {
  try {
    localStorage.removeItem(TOKEN_STORAGE_KEY)
  } catch {
    return
  }
}

interface RequestOptions extends Omit<RequestInit, "body"> {
  body?: unknown
  params?: Record<string, string | number | boolean | undefined>
}

export async function request<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const { params, headers: customHeaders, body, ...fetchOptions } = options

  let url = path
  if (params) {
    const searchParams = new URLSearchParams()
    for (const [key, value] of Object.entries(params)) {
      if (value !== undefined) {
        searchParams.append(key, String(value))
      }
    }
    const queryString = searchParams.toString()
    if (queryString) {
      url += (url.includes("?") ? "&" : "?") + queryString
    }
  }

  const headers = new Headers(customHeaders)
  if (!headers.has("Accept")) {
    headers.set("Accept", "application/json")
  }

  const isFormData = typeof FormData !== "undefined" && body instanceof FormData
  if (!isFormData && body && !headers.has("Content-Type")) {
    headers.set("Content-Type", "application/json")
  }

  const token = getStoredToken()
  if (token && !headers.has("Authorization")) {
    headers.set("Authorization", `Bearer ${token}`)
  }

  const requestBody = isFormData || typeof body === "string" || body === undefined
    ? (body as BodyInit | undefined)
    : JSON.stringify(body)

  let response: Response
  try {
    response = await fetch(url, {
      ...fetchOptions,
      headers,
      body: requestBody
    })
  } catch (networkError) {
    const message = networkError instanceof Error ? networkError.message : "Network connection error"
    throw new ApiError(message, 0, networkError)
  }

  if (response.status === 204) {
    return {} as T
  }

  const contentType = response.headers.get("Content-Type") || ""
  const isJson = contentType.includes("application/json")
  const parsedData = isJson ? await response.json().catch(() => null) : await response.text().catch(() => null)

  if (!response.ok) {
    let errorMessage = "An unexpected error occurred"
    if (parsedData && typeof parsedData === "object") {
      if ("message" in parsedData && typeof parsedData.message === "string") {
        errorMessage = parsedData.message
      } else if ("error" in parsedData && typeof parsedData.error === "string") {
        errorMessage = parsedData.error
      }
    } else if (typeof parsedData === "string" && parsedData.length > 0) {
      errorMessage = parsedData
    } else if (response.statusText) {
      errorMessage = response.statusText
    }
    throw new ApiError(errorMessage, response.status, parsedData)
  }

  return parsedData as T
}

export const http = {
  get<T>(path: string, options?: RequestOptions): Promise<T> {
    return request<T>(path, { ...options, method: "GET" })
  },
  post<T>(path: string, body?: unknown, options?: RequestOptions): Promise<T> {
    return request<T>(path, { ...options, method: "POST", body })
  },
  put<T>(path: string, body?: unknown, options?: RequestOptions): Promise<T> {
    return request<T>(path, { ...options, method: "PUT", body })
  },
  delete<T>(path: string, options?: RequestOptions): Promise<T> {
    return request<T>(path, { ...options, method: "DELETE" })
  },
  patch<T>(path: string, body?: unknown, options?: RequestOptions): Promise<T> {
    return request<T>(path, { ...options, method: "PATCH", body })
  }
}
