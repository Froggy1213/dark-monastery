package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"dark-monastery/internal/memory"
)

// FileStore is a file-based storage for game sessions in JSON.
type FileStore struct {
	saveDir string
	mem     *MemoryStore // in-memory cache for fast access
}

// NewFileStore creates a file-based storage. Creates the directory if it doesn't exist.
func NewFileStore(saveDir string) (*FileStore, error) {
	if err := os.MkdirAll(saveDir, 0755); err != nil {
		return nil, fmt.Errorf("cannot create save directory %s: %w", saveDir, err)
	}
	return &FileStore{
		saveDir: saveDir,
		mem:     NewMemoryStore(),
	}, nil
}

// SaveDir returns the path to the save directory.
func (s *FileStore) SaveDir() string { return s.saveDir }

// Save saves the game state to a JSON file.
func (s *FileStore) Save(sessionID string, state interface{}, meta *SaveMeta) error {
	return s.SaveWithHistory(sessionID, state, meta, nil)
}

// SaveWithHistory saves the state and turn history to a JSON file.
func (s *FileStore) SaveWithHistory(sessionID string, state interface{}, meta *SaveMeta, history []memory.TurnRecord) error {
	meta.UpdatedAt = time.Now()
	meta.SessionID = sessionID

	sf := saveFile{
		Meta:  meta,
		State: state,
	}
	if history != nil {
		sf.History = history
	}

	savePath := filepath.Join(s.saveDir, sessionID+".json")
	data, err := json.MarshalIndent(sf, "", "  ")
	if err != nil {
		return fmt.Errorf("serialization error: %w", err)
	}

	if err := os.WriteFile(savePath, data, 0644); err != nil {
		return fmt.Errorf("file write error: %w", err)
	}

	// Update in-memory cache
	jsonState, _ := json.Marshal(state)
	_ = s.mem.Save(sessionID, jsonState, meta)
	return nil
}

// Load loads the game state from a JSON file.
func (s *FileStore) Load(sessionID string, state interface{}) (*SaveMeta, error) {
	meta, _, err := s.LoadWithHistory(sessionID, state)
	return meta, err
}

// LoadWithHistory loads the state and turn history from a JSON file.
func (s *FileStore) LoadWithHistory(sessionID string, state interface{}) (*SaveMeta, []memory.TurnRecord, error) {
	savePath := filepath.Join(s.saveDir, sessionID+".json")
	data, err := os.ReadFile(savePath)
	if err != nil {
		return nil, nil, fmt.Errorf("cannot read save file: %w", err)
	}

	var sf saveFile
	if err := json.Unmarshal(data, &sf); err != nil {
		return nil, nil, fmt.Errorf("deserialization error: %w", err)
	}

	// Restore state
	stateBytes, _ := json.Marshal(sf.State)
	if err := json.Unmarshal(stateBytes, state); err != nil {
		return nil, nil, fmt.Errorf("state restoration error: %w", err)
	}

	// Cache in memory
	_ = s.mem.Save(sessionID, stateBytes, sf.Meta)

	return sf.Meta, sf.History, nil
}

// List returns a list of all saves.
func (s *FileStore) List() ([]*SaveMeta, error) {
	entries, err := os.ReadDir(s.saveDir)
	if err != nil {
		return nil, fmt.Errorf("cannot read save directory: %w", err)
	}

	var metas []*SaveMeta
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		sessionID := entry.Name()[:len(entry.Name())-5]
		data, err := os.ReadFile(filepath.Join(s.saveDir, entry.Name()))
		if err != nil {
			continue
		}
		var sf saveFile
		if err := json.Unmarshal(data, &sf); err != nil {
			continue
		}
		if sf.Meta != nil {
			sf.Meta.SessionID = sessionID
			metas = append(metas, sf.Meta)
		}
	}
	return metas, nil
}

// Delete deletes a save.
func (s *FileStore) Delete(sessionID string) error {
	_ = s.mem.Delete(sessionID)
	savePath := filepath.Join(s.saveDir, sessionID+".json")
	if err := os.Remove(savePath); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// saveFile is the structure for storing in a JSON file.
type saveFile struct {
	Meta    *SaveMeta           `json:"meta"`
	State   interface{}         `json:"state"`
	History []memory.TurnRecord `json:"history,omitempty"`
}
