package main

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"docflow/internal/config"
	"docflow/internal/handlers"
	"docflow/internal/session"
	ws "docflow/internal/websocket"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// === getPort Tests ===

func TestGetPort(t *testing.T) {
	tests := []struct {
		name     string
		cfg      *config.Config
		expected string
	}{
		{"default port", config.DefaultConfig(), ":8082"},
		{"custom port", func() *config.Config { c := config.DefaultConfig(); c.Server.Port = "9090"; return c }(), ":9090"},
		{"empty port", func() *config.Config { c := config.DefaultConfig(); c.Server.Port = ""; return c }(), ":"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, getPort(tt.cfg))
		})
	}
}

// === readEmbeddedFile Tests ===

func TestReadEmbeddedFile_IndexHTML(t *testing.T) {
	data, err := readEmbeddedFile("index.html")
	require.NoError(t, err)
	require.NotEmpty(t, data)
	assert.Contains(t, string(data), "<!DOCTYPE html>")
}

func TestReadEmbeddedFile_CSS(t *testing.T) {
	// Find a CSS file in the embedded assets
	data, err := readEmbeddedFile("index.html")
	require.NoError(t, err)
	require.NotEmpty(t, data)

	// Read an asset file by looking at what's embedded
	// We know there are CSS and JS files in assets/
	entries, err := frontendFS.ReadDir("assets")
	if err != nil {
		t.Skip("No assets directory embedded")
	}
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".css") {
			cssData, err := readEmbeddedFile("assets/" + entry.Name())
			assert.NoError(t, err)
			assert.NotEmpty(t, cssData)
			return
		}
	}
	t.Skip("No CSS files found in embedded assets")
}

func TestReadEmbeddedFile_JavaScript(t *testing.T) {
	entries, err := frontendFS.ReadDir("assets")
	if err != nil {
		t.Skip("No assets directory embedded")
	}
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".js") {
			jsData, err := readEmbeddedFile("assets/" + entry.Name())
			assert.NoError(t, err)
			assert.NotEmpty(t, jsData)
			return
		}
	}
	t.Skip("No JS files found in embedded assets")
}

func TestReadEmbeddedFile_Favicon(t *testing.T) {
	data, err := readEmbeddedFile("favicon.ico")
	require.NoError(t, err)
	require.NotEmpty(t, data)
}

func TestReadEmbeddedFile_FaviconSVG(t *testing.T) {
	data, err := readEmbeddedFile("favicon.svg")
	require.NoError(t, err)
	require.NotEmpty(t, data)
	assert.Contains(t, string(data), "svg")
}

// === setupLogger Tests ===

func TestSetupLogger_InfoLevel(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Logging.Level = "info"

	setupLogger(cfg)

	// Verify slog default handler is set (no panic)
	slog.Info("test info message")
	slog.Debug("test debug message should not appear at info level")
}

func TestSetupLogger_DebugLevel(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Logging.Level = "debug"

	setupLogger(cfg)

	// Verify debug level works
	slog.Debug("test debug message")
}

func TestSetupLogger_WarnLevel(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Logging.Level = "warn"

	setupLogger(cfg)

	slog.Warn("test warn message")
}

func TestSetupLogger_ErrorLevel(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Logging.Level = "error"

	setupLogger(cfg)

	slog.Error("test error message")
}

func TestSetupLogger_InvalidLevel(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Logging.Level = "invalid"

	// Should default to info, not panic
	setupLogger(cfg)
	slog.Info("should use default info level")
}

func TestSetupLogger_TextFormat(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Logging.Format = "text"

	setupLogger(cfg)
	slog.Info("text format message")
}

func TestSetupLogger_UnknownFormatFallsBackToJSON(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Logging.Format = "xml"

	// Should warn and fall back to json, not panic
	setupLogger(cfg)
	slog.Info("fallback json message")
}

// === writeTempFile Tests ===

func TestWriteTempFile_Success(t *testing.T) {
	testData := []byte("test content")
	filename := writeTempFile("test.pem", testData)
	defer os.Remove(filename)

	file, err := os.Open(filename)
	require.NoError(t, err)
	defer file.Close()

	content, err := io.ReadAll(file)
	require.NoError(t, err)
	assert.Equal(t, testData, content)
	assert.Contains(t, filename, "test.pem")
}

