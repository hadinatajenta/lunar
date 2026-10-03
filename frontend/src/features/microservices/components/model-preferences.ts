export interface AiModelOption {
  id: string
  name: string
  icon: string
  description: string
}

export const AI_MODEL_OPTIONS: AiModelOption[] = [
  {
    id: "claude",
    name: "Claude Sonnet",
    icon: "C",
    description: "Best for structured specs and long OpenAPI definitions."
  },
  {
    id: "gpt",
    name: "GPT",
    icon: "O",
    description: "Balanced reasoning across routes and payloads."
  },
  {
    id: "gemini",
    name: "Gemini",
    icon: "G",
    description: "Fast generation with strong code understanding."
  },
  {
    id: "deepseek",
    name: "DeepSeek",
    icon: "D",
    description: "Cost-efficient for large repositories."
  },
  {
    id: "mimo",
    name: "MiMo",
    icon: "M",
    description: "Lightweight model for quick exports."
  }
]

const MODEL_PREFERENCE_STORAGE_KEY = "lunar_microservices_model_preferences"

export function isKnownModelId(modelId: string): boolean {
  return AI_MODEL_OPTIONS.some((model) => model.id === modelId)
}

export function findModelOption(modelId: string): AiModelOption | null {
  return AI_MODEL_OPTIONS.find((model) => model.id === modelId) ?? null
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
