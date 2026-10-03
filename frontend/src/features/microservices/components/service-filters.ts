export type ServiceFilter = "all" | "dirty" | "clean"

export interface ServiceFilterOption {
  value: ServiceFilter
  label: string
}

export const SERVICE_FILTER_OPTIONS: ServiceFilterOption[] = [
  { value: "all", label: "All" },
  { value: "dirty", label: "Dirty" },
  { value: "clean", label: "Clean" }
]
