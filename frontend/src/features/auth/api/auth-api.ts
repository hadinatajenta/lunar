import { http } from "@/lib/http"
import type { ApiResponse } from "@/types/api"
import type { AuthResponse, LoginRequest, RegisterRequest, UserProfile } from "../types"

export async function loginUser(payload: LoginRequest): Promise<AuthResponse> {
  const response = await http.post<ApiResponse<AuthResponse> | AuthResponse>("/api/auth/login", payload)
  if ("data" in response && response.data) {
    return response.data
  }
  return response as AuthResponse
}

export async function registerUser(payload: RegisterRequest): Promise<AuthResponse> {
  const response = await http.post<ApiResponse<AuthResponse> | AuthResponse>("/api/auth/register", payload)
  if ("data" in response && response.data) {
    return response.data
  }
  return response as AuthResponse
}

export async function getCurrentUser(): Promise<UserProfile> {
  const response = await http.get<ApiResponse<UserProfile> | UserProfile>("/api/auth/me")
  if ("data" in response && response.data) {
    return response.data
  }
  return response as UserProfile
}

export async function logoutUser(): Promise<void> {
  try {
    await http.post<void>("/api/auth/logout")
  } catch {
    return
  }
}
