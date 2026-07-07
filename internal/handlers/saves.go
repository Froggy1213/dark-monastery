package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"dark-monastery/internal/game"
	"dark-monastery/internal/storage"
)

// --- Обработчики сохранений ---

func (s *Server) handleSaves(w http.ResponseWriter, r *http.Request) {
	saves, err := s.fileStore.List()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if saves == nil {
		saves = []*storage.SaveMeta{}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"saves": saves})
}

func (s *Server) handleSave(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Метод не поддерживается", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		SessionID string `json:"session_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Невалидный JSON"})
		return
	}

	s.mu.RLock()
	sess, ok := s.sessions[req.SessionID]
	s.mu.RUnlock()

	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "Сессия не найдена"})
		return
	}

	meta := &storage.SaveMeta{
		SessionID:  req.SessionID,
		PlayerName: game.DefaultPlayerName,
		Location:   sess.State.Location,
		TurnCount:  sess.TurnCount,
	}
	if err := s.fileStore.SaveWithHistory(req.SessionID, sess.State, meta, memoryToStorageRecords(sess.Memory)); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "session_id": req.SessionID})
}

func (s *Server) handleLoad(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Метод не поддерживается", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		SessionID string `json:"session_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Невалидный JSON"})
		return
	}

	state := &game.GameState{}
	meta, records, err := s.fileStore.LoadWithHistory(req.SessionID, state)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "Сохранение не найдено: " + err.Error()})
		return
	}

	mem := s.createMemoryManager(req.SessionID)
	if mem != nil {
		// Восстанавливаем краткосрочную память из последних записей файлового сохранения
		for _, r := range records {
			mem.ShortTermHistory().Add(r.PlayerAction, r.AIResponse)
		}
		if meta != nil {
			mem.SetTurnCounter(meta.TurnCount)
		}
	}

	turnCount := 0
	if meta != nil {
		turnCount = meta.TurnCount
	}

	sess := &Session{
		State:     state,
		Memory:    mem,
		TurnCount: turnCount,
		CreatedAt: time.Now(),
	}

	s.mu.Lock()
	s.sessions[req.SessionID] = sess
	s.mu.Unlock()

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"session_id": req.SessionID,
		"state":      state,
		"turn_count": sess.TurnCount,
	})
}
