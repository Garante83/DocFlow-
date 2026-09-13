package handlers

import (
	"crypto/subtle"
	"encoding/json"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"docflow/internal/config"
	ws "docflow/internal/websocket"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// WebSocket configuration constants
const (
	websocketReadBufferSize  = 1024
	websocketWriteBufferSize = 1024
	// Deadline for the first message after connecting: it must be the auth message
	websocketAuthDeadline = 10 * time.Second
)

// isPrivateIP checks if the given IP address is a private IP
func isPrivateIP(ip string) bool {
	// Remove port if present
	if strings.Contains(ip, ":") {
		ip = strings.Split(ip, ":")[0]
	}

	// localhost always counts as local
	if ip == "localhost" {
		return true
	}

	// Parse the IP
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return false
	}

	// Check for private IP ranges
	// 10.0.0.0/8
	if parsedIP.IsPrivate() {
		return true
	}

	// 100.64.0.0/10 (Carrier-grade NAT)
	if ip4 := parsedIP.To4(); ip4 != nil {
		if ip4[0] == 100 && (ip4[1]&0xC0) == 64 {
			return true
		}
	}

	// 192.168.0.0/16
	if ip4 := parsedIP.To4(); ip4 != nil {
		if ip4[0] == 192 && ip4[1] == 168 {
			return true
		}
	}

	// 172.16.0.0/12
	if ip4 := parsedIP.To4(); ip4 != nil {
		if ip4[0] == 172 && (ip4[1]&0xF0) == 16 {
			return true
		}
	}

	// 169.254.0.0/16 (Link-local)
	if ip4 := parsedIP.To4(); ip4 != nil {
		if ip4[0] == 169 && ip4[1] == 254 {
			return true
		}
	}

	return false
}

// createOriginChecker creates an origin checker function based on configuration
func createOriginChecker(cfg *config.Config) func(r *http.Request) bool {
	if cfg == nil {
		// Allow all origins if no config (fallback for tests)
		return func(r *http.Request) bool {
			return true
		}
	}

	allowedOrigins := cfg.WebSocket.AllowedOrigins
	allowPrivateIPs := cfg.WebSocket.AllowPrivateIPs

	return func(r *http.Request) bool {
		origin := r.Header.Get("Origin")

		// If no Origin header, we can't verify - reject
		if origin == "" {
			return false
		}

		// If the origin is in the allowed list, accept
		for _, allowedOrigin := range allowedOrigins {
			if origin == allowedOrigin {
				return true
			}
		}

		// If private IPs are allowed, check if the origin's IP is private
		if allowPrivateIPs {
			// Parse the origin URL to extract host
			url, err := parseOrigin(origin)
			if err != nil {
				return false
			}

			// Check if the host is a private IP
			if isPrivateIP(url.Host) {
				return true
			}
		}

		return false
	}
}

// parseOrigin parses an origin string and returns the URL components
func parseOrigin(origin string) (*struct {
	Scheme string
	Host   string
	Port   string
}, error) {
	// Remove any trailing slash
	origin = strings.TrimSuffix(origin, "/")

	// Check if it's a valid URL
	if !strings.Contains(origin, "://") {
		// Assume http if no scheme
		// Also handle //example.com case (protocol-relative URL)
		if strings.HasPrefix(origin, "//") {
			origin = "http:" + origin
		} else {
			origin = "http://" + origin
		}
	}

	url, err := url.Parse(origin)
	if err != nil {
		return nil, err
	}

	return &struct {
		Scheme string
		Host   string
		Port   string
	}{
		Scheme: url.Scheme,
		Host:   url.Hostname(),
		Port:   url.Port(),
	}, nil
}