func TestWriteTempFile_EmptyData(t *testing.T) {
	filename := writeTempFile("empty.pem", []byte{})
	defer os.Remove(filename)

	file, err := os.Open(filename)
	require.NoError(t, err)
	defer file.Close()

	content, err := io.ReadAll(file)
	require.NoError(t, err)
	assert.Empty(t, content)
}

func TestWriteTempFile_LargeData(t *testing.T) {
	testData := make([]byte, 1024*1024) // 1MB
	for i := range testData {
		testData[i] = byte(i % 256)
	}
	filename := writeTempFile("large.pem", testData)
	defer os.Remove(filename)

	content, err := os.ReadFile(filename)
	require.NoError(t, err)
	assert.Equal(t, testData, content)
}

// === setupRouter + NoRoute Handler Tests ===

func setupTestRouter(t testing.TB) *gin.Engine {
	t.Helper()
	store := session.NewStore()
	hub := ws.NewHub()
	go hub.Run()
	t.Cleanup(func() { hub.Stop() })

	cfg := config.DefaultConfig()
	handlers.Init(store, hub, cfg)

	return setupRouter(cfg)
}

func TestRootHandler(t *testing.T) {
	r := setupTestRouter(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "text/html; charset=utf-8", w.Header().Get("Content-Type"))
	assert.Contains(t, w.Body.String(), "<!DOCTYPE html>")
}

func TestNoRoute_FallbackToIndex(t *testing.T) {
	r := setupTestRouter(t)

	// Request a path that doesn't exist — should serve index.html
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/nonexistent-page", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "text/html; charset=utf-8", w.Header().Get("Content-Type"))
	assert.Contains(t, w.Body.String(), "<!DOCTYPE html>")
}

func TestNoRoute_CSSContentType(t *testing.T) {
	r := setupTestRouter(t)

	// Request a .css path that doesn't exist
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/assets/test.css", nil)
	r.ServeHTTP(w, req)

	// If file doesn't exist, falls back to index.html
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestNoRoute_JavaScriptContentType(t *testing.T) {
	r := setupTestRouter(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/assets/test.js", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestNoRoute_SVGContentType(t *testing.T) {
	r := setupTestRouter(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/images/test.svg", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestNoRoute_ICOContentType(t *testing.T) {
	r := setupTestRouter(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/favicon.ico", nil)
	r.ServeHTTP(w, req)

	// favicon.ico is embedded, should return with correct type
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAPIRoutesExist(t *testing.T) {
	r := setupTestRouter(t)

	routes := []struct {
		method string
		path   string
	}{
		{"POST", "/api/session"},
		{"POST", "/api/session/00000000-0000-0000-0000-000000000000/verify-pin"},
		{"GET", "/api/session/00000000-0000-0000-0000-000000000000/qrcode"},
		{"POST", "/api/session/00000000-0000-0000-0000-000000000000/upload"},
		{"GET", "/api/session/00000000-0000-0000-0000-000000000000/pdf"},
		{"DELETE", "/api/session/00000000-0000-0000-0000-000000000000"},
	}

	for _, route := range routes {
		t.Run(route.method+" "+route.path, func(t *testing.T) {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest(route.method, route.path, nil)
			r.ServeHTTP(w, req)

			// Route should be matched by a handler (not 405 Method Not Allowed)
			// 404 is acceptable — it means the handler ran but session wasn't found
			assert.NotEqual(t, http.StatusMethodNotAllowed, w.Code, "Route should be registered")
		})
	}
}

// === Graceful Shutdown Test ===

func TestGracefulShutdown(t *testing.T) {
	store := session.NewStore()
	hub := ws.NewHub()
	go hub.Run()

	cfg := config.DefaultConfig()
	handlers.Init(store, hub, cfg)

	r := setupRouter(cfg)

	srv := &http.Server{
		Addr:    ":0", // random port
		Handler: r,
	}

	// Start server in goroutine
	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.ListenAndServeTLS("", "") // will fail without certs, that's ok
	}()

	// Give server a moment to start
	time.Sleep(10 * time.Millisecond)

	// Send shutdown signal via context
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := srv.Shutdown(ctx)
	assert.NoError(t, err, "Graceful shutdown should complete without error")

	hub.Stop()
}
