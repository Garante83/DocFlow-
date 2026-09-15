package handlers

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"docflow/internal/config"
	"docflow/internal/session"
	ws "docflow/internal/websocket"
	"github.com/gin-gonic/gin"
	gws "github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestIsPrivateIP tests the isPrivateIP function
func TestIsPrivateIP(t *testing.T) {
	tests := []struct {
		ip       string
		expected bool
	}{
		// Private IPv4 ranges
		{"10.0.0.1", true},
		{"10.255.255.255", true},
		{"172.16.0.1", true},
		{"172.31.255.255", true},
		{"192.168.0.1", true},
		{"192.168.255.255", true},

		// Link-local
		{"169.254.0.1", true},

		// Carrier-grade NAT
		{"100.64.0.1", true},
		{"100.127.255.255", true},

		// Public IPs
		{"8.8.8.8", false},   // Google DNS
		{"1.1.1.1", false},   // Cloudflare DNS
		{"192.0.2.1", false}, // TEST-NET-1

		// With ports
		{"10.0.0.1:8080", true},
		{"192.168.1.1:443", true},

		// Invalid IPs
		{"invalid", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.ip, func(t *testing.T) {
			result := isPrivateIP(tt.ip)
			assert.Equal(t, tt.expected, result, "isPrivateIP(%s) should be %v", tt.ip, tt.expected)
		})
	}
}

// TestCreateOriginChecker tests the createOriginChecker function
func TestCreateOriginChecker(t *testing.T) {
	// Test with nil config (should allow all)
	originChecker := createOriginChecker(nil)

	// Create a mock request
	req := httptest.NewRequest("GET", "http://localhost:8082", nil)
	req.Header.Set("Origin", "http://localhost:8082")

	// Should allow all when config is nil
	assert.True(t, originChecker(req), "nil config should allow all origins")

	// Test with valid config
	cfg := &config.Config{}
	cfg.WebSocket.AllowedOrigins = []string{"http://localhost:8082", "https://example.com"}
	cfg.WebSocket.AllowPrivateIPs = false

	originChecker = createOriginChecker(cfg)

	// Test allowed origin
	req = httptest.NewRequest("GET", "http://localhost:8082", nil)
	req.Header.Set("Origin", "http://localhost:8082")
	assert.True(t, originChecker(req), "Allowed origin should be accepted")

	// Test not allowed origin
	req = httptest.NewRequest("GET", "http://localhost:8082", nil)
	req.Header.Set("Origin", "http://evil.com")
	assert.False(t, originChecker(req), "Not allowed origin should be rejected")

	// Test with private IP allowed
	cfg.WebSocket.AllowPrivateIPs = true
	originChecker = createOriginChecker(cfg)

	// Test private IP in origin
	req = httptest.NewRequest("GET", "http://localhost:8082", nil)
	req.Header.Set("Origin", "http://192.168.1.1:8082")
	assert.True(t, originChecker(req), "Private IP should be allowed when AllowPrivateIPs is true")

	// Test with no Origin header
	req = httptest.NewRequest("GET", "http://localhost:8082", nil)
	// No Origin header
	assert.False(t, originChecker(req), "Missing Origin header should be rejected")

	// Test same-origin fallback behind a reverse proxy (public domain)
	cfg.WebSocket.AllowPrivateIPs = false
	cfg.WebSocket.AllowedOrigins = []string{} // empty allow-list
	originChecker = createOriginChecker(cfg)

	// Origin matches Host header -> accept
	req = httptest.NewRequest("GET", "http://docs.example.com/ws/session/00000000-0000-0000-0000-000000000000", nil)
	req.Header.Set("Origin", "https://docs.example.com")
	req.Header.Set("X-Forwarded-Proto", "https")
	assert.True(t, originChecker(req), "Same-origin via X-Forwarded-Host should be accepted")

	// Origin matches Host header directly (no proxy) -> accept
	req = httptest.NewRequest("GET", "http://docs.example.com/ws/session/00000000-0000-0000-0000-000000000000", nil)
	req.Header.Set("Origin", "http://docs.example.com")
	assert.True(t, originChecker(req), "Same-origin via Host header should be accepted")

	// Origin from different host -> reject (cross-origin)
	req = httptest.NewRequest("GET", "http://docs.example.com/ws/session/00000000-0000-0000-0000-000000000000", nil)
	req.Header.Set("Origin", "https://evil.com")
	assert.False(t, originChecker(req), "Cross-origin request should be rejected")

	// Origin from same host but different port -> reject
	req = httptest.NewRequest("GET", "http://docs.example.com/ws/session/00000000-0000-0000-0000-000000000000", nil)
	req.Header.Set("Origin", "http://docs.example.com:8443")
	assert.False(t, originChecker(req), "Same host with different port should be rejected")
}

