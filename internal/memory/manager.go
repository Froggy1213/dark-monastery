package memory

import (
	"context"
	"fmt"
	"log"
	"strings"
)

// MemoryManager combines short-term (ring buffer) and long-term (pgvector) memory.
// This is the main facade that handlers work with.
type MemoryManager struct {
	shortTerm   *History          // ring buffer, 3 turns
	store       MemoryStore       // pgvector (storage.PgStore)
	embedClient EmbeddingProvider // ai.EmbeddingClient
	extractor   *Extractor        // async pipeline
	sessionID   string
	turnCounter int
}

// NewMemoryManager creates a new memory manager for a session.
func NewMemoryManager(
	store MemoryStore,
	embedClient EmbeddingProvider,
	extractor *Extractor,
	sessionID string,
) *MemoryManager {
	return &MemoryManager{
		shortTerm:   NewHistory(3), // short-term: last 3 turns
		store:       store,
		embedClient: embedClient,
		extractor:   extractor,
		sessionID:   sessionID,
		turnCounter: 0,
	}
}

// Add saves a turn to both memories:
// 1. Short-term — ring buffer (instant)
// 2. Long-term — via async extractor -> pgvector (background)
func (m *MemoryManager) Add(action, response, location string) {
	m.turnCounter++

	// Short-term
	m.shortTerm.Add(action, response)

	// Long-term — build text for embedding
	content := fmt.Sprintf("Action: %s | Response: %s", action, truncate(response, 500))

	// Determine action type
	actionType := classifyAction(action, response)

	// Submit to async pipeline
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

// BuildContext builds a text memory context for the AI prompt.
// Synchronous — performs a search in pgvector.
//
// Returns a string of the form:
//
//	SHORT-TERM MEMORY: ...
//	LONG-TERM MEMORY: ...
func (m *MemoryManager) BuildContext(ctx context.Context, currentAction, currentLocation string) (string, error) {
	var sb strings.Builder

	// 1. Short-term memory — last 3 turns
	shortCtx := m.shortTerm.RecentContext(3)
	if shortCtx != "" {
		sb.WriteString("SHORT-TERM MEMORY (recent events):\n")
		sb.WriteString(shortCtx)
		sb.WriteString("\n")
	}

	// 2. Long-term memory — semantic search via pgvector
	if m.store != nil && m.embedClient != nil {
		queryText := fmt.Sprintf("Location: %s. Player action: %s", currentLocation, currentAction)

		queryEmbedding, err := m.embedClient.Embed(ctx, queryText)
		if err != nil {
			log.Printf("[MemoryManager] Embedding query error: %v — using only short-term memory", err)
			return sb.String(), nil // graceful fallback
		}

		memories, err := m.store.SearchSimilar(ctx, queryEmbedding, m.sessionID, 5)
		if err != nil {
			log.Printf("[MemoryManager] pgvector search error: %v", err)
			return sb.String(), nil // graceful fallback
		}

		// Filter — don't duplicate what's already in short-term memory
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
			sb.WriteString("LONG-TERM MEMORY (relevant past memories):\n")
			for _, mem := range longTermMemories {
				sb.WriteString(fmt.Sprintf("[Turn %d, %s, similarity: %.2f] %s\n",
					mem.TurnNumber, mem.Location, mem.Similarity, mem.Content))
			}
			sb.WriteString("\nUse long-term memory to maintain narrative continuity and world consistency.\n")
		}
	}

	return sb.String(), nil
}

// SetSession switches the manager to a different session.
func (m *MemoryManager) SetSession(sessionID string) {
	m.sessionID = sessionID
	m.shortTerm.Clear()
	m.turnCounter = 0
}

// SetTurnCounter sets the turn counter (when loading a save).
func (m *MemoryManager) SetTurnCounter(n int) {
	m.turnCounter = n
}

// Clear clears short-term memory. Long-term memory stays in PostgreSQL.
func (m *MemoryManager) Clear() {
	m.shortTerm.Clear()
	m.turnCounter = 0
}

// SessionID returns the current session ID.
func (m *MemoryManager) SessionID() string {
	return m.sessionID
}

// TurnCounter returns the current turn number.
func (m *MemoryManager) TurnCounter() int {
	return m.turnCounter
}

// --- Implementation of game.HistoryProvider (backward compatibility) ---

// RecentContext returns the last N turns from short-term memory.
func (m *MemoryManager) RecentContext(n int) string {
	return m.shortTerm.RecentContext(n)
}

// Len returns the number of entries in short-term memory.
func (m *MemoryManager) Len() int {
	return m.shortTerm.Len()
}

// ShortTermHistory returns access to the short-term history.
func (m *MemoryManager) ShortTermHistory() *History {
	return m.shortTerm
}

// --- Helper functions ---

// classifyAction determines the action type based on keywords.
func classifyAction(action, response string) string {
	lower := strings.ToLower(action + " " + response)

	if strings.Contains(lower, "death") || strings.Contains(lower, "perish") || strings.Contains(lower, "die") {
		return "death"
	}
	if strings.Contains(lower, "quest") || strings.Contains(lower, "task") || strings.Contains(lower, "mission") {
		return "quest"
	}
	if strings.Contains(lower, "legend") || strings.Contains(lower, "history") || strings.Contains(lower, "lore") ||
		strings.Contains(lower, "ancient") || strings.Contains(lower, "scroll") || strings.Contains(lower, "book") {
		return "lore"
	}

	return "turn"
}

// truncate truncates a string to maxLen characters.
func truncate(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen]) + "..."
}
