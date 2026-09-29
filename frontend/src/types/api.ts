export interface ApiResponse<T = unknown> {
  success: boolean
  message?: string
  data?: T
  error?: string
}

export interface ApiErrorDetail {
  field?: string
  message: string
}

export interface ApiErrorResponse {
  success: false
  message: string
  errors?: ApiErrorDetail[]
  statusCode?: number
}
