-- Миграция 001: Создание таблицы воспоминаний с pgvector.
-- Запускается автоматически при старте сервера (PgStore.Migrate).

CREATE EXTENSION IF NOT EXISTS vector;

CREATE TABLE IF NOT EXISTS memories (
    id          BIGSERIAL PRIMARY KEY,
    session_id  TEXT NOT NULL,
    turn_number INT NOT NULL,

    -- Текст воспоминания: "Действие: X | Ответ: Y"
    content     TEXT NOT NULL,

    -- Метаданные для фильтрации
    location    TEXT NOT NULL DEFAULT '',
    action_type TEXT NOT NULL DEFAULT 'turn',

    -- Вектор (768 размерностей — reduced от 3072)
    embedding   vector(768),

    -- Флаг: эмбеддинг ещё не вычислен (async fallback)
    embedded    BOOLEAN NOT NULL DEFAULT FALSE,

    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT memories_session_turn UNIQUE(session_id, turn_number)
);

-- HNSW-индекс для быстрого cosine-поиска
CREATE INDEX IF NOT EXISTS memories_embedding_idx
    ON memories USING hnsw (embedding vector_cosine_ops)
    WITH (m = 16, ef_construction = 64);

-- Индекс по session_id
CREATE INDEX IF NOT EXISTS memories_session_idx ON memories(session_id);

-- Индекс для async обработки pending embeddings
CREATE INDEX IF NOT EXISTS memories_pending_idx ON memories(embedded) WHERE embedded = FALSE;
