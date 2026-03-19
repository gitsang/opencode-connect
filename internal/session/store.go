package session

import (
	"maps"
	"sync"
)

type Store interface {
	Get(chatSessionID string) (string, bool)
	Set(chatSessionID string, opencodeSessionID string)
	ListBindings() map[string]string
}

type MemoryStore struct {
	mu       sync.RWMutex
	bindings map[string]string
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		bindings: map[string]string{},
	}
}

func (s *MemoryStore) Get(chatSessionID string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.bindings[chatSessionID]
	return v, ok
}

func (s *MemoryStore) Set(chatSessionID string, opencodeSessionID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.bindings[chatSessionID] = opencodeSessionID
}

func (s *MemoryStore) ListBindings() map[string]string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make(map[string]string, len(s.bindings))
	maps.Copy(out, s.bindings)
	return out
}
