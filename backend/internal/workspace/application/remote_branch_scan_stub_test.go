package application

import (
	"context"
	"errors"
	"sync"
	"time"
)

var errRemoteUnreachable = errors.New("the remote is unreachable")

type stubRemoteBranchScanner struct {
	mutex        sync.Mutex
	calls        []string
	listCalls    []string
	branches     map[string][]string
	listBranches map[string][]string
	errs         map[string]error
	listErrs     map[string]error
	searchDelay  time.Duration
	listDelay    time.Duration
	beforeReturn func()
	inFlight     int
	maxInFlight  int
}

var _ interface {
	SearchBranches(ctx context.Context, userID string, repo string, filterText string) ([]string, error)
	ListBranches(ctx context.Context, userID string, repo string) ([]string, error)
	DefaultBranch(ctx context.Context, userID string, repo string) (string, error)
} = (*stubRemoteBranchScanner)(nil)

func (s *stubRemoteBranchScanner) SearchBranches(ctx context.Context, userID string, repo string, filterText string) ([]string, error) {
	s.mutex.Lock()
	s.calls = append(s.calls, repo)
	s.inFlight++
	if s.inFlight > s.maxInFlight {
		s.maxInFlight = s.inFlight
	}
	s.mutex.Unlock()

	defer func() {
		s.mutex.Lock()
		s.inFlight--
		s.mutex.Unlock()
	}()

	if s.searchDelay > 0 {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(s.searchDelay):
		}
	}
	if s.beforeReturn != nil {
		s.beforeReturn()
	}
	if err := s.errs[repo]; err != nil {
		return nil, err
	}

	return s.branches[repo], nil
}

func (s *stubRemoteBranchScanner) ListBranches(ctx context.Context, userID string, repo string) ([]string, error) {
	s.mutex.Lock()
	s.listCalls = append(s.listCalls, repo)
	s.inFlight++
	if s.inFlight > s.maxInFlight {
		s.maxInFlight = s.inFlight
	}
	s.mutex.Unlock()

	defer func() {
		s.mutex.Lock()
		s.inFlight--
		s.mutex.Unlock()
	}()

	if s.listDelay > 0 {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(s.listDelay):
		}
	}
	if s.beforeReturn != nil {
		s.beforeReturn()
	}
	if err := s.listErrs[repo]; err != nil {
		return nil, err
	}

	return s.listBranches[repo], nil
}

func (s *stubRemoteBranchScanner) DefaultBranch(ctx context.Context, userID string, repo string) (string, error) {
	return "", nil
}

func (s *stubRemoteBranchScanner) calledRepositories() []string {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	called := make([]string, len(s.calls))
	copy(called, s.calls)
	return called
}

func (s *stubRemoteBranchScanner) calledListRepositories() []string {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	called := make([]string, len(s.listCalls))
	copy(called, s.listCalls)
	return called
}

func (s *stubRemoteBranchScanner) peakInFlight() int {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	return s.maxInFlight
}
