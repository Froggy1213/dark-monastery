package storage

import "time"

// SaveMeta contains save metadata.
type SaveMeta struct {
	SessionID  string    `json:"session_id"`
	PlayerName string    `json:"player_name"`
	Location   string    `json:"location"`
	TurnCount  int       `json:"turn_count"`
	UpdatedAt  time.Time `json:"updated_at"`
}
