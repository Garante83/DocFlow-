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

	"dokumentenscanner/internal/config"
	"dokumentenscanner/internal/session"
	ws "dokumentenscanner/internal/websocket"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupFinalizeTestRouter(t testing.TB) *gin.Engine {
	t.Helper()
	store := session.NewStore()
	hub := ws.NewHub()
	go hub.Run()
	t.Cleanup(func() { hub.Stop() })

	cfg := config.DefaultConfig()
	cfg.Upload.MaxFileSizeMB = 10
	cfg.Upload.AllowedTypes = []string{"image/jpeg", "image/png", "image/webp"}
	cfg.PDF.MaxPages = 5
	cfg.PDF.JPEGQuality = 85
	cfg.PDF.CompressOutput = true

	Init(store, hub, cfg)

	router := gin.Default()
	return router
}

func createAndVerifySession(t *testing.T, router *gin.Engine) string {
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

func uploadTestImage(t *testing.T, router *gin.Engine, sessionID string) int {
	t.Helper()
	img := image.NewGray(image.Rect(0, 0, 100, 100))
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
	return w.Code
}

// === Finalize Tests ===

func TestFinalizeSuccess(t *testing.T) {
	router := setupFinalizeTestRouter(t)
	router.POST("/api/session", CreateSessionHandler)
	router.POST("/api/session/:id/verify-pin", VerifyPINHandler)
	router.POST("/api/session/:id/upload", UploadHandler)
	router.POST("/api/session/:id/finalize", FinalizeHandler)

	sessionID := createAndVerifySession(t, router)

	// Upload 2 images
	assert.Equal(t, http.StatusOK, uploadTestImage(t, router, sessionID))
	assert.Equal(t, http.StatusOK, uploadTestImage(t, router, sessionID))

	// Finalize
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/session/"+sessionID+"/finalize", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, "PDF generated successfully", resp["message"])
	assert.Equal(t, float64(2), resp["page_count"])
	assert.NotNil(t, resp["pdf_size"])
}

func TestFinalizeNoImages(t *testing.T) {
	router := setupFinalizeTestRouter(t)
	router.POST("/api/session", CreateSessionHandler)
	router.POST("/api/session/:id/verify-pin", VerifyPINHandler)
	router.POST("/api/session/:id/finalize", FinalizeHandler)

	sessionID := createAndVerifySession(t, router)

	// Finalize without uploading any images
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/session/"+sessionID+"/finalize", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestFinalizeInvalidSession(t *testing.T) {
	router := setupFinalizeTestRouter(t)
	router.POST("/api/session/:id/finalize", FinalizeHandler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/session/00000000-0000-0000-0000-000000000000/finalize", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestFinalizeInvalidUUID(t *testing.T) {
	router := setupFinalizeTestRouter(t)
	router.POST("/api/session/:id/finalize", FinalizeHandler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/session/not-a-uuid/finalize", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// === Multi-Page Upload Tests ===

func TestMultipleUploadsAppendImages(t *testing.T) {
	router := setupFinalizeTestRouter(t)
	router.POST("/api/session", CreateSessionHandler)
	router.POST("/api/session/:id/verify-pin", VerifyPINHandler)
	router.POST("/api/session/:id/upload", UploadHandler)

	sessionID := createAndVerifySession(t, router)

	// Upload 3 images
	for i := 0; i < 3; i++ {
		code := uploadTestImage(t, router, sessionID)
		assert.Equal(t, http.StatusOK, code)
	}
}

func TestMaxPagesLimit(t *testing.T) {
	store := session.NewStore()
	hub := ws.NewHub()
	go hub.Run()
	t.Cleanup(func() { hub.Stop() })

	cfg := config.DefaultConfig()
	cfg.PDF.MaxPages = 2

	Init(store, hub, cfg)

	router := gin.Default()
	router.POST("/api/session", CreateSessionHandler)
	router.POST("/api/session/:id/verify-pin", VerifyPINHandler)
	router.POST("/api/session/:id/upload", UploadHandler)

	sessionID := createAndVerifySession(t, router)

	// Upload 2 images (at limit)
	assert.Equal(t, http.StatusOK, uploadTestImage(t, router, sessionID))
	assert.Equal(t, http.StatusOK, uploadTestImage(t, router, sessionID))

	// Third upload should hit max pages limit
	w := httptest.NewRecorder()
	img := image.NewGray(image.Rect(0, 0, 100, 100))
	var imgBuf bytes.Buffer
	png.Encode(&imgBuf, img)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("image", "test.png")
	part.Write(imgBuf.Bytes())
	writer.Close()

	req, _ := http.NewRequest("POST", "/api/session/"+sessionID+"/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Contains(t, resp["error"], "maximum")
}
