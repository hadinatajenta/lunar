package app

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"time"

	"lunar/backend/internal/auth/application"
	"lunar/backend/internal/auth/domain"
	"lunar/backend/internal/auth/infrastructure"
	"lunar/backend/internal/auth/transport"
	bitbucketApp "lunar/backend/internal/bitbucket/application"
	bitbucketInfra "lunar/backend/internal/bitbucket/infrastructure"
	bitbucketTransport "lunar/backend/internal/bitbucket/transport"
	"lunar/backend/internal/config"
	confluenceApp "lunar/backend/internal/confluence/application"
	confluenceInfra "lunar/backend/internal/confluence/infrastructure"
	confluenceTransport "lunar/backend/internal/confluence/transport"
	copilotApp "lunar/backend/internal/copilot/application"
	copilotInfra "lunar/backend/internal/copilot/infrastructure"
	copilotTransport "lunar/backend/internal/copilot/transport"
	dashboardApp "lunar/backend/internal/dashboard/application"
	dashboardTransport "lunar/backend/internal/dashboard/transport"
	jiraApp "lunar/backend/internal/jira/application"
	jiraInfra "lunar/backend/internal/jira/infrastructure"
	jiraTransport "lunar/backend/internal/jira/transport"
	sharedAuth "lunar/backend/internal/shared/auth"
	sharedDatabase "lunar/backend/internal/shared/database"
	sharedHttp "lunar/backend/internal/shared/http"
)

type Application struct {
	Server *http.Server
	DB     *sql.DB
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start))
	})
}

