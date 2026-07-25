package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"dokumentenscanner/internal/config"
	"dokumentenscanner/internal/session"
	ws "dokumentenscanner/internal/websocket"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func setupTestRouter(t testing.TB) *gin.Engine {
	// Initialize dependencies for qrcode tests
	store := session.NewStore()
	hub := ws.NewHub()
	go hub.Run()
	
	// Ensure hub is stopped after test completes
	t.Cleanup(func() {
		hub.Stop()
	})

	cfg := config.DefaultConfig()
	Init(store, hub, cfg)

	router := gin.Default()
	api := router.Group("/api")
	{
		api.POST("/session", CreateSessionHandler)
		api.POST("/session/:id/verify-pin", VerifyPINHandler)
		api.GET("/session/:id/qrcode", QRCodeHandler)
		api.POST("/session/:id/upload", UploadHandler)
		api.GET("/session/:id/pdf", PDFHandler)
		api.DELETE("/api/session/:id", DeleteSessionHandler)
	}
	return router
}

// TestGetFrontendURL tests the getFrontendURL function
func TestGetFrontendURL(t *testing.T) {
	// Save old env
	oldURL := os.Getenv("FRONTEND_URL")
	defer func() {
		if oldURL != "" {
			os.Setenv("FRONTEND_URL", oldURL)
		} else {
			os.Unsetenv("FRONTEND_URL")
		}
	}()

	// Test with FRONTEND_URL set
	os.Setenv("FRONTEND_URL", "https://custom.frontend.com")
	url := getFrontendURL()
	assert.Equal(t, "https://custom.frontend.com", url)

	// Test without FRONTEND_URL (auto-detect)
	os.Unsetenv("FRONTEND_URL")
	url = getFrontendURL()
	// Should return LAN IP + port
	assert.NotEmpty(t, url)
	assert.Contains(t, url, "https://")
}

// TestGetDefaultPort tests the getDefaultPort function
func TestGetDefaultPort(t *testing.T) {
	// Save old env
	oldPort := os.Getenv("PORT")
	defer func() {
		if oldPort != "" {
			os.Setenv("PORT", oldPort)
		} else {
			os.Unsetenv("PORT")
		}
	}()

	// Test with PORT set
	os.Setenv("PORT", "9090")
	port := getDefaultPort()
	assert.Equal(t, ":9090", port)

	// Test without PORT (default)
	os.Unsetenv("PORT")
	port = getDefaultPort()
	assert.Equal(t, ":8082", port)
}

// TestGetLocalIP tests the getLocalIP function
func TestGetLocalIP(t *testing.T) {
	ip := getLocalIP()
	assert.NotEmpty(t, ip)
	// Should be a valid IP address (either localhost or LAN IP)
	assert.True(t, strings.Contains(ip, ".") || ip == "localhost")
}

// TestQRCodeHandler tests the QRCodeHandler
func TestQRCodeHandler(t *testing.T) {
	router := setupTestRouter(t)

	// Create a session first using the router's store
	req, _ := http.NewRequest("POST", "/api/session", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	require.Equal(t, http.StatusOK, w.Code)
	
	var sessionResp map[string]string
	json.Unmarshal(w.Body.Bytes(), &sessionResp)
	sessionID := sessionResp["session_id"]
	
	// Now test QRCodeHandler with the valid session
	req = httptest.NewRequest("GET", "/api/session/"+sessionID+"/qrcode", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "image/png", w.Header().Get("Content-Type"))
	assert.NotEmpty(t, w.Body.Bytes(), "QR code should not be empty")
	// PNG files should be > 100 bytes
	assert.True(t, len(w.Body.Bytes()) > 100, "QR code PNG should be > 100 bytes")
}

// TestQRCodeHandlerInvalidSession tests QRCodeHandler with invalid session
func TestQRCodeHandlerInvalidSession(t *testing.T) {
	router := setupTestRouter(t)

	// Test with non-existent session ID
	req := httptest.NewRequest("GET", "/api/session/"+uuid.New().String()+"/qrcode", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// TestQRCodeHandlerInvalidID tests QRCodeHandler with invalid UUID
func TestQRCodeHandlerInvalidID(t *testing.T) {
	router := setupTestRouter(t)

	// Test with invalid UUID format
	req := httptest.NewRequest("GET", "/api/session/invalid-uuid/qrcode", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
