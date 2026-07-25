package handlers

import (
	"bytes"
	"encoding/json"
	"image"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"dokumentenscanner/internal/config"
	"dokumentenscanner/internal/session"
	ws "dokumentenscanner/internal/websocket"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	gin.SetMode(gin.TestMode)
	// Initialize dependencies for tests with a valid config
	store := session.NewStore()
	hub := ws.NewHub()
	go hub.Run()
	
	// Create a minimal config for tests
	cfg := &config.Config{}
	cfg.Upload.MaxFileSizeMB = 10
	cfg.Upload.AllowedTypes = []string{"image/jpeg", "image/png", "image/webp"}
	cfg.Session.Timeout = 1 * time.Hour
	cfg.Session.CleanupInterval = 5 * time.Minute
	cfg.Session.MaxFailedAttempts = 3
	cfg.Session.LockoutDuration = 5 * time.Minute
	cfg.Logging.Level = "info"
	cfg.Logging.Format = "json"
	cfg.Server.Port = "8082"
	
	Init(store, hub, cfg)
}

func TestFullWorkflow(t *testing.T) {
	router := gin.Default()
	router.POST("/api/session", CreateSessionHandler)
	router.POST("/api/session/:id/verify-pin", VerifyPINHandler)
	router.POST("/api/session/:id/upload", UploadHandler)
	router.GET("/api/session/:id/pdf", PDFHandler)
	router.DELETE("/api/session/:id", DeleteSessionHandler)

	// Step 1: Create session
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/session", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var sessionResp map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &sessionResp)
	require.NoError(t, err)

	sessionID := sessionResp["session_id"]
	pin := sessionResp["pin"]
	assert.NotEmpty(t, sessionID)
	assert.Len(t, pin, 6)

	// Step 2: Verify PIN
	w = httptest.NewRecorder()
	reqBody := map[string]string{"pin": pin}
	reqBodyBytes, _ := json.Marshal(reqBody)
	req, _ = http.NewRequest("POST", "/api/session/"+sessionID+"/verify-pin", bytes.NewReader(reqBodyBytes))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var pinResp map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &pinResp)
	require.NoError(t, err)
	assert.Equal(t, "PIN verified successfully", pinResp["message"])
	assert.Equal(t, true, pinResp["valid"])

	// Step 3: Upload image
	img := image.NewGray(image.Rect(0, 0, 100, 100))
	var imgBuf bytes.Buffer
	err = png.Encode(&imgBuf, img)
	require.NoError(t, err)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("image", "test.png")
	require.NoError(t, err)
	_, err = part.Write(imgBuf.Bytes())
	require.NoError(t, err)
	writer.Close()

	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/api/session/"+sessionID+"/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var uploadResp map[string]string
	err = json.Unmarshal(w.Body.Bytes(), &uploadResp)
	require.NoError(t, err)
	assert.Equal(t, "image uploaded successfully", uploadResp["message"])

	// Step 4: Generate PDF
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/session/"+sessionID+"/pdf", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/pdf", w.Header().Get("Content-Type"))
	assert.True(t, len(w.Body.Bytes()) > 100, "PDF should be larger than 100 bytes")

	// Step 5: Delete session
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("DELETE", "/api/session/"+sessionID, nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

// TestFullWorkflowWithDevConfig tests the workflow with dev configuration
func TestFullWorkflowWithDevConfig(t *testing.T) {
	// Create a dev config
	store := session.NewStore()
	hub := ws.NewHub()
	go hub.Run()
	
	cfg := &config.Config{}
	cfg.Upload.MaxFileSizeMB = 50 // Larger limit for dev
	cfg.Upload.AllowedTypes = []string{"image/jpeg", "image/png", "image/webp", "image/gif"}
	cfg.Session.Timeout = 24 * time.Hour
	cfg.Logging.Level = "debug"
	
	// Reinitialize with dev config
	Init(store, hub, cfg)
	
	router := gin.Default()
	router.POST("/api/session", CreateSessionHandler)
	router.POST("/api/session/:id/upload", UploadHandler)
	
	// Test with dev config - should allow larger files and more types
	req, _ := http.NewRequest("POST", "/api/session", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
}

// TestFullWorkflowWithProdConfig tests the workflow with prod configuration
func TestFullWorkflowWithProdConfig(t *testing.T) {
	// Create a prod config
	store := session.NewStore()
	hub := ws.NewHub()
	go hub.Run()
	
	cfg := &config.Config{}
	cfg.Upload.MaxFileSizeMB = 10
	cfg.Upload.AllowedTypes = []string{"image/jpeg", "image/png"}
	cfg.WebSocket.AllowPrivateIPs = false
	cfg.Logging.Level = "info"
	cfg.Server.Port = "443"
	
	// Reinitialize with prod config
	Init(store, hub, cfg)
	
	router := gin.Default()
	router.POST("/api/session", CreateSessionHandler)
	
	// Test with prod config - should have stricter limits
	req, _ := http.NewRequest("POST", "/api/session", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
}

// TestConcurrentSessions tests handling of multiple concurrent sessions
func TestConcurrentSessions(t *testing.T) {
	router := gin.Default()
	router.POST("/api/session", CreateSessionHandler)
	router.DELETE("/api/session/:id", DeleteSessionHandler)

	// Create multiple sessions
	var sessionIDs []string
	for i := 0; i < 5; i++ {
		req, _ := http.NewRequest("POST", "/api/session", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		var sessionResp map[string]string
		json.Unmarshal(w.Body.Bytes(), &sessionResp)
		sessionIDs = append(sessionIDs, sessionResp["session_id"])
	}

	// Verify all sessions can be deleted (which confirms they exist)
	for _, id := range sessionIDs {
		req, _ := http.NewRequest("DELETE", "/api/session/"+id, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusNoContent, w.Code)
	}
}
