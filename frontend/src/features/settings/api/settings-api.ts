import { http } from "../../../lib/http"
import type { RedactedSecrets, SystemConfig, UserSecretsInput } from "../types"

export const getSystemConfig = async (): Promise<SystemConfig> => {
  return http.get<SystemConfig>("/api/config")
}

export const getUserSecrets = async (): Promise<RedactedSecrets> => {
  return http.get<RedactedSecrets>("/api/auth/secrets")
}

export const saveUserSecrets = async (secrets: UserSecretsInput): Promise<RedactedSecrets> => {
  return http.put<RedactedSecrets>("/api/auth/secrets", secrets)
}
