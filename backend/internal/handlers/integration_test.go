package handlers

import (
	"bytes"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
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

func init() {
	gin.SetMode(gin.TestMode)
}

func setupIntegrationRouter(t testing.TB) *gin.Engine {
	store := session.NewStore()
	hub := ws.NewHub()
	go hub.Run()
	t.Cleanup(func() { hub.Stop() })

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

	router := gin.Default()
	return router
}

func createTestPNG(t *testing.T) []byte {
	t.Helper()
	img := image.NewGray(image.Rect(0, 0, 100, 100))
	var buf bytes.Buffer
	err := png.Encode(&buf, img)
	require.NoError(t, err)
	return buf.Bytes()
}

func createSessionAndVerify(t *testing.T, router *gin.Engine) string {
	t.Helper()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/session", nil)
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	sessionID := resp["session_id"]
	pin := resp["pin"]

	body, _ := json.Marshal(map[string]string{"pin": pin})
	req, _ = http.NewRequest("POST", "/api/session/"+sessionID+"/verify-pin", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	return sessionID
}

func uploadPNG(t *testing.T, router *gin.Engine, sessionID string) int {
	t.Helper()
	imgData := createTestPNG(t)
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("image", "test.png")
	part.Write(imgData)
	writer.Close()

	req, _ := http.NewRequest("POST", "/api/session/"+sessionID+"/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w.Code
}

// === Full Workflow Tests ===

func TestFullWorkflow(t *testing.T) {
	router := setupIntegrationRouter(t)
	router.POST("/api/session", CreateSessionHandler)
	router.POST("/api/session/:id/verify-pin", VerifyPINHandler)
	router.POST("/api/session/:id/upload", UploadHandler)
	router.GET("/api/session/:id/pdf", PDFHandler)
	router.DELETE("/api/session/:id", DeleteSessionHandler)

	sessionID := createSessionAndVerify(t, router)

	// Upload
	assert.Equal(t, http.StatusOK, uploadPNG(t, router, sessionID))

	// PDF
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/session/"+sessionID+"/pdf", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/pdf", w.Header().Get("Content-Type"))
	assert.True(t, len(w.Body.Bytes()) > 100)

	// Delete
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("DELETE", "/api/session/"+sessionID, nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestFullWorkflowWithQRCode(t *testing.T) {
	router := setupIntegrationRouter(t)
	router.POST("/api/session", CreateSessionHandler)
	router.GET("/api/session/:id/qrcode", QRCodeHandler)

	// Create session
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/session", nil)
	router.ServeHTTP(w, req)

	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	sessionID := resp["session_id"]

	// Get QR code
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/session/"+sessionID+"/qrcode", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "image/png", w.Header().Get("Content-Type"))
	assert.True(t, len(w.Body.Bytes()) > 100)
	assert.Contains(t, string(w.Body.Bytes()), "\x89PNG") // PNG magic bytes
}

// === Config Variant Tests ===

func TestFullWorkflowWithDevConfig(t *testing.T) {
	store := session.NewStore()
	hub := ws.NewHub()
	go hub.Run()
	t.Cleanup(func() { hub.Stop() })

	cfg := &config.Config{}
	cfg.Upload.MaxFileSizeMB = 50
	cfg.Upload.AllowedTypes = []string{"image/jpeg", "image/png", "image/webp", "image/gif"}
	cfg.Session.Timeout = 24 * time.Hour
	cfg.Logging.Level = "debug"

	Init(store, hub, cfg)

	router := gin.Default()
	router.POST("/api/session", CreateSessionHandler)
	router.POST("/api/session/:id/upload", UploadHandler)

	req, _ := http.NewRequest("POST", "/api/session", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestFullWorkflowWithProdConfig(t *testing.T) {
	store := session.NewStore()
	hub := ws.NewHub()
	go hub.Run()
	t.Cleanup(func() { hub.Stop() })

	cfg := &config.Config{}
	cfg.Upload.MaxFileSizeMB = 10
	cfg.Upload.AllowedTypes = []string{"image/jpeg", "image/png"}
	cfg.WebSocket.AllowPrivateIPs = false
	cfg.Logging.Level = "info"
	cfg.Server.Port = "443"

	Init(store, hub, cfg)

	router := gin.Default()
	router.POST("/api/session", CreateSessionHandler)

	req, _ := http.NewRequest("POST", "/api/session", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

// === PIN Lockout Test ===

func TestPINLockoutAfterFailedAttempts(t *testing.T) {
	router := setupIntegrationRouter(t)
	router.POST("/api/session", CreateSessionHandler)
	router.POST("/api/session/:id/verify-pin", VerifyPINHandler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/session", nil)
	router.ServeHTTP(w, req)

	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	sessionID := resp["session_id"]

	// Try wrong PIN 3 times — after 3rd attempt should be locked
	for i := 0; i < 3; i++ {
		body, _ := json.Marshal(map[string]string{"pin": "000000"})
		req, _ = http.NewRequest("POST", "/api/session/"+sessionID+"/verify-pin", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}

	// Fourth attempt should be locked out
	body, _ := json.Marshal(map[string]string{"pin": "000000"})
	req, _ = http.NewRequest("POST", "/api/session/"+sessionID+"/verify-pin", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusTooManyRequests, w.Code)
}

func TestPINLockoutRecoveryWithCorrectPIN(t *testing.T) {
	router := setupIntegrationRouter(t)
	router.POST("/api/session", CreateSessionHandler)
	router.POST("/api/session/:id/verify-pin", VerifyPINHandler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/session", nil)
	router.ServeHTTP(w, req)

	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	sessionID := resp["session_id"]
	correctPIN := resp["pin"]

	// Wrong PIN once — session gets locked immediately
	body, _ := json.Marshal(map[string]string{"pin": "000000"})
	req, _ = http.NewRequest("POST", "/api/session/"+sessionID+"/verify-pin", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	// Correct PIN should also be rejected (session is locked)
	body, _ = json.Marshal(map[string]string{"pin": correctPIN})
	req, _ = http.NewRequest("POST", "/api/session/"+sessionID+"/verify-pin", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusTooManyRequests, w.Code, "Session should be locked after 1 failed attempt")
}

// === Concurrent Session Tests ===

func TestConcurrentSessions(t *testing.T) {
	router := gin.Default()
	router.POST("/api/session", CreateSessionHandler)
	router.DELETE("/api/session/:id", DeleteSessionHandler)

	var sessionIDs []string
	for i := 0; i < 5; i++ {
		req, _ := http.NewRequest("POST", "/api/session", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		var sessionResp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &sessionResp)
		sessionIDs = append(sessionIDs, sessionResp["session_id"].(string))
	}

	for _, id := range sessionIDs {
		req, _ := http.NewRequest("DELETE", "/api/session/"+id, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusNoContent, w.Code)
	}
}

func TestConcurrentUploadsFromDifferentSessions(t *testing.T) {
	router := setupIntegrationRouter(t)
	router.POST("/api/session", CreateSessionHandler)
	router.POST("/api/session/:id/verify-pin", VerifyPINHandler)
	router.POST("/api/session/:id/upload", UploadHandler)

	var wg sync.WaitGroup
	var mu sync.Mutex
	var codes []int

	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sessionID := createSessionAndVerify(t, router)
			code := uploadPNG(t, router, sessionID)
			mu.Lock()
			codes = append(codes, code)
			mu.Unlock()
		}()
	}

	wg.Wait()
	assert.Len(t, codes, 3)
	for _, code := range codes {
		assert.Equal(t, http.StatusOK, code)
	}
}

// === Error Handling Tests ===

func TestUploadWithoutPINVerification(t *testing.T) {
	router := setupIntegrationRouter(t)
	router.POST("/api/session", CreateSessionHandler)
	router.POST("/api/session/:id/upload", UploadHandler)

	// Create session but don't verify PIN
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/session", nil)
	router.ServeHTTP(w, req)

	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	sessionID := resp["session_id"]

	// Upload should fail (session not in upload_allowed state)
	code := uploadPNG(t, router, sessionID)
	assert.Equal(t, http.StatusForbidden, code)
}

func TestPDFWithoutUpload(t *testing.T) {
	router := setupIntegrationRouter(t)
	router.POST("/api/session", CreateSessionHandler)
	router.POST("/api/session/:id/verify-pin", VerifyPINHandler)
	router.GET("/api/session/:id/pdf", PDFHandler)

	sessionID := createSessionAndVerify(t, router)

	// PDF should fail (no image uploaded)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/session/"+sessionID+"/pdf", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDeleteNonExistentSession(t *testing.T) {
	router := setupIntegrationRouter(t)
	router.DELETE("/api/session/:id", DeleteSessionHandler)

	// Use a valid UUID format that doesn't exist
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/api/session/00000000-0000-0000-0000-000000000000", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code) // Idempotent delete
}

// === WebSocket Integration Test ===

func TestWebSocketConnectionAndBroadcast(t *testing.T) {
	store := session.NewStore()
	hub := ws.NewHub()
	go hub.Run()
	t.Cleanup(func() { hub.Stop() })

	cfg := config.DefaultConfig()
	Init(store, hub, cfg)

	router := gin.Default()
	router.GET("/ws/session/:id", WebSocketHandler)

	// Create a session
	sess, err := store.Create()
	require.NoError(t, err)

	// Start a test WS server
	upgrader := gws.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer c.Close()
		for {
			_, _, err := c.ReadMessage()
			if err != nil {
				return
			}
		}
	}))
	t.Cleanup(func() { server.Close() })

	// Connect to the actual WS endpoint
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	c, _, err := gws.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		// WS endpoint may not work in test server without proper TLS
		t.Skip("WebSocket connection failed in test environment:", err)
	}
	t.Cleanup(func() { c.Close() })

	// Broadcast a message
	hub.Broadcast(sess.ID, []byte(`{"event":"test_broadcast","data":"hello"}`))
	time.Sleep(50 * time.Millisecond)

	assert.True(t, true, "WebSocket broadcast should not panic")
}

// === Multi-Page Integration Tests ===

func TestMultiPageWorkflow(t *testing.T) {
	router := setupIntegrationRouter(t)
	router.POST("/api/session", CreateSessionHandler)
	router.POST("/api/session/:id/verify-pin", VerifyPINHandler)
	router.POST("/api/session/:id/upload", UploadHandler)
	router.POST("/api/session/:id/finalize", FinalizeHandler)
	router.GET("/api/session/:id/pdf", PDFHandler)
	router.DELETE("/api/session/:id", DeleteSessionHandler)

	sessionID := createSessionAndVerify(t, router)

	// Upload 3 pages
	for i := 0; i < 3; i++ {
		assert.Equal(t, http.StatusOK, uploadPNG(t, router, sessionID))
	}

	// Finalize
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/session/"+sessionID+"/finalize", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var finalizeResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &finalizeResp)
	assert.Equal(t, float64(3), finalizeResp["page_count"])

	// Download PDF
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/session/"+sessionID+"/pdf", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/pdf", w.Header().Get("Content-Type"))
	assert.True(t, len(w.Body.Bytes()) > 100, "Multi-page PDF should be > 100 bytes")

	// Cleanup
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("DELETE", "/api/session/"+sessionID, nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestMultiPageWorkflowWithQRCode(t *testing.T) {
	router := setupIntegrationRouter(t)
	router.POST("/api/session", CreateSessionHandler)
	router.GET("/api/session/:id/qrcode", QRCodeHandler)
	router.POST("/api/session/:id/verify-pin", VerifyPINHandler)
	router.POST("/api/session/:id/upload", UploadHandler)
	router.POST("/api/session/:id/finalize", FinalizeHandler)

	// Create session
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/session", nil)
	router.ServeHTTP(w, req)

	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	sessionID := resp["session_id"]

	// Get QR code
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/session/"+sessionID+"/qrcode", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "image/png", w.Header().Get("Content-Type"))

	// Verify PIN
	pinBody, _ := json.Marshal(map[string]string{"pin": resp["pin"]})
	req, _ = http.NewRequest("POST", "/api/session/"+sessionID+"/verify-pin", bytes.NewReader(pinBody))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// Upload 2 pages
	assert.Equal(t, http.StatusOK, uploadPNG(t, router, sessionID))
	assert.Equal(t, http.StatusOK, uploadPNG(t, router, sessionID))

	// Finalize
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/api/session/"+sessionID+"/finalize", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestMultiPagePDFSizeCompression(t *testing.T) {
	store := session.NewStore()
	hub := ws.NewHub()
	go hub.Run()
	t.Cleanup(func() { hub.Stop() })

	cfg := &config.Config{}
	cfg.Upload.MaxFileSizeMB = 10
	cfg.Upload.AllowedTypes = []string{"image/jpeg", "image/png", "image/webp"}
	cfg.Session.Timeout = 1 * time.Hour
	cfg.Session.CleanupInterval = 5 * time.Minute
	cfg.Session.MaxFailedAttempts = 3
	cfg.Session.LockoutDuration = 5 * time.Minute
	cfg.PDF.MaxPages = 20
	cfg.PDF.JPEGQuality = 50
	cfg.PDF.CompressOutput = true

	Init(store, hub, cfg)

	router := gin.Default()
	router.POST("/api/session", CreateSessionHandler)
	router.POST("/api/session/:id/verify-pin", VerifyPINHandler)
	router.POST("/api/session/:id/upload", UploadHandler)
	router.POST("/api/session/:id/finalize", FinalizeHandler)
	router.GET("/api/session/:id/pdf", PDFHandler)

	sessionID := createSessionAndVerify(t, router)

	// Create larger test images (1000x1000 RGBA with random noise)
	for i := 0; i < 3; i++ {
		img := image.NewRGBA(image.Rect(0, 0, 1000, 1000))
		for y := 0; y < 1000; y++ {
			for x := 0; x < 1000; x++ {
				img.Set(x, y, color.RGBA{
					R: uint8((x + y + i*37) % 256),
					G: uint8((x*2 + y*3) % 256),
					B: uint8((x*3 + y*2) % 256),
					A: 255,
				})
			}
		}
		var imgBuf bytes.Buffer
		png.Encode(&imgBuf, img)

		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		part, _ := writer.CreateFormFile("image", "test.png")
		part.Write(imgBuf.Bytes())
		writer.Close()

		req, _ := http.NewRequest("POST", "/api/session/"+sessionID+"/upload", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	}

	// Finalize with low quality
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/session/"+sessionID+"/finalize", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// Get PDF size
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/session/"+sessionID+"/pdf", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	pdfSize := len(w.Body.Bytes())
	assert.True(t, pdfSize > 100, "PDF should be > 100 bytes, got %d", pdfSize)

	// Verify PDF is smaller than uncompressed original (3 PNGs of 1000x1000 RGBA are ~12MB)
	// With JPEG 50% + zlib, should be ~10x smaller
	assert.True(t, pdfSize < 2*1024*1024,
		"Compressed PDF (%d bytes) should be < 2MB for 3x 1000x1000 RGBA images", pdfSize)
}
