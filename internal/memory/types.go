package memory

import (
	"context"
	"time"
)

// Memory — a single memory stored in pgvector.
type Memory struct {
	ID         int64     `json:"id"`
	SessionID  string    `json:"session_id"`
	TurnNumber int       `json:"turn_number"`
	Content    string    `json:"content"`     // "Action: X | Response: Y"
	Location   string    `json:"location"`
	ActionType string    `json:"action_type"` // "turn", "lore", "quest", "death"
	Embedding  []float32 `json:"-"`           // 768d vector, not serialized to JSON
	Embedded   bool      `json:"embedded"`
	CreatedAt  time.Time `json:"created_at"`
	Similarity float32   `json:"similarity,omitempty"` // only during search
}

// EmbeddingProvider is an interface for generating embeddings.
// Implemented in ai.EmbeddingClient (avoids circular import).
type EmbeddingProvider interface {
	Embed(ctx context.Context, text string) ([]float32, error)
}

// MemoryStore is an interface for storing/retrieving memories.
// Implemented in storage.PgStore (avoids circular import).
type MemoryStore interface {
	InsertMemory(ctx context.Context, m *Memory) error
	UpdateEmbedding(ctx context.Context, id int64, embedding []float32) error
	SearchSimilar(ctx context.Context, embedding []float32, sessionID string, topK int) ([]*Memory, error)
	PendingEmbeddings(ctx context.Context, limit int) ([]*Memory, error)
	ForSession(ctx context.Context, sessionID string) ([]*Memory, error)
}