// WebSocketHandler handles WebSocket connections for sessions.
// Authentication is NOT done via URL query (the token would end up in access
// logs and reverse-proxy logs). Instead, the client must send an
// {"type":"auth","token":"<PIN>"} message as its first message; the
// connection is registered with the hub only after a valid token.
func WebSocketHandler(c *gin.Context) {
	deps := getDeps()

	// Get the config, use DefaultConfig if nil
	cfg := deps.Config
	if cfg == nil {
		cfg = config.DefaultConfig()
	}

	// Create upgrader with origin checker from config
	upgrader := &websocket.Upgrader{
		ReadBufferSize:  websocketReadBufferSize,
		WriteBufferSize: websocketWriteBufferSize,
		CheckOrigin:     createOriginChecker(cfg),
	}

	// Validate session before upgrading
	sessionIDStr := c.Param("id")
	sessionID, err := uuid.Parse(sessionIDStr)
	if err != nil {
		slog.Warn("Invalid session ID", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid session ID"})
		return
	}

	if _, exists := deps.SessionStore.Get(sessionID); !exists {
		slog.Warn("Session not found", "session_id", sessionID)
		c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
		return
	}

	// Upgrade to WebSocket (unauthenticated state until the auth message)
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		slog.Error("Failed to upgrade to WebSocket", "error", err)
		return
	}

	if !authenticateWebSocket(conn, deps, sessionID) {
		return
	}

	client := &ws.Client{Conn: conn, SessionID: sessionID, Hub: deps.WebSocketHub}
	deps.WebSocketHub.Register(client)
	defer deps.WebSocketHub.Unregister(client)

	// Get WebSocket config, use defaults if not set
	wsConfig := cfg.WebSocket
	if wsConfig.ReadDeadline <= 0 {
		wsConfig.ReadDeadline = 60 * time.Second
	}
	if wsConfig.PingInterval <= 0 {
		wsConfig.PingInterval = 30 * time.Second
	}

	// Reset read deadline for the authenticated connection lifetime
	conn.SetReadDeadline(time.Now().Add(wsConfig.ReadDeadline))

	// Handle pong messages to reset read deadline
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(wsConfig.ReadDeadline))
		return nil
	})

	// Start ping goroutine to keep connection alive
	go func() {
		ticker := time.NewTicker(wsConfig.PingInterval)
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
				slog.Warn("WebSocket unexpected close", "error", err)
			}
			break
		}

		// Validate message is valid JSON
		var msg map[string]interface{}
		if err := json.Unmarshal(message, &msg); err != nil {
			slog.Warn("Invalid WebSocket message format", "session_id", sessionID, "error", err)
			continue
		}

		// Never rebroadcast auth messages (they carry the PIN)
		if msg["type"] == "auth" || msg["event"] == "auth" {
			continue
		}

		slog.Debug("WebSocket message received", "session_id", sessionID, "length", len(message))
		deps.WebSocketHub.Broadcast(sessionID, message)
	}
}

// authenticateWebSocket reads the first message after the upgrade and
// validates the token (constant-time) against the session PIN. On any
// failure the connection is closed and false is returned.
func authenticateWebSocket(conn *websocket.Conn, deps *HandlerDeps, sessionID uuid.UUID) bool {
	// Short deadline for the auth handshake, plus a small message size limit
	conn.SetReadDeadline(time.Now().Add(websocketAuthDeadline))
	conn.SetReadLimit(4096)

	_, message, err := conn.ReadMessage()
	if err != nil {
		slog.Warn("WebSocket closed before auth", "session_id", sessionID)
		conn.Close()
		return false
	}

	var auth struct {
		Type  string `json:"type"`
		Event string `json:"event"`
		Token string `json:"token"`
	}
	if err := json.Unmarshal(message, &auth); err != nil {
		slog.Warn("WebSocket auth message not JSON", "session_id", sessionID)
		closeUnauthenticated(conn)
		return false
	}

	if auth.Type != "auth" && auth.Event != "auth" {
		slog.Warn("WebSocket first message was not auth", "session_id", sessionID)
		closeUnauthenticated(conn)
		return false
	}

	sess, exists := deps.SessionStore.Get(sessionID)
	if !exists {
		slog.Warn("Session gone before auth", "session_id", sessionID)
		closeUnauthenticated(conn)
		return false
	}

	if subtle.ConstantTimeCompare([]byte(sess.PIN), []byte(auth.Token)) != 1 {
		slog.Warn("WebSocket auth failed", "session_id", sessionID)
		closeUnauthenticated(conn)
		return false
	}

	return true
}

// closeUnauthenticated sends a policy-violation close frame and drops the
// connection (never log the reason together with a session ID; the UUID is
// random and unlinkable to a person, the PIN is never logged).
func closeUnauthenticated(conn *websocket.Conn) {
	_ = conn.WriteControl(
		websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "auth required"),
		time.Now().Add(time.Second),
	)
	_ = conn.Close()
}
