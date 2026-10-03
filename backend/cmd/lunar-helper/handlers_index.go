package main

import (
	"context"
	"net/http"
	"strings"
	"sync"

	codeindexapplication "lunar/backend/internal/codeindex/application"
	sharedHttp "lunar/backend/internal/shared/http"
)

const (
	indexStatusPath = "/index/status"
	indexBuildPath  = "/index/build"
	maxBuildRepos   = 128
)

type indexStatusResponse struct {
	Indexed         bool   `json:"indexed"`
	IsBuilding      bool   `json:"is_building"`
	RepoCount       int    `json:"repo_count"`
	FileCount       int    `json:"file_count"`
	ChunkCount      int    `json:"chunk_count"`
	UpdatedAt       string `json:"updated_at"`
	ProgressPercent int    `json:"progress_percent"`
}

type indexBuildRequest struct {
	Root    string   `json:"root"`
	Repos   []string `json:"repos"`
	Rebuild bool     `json:"rebuild"`
}

type indexBuildResponse struct {
	Started bool `json:"started"`
}

type indexedFileCounter interface {
	CountIndexedFiles(ctx context.Context) (int, error)
}

type indexBuildTracker struct {
	mutex     sync.Mutex
	building  bool
	completed int
	total     int
}

func (t *indexBuildTracker) start(total int) bool {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	if t.building {
		return false
	}
	t.building = true
	t.completed = 0
	t.total = total
	return true
}

func (t *indexBuildTracker) advance() {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	t.completed++
}

func (t *indexBuildTracker) finish() {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	t.building = false
	t.completed = 0
	t.total = 0
}

func (t *indexBuildTracker) snapshot() (bool, int) {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	if !t.building {
		return false, 0
	}
	if t.total <= 0 {
		return true, 0
	}
	percent := t.completed * 100 / t.total
	if percent > 100 {
		percent = 100
	}
	return true, percent
}

func (s *helperServer) handleIndexStatus(w http.ResponseWriter, r *http.Request) {
	if s.indexStore == nil {
		sharedHttp.WriteError(w, http.StatusServiceUnavailable, "the code index is not available")
		return
	}

	statuses, err := s.indexStore.GetIndexStatus(r.Context())
	if err != nil {
		s.logger.Error("cannot read the code index status", "error", err)
		sharedHttp.WriteError(w, http.StatusInternalServerError, "cannot read the code index status")
		return
	}

	response := indexStatusResponse{}
	for _, status := range statuses {
		if status.ChunkCount == 0 {
			continue
		}
		response.RepoCount++
		response.ChunkCount += status.ChunkCount
		if status.IndexedAt > response.UpdatedAt {
			response.UpdatedAt = status.IndexedAt
		}
	}
	response.Indexed = response.RepoCount > 0

	if counter, isCounter := s.indexStore.(indexedFileCounter); isCounter {
		fileCount, counterError := counter.CountIndexedFiles(r.Context())
		if counterError != nil {
			s.logger.Warn("cannot count indexed files", "error", counterError)
		} else {
			response.FileCount = fileCount
		}
	}

	response.IsBuilding, response.ProgressPercent = s.buildTracker.snapshot()
	sharedHttp.WriteJSON(w, http.StatusOK, response)
}

func (s *helperServer) handleIndexBuild(w http.ResponseWriter, r *http.Request) {
	if s.indexer == nil || s.indexStore == nil {
		sharedHttp.WriteError(w, http.StatusServiceUnavailable, "the code index is not available")
		return
	}

	var request indexBuildRequest
	if err := sharedHttp.ParseJSON(r, &request); err != nil {
		sharedHttp.WriteError(w, http.StatusBadRequest, "invalid index build request")
		return
	}

	rootPath := strings.TrimSpace(request.Root)
	if rootPath == "" {
		sharedHttp.WriteError(w, http.StatusBadRequest, "a repository root is required")
		return
	}
	if len(request.Repos) > maxBuildRepos {
		sharedHttp.WriteError(w, http.StatusBadRequest, "too many repositories requested")
		return
	}

	if !s.buildTracker.start(len(request.Repos)) {
		sharedHttp.WriteError(w, http.StatusConflict, "an index build is already running")
		return
	}

	repositories := append([]string(nil), request.Repos...)
	go s.runIndexBuild(rootPath, repositories, request.Rebuild)

	sharedHttp.WriteJSON(w, http.StatusAccepted, indexBuildResponse{Started: true})
}

func (s *helperServer) runIndexBuild(rootPath string, repositories []string, rebuild bool) {
	defer s.buildTracker.finish()

	if len(repositories) == 0 {
		results, err := s.indexer.IndexWorkspace(s.buildContext, rootPath, s.allowedRoots)
		if err != nil {
			s.logger.Warn("index build failed", "error", err)
			return
		}
		s.logIndexResults(results)
		return
	}

	for _, repoName := range repositories {
		if s.buildContext.Err() != nil {
			return
		}
		if rebuild {
			if err := s.indexStore.ClearRepo(s.buildContext, repoName); err != nil {
				s.logger.Warn("cannot clear repository before rebuild", "repository", repoName, "error", err)
			}
		}

		result, err := s.indexer.IndexRepository(s.buildContext, rootPath, repoName)
		if err != nil {
			s.logger.Warn("index build failed for repository", "repository", repoName, "error", err)
		} else {
			s.logIndexResult(result)
		}
		s.buildTracker.advance()
	}
}

func (s *helperServer) logIndexResults(results []codeindexapplication.IndexResult) {
	for _, result := range results {
		s.logIndexResult(result)
	}
}

func (s *helperServer) logIndexResult(result codeindexapplication.IndexResult) {
	s.logger.Info("indexed repository",
		"repository", result.RepoName,
		"indexed_files", result.IndexedFiles,
		"unchanged_files", result.UnchangedFiles,
		"removed_files", result.RemovedFiles,
		"chunks", result.ChunkCount)
}
