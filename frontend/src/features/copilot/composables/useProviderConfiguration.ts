import { computed, ref, type Ref } from "vue"
import { http } from "../../../lib/http"
import { getConfiguredModels, getDefaultModel, isModelConfigured } from "../utils/models"

const configuredProviders = ref<string[]>([])
const isCheckingProviders = ref(true)

const collectProviderNames = (...sources: Array<string[] | undefined>): string[] =>
  Array.from(new Set(sources.flatMap((source) => source ?? []).map((name) => name.toLowerCase())))

export const useProviderConfiguration = (selectedModel: Ref<string>) => {
  const refreshConfiguredProviders = async (): Promise<void> => {
    isCheckingProviders.value = true
    try {
      const [secretsResp, configResp] = await Promise.all([
        http.get<{ configured_ai_providers?: string[]; has_ai_keys?: boolean }>("/api/auth/secrets").catch(() => null),
        http.get<{ system_ai_providers?: string[] }>("/api/config").catch(() => null)
      ])

      configuredProviders.value = collectProviderNames(secretsResp?.configured_ai_providers, configResp?.system_ai_providers)

      if (!isModelConfigured(selectedModel.value, configuredProviders.value)) {
        selectedModel.value = getDefaultModel(configuredProviders.value)
      }
    } catch {
      configuredProviders.value = []
    } finally {
      isCheckingProviders.value = false
    }
  }

  const hasConfiguredAI = computed(() => configuredProviders.value.length > 0)

  const configuredModels = computed(() => getConfiguredModels(configuredProviders.value))

  return { configuredProviders, isCheckingProviders, hasConfiguredAI, configuredModels, refreshConfiguredProviders }
}
