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

// Server holds all dependencies for the HTTP API.
type Server struct {
	engine    *game.Engine
	fileStore *storage.FileStore
	lore      *game.LoreBook
	sessions  map[string]*Session
	mu        sync.RWMutex

	// RAG components (nil if PostgreSQL is unavailable)
	pgStore     *storage.PgStore
	embedClient *ai.EmbeddingClient
	extractor   *memory.Extractor
}

// Session is an active game session.
type Session struct {
	State     *game.GameState       `json:"state"`
	Memory    *memory.MemoryManager `json:"-"`
	TurnCount int                   `json:"turn_count"`
	CreatedAt time.Time             `json:"created_at"`
}

// NewServer creates a new HTTP server with the game engine.
// pgStore, embedClient, extractor can be nil — then legacy mode is used.
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

// createMemoryManager creates a MemoryManager for a session.
// If RAG components are unavailable, returns nil.
func (s *Server) createMemoryManager(sessionID string) *memory.MemoryManager {
	if s.pgStore == nil || s.embedClient == nil {
		return nil
	}
	return memory.NewMemoryManager(s.pgStore, s.embedClient, s.extractor, sessionID)
}

// RegisterRoutes registers all endpoints on the given ServeMux.
func (s *Server) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/game/new", s.handleNewGame)
	mux.HandleFunc("/api/game/action", s.handleAction)
	mux.HandleFunc("/api/game/state", s.handleState)
	mux.HandleFunc("/api/game/saves", s.handleSaves)
	mux.HandleFunc("/api/game/save", s.handleSave)
	mux.HandleFunc("/api/game/load", s.handleLoad)
	mux.HandleFunc("/ws/game", s.handleWebSocket)
}
