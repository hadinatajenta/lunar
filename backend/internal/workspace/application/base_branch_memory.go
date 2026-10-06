package application

import "sync"

type RememberedBaseMemory struct {
	mutex   sync.RWMutex
	choices map[string]map[string]string
}

func NewRememberedBaseMemory() *RememberedBaseMemory {
	return &RememberedBaseMemory{choices: make(map[string]map[string]string)}
}

func (m *RememberedBaseMemory) Remember(userID string, repoName string, baseBranch string) {
	if m == nil || userID == "" || repoName == "" || baseBranch == "" {
		return
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	userChoices := m.choices[userID]
	if userChoices == nil {
		userChoices = make(map[string]string)
		m.choices[userID] = userChoices
	}
	userChoices[repoName] = baseBranch
}

func (m *RememberedBaseMemory) Recall(userID string, repoName string) string {
	if m == nil {
		return ""
	}

	m.mutex.RLock()
	defer m.mutex.RUnlock()

	return m.choices[userID][repoName]
}
