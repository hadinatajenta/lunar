const DEFAULT_BACKEND_URL = "http://localhost:8081"

export const resolveBackendURL = (): string => {
  const configured = import.meta.env.VITE_LUNAR_BACKEND_URL
  if (typeof configured === "string" && configured.trim().length > 0) {
    return configured.trim().replace(/\/+$/, "")
  }
  return DEFAULT_BACKEND_URL
}
