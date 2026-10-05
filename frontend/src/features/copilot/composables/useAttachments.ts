import { computed, onUnmounted, ref } from "vue"
import { formatErrorMessage } from "../../../lib/utils"
import { uploadCopilotAttachment } from "../api/attachment-api"
import type { AttachmentUpload } from "../api/attachment-api"
import {
  describeAttachmentRejection,
  isImageAttachmentFile,
  UPLOAD_FAILED_MESSAGE
} from "../utils/attachment-rules"

export type AttachmentDraftStatus = "uploading" | "ready" | "failed"

export interface AttachmentDraft {
  localId: string
  file: File
  isImage: boolean
  thumbnailUrl: string | null
  status: AttachmentDraftStatus
  upload: AttachmentUpload | null
  error: string | null
}

const DRAFT_ID_PREFIX = "attachment-draft-"

const createThumbnailUrl = (file: File): string | null => {
  if (!isImageAttachmentFile(file)) return null
  try {
    if (typeof URL === "undefined" || typeof URL.createObjectURL !== "function") return null
    return URL.createObjectURL(file)
  } catch {
    return null
  }
}

const releaseThumbnailUrl = (thumbnailUrl: string | null): void => {
  if (thumbnailUrl === null) return
  try {
    URL.revokeObjectURL(thumbnailUrl)
  } catch {
    return
  }
}

export const useAttachments = () => {
  const drafts = ref<AttachmentDraft[]>([])
  let draftSequence = 0

  const nextLocalId = (): string => {
    draftSequence += 1
    return DRAFT_ID_PREFIX + draftSequence
  }

  const replaceDraft = (localId: string, changes: Partial<AttachmentDraft>): void => {
    drafts.value = drafts.value.map((draft) =>
      draft.localId === localId ? { ...draft, ...changes } : draft
    )
  }

  const appendDraft = (draft: AttachmentDraft): void => {
    drafts.value = [...drafts.value, draft]
  }

  const uploadDraft = async (file: File, localId: string): Promise<void> => {
    try {
      const upload = await uploadCopilotAttachment(file)
      replaceDraft(localId, {
        status: "ready",
        upload,
        isImage: upload.kind === "image",
        error: null
      })
    } catch (thrownError) {
      replaceDraft(localId, {
        status: "failed",
        error: formatErrorMessage(thrownError, UPLOAD_FAILED_MESSAGE)
      })
    }
  }

  const addFile = (file: File): void => {
    const rejection = describeAttachmentRejection(file)
    if (rejection !== null) {
      appendDraft({
        localId: nextLocalId(),
        file,
        isImage: isImageAttachmentFile(file),
        thumbnailUrl: null,
        status: "failed",
        upload: null,
        error: rejection
      })
      return
    }

    const localId = nextLocalId()
    appendDraft({
      localId,
      file,
      isImage: isImageAttachmentFile(file),
      thumbnailUrl: createThumbnailUrl(file),
      status: "uploading",
      upload: null,
      error: null
    })
    void uploadDraft(file, localId)
  }

  const addFiles = (files: File[]): void => {
    files.forEach(addFile)
  }

  const removeDraft = (localId: string): void => {
    const removedDraft = drafts.value.find((draft) => draft.localId === localId)
    if (removedDraft !== undefined) {
      releaseThumbnailUrl(removedDraft.thumbnailUrl)
    }
    drafts.value = drafts.value.filter((draft) => draft.localId !== localId)
  }

  const clearDrafts = (): void => {
    drafts.value.forEach((draft) => releaseThumbnailUrl(draft.thumbnailUrl))
    drafts.value = []
  }

  onUnmounted(clearDrafts)

  const uploadingCount = computed(() =>
    drafts.value.filter((draft) => draft.status === "uploading").length
  )

  const failedCount = computed(() =>
    drafts.value.filter((draft) => draft.status === "failed").length
  )

  const hasImageDraft = computed(() => drafts.value.some((draft) => draft.isImage))

  const readyAttachmentIds = computed<string[]>(() =>
    drafts.value.flatMap((draft) =>
      draft.status === "ready" && draft.upload !== null ? [draft.upload.id] : []
    )
  )

  return {
    drafts,
    uploadingCount,
    failedCount,
    hasImageDraft,
    readyAttachmentIds,
    addFiles,
    removeDraft,
    clearDrafts
  }
}
