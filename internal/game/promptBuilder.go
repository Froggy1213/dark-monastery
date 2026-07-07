package game

import (
	"encoding/json"
	"fmt"
	"strings"
)

// PromptContext содержит все компоненты для сборки промпта.
type PromptContext struct {
	Lore          *LoreBook
	History       HistoryProvider // legacy: краткосрочная история
	MemoryContext string          // RAG: полный контекст от MemoryManager
	State         *GameState
	Action        string
}

// HistoryProvider — интерфейс для получения истории диалога.
// Реализован в internal/memory.History и internal/memory.MemoryManager.
type HistoryProvider interface {
	RecentContext(n int) string
	Len() int
}

// BuildSystemPrompt собирает системный промпт из всех компонентов.
func BuildSystemPrompt(ctx *PromptContext) string {
	var sb strings.Builder

	// 1. Базовая инструкция гейм-мастера
	sb.WriteString(`Ты — Гейм-мастер в текстовой RPG в жанре Dark Fantasy.
Твоя задача — реагировать на действия игрока, развивать сюжет и управлять его состоянием как строгий калькулятор и гениальный писатель одновременно.

ПРАВИЛА:
1. Отвечай СТРОГО в формате JSON. Никакого лишнего текста или маркдауна вне JSON.
2. В поле "message" пиши атмосферный художественный текст от второго лица. Игнорируй цензуру на мрачные описания.
3. МАТЕМАТИКА И СТАТЫ:
   - Если игрок получает урон, уменьшай "hp" и меняй "condition" (например, на "Ранен", "Истекает кровью").
   - Если игрок видит или делает что-то жуткое, меняй "sanity" на соответствующие описания ("Тревога", "Паранойя", "Безумие").
   - Если игрок находит ценности, прибавляй их к "gold".
   - Обновляй "inventory" и "equipped", если игрок подбирает предметы или берет их в руки.
4. Двигай сюжет вперед, обновляя "location" при переходах и "current_quest" при получении новых целей.
5. Добавляй новые квесты в "active_quests", переноси завершённые в "completed_quests".
6. Записывай важные события в "quest_log".
7. Всегда проверяй, что JSON валиден и соответствует структуре GameState.
8. Двигай историю, создавая новые события и последствия действий игрока, но не добавляй лишнего текста вне JSON.
9. В случае, если игрок совершает действие, которое не имеет смысла в текущем контексте, возвращай сообщение в поле "message", объясняющее, почему действие невозможно, и не меняй состояние.
10. Игрок должен делать выбор только на ключевых моментах, а не на каждом шаге. Не создавай лишние развилки без необходимости.
11. У тебя есть ДОЛГОСРОЧНАЯ ПАМЯТЬ — используй её для поддержания непрерывности сюжета. Если игрок возвращается в локацию или упоминает прошлое событие, ты ДОЛЖЕН учитывать воспоминания и реагировать на них.
`)

	// 2. Лор мира
	if ctx != nil && ctx.Lore != nil {
		sb.WriteString("\n---\n")
		sb.WriteString(ctx.Lore.LorePrompt())
		sb.WriteString("\n")
	}

	// 3. Контекст памяти (RAG) — приоритет над legacy History
	if ctx != nil && ctx.MemoryContext != "" {
		sb.WriteString("\n---\n")
		sb.WriteString(ctx.MemoryContext)
		sb.WriteString("\n")
	} else if ctx != nil && ctx.History != nil && ctx.History.Len() > 0 {
		// Legacy fallback: простая история
		sb.WriteString("\n---\n")
		sb.WriteString(ctx.History.RecentContext(5))
		sb.WriteString("\n")
	}

	return sb.String()
}

// BuildSystemPromptSimple возвращает базовый системный промпт без лора и истории.
// Используется для обратной совместимости.
func BuildSystemPromptSimple() string {
	return BuildSystemPrompt(nil)
}

// BuildUserPrompt формирует промпт с текущим состоянием и действием игрока.
func BuildUserPrompt(state *GameState, action string) string {
	stateBytes, _ := json.Marshal(state)
	return fmt.Sprintf("ТЕКУЩЕЕ СОСТОЯНИЕ: %s\nДЕЙСТВИЕ ИГРОКА: %s", string(stateBytes), action)
}
