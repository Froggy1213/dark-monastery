package handlers

import (
	"encoding/json"
	"net/http"

	"dark-monastery/internal/memory"
)

// writeJSON serializes data as JSON and sends it to the client.
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// memoryToStorageRecords converts short-term memory into a slice of records.
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
