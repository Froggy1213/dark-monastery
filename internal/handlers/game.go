package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"dark-monastery/internal/ai"
	"dark-monastery/internal/game"
	"dark-monastery/internal/storage"
)

// --- Обработчики игровых действий ---

func (s *Server) handleNewGame(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Метод не поддерживается", http.StatusMethodNotAllowed)
		return
	}

	state := game.NewPlayer()
	sessionID := fmt.Sprintf("session-%d", time.Now().UnixNano())

	sess := &Session{
		State:     state,
		Memory:    s.createMemoryManager(sessionID),
		TurnCount: 0,
		CreatedAt: time.Now(),
	}

	s.mu.Lock()
	s.sessions[sessionID] = sess
	s.mu.Unlock()

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"session_id": sessionID,
		"state":      state,
		"turn_count": 0,
	})
}

func (s *Server) handleAction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Метод не поддерживается", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		SessionID string `json:"session_id"`
		Action    string `json:"action"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Невалидный JSON"})
		return
	}

	if req.SessionID == "" || req.Action == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "session_id и action обязательны"})
		return
	}

	s.mu.RLock()
	sess, ok := s.sessions[req.SessionID]
	s.mu.RUnlock()

	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "Сессия не найдена"})
		return
	}

	newState, err := s.processActionWithMemory(r.Context(), sess, req.Action)
	if err != nil {
		log.Printf("Ошибка AI: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Ошибка нейросети: " + err.Error()})
		return
	}

	sess.TurnCount++

	// RAG: сохраняем в обе памяти
	if sess.Memory != nil {
		sess.Memory.Add(req.Action, newState.Message, newState.Location)
	}

	sess.State = newState

	// Автосохранение (файловое)
	meta := &storage.SaveMeta{
		SessionID:  req.SessionID,
		PlayerName: game.DefaultPlayerName,
		Location:   newState.Location,
		TurnCount:  sess.TurnCount,
	}
	_ = s.fileStore.SaveWithHistory(req.SessionID, newState, meta, memoryToStorageRecords(sess.Memory))

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"state":      newState,
		"turn_count": sess.TurnCount,
	})
}

func (s *Server) handleState(w http.ResponseWriter, r *http.Request) {
	sessionID := r.URL.Query().Get("session_id")
	if sessionID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "session_id обязателен"})
		return
	}

	s.mu.RLock()
	sess, ok := s.sessions[sessionID]
	s.mu.RUnlock()

	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "Сессия не найдена"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"state":      sess.State,
		"turn_count": sess.TurnCount,
	})
}

// processActionWithMemory выполняет действие с RAG-контекстом.
// Используется в handleAction и handleWebSocket.
func (s *Server) processActionWithMemory(ctx context.Context, sess *Session, action string) (*game.GameState, error) {
	// RAG: строим контекст памяти
	if sess.Memory != nil {
		memCtx, err := sess.Memory.BuildContext(ctx, action, sess.State.Location)
		if err != nil {
			log.Printf("[ProcessAction] Ошибка BuildContext: %v", err)
		}
		if gemini, ok := s.engine.AIClient().(*ai.GeminiClient); ok {
			gemini.SetMemoryContext(memCtx)
		}
	}

	return s.engine.ProcessTurn(sess.State, action)
}
