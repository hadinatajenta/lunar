import type { ModelOption } from "../types"

export const allSupportedModels: ModelOption[] = [
  {
    id: "deepseek-v4-pro",
    name: "DeepSeek-V4 Pro (Thinking)",
    provider: "deepseek",
    description: "DeepSeek chain-of-thought high reasoning model",
    hasThinking: true
  },
  {
    id: "deepseek-flash",
    name: "DeepSeek Flash",
    provider: "deepseek",
    description: "Fast low-latency code and general response model",
    hasThinking: false
  },
  {
    id: "gpt-6-astra",
    name: "GPT-6 Astra (Reasoning)",
    provider: "openai",
    description: "OpenAI high-reasoning coding model",
    hasThinking: true
  },
  {
    id: "gpt-6-sol",
    name: "GPT-6 Sol",
    provider: "openai",
    description: "OpenAI general development model",
    hasThinking: false
  },
  {
    id: "claude-opus-5-5",
    name: "Claude Opus 5.5 (Adaptive Thinking)",
    provider: "claude",
    description: "Anthropic adaptive systems thinking model",
    hasThinking: true
  },
  {
    id: "claude-sonnet-5-5",
    name: "Claude Sonnet 5.5",
    provider: "claude",
    description: "Anthropic high-accuracy engineering assistant",
    hasThinking: false
  },
  {
    id: "claude-fable-5-1",
    name: "Claude Fable 5.1 (Deep Reasoning)",
    provider: "claude",
    description: "Anthropic deep reasoning model",
    hasThinking: true
  },
  {
    id: "gemini-3-8-flash",
    name: "Gemini 3.8 Flash (Extended Thinking)",
    provider: "gemini",
    description: "Google multimodal extended thinking model",
    hasThinking: true
  },
  {
    id: "gemini-3-1-pro",
    name: "Gemini 3.1 Pro",
    provider: "gemini",
    description: "Google enterprise engineering model",
    hasThinking: false
  },
  {
    id: "mimo-v2-5-pro",
    name: "Xiaomi MiMo-V2.5 Pro",
    provider: "mimo",
    description: "Xiaomi specialized engineering reasoning model",
    hasThinking: true
  }
]

export const isProviderConfigured = (
  provider: string,
  configuredProviders: string[] = []
): boolean => {
  const normalized = configuredProviders.map((p) => p.toLowerCase())
  return normalized.includes(provider.toLowerCase())
}

export const isModelConfigured = (
  modelName: string,
  configuredProviders: string[] = []
): boolean => {
  const found = allSupportedModels.find((m) => m.name === modelName || m.id === modelName)
  if (!found) return false
  return isProviderConfigured(found.provider, configuredProviders)
}

export const getConfiguredModels = (
  configuredProviders: string[] = []
): ModelOption[] => {
  return allSupportedModels.filter((m) =>
    isProviderConfigured(m.provider, configuredProviders)
  )
}

export const getDefaultModel = (
  configuredProviders: string[] = []
): string => {
  const configured = getConfiguredModels(configuredProviders)
  if (configured.length > 0) {
    return configured[0].name
  }
  return "DeepSeek-V4 Pro (Thinking)"
}
