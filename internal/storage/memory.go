package storage

import (
	"fmt"
	"sync"
	"time"
)

// MemoryStore — in-memory хранилище игровых сессий.
type MemoryStore struct {
	mu     sync.RWMutex
	games  map[string][]byte // sessionID → JSON GameState
	metas  map[string]*SaveMeta
}

// NewMemoryStore создаёт новое in-memory хранилище.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		games: make(map[string][]byte),
		metas: make(map[string]*SaveMeta),
	}
}

// Save сохраняет состояние игры в память.
func (s *MemoryStore) Save(sessionID string, stateJSON []byte, meta *SaveMeta) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.games[sessionID] = stateJSON
	meta.UpdatedAt = time.Now()
	s.metas[sessionID] = meta
	return nil
}

// Load загружает состояние игры из памяти.
func (s *MemoryStore) Load(sessionID string) ([]byte, *SaveMeta, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	data, ok := s.games[sessionID]
	if !ok {
		return nil, nil, fmt.Errorf("сохранение %s не найдено", sessionID)
	}
	return data, s.metas[sessionID], nil
}

// List возвращает список всех сохранений.
func (s *MemoryStore) List() []*SaveMeta {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*SaveMeta, 0, len(s.metas))
	for _, m := range s.metas {
		result = append(result, m)
	}
	return result
}

// Delete удаляет сохранение.
func (s *MemoryStore) Delete(sessionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.games, sessionID)
	delete(s.metas, sessionID)
	return nil
}
