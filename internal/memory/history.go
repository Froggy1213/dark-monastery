package memory

import (
	"sync"
)

// TurnRecord хранит один ход диалога: действие игрока и ответ AI.
type TurnRecord struct {
	PlayerAction string `json:"player_action"`
	AIResponse   string `json:"ai_response"`
}

// History — кольцевой буфер для хранения последних N ходов.
// Позволяет Gemini помнить недавние события.
type History struct {
	mu       sync.RWMutex
	buffer   []TurnRecord
	capacity int
	head     int // позиция для следующей записи
	size     int // текущее количество записей
}

// NewHistory создаёт новый кольцевой буфер заданной ёмкости.
func NewHistory(capacity int) *History {
	return &History{
		buffer:   make([]TurnRecord, capacity),
		capacity: capacity,
	}
}

// Add добавляет запись о ходе в историю.
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

// Recent возвращает последние N записей в хронологическом порядке.
// Если записей меньше N, возвращает все имеющиеся.
func (h *History) Recent(n int) []TurnRecord {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if n > h.size {
		n = h.size
	}

	result := make([]TurnRecord, n)
	// Самая старая запись находится в позиции (head - size + capacity) % capacity
	start := (h.head - h.size + h.capacity) % h.capacity

	// Если n < size, берём только последние n
	if n < h.size {
		start = (h.head - n + h.capacity) % h.capacity
	}

	for i := 0; i < n; i++ {
		idx := (start + i) % h.capacity
		result[i] = h.buffer[idx]
	}
	return result
}

// RecentContext возвращает последние N ходов в виде текста для промпта.
func (h *History) RecentContext(n int) string {
	records := h.Recent(n)
	if len(records) == 0 {
		return ""
	}

	ctx := "НЕДАВНИЕ СОБЫТИЯ:\n"
	for i, r := range records {
		ctx += "Ход " + string(rune('1'+i)) + ":\n"
		ctx += "  Действие игрока: " + r.PlayerAction + "\n"
		ctx += "  Ответ мастера: " + r.AIResponse + "\n"
	}
	return ctx
}

// Len возвращает количество записей в буфере.
func (h *History) Len() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.size
}

// Clear очищает историю.
func (h *History) Clear() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.head = 0
	h.size = 0
}
