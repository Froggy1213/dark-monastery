package storage

import (
	"context"
	"fmt"
	"log"
	"time"

	"dark-monastery/internal/memory"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	pgxvec "github.com/pgvector/pgvector-go/pgx"
	"github.com/pgvector/pgvector-go"
)

// migrationSQL содержит SQL для создания таблицы memories с pgvector.
// Дублирует migrations/001_create_memories.sql для удобства go:embed не позволяет ..
const migrationSQL = `
CREATE EXTENSION IF NOT EXISTS vector;

CREATE TABLE IF NOT EXISTS memories (
    id          BIGSERIAL PRIMARY KEY,
    session_id  TEXT NOT NULL,
    turn_number INT NOT NULL,
    content     TEXT NOT NULL,
    location    TEXT NOT NULL DEFAULT '',
    action_type TEXT NOT NULL DEFAULT 'turn',
    embedding   vector(768),
    embedded    BOOLEAN NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT memories_session_turn UNIQUE(session_id, turn_number)
);

CREATE INDEX IF NOT EXISTS memories_embedding_idx
    ON memories USING hnsw (embedding vector_cosine_ops)
    WITH (m = 16, ef_construction = 64);

CREATE INDEX IF NOT EXISTS memories_session_idx ON memories(session_id);

CREATE INDEX IF NOT EXISTS memories_pending_idx ON memories(embedded) WHERE embedded = FALSE;
`
type PgStore struct {
	pool *pgxpool.Pool
}

// NewPgStore создаёт новый PostgreSQL store.
func NewPgStore(ctx context.Context, connString string) (*PgStore, error) {
	config, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, fmt.Errorf("ошибка разбора строки подключения: %w", err)
	}

	config.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		return pgxvec.RegisterTypes(ctx, conn)
	}

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("ошибка подключения к PostgreSQL: %w", err)
	}

	// Проверяем соединение
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("PostgreSQL недоступен: %w", err)
	}

	log.Println("[PgStore] Подключено к PostgreSQL")
	return &PgStore{pool: pool}, nil
}

// Migrate запускает SQL-миграцию (создание таблиц и индексов).
func (p *PgStore) Migrate(ctx context.Context) error {
	_, err := p.pool.Exec(ctx, migrationSQL)
	if err != nil {
		return fmt.Errorf("ошибка миграции: %w", err)
	}
	log.Println("[PgStore] Миграция выполнена")
	return nil
}

// Close закрывает пул соединений.
func (p *PgStore) Close() {
	p.pool.Close()
}

// InsertMemory вставляет воспоминание в таблицу memories.
// Если embedding == nil, запись создаётся с embedded=false для async обработки.
func (p *PgStore) InsertMemory(ctx context.Context, m *memory.Memory) error {
	var embeddingVal interface{}
	if len(m.Embedding) > 0 {
		embeddingVal = pgvector.NewVector(m.Embedding)
	}

	query := `
		INSERT INTO memories (session_id, turn_number, content, location, action_type, embedding, embedded, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (session_id, turn_number)
		DO UPDATE SET content = EXCLUDED.content,
		              location = EXCLUDED.location,
		              action_type = EXCLUDED.action_type,
		              embedding = COALESCE(EXCLUDED.embedding, memories.embedding),
		              embedded = EXCLUDED.embedded
		RETURNING id
	`

	createdAt := m.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now()
	}

	err := p.pool.QueryRow(ctx, query,
		m.SessionID,
		m.TurnNumber,
		m.Content,
		m.Location,
		m.ActionType,
		embeddingVal,
		m.Embedded,
		createdAt,
	).Scan(&m.ID)

	if err != nil {
		return fmt.Errorf("ошибка вставки memory: %w", err)
	}
	return nil
}

// UpdateEmbedding обновляет вектор для записи (async pipeline).
func (p *PgStore) UpdateEmbedding(ctx context.Context, id int64, embedding []float32) error {
	query := `UPDATE memories SET embedding = $1, embedded = TRUE WHERE id = $2`
	_, err := p.pool.Exec(ctx, query, pgvector.NewVector(embedding), id)
	if err != nil {
		return fmt.Errorf("ошибка обновления embedding: %w", err)
	}
	return nil
}

// SearchSimilar ищет top-K похожих воспоминаний по cosine similarity.
// Если sessionID пустой — ищет по всем сессиям.
func (p *PgStore) SearchSimilar(ctx context.Context, embedding []float32, sessionID string, topK int) ([]*memory.Memory, error) {
	var query string
	var args []interface{}

	vec := pgvector.NewVector(embedding)

	if sessionID != "" {
		query = `
			SELECT id, session_id, turn_number, content, location, action_type, embedded, created_at,
			       1 - (embedding <=> $1) AS similarity
			FROM memories
			WHERE session_id = $2 AND embedded = TRUE
			ORDER BY embedding <=> $1
			LIMIT $3
		`
		args = []interface{}{vec, sessionID, topK}
	} else {
		query = `
			SELECT id, session_id, turn_number, content, location, action_type, embedded, created_at,
			       1 - (embedding <=> $1) AS similarity
			FROM memories
			WHERE embedded = TRUE
			ORDER BY embedding <=> $1
			LIMIT $2
		`
		args = []interface{}{vec, topK}
	}

	rows, err := p.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("ошибка поиска memories: %w", err)
	}
	defer rows.Close()

	var results []*memory.Memory
	for rows.Next() {
		m := &memory.Memory{}
		if err := rows.Scan(
			&m.ID, &m.SessionID, &m.TurnNumber, &m.Content,
			&m.Location, &m.ActionType, &m.Embedded, &m.CreatedAt,
			&m.Similarity,
		); err != nil {
			return nil, fmt.Errorf("ошибка сканирования memory: %w", err)
		}
		results = append(results, m)
	}

	return results, rows.Err()
}

// PendingEmbeddings возвращает записи без эмбеддингов (для async пересчёта).
func (p *PgStore) PendingEmbeddings(ctx context.Context, limit int) ([]*memory.Memory, error) {
	query := `
		SELECT id, session_id, turn_number, content, location, action_type, embedded, created_at
		FROM memories
		WHERE embedded = FALSE
		ORDER BY created_at ASC
		LIMIT $1
	`

	rows, err := p.pool.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("ошибка загрузки pending: %w", err)
	}
	defer rows.Close()

	var results []*memory.Memory
	for rows.Next() {
		m := &memory.Memory{}
		if err := rows.Scan(
			&m.ID, &m.SessionID, &m.TurnNumber, &m.Content,
			&m.Location, &m.ActionType, &m.Embedded, &m.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("ошибка сканирования pending: %w", err)
		}
		results = append(results, m)
	}

	return results, rows.Err()
}

// ForSession возвращает все воспоминания сессии.
func (p *PgStore) ForSession(ctx context.Context, sessionID string) ([]*memory.Memory, error) {
	query := `
		SELECT id, session_id, turn_number, content, location, action_type, embedded, created_at
		FROM memories
		WHERE session_id = $1
		ORDER BY turn_number ASC
	`

	rows, err := p.pool.Query(ctx, query, sessionID)
	if err != nil {
		return nil, fmt.Errorf("ошибка загрузки memories сессии: %w", err)
	}
	defer rows.Close()

	var results []*memory.Memory
	for rows.Next() {
		m := &memory.Memory{}
		if err := rows.Scan(
			&m.ID, &m.SessionID, &m.TurnNumber, &m.Content,
			&m.Location, &m.ActionType, &m.Embedded, &m.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("ошибка сканирования memory: %w", err)
		}
		results = append(results, m)
	}

	return results, rows.Err()
}
