package handlers

import (
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"dokumentenscanner/internal/config"
	ws "dokumentenscanner/internal/websocket"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// WebSocket configuration constants
const (
	websocketReadBufferSize  = 1024
	websocketWriteBufferSize = 1024
)

// isPrivateIP checks if the given IP address is a private IP
func isPrivateIP(ip string) bool {
	// Remove port if present
	if strings.Contains(ip, ":") {
		ip = strings.Split(ip, ":")[0]
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
	
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("Failed to upgrade to WebSocket: %v", err)
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
	_, exists := deps.SessionStore.Get(sessionID)
	if !exists {
		log.Println("Session not found:", sessionID)
		conn.Close()
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
	
	// Set initial read deadline
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
				log.Println("WebSocket error:", err)
			}
			break
		}

		log.Printf("Received message from session %s: %s", sessionID, message)
		deps.WebSocketHub.Broadcast(sessionID, message)
	}
}
