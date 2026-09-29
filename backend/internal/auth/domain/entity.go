package domain

import (
	"time"
)

type User struct {
	ID           string
	Email        string
	PasswordHash string
	Salt         string
	FullName     string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (u *User) ToPublic() UserPublicProfile {
	return UserPublicProfile{
		ID:        u.ID,
		Email:     u.Email,
		FullName:  u.FullName,
		CreatedAt: u.CreatedAt,
	}
}

type UserPublicProfile struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	FullName  string    `json:"full_name"`
	CreatedAt time.Time `json:"created_at"`
}

type UserSecrets struct {
	UserID            string
	JiraPATEnc        string
	JiraUsername      string
	BitbucketPATEnc   string
	BitbucketUsername string
	ConfluencePATEnc  string
	AIKeysJSONEnc     string
	UpdatedAt         time.Time
}

type RedactedSecrets struct {
	UserID                string    `json:"user_id"`
	HasJiraPAT            bool      `json:"has_jira_pat"`
	JiraUsername          string    `json:"jira_username"`
	HasBitbucketPAT       bool      `json:"has_bitbucket_pat"`
	BitbucketUsername     string    `json:"bitbucket_username"`
	HasConfluencePAT      bool      `json:"has_confluence_pat"`
	HasAIKeys             bool      `json:"has_ai_keys"`
	ConfiguredAIProviders []string  `json:"configured_ai_providers"`
	UpdatedAt             time.Time `json:"updated_at"`
}

type DecryptedSecrets struct {
	UserID            string            `json:"user_id"`
	JiraPAT           string            `json:"jira_pat"`
	JiraUsername      string            `json:"jira_username"`
	BitbucketPAT      string            `json:"bitbucket_pat"`
	BitbucketUsername string            `json:"bitbucket_username"`
	ConfluencePAT     string            `json:"confluence_pat"`
	AIKeys            map[string]string `json:"ai_keys"`
	UpdatedAt         time.Time         `json:"updated_at"`
}

type SaveSecretsInput struct {
	JiraPAT           string            `json:"jira_pat"`
	JiraUsername      string            `json:"jira_username"`
	BitbucketPAT      string            `json:"bitbucket_pat"`
	BitbucketUsername string            `json:"bitbucket_username"`
	ConfluencePAT     string            `json:"confluence_pat"`
	AIKeys            map[string]string `json:"ai_keys"`
}
