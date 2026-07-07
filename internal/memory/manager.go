package memory

import (
	"context"
	"fmt"
	"log"
	"strings"
)

// MemoryManager объединяет краткосрочную (ring buffer) и долгосрочную (pgvector) память.
// Это основной фасад, с которым работают хендлеры.
type MemoryManager struct {
	shortTerm   *History          // кольцевой буфер, 3 хода
	store       MemoryStore       // pgvector (storage.PgStore)
	embedClient EmbeddingProvider // ai.EmbeddingClient
	extractor   *Extractor        // async pipeline
	sessionID   string
	turnCounter int
}

// NewMemoryManager создаёт новый менеджер памяти для сессии.
func NewMemoryManager(
	store MemoryStore,
	embedClient EmbeddingProvider,
	extractor *Extractor,
	sessionID string,
) *MemoryManager {
	return &MemoryManager{
		shortTerm:   NewHistory(3), // краткосрочная: последние 3 хода
		store:       store,
		embedClient: embedClient,
		extractor:   extractor,
		sessionID:   sessionID,
		turnCounter: 0,
	}
}

// Add сохраняет ход в обе памяти:
// 1. Краткосрочная — ring buffer (мгновенно)
// 2. Долгосрочная — через async extractor → pgvector (фоном)
func (m *MemoryManager) Add(action, response, location string) {
	m.turnCounter++

	// Краткосрочная
	m.shortTerm.Add(action, response)

	// Долгосрочная — формируем текст для embedding
	content := fmt.Sprintf("Действие: %s | Ответ: %s", action, truncate(response, 500))

	// Определяем тип действия
	actionType := classifyAction(action, response)

	// Отправляем в async pipeline
	if m.extractor != nil {
		m.extractor.Submit(ExtractJob{
			SessionID:  m.sessionID,
			TurnNumber: m.turnCounter,
			Content:    content,
			Location:   location,
			ActionType: actionType,
		})
	}
}

// BuildContext формирует текстовый контекст памяти для промпта AI.
// Синхронный — выполняет поиск в pgvector.
//
// Возвращает строку вида:
//
//	КРАТКОСРОЧНАЯ ПАМЯТЬ: ...
//	ДОЛГОСРОЧНАЯ ПАМЯТЬ: ...
func (m *MemoryManager) BuildContext(ctx context.Context, currentAction, currentLocation string) (string, error) {
	var sb strings.Builder

	// 1. Краткосрочная память — последние 3 хода
	shortCtx := m.shortTerm.RecentContext(3)
	if shortCtx != "" {
		sb.WriteString("КРАТКОСРОЧНАЯ ПАМЯТЬ (последние события):\n")
		sb.WriteString(shortCtx)
		sb.WriteString("\n")
	}

	// 2. Долгосрочная память — семантический поиск по pgvector
	if m.store != nil && m.embedClient != nil {
		queryText := fmt.Sprintf("Локация: %s. Действие игрока: %s", currentLocation, currentAction)

		queryEmbedding, err := m.embedClient.Embed(ctx, queryText)
		if err != nil {
			log.Printf("[MemoryManager] Ошибка embedding запроса: %v — используем только краткосрочную память", err)
			return sb.String(), nil // graceful fallback
		}

		memories, err := m.store.SearchSimilar(ctx, queryEmbedding, m.sessionID, 5)
		if err != nil {
			log.Printf("[MemoryManager] Ошибка поиска в pgvector: %v", err)
			return sb.String(), nil // graceful fallback
		}

		// Фильтруем — не дублируем то, что уже в краткосрочной памяти
		recentTurns := make(map[int]bool)
		for i := m.turnCounter; i > m.turnCounter-3 && i > 0; i-- {
			recentTurns[i] = true
		}

		var longTermMemories []*Memory
		for _, mem := range memories {
			if !recentTurns[mem.TurnNumber] {
				longTermMemories = append(longTermMemories, mem)
			}
		}

		if len(longTermMemories) > 0 {
			sb.WriteString("ДОЛГОСРОЧНАЯ ПАМЯТЬ (релевантные воспоминания из прошлого):\n")
			for _, mem := range longTermMemories {
				sb.WriteString(fmt.Sprintf("[Ход %d, %s, совпадение: %.2f] %s\n",
					mem.TurnNumber, mem.Location, mem.Similarity, mem.Content))
			}
			sb.WriteString("\nИспользуй долгосрочную память для поддержания непрерывности сюжета и мира.\n")
		}
	}

	return sb.String(), nil
}

// SetSession переключает менеджер на другую сессию.
func (m *MemoryManager) SetSession(sessionID string) {
	m.sessionID = sessionID
	m.shortTerm.Clear()
	m.turnCounter = 0
}

// SetTurnCounter устанавливает счётчик ходов (при загрузке сохранения).
func (m *MemoryManager) SetTurnCounter(n int) {
	m.turnCounter = n
}

// Clear очищает краткосрочную память. Долгосрочная остаётся в PostgreSQL.
func (m *MemoryManager) Clear() {
	m.shortTerm.Clear()
	m.turnCounter = 0
}

// SessionID возвращает текущий ID сессии.
func (m *MemoryManager) SessionID() string {
	return m.sessionID
}

// TurnCounter возвращает текущий номер хода.
func (m *MemoryManager) TurnCounter() int {
	return m.turnCounter
}

// --- Реализация game.HistoryProvider (обратная совместимость) ---

// RecentContext возвращает последние N ходов из краткосрочной памяти.
func (m *MemoryManager) RecentContext(n int) string {
	return m.shortTerm.RecentContext(n)
}

// Len возвращает количество записей в краткосрочной памяти.
func (m *MemoryManager) Len() int {
	return m.shortTerm.Len()
}

// ShortTermHistory возвращает доступ к краткосрочной истории.
func (m *MemoryManager) ShortTermHistory() *History {
	return m.shortTerm
}

// --- Вспомогательные функции ---

// classifyAction определяет тип действия на основе ключевых слов.
func classifyAction(action, response string) string {
	lower := strings.ToLower(action + " " + response)

	if strings.Contains(lower, "смерт") || strings.Contains(lower, "погиб") || strings.Contains(lower, "умер") {
		return "death"
	}
	if strings.Contains(lower, "квест") || strings.Contains(lower, "задан") || strings.Contains(lower, "миссия") {
		return "quest"
	}
	if strings.Contains(lower, "легенд") || strings.Contains(lower, "истори") || strings.Contains(lower, "лор") ||
		strings.Contains(lower, "древн") || strings.Contains(lower, "свиток") || strings.Contains(lower, "книг") {
		return "lore"
	}

	return "turn"
}

// truncate обрезает строку до maxLen символов.
func truncate(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen]) + "..."
}
