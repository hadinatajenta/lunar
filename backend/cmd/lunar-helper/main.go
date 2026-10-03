package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	codeindexapplication "lunar/backend/internal/codeindex/application"
	"lunar/backend/internal/config"
	"lunar/backend/internal/workspace/infrastructure"
)

const (
	helperVersion      = "1.0.0"
	helperAddressEnv   = "LUNAR_HELPER_ADDR"
	helperOriginsEnv   = "LUNAR_HELPER_ORIGINS"
	serverReadTimeout  = 15 * time.Second
	serverWriteTimeout = 60 * time.Second
	serverIdleTimeout  = 120 * time.Second
	shutdownTimeout    = 10 * time.Second
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	slog.SetDefault(logger)

	tokenPath, err := helperTokenPath()
	if err != nil {
		logger.Error("cannot resolve the helper token path", "error", err)
		os.Exit(1)
	}

	token, err := loadOrCreateToken(tokenPath)
	if err != nil {
		logger.Error("cannot load the helper token", "error", err)
		os.Exit(1)
	}

	workspaceConfig := config.LoadConfig()
	inspector := infrastructure.NewGitInspector(workspaceConfig.WorkspaceAllowedRoots)

	indexStore, indexError := openHelperIndex()
	if indexError != nil {
		logger.Warn("the code index is unavailable, so code search endpoints are disabled", "error", indexError)
	}

	dependencies := helperDependencies{
		reader:         inspector,
		allowedRoots:   workspaceConfig.WorkspaceAllowedRoots,
		token:          token,
		allowedOrigins: parseOriginList(os.Getenv(helperOriginsEnv)),
		version:        helperVersion,
		logger:         logger,
	}
	if indexStore != nil {
		dependencies.indexStore = indexStore
		dependencies.indexer = codeindexapplication.NewIndexer(indexStore)
		defer func() {
			if err := indexStore.Close(); err != nil {
				logger.Warn("cannot close the code index", "error", err)
			}
		}()
	}

	handler, stopHelper := newHelperHandler(dependencies)
	defer stopHelper()

	address := resolveListenAddress(os.Getenv(helperAddressEnv))
	server := &http.Server{
		Addr:              address,
		Handler:           handler,
		ReadHeaderTimeout: serverReadTimeout,
		ReadTimeout:       serverReadTimeout,
		WriteTimeout:      serverWriteTimeout,
		IdleTimeout:       serverIdleTimeout,
	}

	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, os.Interrupt, syscall.SIGTERM)

	serverErrors := make(chan error, 1)
	go func() {
		logger.Info("lunar helper listening", "address", address, "token_path", tokenPath)
		serverErrors <- server.ListenAndServe()
	}()

	select {
	case <-stopChan:
		logger.Info("lunar helper shutting down")
	case err := <-serverErrors:
		if !errors.Is(err, http.ErrServerClosed) {
			logger.Error("lunar helper server failure", "error", err)
			os.Exit(1)
		}
	}

	shutdownContext, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := server.Shutdown(shutdownContext); err != nil {
		logger.Error("lunar helper graceful shutdown failed", "error", err)
		os.Exit(1)
	}

	logger.Info("lunar helper stopped")
}
