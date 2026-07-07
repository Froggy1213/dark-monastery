package handlers

import (
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
	engine    *game.Engine
	fileStore *storage.FileStore
	lore      *game.LoreBook
	sessions  map[string]*Session
	mu        sync.RWMutex

	// RAG компоненты (nil если PostgreSQL недоступен)
	pgStore     *storage.PgStore
	embedClient *ai.EmbeddingClient
	extractor   *memory.Extractor
}

// Session — активная игровая сессия.
type Session struct {
	State     *game.GameState       `json:"state"`
	Memory    *memory.MemoryManager `json:"-"`
	TurnCount int                   `json:"turn_count"`
	CreatedAt time.Time             `json:"created_at"`
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
