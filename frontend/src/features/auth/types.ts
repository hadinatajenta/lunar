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