func NewApplication(cfg *config.Config) (*Application, error) {
	db, err := sharedDatabase.OpenDB(cfg.DBPath)
	if err != nil {
		return nil, err
	}

	if err := sharedDatabase.AutoMigrate(db); err != nil {
		_ = db.Close()
		return nil, err
	}

	userRepo := infrastructure.NewSQLiteUserRepository(db)
	vaultRepo := infrastructure.NewSQLiteVaultRepository(db)

	authService := application.NewAuthService(
		userRepo,
		vaultRepo,
		cfg.EncryptionKey,
		cfg.JWTSecret,
		cfg.JWTTTL,
	)

	seedInitialUser(userRepo, authService, cfg)

	authHandler := transport.NewAuthHandler(authService)

	systemAIProviders := map[string]string{
		"deepseek": cfg.SystemDeepseekKey,
		"gemini":   cfg.SystemGeminiKey,
		"openai":   cfg.SystemOpenAIKey,
		"claude":   cfg.SystemClaudeKey,
		"mimo":     cfg.SystemMimoKey,
	}

	copilotRepo := copilotInfra.NewSQLiteCopilotRepository(db)
	llmClient := copilotInfra.NewLLMClient(cfg.SystemDeepseekURL, cfg.SystemMimoBaseURL)
	copilotService := copilotApp.NewCopilotService(copilotRepo, llmClient, authService, systemAIProviders)
	copilotHandler := copilotTransport.NewCopilotHandler(copilotService)

	bitbucketClient := bitbucketInfra.NewAtlassianBitbucketClient(cfg.BitbucketBaseURL)
	bitbucketService := bitbucketApp.NewBitbucketService(bitbucketClient, authService, copilotService)
	bitbucketHandler := bitbucketTransport.NewBitbucketHandler(bitbucketService)

	jiraClient := jiraInfra.NewAtlassianJiraClient(cfg.JiraBaseURL)
	authService.SetJiraVerifier(&jiraVerifierAdapter{client: jiraClient})
	jiraService := jiraApp.NewJiraService(jiraClient, authService)
	jiraHandler := jiraTransport.NewJiraHandler(jiraService)

	confluenceClient := confluenceInfra.NewConfluenceClient(cfg.ConfluenceBaseURL)
	confluenceService := confluenceApp.NewConfluenceService(confluenceClient, authService, jiraService)
	confluenceHandler := confluenceTransport.NewConfluenceHandler(confluenceService)

	dashboardService := dashboardApp.NewDashboardService(jiraService, bitbucketService, copilotService, authService)
	dashboardHandler := dashboardTransport.NewDashboardHandler(dashboardService)

	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		sharedHttp.WriteJSON(w, http.StatusOK, map[string]string{"status": "healthy"})
	})

	mux.HandleFunc("GET /api/config", func(w http.ResponseWriter, r *http.Request) {
		var systemAI []string
		if cfg.SystemDeepseekKey != "" {
			systemAI = append(systemAI, "deepseek")
		}
		if cfg.SystemGeminiKey != "" {
			systemAI = append(systemAI, "gemini")
		}
		if cfg.SystemOpenAIKey != "" {
			systemAI = append(systemAI, "openai")
		}
		if cfg.SystemClaudeKey != "" {
			systemAI = append(systemAI, "claude")
		}
		if cfg.SystemMimoKey != "" {
			systemAI = append(systemAI, "mimo")
		}

		sharedHttp.WriteJSON(w, http.StatusOK, map[string]any{
			"jira_base_url":       cfg.JiraBaseURL,
			"bitbucket_base_url":  cfg.BitbucketBaseURL,
			"confluence_base_url": cfg.ConfluenceBaseURL,
			"system_ai_providers": systemAI,
		})
	})

	mux.HandleFunc("POST /api/auth/register", authHandler.Register)
	mux.HandleFunc("POST /api/auth/login", authHandler.Login)
	mux.HandleFunc("POST /api/auth/verify-jira-pat", authHandler.VerifyJiraPAT)
	mux.HandleFunc("POST /api/auth/register-with-jira", authHandler.RegisterWithJira)

	authMiddleware := sharedAuth.RequireAuth(cfg.JWTSecret)
	mux.Handle("GET /api/auth/me", authMiddleware(http.HandlerFunc(authHandler.Me)))
	mux.Handle("GET /api/auth/secrets", authMiddleware(http.HandlerFunc(authHandler.GetSecrets)))
	mux.Handle("PUT /api/auth/secrets", authMiddleware(http.HandlerFunc(authHandler.SaveSecrets)))

	mux.Handle("GET /api/copilot/models", authMiddleware(http.HandlerFunc(copilotHandler.ListModels)))
	mux.Handle("POST /api/copilot/chat", authMiddleware(http.HandlerFunc(copilotHandler.Chat)))
	mux.Handle("GET /api/copilot/sessions", authMiddleware(http.HandlerFunc(copilotHandler.ListSessions)))
	mux.Handle("DELETE /api/copilot/sessions/{id}", authMiddleware(http.HandlerFunc(copilotHandler.DeleteSession)))
	mux.Handle("GET /api/copilot/sessions/{id}/messages", authMiddleware(http.HandlerFunc(copilotHandler.ListMessages)))

	mux.Handle("GET /api/bitbucket/pushes", authMiddleware(http.HandlerFunc(bitbucketHandler.ListPushes)))
	mux.Handle("GET /api/bitbucket/prs", authMiddleware(http.HandlerFunc(bitbucketHandler.ListPullRequests)))
	mux.Handle("GET /api/bitbucket/prs/{id}/diff", authMiddleware(http.HandlerFunc(bitbucketHandler.GetDiff)))
	mux.Handle("POST /api/bitbucket/prs", authMiddleware(http.HandlerFunc(bitbucketHandler.CreatePullRequest)))
	mux.Handle("POST /api/bitbucket/prs/{id}/ai-review", authMiddleware(http.HandlerFunc(bitbucketHandler.GenerateAIReview)))
	mux.Handle("POST /api/bitbucket/prs/{id}/comments", authMiddleware(http.HandlerFunc(bitbucketHandler.PostComment)))
	mux.Handle("POST /api/bitbucket/prs/{id}/action", authMiddleware(http.HandlerFunc(bitbucketHandler.ApplyAction)))
	mux.HandleFunc("POST /api/bitbucket/reset", bitbucketHandler.Reset)

	mux.Handle("GET /api/jira/issues", authMiddleware(http.HandlerFunc(jiraHandler.ListMyIssues)))
	mux.Handle("GET /api/jira/backlog", authMiddleware(http.HandlerFunc(jiraHandler.GetBacklog)))
	mux.Handle("GET /api/jira/issues/remotelinks", authMiddleware(http.HandlerFunc(jiraHandler.ListIssueRemoteLinks)))
	mux.Handle("GET /api/jira/issues/{key}", authMiddleware(http.HandlerFunc(jiraHandler.GetIssueDetail)))

	mux.Handle("GET /api/confluence/documents", authMiddleware(http.HandlerFunc(confluenceHandler.ListDocuments)))
	mux.Handle("GET /api/confluence/documents/{id}", authMiddleware(http.HandlerFunc(confluenceHandler.GetDocument)))

	mux.Handle("GET /api/dashboard/summary", authMiddleware(http.HandlerFunc(dashboardHandler.GetSummary)))

	handler := sharedHttp.CORSMiddleware(cfg.CORSOrigin)(loggingMiddleware(mux))

	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return &Application{
		Server: server,
		DB:     db,
	}, nil
}

func (a *Application) Shutdown(ctx context.Context) error {
	serverErr := a.Server.Shutdown(ctx)
	dbErr := a.DB.Close()
	if serverErr != nil {
		return serverErr
	}
	return dbErr
}

func seedInitialUser(userRepo domain.UserRepository, authService *application.AuthService, cfg *config.Config) {
	if cfg.SeedUserEmail == "" {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	existing, err := userRepo.GetUserByEmail(ctx, cfg.SeedUserEmail)
	if err == nil && existing != nil {
		return
	}

	_, _ = authService.Register(ctx, application.RegisterRequest{
		Email:    cfg.SeedUserEmail,
		Password: cfg.SeedUserPassword,
		FullName: cfg.SeedUserName,
	})
}

type jiraVerifierAdapter struct {
	client *jiraInfra.AtlassianJiraClient
}

func (a *jiraVerifierAdapter) VerifyPAT(ctx context.Context, pat string) (*application.JiraProfile, error) {
	u, err := a.client.VerifyPAT(ctx, pat)
	if err != nil {
		return nil, err
	}
	return &application.JiraProfile{
		DisplayName: u.DisplayName,
		Email:       u.EmailAddress,
		Username:    u.Name,
	}, nil
}
