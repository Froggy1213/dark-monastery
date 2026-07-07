package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"dark-monastery/internal/memory"
)

// FileStore — файловое хранилище игровых сессий в JSON.
type FileStore struct {
	saveDir string
	mem     *MemoryStore // in-memory кеш для быстрого доступа
}

// NewFileStore создаёт файловое хранилище. Создаёт директорию, если её нет.
func NewFileStore(saveDir string) (*FileStore, error) {
	if err := os.MkdirAll(saveDir, 0755); err != nil {
		return nil, fmt.Errorf("не могу создать директорию сохранений %s: %w", saveDir, err)
	}
	return &FileStore{
		saveDir: saveDir,
		mem:     NewMemoryStore(),
	}, nil
}

// SaveDir возвращает путь к директории сохранений.
func (s *FileStore) SaveDir() string { return s.saveDir }

// Save сохраняет состояние игры в JSON-файл.
func (s *FileStore) Save(sessionID string, state interface{}, meta *SaveMeta) error {
	return s.SaveWithHistory(sessionID, state, meta, nil)
}

// SaveWithHistory сохраняет состояние и историю ходов в JSON-файл.
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
		return fmt.Errorf("ошибка сериализации: %w", err)
	}

	if err := os.WriteFile(savePath, data, 0644); err != nil {
		return fmt.Errorf("ошибка записи файла: %w", err)
	}

	// Обновляем кеш в памяти
	jsonState, _ := json.Marshal(state)
	_ = s.mem.Save(sessionID, jsonState, meta)
	return nil
}

// Load загружает состояние игры из JSON-файла.
func (s *FileStore) Load(sessionID string, state interface{}) (*SaveMeta, error) {
	meta, _, err := s.LoadWithHistory(sessionID, state)
	return meta, err
}

// LoadWithHistory загружает состояние и историю ходов из JSON-файла.
func (s *FileStore) LoadWithHistory(sessionID string, state interface{}) (*SaveMeta, []memory.TurnRecord, error) {
	savePath := filepath.Join(s.saveDir, sessionID+".json")
	data, err := os.ReadFile(savePath)
	if err != nil {
		return nil, nil, fmt.Errorf("не могу прочитать сохранение: %w", err)
	}

	var sf saveFile
	if err := json.Unmarshal(data, &sf); err != nil {
		return nil, nil, fmt.Errorf("ошибка десериализации: %w", err)
	}

	// Восстанавливаем состояние
	stateBytes, _ := json.Marshal(sf.State)
	if err := json.Unmarshal(stateBytes, state); err != nil {
		return nil, nil, fmt.Errorf("ошибка восстановления состояния: %w", err)
	}

	// Кешируем в память
	_ = s.mem.Save(sessionID, stateBytes, sf.Meta)

	return sf.Meta, sf.History, nil
}

// List возвращает список всех сохранений.
func (s *FileStore) List() ([]*SaveMeta, error) {
	entries, err := os.ReadDir(s.saveDir)
	if err != nil {
		return nil, fmt.Errorf("не могу прочитать директорию сохранений: %w", err)
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

// Delete удаляет сохранение.
func (s *FileStore) Delete(sessionID string) error {
	_ = s.mem.Delete(sessionID)
	savePath := filepath.Join(s.saveDir, sessionID+".json")
	if err := os.Remove(savePath); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// saveFile — структура для хранения в JSON-файле.
type saveFile struct {
	Meta    *SaveMeta           `json:"meta"`
	State   interface{}         `json:"state"`
	History []memory.TurnRecord `json:"history,omitempty"`
}
