package handlers

import (
	"log"
	"net/http"
	"time"

	ws "dokumentenscanner/internal/websocket"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// WebSocket configuration constants
const (
	websocketReadBufferSize  = 1024
	websocketWriteBufferSize = 1024
	websocketReadDeadline    = 60 * time.Second
	websocketPingInterval    = 30 * time.Second
)

// WebSocketHub is a global WebSocket hub instance.
var WebSocketHub = ws.NewHub()

// WebSocketHandler handles WebSocket connections for sessions.
func WebSocketHandler(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Println("Failed to upgrade to WebSocket:", err)
		return
	}

	sessionIDStr := c.Param("id")
	sessionID, err := uuid.Parse(sessionIDStr)
	if err != nil {
		log.Println("Invalid session ID:", err)
		conn.Close()
		return
	}

	// Check if the session exists
	_, exists := SessionStore.Get(sessionID)
	if !exists {
		log.Println("Session not found:", sessionID)
		conn.Close()
		return
	}

	client := &ws.Client{Conn: conn, SessionID: sessionID, Hub: WebSocketHub}
	WebSocketHub.Register(client)
	defer WebSocketHub.Unregister(client)

	// Set initial read deadline
	conn.SetReadDeadline(time.Now().Add(websocketReadDeadline))

	// Handle pong messages to reset read deadline
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(websocketReadDeadline))
		return nil
	})

	// Start ping goroutine to keep connection alive
	go func() {
		ticker := time.NewTicker(websocketPingInterval)
		defer ticker.Stop()
		for range ticker.C {
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}()

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				log.Println("WebSocket error:", err)
			}
			break
		}

		log.Printf("Received message from session %s: %s", sessionID, message)
		WebSocketHub.Broadcast(sessionID, message)
	}
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  websocketReadBufferSize,
	WriteBufferSize: websocketWriteBufferSize,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}
