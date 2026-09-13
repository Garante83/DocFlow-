package websocket

import (
	"sync"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// Hub manages WebSocket connections for sessions.
type Hub struct {
	clients    map[uuid.UUID]map[*Client]bool
	register   chan *Client
	unregister chan *Client
	broadcast  chan *Message
	mu         sync.Mutex
	stopChan   chan struct{}
}

// Client represents a WebSocket client.
type Client struct {
	Conn      *websocket.Conn
	SessionID uuid.UUID
	Hub       *Hub
}

// Message represents a WebSocket message.
type Message struct {
	sessionID uuid.UUID
	data      []byte
}

// NewHub creates a new WebSocket hub.
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[uuid.UUID]map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client, 100), // Buffered to prevent deadlocks
		broadcast:  make(chan *Message),
		stopChan:   make(chan struct{}),
	}
}

// Run starts the hub's main loop.
func (h *Hub) Run() {
	for {
		select {
		case <-h.stopChan:
			return
		case client := <-h.register:
			h.registerClient(client)
		case client := <-h.unregister:
			h.unregisterClient(client)
		case message := <-h.broadcast:
			h.broadcastMessage(message)
		}
	}
}

// registerClient registers a new client with the hub.
func (h *Hub) registerClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, exists := h.clients[client.SessionID]; !exists {
		h.clients[client.SessionID] = make(map[*Client]bool)
	}
	h.clients[client.SessionID][client] = true
}

// unregisterClient unregisters a client from the hub.
func (h *Hub) unregisterClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, exists := h.clients[client.SessionID]; exists {
		if _, ok := h.clients[client.SessionID][client]; ok {
			delete(h.clients[client.SessionID], client)
			client.Conn.Close()
		}
	}
}

// broadcastMessage broadcasts a message to all clients of a session.
func (h *Hub) broadcastMessage(message *Message) {
	h.mu.Lock()
	clients, exists := h.clients[message.sessionID]
	if !exists {
		h.mu.Unlock()
		return
	}
	// Take a snapshot of clients under the lock to avoid concurrent map iteration
	snapshot := make([]*Client, 0, len(clients))
	for client := range clients {
		snapshot = append(snapshot, client)
	}
	h.mu.Unlock()

	var failed []*Client
	for _, client := range snapshot {
		if err := client.Conn.WriteMessage(websocket.TextMessage, message.data); err != nil {
			failed = append(failed, client)
		}
	}

	for _, client := range failed {
		h.Unregister(client)
	}
}

// Broadcast sends a message to all clients of a session.
func (h *Hub) Broadcast(sessionID uuid.UUID, data []byte) {
	h.broadcast <- &Message{sessionID: sessionID, data: data}
}

// Register registers a new client with the hub.
func (h *Hub) Register(client *Client) {
	h.register <- client
}

// Unregister unregisters a client from the hub.
func (h *Hub) Unregister(client *Client) {
	h.unregister <- client
}

// Stop stops the hub's main loop.
func (h *Hub) Stop() {
	close(h.stopChan)
}
