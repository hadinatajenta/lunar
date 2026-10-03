import { computed, ref } from "vue"
import { DEFAULT_HELPER_URL } from "../../microservices/api/helper-api"
import { useMicroservices } from "../../microservices/composables/useMicroservices"
import type { HelperConnection } from "../../microservices/types"

export function useHelperConnection() {
  const {
    workspace,
    helperStatus,
    helperVersion,
    hasHelperToken,
    error,
    checkHelperHealth,
    connectHelper,
    validateRootPath,
    clearError
  } = useMicroservices()

  const helperUrl = ref("")
  const helperToken = ref("")
  const isCheckingHelper = ref(false)
  const isFolderChecking = ref(false)
  const isFolderValid = ref<boolean | null>(null)
  const folderValidationMessage = ref("")
  const helperTestMessage = ref("")

  const isHelperOnline = computed(() => helperStatus.value === "online")
  const isTokenSaved = computed(() => hasHelperToken.value)

  const tokenPlaceholder = computed(() =>
    isTokenSaved.value
      ? "•••••••••••• (saved, leave blank to keep it)"
      : `Paste the token from ~/.lunar/helper-token`
  )

  const tokenStatusMessage = computed(() => {
    if (isTokenSaved.value) {
      return "Token saved for this browser session. It is never displayed again."
    }
    if (workspace.value?.has_helper_token === true) {
      return "A token is stored on your account, but this browser session cannot use it yet. Enter it again to read repositories from the helper."
    }
    return "No helper token is stored yet."
  })

  const resolveConnectionOverride = (): HelperConnection | undefined => {
    const trimmedToken = helperToken.value.trim()
    if (trimmedToken.length === 0) {
      return undefined
    }
    return { helperUrl: helperUrl.value.trim(), helperToken: trimmedToken }
  }

  const resolveHelperUrl = (): string => helperUrl.value.trim() || DEFAULT_HELPER_URL

  const handleTestConnection = async () => {
    clearError()
    isCheckingHelper.value = true
    helperTestMessage.value = ""

    const isOnline = await checkHelperHealth(resolveConnectionOverride())
    if (isOnline) {
      const versionLabel = helperVersion.value.length > 0 ? ` version ${helperVersion.value}` : ""
      helperTestMessage.value = `The helper responded with${versionLabel}. Repository metadata can be read from this machine.`
    } else {
      helperTestMessage.value = error.value ?? "The helper did not respond."
    }

    isCheckingHelper.value = false
  }

  const handleConnectHelper = async () => {
    clearError()
    helperTestMessage.value = ""

    const isConnected = await connectHelper(resolveHelperUrl())
    if (isConnected) {
      helperTestMessage.value = "Connected. The helper token was stored for this browser session."
      return
    }
    helperTestMessage.value = error.value ?? "The helper did not respond."
  }

  const handleCheckHelper = async () => {
    clearError()
    helperTestMessage.value = ""
    await checkHelperHealth(resolveConnectionOverride())
  }

  const handleValidateFolder = async (rootPath: string) => {
    clearError()
    folderValidationMessage.value = ""
    isFolderValid.value = null

    if (rootPath.trim().length === 0) {
      folderValidationMessage.value = "Enter the folder that holds your repositories."
      return
    }

    isFolderChecking.value = true
    const validation = await validateRootPath(rootPath.trim(), resolveConnectionOverride())
    isFolderValid.value = validation.valid
    folderValidationMessage.value = validation.valid
      ? "The helper can read this folder."
      : validation.reason || "The helper could not find this folder on this machine."
    isFolderChecking.value = false
  }

  const resetFolderValidation = () => {
    isFolderValid.value = null
    folderValidationMessage.value = ""
  }

  return {
    helperUrl,
    helperToken,
    isCheckingHelper,
    isFolderChecking,
    isFolderValid,
    folderValidationMessage,
    helperTestMessage,
    isHelperOnline,
    isTokenSaved,
    tokenPlaceholder,
    tokenStatusMessage,
    handleTestConnection,
    handleConnectHelper,
    handleCheckHelper,
    handleValidateFolder,
    resetFolderValidation
  }
}
