import { allSupportedModels } from "./models"

export const ATTACH_CONTROL_LABEL = "Attach files"
export const UPLOAD_FAILED_MESSAGE = "The file could not be uploaded"
export const ACCEPTED_TYPES_MESSAGE = "Accepted file types are PNG, JPEG, WebP, PDF, XLSX, XLS and DOCX"
export const MAX_ATTACHMENT_SIZE_LABEL = "10 MB"
export const ATTACHMENT_ACCEPT_ATTRIBUTE = "image/png,image/jpeg,image/webp,application/pdf,.xlsx,.xls,.docx"

const MAX_ATTACHMENT_BYTES = 10 * 1024 * 1024

const ACCEPTED_MIME_TYPES = ["image/png", "image/jpeg", "image/webp", "application/pdf"]
const ACCEPTED_FILE_EXTENSIONS = ["png", "jpg", "jpeg", "webp", "pdf", "xlsx", "xls", "docx"]
const IMAGE_FILE_EXTENSIONS = ["png", "jpg", "jpeg", "webp"]
const IMAGE_CAPABLE_MODEL_IDS = ["deepseek-flash"]
const BYTE_UNITS = ["B", "KB", "MB"]
const BYTES_PER_UNIT = 1024
const UNIT_DECIMAL_FACTOR = 10

const resolveFileExtension = (fileName: string): string => {
  const lastDotIndex = fileName.lastIndexOf(".")
  if (lastDotIndex < 0) return ""
  return fileName.slice(lastDotIndex + 1).toLowerCase()
}

const isAcceptedAttachmentFile = (file: File): boolean => {
  if (ACCEPTED_FILE_EXTENSIONS.includes(resolveFileExtension(file.name))) return true
  return ACCEPTED_MIME_TYPES.includes(file.type)
}

export const isImageAttachmentFile = (file: File): boolean => {
  if (file.type.startsWith("image/")) return true
  return IMAGE_FILE_EXTENSIONS.includes(resolveFileExtension(file.name))
}

export const describeAttachmentRejection = (file: File): string | null => {
  if (!isAcceptedAttachmentFile(file)) {
    return `Unsupported file type. ${ACCEPTED_TYPES_MESSAGE}`
  }
  if (file.size > MAX_ATTACHMENT_BYTES) {
    return `The file is larger than ${MAX_ATTACHMENT_SIZE_LABEL}`
  }
  return null
}

export const formatAttachmentSize = (bytes: number): string => {
  if (!Number.isFinite(bytes) || bytes < 0) return "-"
  let value = bytes
  let unitIndex = 0
  while (value >= BYTES_PER_UNIT && unitIndex < BYTE_UNITS.length - 1) {
    value /= BYTES_PER_UNIT
    unitIndex += 1
  }
  const roundedValue = unitIndex === 0
    ? Math.round(value)
    : Math.round(value * UNIT_DECIMAL_FACTOR) / UNIT_DECIMAL_FACTOR
  return `${roundedValue} ${BYTE_UNITS[unitIndex]}`
}

export const modelAcceptsImages = (modelName: string): boolean => {
  const matchedModel = allSupportedModels.find(
    (model) => model.name === modelName || model.id === modelName
  )
  if (matchedModel === undefined) return false
  return IMAGE_CAPABLE_MODEL_IDS.includes(matchedModel.id)
}

export const imageCapableModelNames = (): string[] =>
  allSupportedModels
    .filter((model) => IMAGE_CAPABLE_MODEL_IDS.includes(model.id))
    .map((model) => model.name)
