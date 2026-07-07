package memory

import (
	"context"
	"time"
)

// Memory — одно воспоминание, хранящееся в pgvector.
type Memory struct {
	ID         int64     `json:"id"`
	SessionID  string    `json:"session_id"`
	TurnNumber int       `json:"turn_number"`
	Content    string    `json:"content"`     // "Действие: X | Ответ: Y"
	Location   string    `json:"location"`
	ActionType string    `json:"action_type"` // "turn", "lore", "quest", "death"
	Embedding  []float32 `json:"-"`           // 768d вектор, не сериализуется в JSON
	Embedded   bool      `json:"embedded"`
	CreatedAt  time.Time `json:"created_at"`
	Similarity float32   `json:"similarity,omitempty"` // только при поиске
}

// EmbeddingProvider — интерфейс для генерации эмбеддингов.
// Реализован в ai.EmbeddingClient (избегает циклического импорта).
type EmbeddingProvider interface {
	Embed(ctx context.Context, text string) ([]float32, error)
}

// MemoryStore — интерфейс для хранения/поиска воспоминаний.
// Реализован в storage.PgStore (избегает циклического импорта).
type MemoryStore interface {
	InsertMemory(ctx context.Context, m *Memory) error
	UpdateEmbedding(ctx context.Context, id int64, embedding []float32) error
	SearchSimilar(ctx context.Context, embedding []float32, sessionID string, topK int) ([]*Memory, error)
	PendingEmbeddings(ctx context.Context, limit int) ([]*Memory, error)
	ForSession(ctx context.Context, sessionID string) ([]*Memory, error)
}
