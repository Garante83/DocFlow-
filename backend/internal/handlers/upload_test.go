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
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupUploadTestRouter() *gin.Engine {
	// Initialize dependencies
	store := session.NewStore()
	hub := ws.NewHub()
	go hub.Run()
	
	cfg := &config.Config{}
	cfg.Upload.MaxFileSizeMB = 10
	cfg.Upload.AllowedTypes = []string{"image/jpeg", "image/png", "image/webp"}
	
	Init(store, hub, cfg)
	
	router := gin.Default()
	router.POST("/api/session", CreateSessionHandler)
	router.POST("/api/session/:id/verify-pin", VerifyPINHandler)
	router.POST("/api/session/:id/upload", UploadHandler)
	return router
}

// TestUploadHandlerSuccess tests successful image upload
func TestUploadHandlerSuccess(t *testing.T) {
	router := setupUploadTestRouter()
	
	// Create a session first
	req, _ := http.NewRequest("POST", "/api/session", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	require.Equal(t, http.StatusOK, w.Code)
	
	var sessionResp map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &sessionResp)
	require.NoError(t, err)
	
	sessionID := sessionResp["session_id"]
	pin := sessionResp["pin"]
	
	// Verify PIN
	reqBody := map[string]string{"pin": pin}
	reqBodyBytes, _ := json.Marshal(reqBody)
	req = httptest.NewRequest("POST", "/api/session/"+sessionID+"/verify-pin", bytes.NewReader(reqBodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	require.Equal(t, http.StatusOK, w.Code)
	
	// Now upload an image
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
	
	req = httptest.NewRequest("POST", "/api/session/"+sessionID+"/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	
	var uploadResp map[string]string
	err = json.Unmarshal(w.Body.Bytes(), &uploadResp)
	require.NoError(t, err)
	assert.Equal(t, "image uploaded successfully", uploadResp["message"])
}

// TestUploadHandlerInvalidSession tests upload with non-existent session
func TestUploadHandlerInvalidSession(t *testing.T) {
	router := setupUploadTestRouter()
	
	// Try to upload to non-existent session
	img := image.NewGray(image.Rect(0, 0, 10, 10))
	var imgBuf bytes.Buffer
	png.Encode(&imgBuf, img)
	
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("image", "test.png")
	part.Write(imgBuf.Bytes())
	writer.Close()
	
	req := httptest.NewRequest("POST", "/api/session/"+uuid.New().String()+"/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// TestUploadHandlerNotAllowed tests upload when session not allowed
func TestUploadHandlerNotAllowed(t *testing.T) {
	router := setupUploadTestRouter()
	
	// Create a session but don't verify PIN (session is not in upload_allowed state)
	req, _ := http.NewRequest("POST", "/api/session", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	require.Equal(t, http.StatusOK, w.Code)
	
	var sessionResp map[string]string
	json.Unmarshal(w.Body.Bytes(), &sessionResp)
	sessionID := sessionResp["session_id"]
	
	// Try to upload without verifying PIN first
	img := image.NewGray(image.Rect(0, 0, 10, 10))
	var imgBuf bytes.Buffer
	png.Encode(&imgBuf, img)
	
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("image", "test.png")
	part.Write(imgBuf.Bytes())
	writer.Close()
	
	req = httptest.NewRequest("POST", "/api/session/"+sessionID+"/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusForbidden, w.Code)
}

// TestUploadHandlerInvalidFile tests upload with invalid file
func TestUploadHandlerInvalidFile(t *testing.T) {
	router := setupUploadTestRouter()
	
	// Create and verify session
	req, _ := http.NewRequest("POST", "/api/session", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	var sessionResp map[string]string
	json.Unmarshal(w.Body.Bytes(), &sessionResp)
	sessionID := sessionResp["session_id"]
	pin := sessionResp["pin"]
	
	// Verify PIN
	reqBody := map[string]string{"pin": pin}
	reqBodyBytes, _ := json.Marshal(reqBody)
	req = httptest.NewRequest("POST", "/api/session/"+sessionID+"/verify-pin", bytes.NewReader(reqBodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	// Upload non-image file (text file)
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("image", "test.txt")
	part.Write([]byte("not an image"))
	writer.Close()
	
	req = httptest.NewRequest("POST", "/api/session/"+sessionID+"/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	// Should fail because file is not a valid image
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestUploadHandlerFileTooLarge tests upload with file exceeding size limit
func TestUploadHandlerFileTooLarge(t *testing.T) {
	router := setupUploadTestRouter()
	
	// Create and verify session
	req, _ := http.NewRequest("POST", "/api/session", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	var sessionResp map[string]string
	json.Unmarshal(w.Body.Bytes(), &sessionResp)
	sessionID := sessionResp["session_id"]
	pin := sessionResp["pin"]
	
	// Verify PIN
	reqBody := map[string]string{"pin": pin}
	reqBodyBytes, _ := json.Marshal(reqBody)
	req = httptest.NewRequest("POST", "/api/session/"+sessionID+"/verify-pin", bytes.NewReader(reqBodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	// Create a large image (> 10MB)
	// Create a 4000x4000 image which is approximately 16MB as PNG
	img := image.NewGray(image.Rect(0, 0, 4000, 4000))
	var imgBuf bytes.Buffer
	png.Encode(&imgBuf, img)
	
	// Verify image is large enough
	assert.True(t, imgBuf.Len() > 10*1024*1024, "Test image should be > 10MB")
	
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("image", "large.png")
	part.Write(imgBuf.Bytes())
	writer.Close()
	
	req = httptest.NewRequest("POST", "/api/session/"+sessionID+"/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	// Should fail because file is too large
	assert.Equal(t, http.StatusBadRequest, w.Code)
	
	// Check error message contains size info
	var errorResp map[string]string
	json.Unmarshal(w.Body.Bytes(), &errorResp)
	assert.Contains(t, errorResp["error"], "file size exceeds")
}

// TestUploadHandlerMissingFormField tests upload with missing image field
func TestUploadHandlerMissingFormField(t *testing.T) {
	router := setupUploadTestRouter()
	
	// Create and verify session
	req, _ := http.NewRequest("POST", "/api/session", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	var sessionResp map[string]string
	json.Unmarshal(w.Body.Bytes(), &sessionResp)
	sessionID := sessionResp["session_id"]
	pin := sessionResp["pin"]
	
	// Verify PIN
	reqBody := map[string]string{"pin": pin}
	reqBodyBytes, _ := json.Marshal(reqBody)
	req = httptest.NewRequest("POST", "/api/session/"+sessionID+"/verify-pin", bytes.NewReader(reqBodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	// Upload without image field
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	writer.Close()
	
	req = httptest.NewRequest("POST", "/api/session/"+sessionID+"/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
