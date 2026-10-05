import { http } from "../../../lib/http"

export type AttachmentKind = "image" | "document"

export interface AttachmentUpload {
  id: string
  name: string
  kind: AttachmentKind
  size: number
  preview: string
}

const ATTACHMENT_UPLOAD_ENDPOINT = "/api/copilot/attachments"
const ATTACHMENT_UPLOAD_FIELD = "file"

export const uploadCopilotAttachment = async (file: File): Promise<AttachmentUpload> => {
  const formData = new FormData()
  formData.append(ATTACHMENT_UPLOAD_FIELD, file, file.name)
  return http.post<AttachmentUpload>(ATTACHMENT_UPLOAD_ENDPOINT, formData)
}
