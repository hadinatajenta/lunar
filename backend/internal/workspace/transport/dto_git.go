package transport

import (
	"lunar/backend/internal/workspace/domain"
)

type branchResponse struct {
	Name            string `json:"name"`
	IsCurrent       bool   `json:"is_current"`
	UpdatedRelative string `json:"updated_relative"`
}

type branchesResponse struct {
	Branches []branchResponse `json:"branches"`
}

type checkoutBranchRequest struct {
	Name   string `json:"name"`
	Branch string `json:"branch"`
}

type repositoryNameRequest struct {
	Name string `json:"name"`
}

type checkoutBranchResponse struct {
	Branch string `json:"branch"`
}

type fetchResponse struct {
	Ahead  int `json:"ahead"`
	Behind int `json:"behind"`
}

type pullResponse struct {
	Branch          string `json:"branch"`
	UpdatedRelative string `json:"updated_relative"`
}

type stashResponse struct {
	StashCount int `json:"stash_count"`
}

type discardResponse struct {
	RestoredCount int `json:"restored_count"`
}

func toBranchesResponse(branches []domain.Branch) branchesResponse {
	branchResponses := make([]branchResponse, 0, len(branches))
	for _, branch := range branches {
		branchResponses = append(branchResponses, branchResponse{
			Name:            branch.Name,
			IsCurrent:       branch.IsCurrent,
			UpdatedRelative: branch.UpdatedRelative,
		})
	}

	return branchesResponse{Branches: branchResponses}
}
