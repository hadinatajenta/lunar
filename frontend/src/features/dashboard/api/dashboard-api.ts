import { http } from "../../../lib/http"
import type { DashboardSummary } from "../types"

export async function fetchDashboardSummary(): Promise<DashboardSummary> {
  return http.get<DashboardSummary>("/api/dashboard/summary")
}
