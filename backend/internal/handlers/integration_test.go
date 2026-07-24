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

	"dokumentenscanner/internal/session"
	ws "dokumentenscanner/internal/websocket"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	gin.SetMode(gin.TestMode)
	// Reinitialize store and hub for each test
	SessionStore = session.NewStore()
	WebSocketHub = ws.NewHub()
	go WebSocketHub.Run()
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
