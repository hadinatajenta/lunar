import { applyQueryReview, requestQueryReview } from "../api/confluence-api"
import { createQueryReviewController, type QueryReviewController } from "./useQueryReview"

const queryReviewController = createQueryReviewController({
  request: (documentId, payload, signal) => requestQueryReview(documentId, payload, signal),
  apply: (documentId, repositories) => applyQueryReview(documentId, repositories)
})

export function useQueryReview(): QueryReviewController {
  return queryReviewController
}
