package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"dark-monastery/internal/ai"
	"dark-monastery/internal/game"
	"dark-monastery/internal/memory"
	"dark-monastery/internal/storage"
)

// Server хранит все зависимости для HTTP API.
type Server struct {
	engine      *game.Engine
	fileStore   *storage.FileStore
	lore        *game.LoreBook
	sessions    map[string]*Session
	mu          sync.RWMutex

	// RAG компоненты (nil если PostgreSQL недоступен)
	pgStore     *storage.PgStore
	embedClient *ai.EmbeddingClient
	extractor   *memory.Extractor
}

// Session — активная игровая сессия.
type Session struct {
	State     *game.GameState      `json:"state"`
	Memory    *memory.MemoryManager `json:"-"`
	TurnCount int                  `json:"turn_count"`
	CreatedAt time.Time            `json:"created_at"`
}

// NewServer создаёт новый HTTP сервер с игровым движком.
// pgStore, embedClient, extractor могут быть nil — тогда используется legacy режим.
func NewServer(
	engine *game.Engine,
	fileStore *storage.FileStore,
	lore *game.LoreBook,
	pgStore *storage.PgStore,
	embedClient *ai.EmbeddingClient,
	extractor *memory.Extractor,
) *Server {
	return &Server{
		engine:      engine,
		fileStore:   fileStore,
		lore:        lore,
		sessions:    make(map[string]*Session),
		pgStore:     pgStore,
		embedClient: embedClient,
		extractor:   extractor,
	}
}

// createMemoryManager создаёт MemoryManager для сессии.
// Если RAG-компоненты недоступны, возвращает nil.
func (s *Server) createMemoryManager(sessionID string) *memory.MemoryManager {
	if s.pgStore == nil || s.embedClient == nil {
		return nil
	}
	return memory.NewMemoryManager(s.pgStore, s.embedClient, s.extractor, sessionID)
}

// RegisterRoutes регистрирует все эндпоинты на переданном ServeMux.
func (s *Server) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/game/new", s.handleNewGame)
	mux.HandleFunc("/api/game/action", s.handleAction)
	mux.HandleFunc("/api/game/state", s.handleState)
	mux.HandleFunc("/api/game/saves", s.handleSaves)
	mux.HandleFunc("/api/game/save", s.handleSave)
	mux.HandleFunc("/api/game/load", s.handleLoad)
	mux.HandleFunc("/ws/game", s.handleWebSocket)
}

// --- Обработчики ---

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

	// RAG: строим контекст памяти перед вызовом AI
	if sess.Memory != nil {
		ctx := r.Context()
		memCtx, err := sess.Memory.BuildContext(ctx, req.Action, sess.State.Location)
		if err != nil {
			log.Printf("[Action] Ошибка BuildContext: %v", err)
		}
		// Устанавливаем контекст памяти в Gemini клиент
		if gemini, ok := s.engine.AIClient().(*ai.GeminiClient); ok {
			gemini.SetMemoryContext(memCtx)
		}
	}

	newState, err := s.engine.ProcessTurn(sess.State, req.Action)
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
		PlayerName: "Игрок",
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
		PlayerName: "Игрок",
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

// --- Вспомогательные функции ---

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// memoryToStorageRecords конвертирует краткосрочную память в записи для файлового сохранения.
func memoryToStorageRecords(mem *memory.MemoryManager) []storage.TurnRecord {
	if mem == nil {
		return nil
	}
	h := mem.ShortTermHistory()
	recent := h.Recent(h.Len())
	records := make([]storage.TurnRecord, len(recent))
	for i, r := range recent {
		records[i] = storage.TurnRecord{
			PlayerAction: r.PlayerAction,
			AIResponse:   r.AIResponse,
		}
	}
	return records
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
