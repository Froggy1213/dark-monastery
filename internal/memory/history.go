package memory

import (
	"sync"
)

// TurnRecord stores one turn of dialogue: the player's action and the AI's response.
type TurnRecord struct {
	PlayerAction string `json:"player_action"`
	AIResponse   string `json:"ai_response"`
}

// History is a ring buffer for storing the last N turns.
// Allows Gemini to remember recent events.
type History struct {
	mu       sync.RWMutex
	buffer   []TurnRecord
	capacity int
	head     int // position for the next write
	size     int // current number of entries
}

// NewHistory creates a new ring buffer with the given capacity.
func NewHistory(capacity int) *History {
	return &History{
		buffer:   make([]TurnRecord, capacity),
		capacity: capacity,
	}
}

// Add adds a turn record to the history.
func (h *History) Add(action, response string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.buffer[h.head] = TurnRecord{
		PlayerAction: action,
		AIResponse:   response,
	}
	h.head = (h.head + 1) % h.capacity
	if h.size < h.capacity {
		h.size++
	}
}

// Recent returns the last N records in chronological order.
// If there are fewer than N records, returns all available.
func (h *History) Recent(n int) []TurnRecord {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if n > h.size {
		n = h.size
	}

	result := make([]TurnRecord, n)
	// The oldest record is at position (head - size + capacity) % capacity
	start := (h.head - h.size + h.capacity) % h.capacity

	// If n < size, take only the last n
	if n < h.size {
		start = (h.head - n + h.capacity) % h.capacity
	}

	for i := 0; i < n; i++ {
		idx := (start + i) % h.capacity
		result[i] = h.buffer[idx]
	}
	return result
}

// RecentContext returns the last N turns as text for the prompt.
func (h *History) RecentContext(n int) string {
	records := h.Recent(n)
	if len(records) == 0 {
		return ""
	}

	ctx := "RECENT EVENTS:\n"
	for i, r := range records {
		ctx += "Turn " + string(rune('1'+i)) + ":\n"
		ctx += "  Player action: " + r.PlayerAction + "\n"
		ctx += "  Master response: " + r.AIResponse + "\n"
	}
	return ctx
}

// Len returns the number of records in the buffer.
func (h *History) Len() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.size
}

// Clear clears the history.
func (h *History) Clear() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.head = 0
	h.size = 0
}
