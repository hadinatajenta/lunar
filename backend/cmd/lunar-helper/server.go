package main

import (
	"context"
	"crypto/subtle"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	codeindexapplication "lunar/backend/internal/codeindex/application"
	codeindexdomain "lunar/backend/internal/codeindex/domain"
	sharedHttp "lunar/backend/internal/shared/http"
	"lunar/backend/internal/workspace/domain"
)

const (
	healthPath       = "/health"
	repositoriesPath = "/repositories"
	servicesPath     = "/services"
	validatePathPath = "/validate-path"
	tokenPath        = "/token"

	authorizationHeader = "Authorization"
	bearerScheme        = "Bearer"
	originHeader        = "Origin"
	rootQueryParameter  = "root"
	namesQueryParameter = "names"

	corsRequestMethods = "GET, POST, OPTIONS"
	corsAllowedHeaders = "Authorization, Content-Type"

	loopbackHost      = "127.0.0.1"
	defaultHelperAddr = "127.0.0.1:5199"
	maxPortNumber     = 65535
)

var defaultHelperOrigins = []string{"http://localhost:5174", "http://127.0.0.1:5174"}

type helperServer struct {
	reader       domain.GitReader
	indexStore   codeindexdomain.IndexStore
	indexer      *codeindexapplication.Indexer
	allowedRoots []string
	token        string
	version      string
	logger       *slog.Logger
	buildTracker indexBuildTracker
	buildContext context.Context
	buildCancel  context.CancelFunc
}

type helperDependencies struct {
	reader         domain.GitReader
	indexStore     codeindexdomain.IndexStore
	indexer        *codeindexapplication.Indexer
	allowedRoots   []string
	token          string
	allowedOrigins []string
	version        string
	logger         *slog.Logger
}

func newHelperHandler(dependencies helperDependencies) (http.Handler, func()) {
	buildContext, buildCancel := context.WithCancel(context.Background())
	server := &helperServer{
		reader:       dependencies.reader,
		indexStore:   dependencies.indexStore,
		indexer:      dependencies.indexer,
		allowedRoots: dependencies.allowedRoots,
		token:        dependencies.token,
		version:      dependencies.version,
		logger:       dependencies.logger,
		buildContext: buildContext,
		buildCancel:  buildCancel,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET "+healthPath, server.handleHealth)
	mux.HandleFunc("GET "+tokenPath, server.handleToken)
	mux.Handle("GET "+repositoriesPath, requireToken(dependencies.token, http.HandlerFunc(server.handleRepositories)))
	mux.Handle("GET "+servicesPath, requireToken(dependencies.token, http.HandlerFunc(server.handleServices)))
	mux.Handle("POST "+validatePathPath, requireToken(dependencies.token, http.HandlerFunc(server.handleValidatePath)))
	mux.Handle("GET "+indexStatusPath, requireToken(dependencies.token, http.HandlerFunc(server.handleIndexStatus)))
	mux.Handle("POST "+indexBuildPath, requireToken(dependencies.token, http.HandlerFunc(server.handleIndexBuild)))
	mux.Handle("GET "+searchPath, requireToken(dependencies.token, http.HandlerFunc(server.handleSearch)))
	mux.Handle("POST "+agentAskPath, requireToken(dependencies.token, http.HandlerFunc(server.handleAgentAsk)))

	handler := loggingMiddleware(dependencies.logger)(corsMiddleware(dependencies.allowedOrigins)(mux))
	return handler, server.stop
}

func (s *helperServer) stop() {
	if s.buildCancel != nil {
		s.buildCancel()
	}
}

func requireToken(token string, next http.Handler) http.Handler {
	expected := []byte(token)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		presented, found := bearerToken(r.Header.Get(authorizationHeader))
		if !found || subtle.ConstantTimeCompare([]byte(presented), expected) != 1 {
			sharedHttp.WriteError(w, http.StatusUnauthorized, messageUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func bearerToken(header string) (string, bool) {
	scheme, value, found := strings.Cut(strings.TrimSpace(header), " ")
	if !found || !strings.EqualFold(scheme, bearerScheme) {
		return "", false
	}

	token := strings.TrimSpace(value)
	if token == "" {
		return "", false
	}
	return token, true
}

func corsMiddleware(allowedOrigins []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get(originHeader)
			if origin != "" {
				if !isAllowedOrigin(origin, allowedOrigins) {
					sharedHttp.WriteError(w, http.StatusForbidden, messageOriginNotAllowed)
					return
				}

				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Methods", corsRequestMethods)
				w.Header().Set("Access-Control-Allow-Headers", corsAllowedHeaders)
				w.Header().Add("Vary", originHeader)
			}

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func isAllowedOrigin(origin string, allowedOrigins []string) bool {
	for _, allowed := range allowedOrigins {
		if allowed == origin {
			return true
		}
	}
	return false
}

func loggingMiddleware(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			startedAt := time.Now()
			recorder := &statusRecorder{ResponseWriter: w, status: http.StatusOK}

			next.ServeHTTP(recorder, r)

			logger.Info(
				"lunar helper request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", recorder.status,
				"duration_ms", time.Since(startedAt).Milliseconds(),
			)
		})
	}
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func (r *statusRecorder) Flush() {
	flusher, isFlushable := r.ResponseWriter.(http.Flusher)
	if !isFlushable {
		return
	}
	flusher.Flush()
}

func parseOriginList(rawOrigins string) []string {
	origins := make([]string, 0)
	for _, entry := range strings.Split(rawOrigins, ",") {
		trimmed := strings.TrimSpace(entry)
		if trimmed != "" {
			origins = append(origins, trimmed)
		}
	}

	if len(origins) == 0 {
		return append([]string(nil), defaultHelperOrigins...)
	}
	return origins
}

func resolveListenAddress(rawAddress string) string {
	trimmed := strings.TrimSpace(rawAddress)
	if trimmed == "" {
		return defaultHelperAddr
	}

	host, port, err := net.SplitHostPort(trimmed)
	if err != nil {
		if isPortNumber(trimmed) {
			return net.JoinHostPort(loopbackHost, trimmed)
		}
		return defaultHelperAddr
	}

	if !isPortNumber(port) {
		return defaultHelperAddr
	}

	if host == loopbackHost || isLoopbackHost(host) {
		return net.JoinHostPort(host, port)
	}
	return net.JoinHostPort(loopbackHost, port)
}

func isPortNumber(value string) bool {
	port, err := strconv.Atoi(value)
	if err != nil {
		return false
	}
	return port >= 0 && port <= maxPortNumber
}

func isLoopbackHost(host string) bool {
	parsed := net.ParseIP(host)
	return parsed != nil && parsed.IsLoopback()
}
