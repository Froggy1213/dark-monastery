package handlers

import (
	"encoding/json"
	"net/http"

	"dark-monastery/internal/memory"
)

// writeJSON сериализует data в JSON и отправляет клиенту.
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// memoryToStorageRecords конвертирует краткосрочную память в слайс записей.
func memoryToStorageRecords(mem *memory.MemoryManager) []memory.TurnRecord {
	if mem == nil {
		return nil
	}
	h := mem.ShortTermHistory()
	recent := h.Recent(h.Len())
	records := make([]memory.TurnRecord, len(recent))
	for i, r := range recent {
		records[i] = memory.TurnRecord{
			PlayerAction: r.PlayerAction,
			AIResponse:   r.AIResponse,
		}
	}
	return records
}
