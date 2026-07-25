package handlers

import (
	"net/http/httptest"
	"testing"

	"dokumentenscanner/internal/config"
	"dokumentenscanner/internal/session"
	ws "dokumentenscanner/internal/websocket"
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
