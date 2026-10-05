import { allSupportedModels } from "../../copilot/utils/models"

export interface AiModelOption {
  id: string
  name: string
}

const MODEL_PREFERENCE_STORAGE_KEY = "lunar_microservices_model_preferences"
const FREE_MODEL_ID_SUFFIX = "-free"

export function findModelOption(modelId: string): AiModelOption | null {
  const catalogModel = allSupportedModels.find((model) => model.id === modelId)
  if (catalogModel === undefined) {
    return null
  }
  return { id: catalogModel.id, name: catalogModel.name }
}

export function isKnownModelId(modelId: string): boolean {
  return allSupportedModels.some((model) => model.id === modelId)
}

export function isFreeModelId(modelId: string): boolean {
  return modelId.endsWith(FREE_MODEL_ID_SUFFIX)
}

export function readModelPreferences(): Record<string, string> {
  try {
    const storedValue = localStorage.getItem(MODEL_PREFERENCE_STORAGE_KEY)
    if (storedValue === null) {
      return {}
    }
    const parsedValue: unknown = JSON.parse(storedValue)
    if (typeof parsedValue !== "object" || parsedValue === null || Array.isArray(parsedValue)) {
      return {}
    }
    const preferences: Record<string, string> = {}
    for (const [repositoryName, modelId] of Object.entries(parsedValue as Record<string, unknown>)) {
      if (typeof modelId === "string" && isKnownModelId(modelId)) {
        preferences[repositoryName] = modelId
      }
    }
    return preferences
  } catch {
    return {}
  }
}

export function storeModelPreference(repositoryName: string, modelId: string): void {
  if (!isKnownModelId(modelId)) {
    return
  }
  const preferences = readModelPreferences()
  preferences[repositoryName] = modelId
  try {
    localStorage.setItem(MODEL_PREFERENCE_STORAGE_KEY, JSON.stringify(preferences))
  } catch {
    return
  }
}
