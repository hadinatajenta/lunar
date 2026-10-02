package config

import (
	"bufio"
	"os"
	"strings"
	"time"
)

type Config struct {
	Port                 string
	DBPath               string
	JWTSecret            string
	JWTTTL               time.Duration
	EncryptionKey        string
	CORSOrigin           string
	JiraBaseURL          string
	BitbucketBaseURL     string
	ConfluenceBaseURL    string
	SystemDeepseekKey    string
	SystemDeepseekURL    string
	SystemGeminiKey      string
	SystemOpenAIKey      string
	SystemClaudeKey      string
	SystemMimoKey        string
	SystemMimoBaseURL    string
	SeedUserEmail        string
	SeedUserPassword     string
	SeedUserName         string
}

func loadDotEnv() {
	candidates := []string{
		".env",
		"backend/.env",
		"../.env",
		"/opt/lunar/.env",
	}

	for _, path := range candidates {
		file, err := os.Open(path)
		if err != nil {
			continue
		}

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}

			parts := strings.SplitN(line, "=", 2)
			if len(parts) != 2 {
				continue
			}

			key := strings.TrimSpace(parts[0])
			value := strings.Trim(strings.TrimSpace(parts[1]), `"'`)

			_ = os.Setenv(key, value)
		}
		_ = file.Close()
		break
	}
}

func getEnvOrDefault(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

func LoadConfig() *Config {
	loadDotEnv()

	jwtSecret := getEnvOrDefault("JWT_SECRET", "lunar-default-secret-key-change-in-production")
	encryptionKey := getEnvOrDefault("ENCRYPTION_KEY", "0123456789abcdef0123456789abcdef")
	port := getEnvOrDefault("PORT", "8081")
	dbPath := getEnvOrDefault("DB_PATH", "data/lunar.db")
	corsOrigin := getEnvOrDefault("CORS_ORIGIN", "*")

	jiraBase := strings.TrimRight(getEnvOrDefault("JIRA_BASE_URL", "https://jira.bri.co.id"), "/")
	bitbucketBase := strings.TrimRight(getEnvOrDefault("BITBUCKET_BASE_URL", "https://bitbucket.bri.co.id"), "/")
	confluenceBase := strings.TrimRight(getEnvOrDefault("CONFLUENCE_BASE_URL", "https://confluence.bri.co.id"), "/")

	return &Config{
		Port:              port,
		DBPath:            dbPath,
		JWTSecret:         jwtSecret,
		JWTTTL:            24 * time.Hour,
		EncryptionKey:     encryptionKey,
		CORSOrigin:        corsOrigin,
		JiraBaseURL:       jiraBase,
		BitbucketBaseURL:  bitbucketBase,
		ConfluenceBaseURL: confluenceBase,
		SystemDeepseekKey: os.Getenv("SYSTEM_DEEPSEEK_KEY"),
		SystemDeepseekURL: getEnvOrDefault("SYSTEM_DEEPSEEK_BASE_URL", "https://api.deepseek.com"),
		SystemGeminiKey:   os.Getenv("SYSTEM_GEMINI_KEY"),
		SystemOpenAIKey:   os.Getenv("SYSTEM_OPENAI_KEY"),
		SystemClaudeKey:   os.Getenv("SYSTEM_CLAUDE_KEY"),
		SystemMimoKey:     os.Getenv("SYSTEM_MIMO_KEY"),
		SystemMimoBaseURL: os.Getenv("SYSTEM_MIMO_BASE_URL"),
		SeedUserEmail:     getEnvOrDefault("SEED_USER_EMAIL", "developer@lunar.dev"),
		SeedUserPassword:  getEnvOrDefault("SEED_USER_PASSWORD", "12345678"),
		SeedUserName:      getEnvOrDefault("SEED_USER_NAME", "Lunar Developer"),
	}
}
