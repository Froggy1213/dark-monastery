package storage

import (
	"fmt"
	"sync"
	"time"
)

// MemoryStore is an in-memory storage for game sessions.
type MemoryStore struct {
	mu    sync.RWMutex
	games map[string][]byte // sessionID -> JSON GameState
	metas map[string]*SaveMeta
}

// NewMemoryStore creates a new in-memory storage.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		games: make(map[string][]byte),
		metas: make(map[string]*SaveMeta),
	}
}

// Save saves the game state to memory.
func (s *MemoryStore) Save(sessionID string, stateJSON []byte, meta *SaveMeta) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.games[sessionID] = stateJSON
	meta.UpdatedAt = time.Now()
	s.metas[sessionID] = meta
	return nil
}

// Load loads the game state from memory.
func (s *MemoryStore) Load(sessionID string) ([]byte, *SaveMeta, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	data, ok := s.games[sessionID]
	if !ok {
		return nil, nil, fmt.Errorf("save %s not found", sessionID)
	}
	return data, s.metas[sessionID], nil
}

// List returns a list of all saves.
func (s *MemoryStore) List() []*SaveMeta {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*SaveMeta, 0, len(s.metas))
	for _, m := range s.metas {
		result = append(result, m)
	}
	return result
}

// Delete deletes a save.
func (s *MemoryStore) Delete(sessionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.games, sessionID)
	delete(s.metas, sessionID)
	return nil
}
