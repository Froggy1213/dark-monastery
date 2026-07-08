package handlers

import (
	"log"
	"net/http"

	"dark-monastery/internal/game"
	"dark-monastery/internal/storage"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// wsMessage — message from the client.
type wsMessage struct {
	Type      string `json:"type"`
	Text      string `json:"text"`
	SessionID string `json:"session_id"`
}

// wsResponse — response to the client.
type wsResponse struct {
	Type      string          `json:"type"`
	State     *game.GameState `json:"state,omitempty"`
	TurnCount int             `json:"turn_count"`
	Message   string          `json:"message,omitempty"`
}

func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}
	defer conn.Close()

	sessionID := r.URL.Query().Get("session_id")

	s.mu.RLock()
	sess, ok := s.sessions[sessionID]
	s.mu.RUnlock()

	if !ok {
		sess = &Session{
			State:  game.NewPlayer(),
			Memory: s.createMemoryManager(sessionID),
		}
		s.mu.Lock()
		s.sessions[sessionID] = sess
		s.mu.Unlock()

		// Auto-generate first turn with AI
		newState, err := s.processActionWithMemory(r.Context(), sess, "START")
		if err != nil {
			log.Printf("First turn generation error: %v", err)
		} else {
			sess.State = newState
			sess.TurnCount++
			if sess.Memory != nil {
				sess.Memory.Add("START", newState.Message, newState.Location)
			}
		}
	}

	// Current state
	writeWS(conn, wsResponse{
		Type:      "update",
		State:     sess.State,
		TurnCount: sess.TurnCount,
		Message:   sess.State.Message,
	})

	ctx := r.Context()

	for {
		var msg wsMessage
		if err := conn.ReadJSON(&msg); err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		switch msg.Type {
		case "ping":
			writeWS(conn, wsResponse{Type: "pong", Message: "pong"})

		case "action":
			if msg.Text == "" {
				writeWS(conn, wsResponse{Type: "error", Message: "Empty action"})
				continue
			}

			writeWS(conn, wsResponse{Type: "thinking", Message: "AI is thinking..."})

			// RAG: build memory context and call AI
			newState, err := s.processActionWithMemory(ctx, sess, msg.Text)
			if err != nil {
				log.Printf("AI error: %v", err)
				writeWS(conn, wsResponse{Type: "error", Message: "AI error: " + err.Error()})
				continue
			}

			sess.TurnCount++

			// RAG: save to both memories
			if sess.Memory != nil {
				sess.Memory.Add(msg.Text, newState.Message, newState.Location)
			}

			sess.State = newState

			// Auto-save (file)
			meta := &storage.SaveMeta{
				SessionID:  sessionID,
				PlayerName: game.DefaultPlayerName,
				Location:   newState.Location,
				TurnCount:  sess.TurnCount,
			}
			_ = s.fileStore.SaveWithHistory(sessionID, newState, meta, memoryToStorageRecords(sess.Memory))

			writeWS(conn, wsResponse{
				Type:      "update",
				State:     newState,
				TurnCount: sess.TurnCount,
				Message:   newState.Message,
			})

		default:
			writeWS(conn, wsResponse{Type: "error", Message: "Unknown type: " + msg.Type})
		}
	}
}

func writeWS(conn *websocket.Conn, resp wsResponse) {
	if err := conn.WriteJSON(resp); err != nil {
		log.Printf("WebSocket send error: %v", err)
	}
}
