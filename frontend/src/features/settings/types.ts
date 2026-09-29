export interface UserSecretsInput {
  jira_pat?: string
  jira_username?: string
  bitbucket_pat?: string
  bitbucket_username?: string
  confluence_pat?: string
  ai_keys?: Record<string, string>
}

export interface RedactedSecrets {
  user_id: string
  has_jira_pat: boolean
  jira_username: string
  has_bitbucket_pat: boolean
  bitbucket_username: string
  has_confluence_pat: boolean
  has_ai_keys: boolean
  configured_ai_providers: string[]
  updated_at: string
}

export interface SystemConfig {
  jira_base_url: string
  bitbucket_base_url: string
  confluence_base_url: string
  system_ai_providers: string[]
}