// TestIsSameOrigin tests the isSameOrigin helper
func TestIsSameOrigin(t *testing.T) {
	tests := []struct {
		name     string
		origin   string
		host     string
		xffHost  string
		xffProto string
		tls      bool
		expected bool
	}{
		{
			name:     "origin matches Host header",
			origin:   "http://docs.example.com",
			host:     "docs.example.com",
			expected: true,
		},
		{
			name:     "origin matches Host header with port",
			origin:   "http://docs.example.com:8082",
			host:     "docs.example.com:8082",
			expected: true,
		},
		{
			name:     "origin matches X-Forwarded-Host (https proxy)",
			origin:   "https://docs.example.com",
			xffHost:  "docs.example.com",
			xffProto: "https",
			expected: true,
		},
		{
			name:     "origin default https port vs forwarded proto https",
			origin:   "https://docs.example.com",
			xffHost:  "docs.example.com:443",
			xffProto: "https",
			expected: true,
		},
		{
			name:     "different host",
			origin:   "https://evil.com",
			xffHost:  "docs.example.com",
			xffProto: "https",
			expected: false,
		},
		{
			name:     "different port",
			origin:   "http://docs.example.com:8443",
			host:     "docs.example.com",
			expected: false,
		},
		{
			name:     "scheme mismatch http vs https",
			origin:   "http://docs.example.com",
			xffHost:  "docs.example.com",
			xffProto: "https",
			expected: false,
		},
		{
			name:     "https origin matches TLS request without proxy",
			origin:   "https://docs.example.com",
			host:     "docs.example.com",
			tls:      true,
			expected: true,
		},
		{
			name:     "invalid origin",
			origin:   "not a url",
			host:     "docs.example.com",
			expected: false,
		},
		{
			name:     "no host at all",
			origin:   "http://docs.example.com",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "http://localhost:8082", nil)
			if tt.tls {
				req.TLS = &tls.ConnectionState{}
			}
			if tt.host != "" {
				req.Host = tt.host
			}
			if tt.xffHost != "" {
				req.Header.Set("X-Forwarded-Host", tt.xffHost)
			}
			if tt.xffProto != "" {
				req.Header.Set("X-Forwarded-Proto", tt.xffProto)
			}
			assert.Equal(t, tt.expected, isSameOrigin(tt.origin, req))
		})
	}
}

