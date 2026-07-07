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

// wsMessage — сообщение от клиента.
type wsMessage struct {
	Type      string `json:"type"`
	Text      string `json:"text"`
	SessionID string `json:"session_id"`
}

// wsResponse — ответ клиенту.
type wsResponse struct {
	Type      string          `json:"type"`
	State     *game.GameState `json:"state,omitempty"`
	TurnCount int             `json:"turn_count,omitempty"`
	Message   string          `json:"message,omitempty"`
}

func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Ошибка WebSocket upgrade: %v", err)
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
	}

	// Текущее состояние
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
				log.Printf("WebSocket ошибка: %v", err)
			}
			break
		}

		switch msg.Type {
		case "ping":
			writeWS(conn, wsResponse{Type: "pong", Message: "pong"})

		case "action":
			if msg.Text == "" {
				writeWS(conn, wsResponse{Type: "error", Message: "Пустое действие"})
				continue
			}

			writeWS(conn, wsResponse{Type: "thinking", Message: "ИИ думает..."})

			// RAG: строим контекст памяти и вызываем AI
			newState, err := s.processActionWithMemory(ctx, sess, msg.Text)
			if err != nil {
				log.Printf("Ошибка AI: %v", err)
				writeWS(conn, wsResponse{Type: "error", Message: "Ошибка нейросети: " + err.Error()})
				continue
			}

			sess.TurnCount++

			// RAG: сохраняем в обе памяти
			if sess.Memory != nil {
				sess.Memory.Add(msg.Text, newState.Message, newState.Location)
			}

			sess.State = newState

			// Автосохранение (файловое)
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
			writeWS(conn, wsResponse{Type: "error", Message: "Неизвестный тип: " + msg.Type})
		}
	}
}

func writeWS(conn *websocket.Conn, resp wsResponse) {
	if err := conn.WriteJSON(resp); err != nil {
		log.Printf("Ошибка отправки WebSocket: %v", err)
	}
}
