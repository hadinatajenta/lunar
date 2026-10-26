package domain

type ApplyQueryReviewRequest struct {
	Repositories []ApplyRepository `json:"repositories"`
}

type ApplyRepository struct {
	Repository string       `json:"repository"`
	Queries    []ApplyQuery `json:"queries"`
}

type ApplyQuery struct {
	No             int    `json:"no"`
	Feature        string `json:"feature"`
	QueryStatement string `json:"query_statement"`
	Index          string `json:"index"`
	CreateIndex    string `json:"create_index"`
}

type ApplyQueryReviewResult struct {
	PageID  string `json:"page_id"`
	Version int    `json:"version"`
	URL     string `json:"url"`
}