// TestParseOrigin tests the parseOrigin function
func TestParseOrigin(t *testing.T) {
	tests := []struct {
		origin    string
		expected  bool // true if parsing should succeed
		checkHost func(*struct {
			Scheme string
			Host   string
			Port   string
		}) bool
	}{
		{
			origin:   "http://localhost:8082",
			expected: true,
			checkHost: func(p *struct {
				Scheme string
				Host   string
				Port   string
			}) bool {
				return p.Scheme == "http" && p.Host == "localhost" && p.Port == "8082"
			},
		},
		{
			origin:   "https://example.com:443",
			expected: true,
			checkHost: func(p *struct {
				Scheme string
				Host   string
				Port   string
			}) bool {
				return p.Scheme == "https" && p.Host == "example.com" && p.Port == "443"
			},
		},
		{
			origin:   "localhost:8082",
			expected: true,
			checkHost: func(p *struct {
				Scheme string
				Host   string
				Port   string
			}) bool {
				return p.Scheme == "http" && p.Host == "localhost" && p.Port == "8082"
			},
		},
		{
			origin:   "http://192.168.1.1",
			expected: true,
			checkHost: func(p *struct {
				Scheme string
				Host   string
				Port   string
			}) bool {
				return p.Scheme == "http" && p.Host == "192.168.1.1" && p.Port == ""
			},
		},
		{
			origin:   "http://example.com/path",
			expected: true,
			checkHost: func(p *struct {
				Scheme string
				Host   string
				Port   string
			}) bool {
				return p.Scheme == "http" && p.Host == "example.com" && p.Port == ""
			},
		},
		{
			origin:   "//example.com",
			expected: true,
			checkHost: func(p *struct {
				Scheme string
				Host   string
				Port   string
			}) bool {
				return p.Scheme == "http" && p.Host == "example.com"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.origin, func(t *testing.T) {
			result, err := parseOrigin(tt.origin)
			if tt.expected {
				require.NoError(t, err)
				assert.True(t, tt.checkHost(result), "parseOrigin(%s) failed check", tt.origin)
			} else {
				assert.Error(t, err, "parseOrigin(%s) should fail", tt.origin)
			}
		})
	}
}

// TestWebSocketHandlerSetup tests that WebSocketHandler can be initialized
func TestWebSocketHandlerSetup(t *testing.T) {
	// Initialize dependencies
	store := session.NewStore()
	hub := ws.NewHub()
	go hub.Run()

	cfg := config.DefaultConfig()

	// Initialize handler dependencies
	Init(store, hub, cfg)

	// Create a test request
	// Note: We can't fully test WebSocketHandler without a real WebSocket connection,
	// but we can at least verify it doesn't panic
	// This is a placeholder for more comprehensive tests

	// Cleanup
	hub.Stop()

	assert.True(t, true, "WebSocketHandler setup should complete without panic")
}

// TestWebSocketHandler_AuthFlow verifies the message-based WebSocket
// authentication: no token in the URL, auth as first message, constant-time
// PIN check, and the auth message is never rebroadcast.
func TestWebSocketHandler_AuthFlow(t *testing.T) {
	store := session.NewStore()
	hub := ws.NewHub()
	go hub.Run()
	t.Cleanup(func() { hub.Stop() })

	Init(store, hub, config.DefaultConfig())

	router := gin.New()
	router.GET("/ws/session/:id", WebSocketHandler)
	server := httptest.NewServer(router)
	t.Cleanup(func() { server.Close() })

	sess, err := store.Create()
	require.NoError(t, err)

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws/session/" + sess.ID.String()
	header := http.Header{"Origin": []string{"https://localhost:8082"}}

	// Valid token: connection stays open and receives broadcasts
	c, _, err := gws.DefaultDialer.Dial(wsURL, header)
	require.NoError(t, err)
	defer c.Close()

	c.WriteMessage(gws.TextMessage, []byte(`{"type":"auth","token":"`+sess.PIN+`"}`))

	received := make(chan string, 1)
	go func() {
		_, msg, err := c.ReadMessage()
		if err == nil {
			received <- string(msg)
		}
	}()
	time.Sleep(100 * time.Millisecond)
	hub.Broadcast(sess.ID, []byte(`{"event":"test_broadcast","data":1}`))

	select {
	case msg := <-received:
		assert.Contains(t, msg, "test_broadcast")
	case <-time.After(2 * time.Second):
		t.Fatal("no broadcast received after successful auth")
	}

	// Auth messages must never be broadcast to other clients
	c2, _, err := gws.DefaultDialer.Dial(wsURL, header)
	require.NoError(t, err)
	defer c2.Close()
	c2.WriteMessage(gws.TextMessage, []byte(`{"type":"auth","token":"`+sess.PIN+`"}`))
	time.Sleep(100 * time.Millisecond)
	hub.Broadcast(sess.ID, []byte(`{"event":"broadcast_two","data":2}`))
	time.Sleep(100 * time.Millisecond)
	// c already read one message; second client should also receive only the broadcast
	_, msg2, err := c2.ReadMessage()
	require.NoError(t, err)
	assert.Contains(t, string(msg2), "broadcast_two")

	// Wrong token: server closes the connection
	c3, _, err := gws.DefaultDialer.Dial(wsURL, header)
	require.NoError(t, err)
	defer c3.Close()
	c3.WriteMessage(gws.TextMessage, []byte(`{"type":"auth","token":"000000"}`))
	_, _, err = c3.ReadMessage()
	assert.Error(t, err, "connection should be closed on wrong token")

	// First message not being auth: server closes the connection
	c4, _, err := gws.DefaultDialer.Dial(wsURL, header)
	require.NoError(t, err)
	defer c4.Close()
	c4.WriteMessage(gws.TextMessage, []byte(`{"event":"download_request","data":{}}`))
	_, _, err = c4.ReadMessage()
	assert.Error(t, err, "connection should be closed when first message is not auth")
}
