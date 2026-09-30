export interface UserProfile {
  id: string | number
  email: string
  full_name?: string
  name?: string
  role?: string
  created_at?: string
  createdAt?: string
}

export interface LoginRequest {
  email: string
  password: string
  remember?: boolean
}

export interface RegisterRequest {
  email: string
  password: string
  full_name?: string
  name?: string
}

export interface AuthResponse {
  token: string
  user: UserProfile
}

export interface VerifyJiraPatRequest {
  pat: string
}

export interface VerifiedJiraProfile {
  display_name: string
  email: string
  username: string
}

export interface RegisterWithJiraRequest {
  pat: string
  password: string
}
