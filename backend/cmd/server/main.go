package main

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"embed"
	"encoding/pem"
	"fmt"
	"io"
	"log"
	"log/slog"
	"math/big"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"docflow/internal/config"
	"docflow/internal/handlers"
	"docflow/internal/session"
	"docflow/internal/websocket"
	"github.com/gin-gonic/gin"
)

//go:embed index.html favicon.ico favicon.svg assets
var frontendFS embed.FS

func getPort(cfg *config.Config) string {
	return ":" + cfg.Server.Port
}

func readEmbeddedFile(name string) ([]byte, error) {
	f, err := frontendFS.Open(name)
	if err != nil {
		return nil, fmt.Errorf("failed to open embedded file %s: %w", name, err)
	}
	defer f.Close()
	data, err := io.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("failed to read embedded file %s: %w", name, err)
	}
	return data, nil
}

// securityHeaders adds common security headers to all responses
func securityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Header("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; img-src 'self' data: blob:")
		c.Next()
	}
}

// rateLimiter implements a simple in-memory rate limiter per IP
type rateLimiter struct {
	mu       sync.Mutex
	clients  map[string]*clientInfo
	enabled  bool
	maxReqs  int
	window   time.Duration
	stopChan chan struct{}
}

type clientInfo struct {
	requests   []time.Time
	lastAccess time.Time
}

func newRateLimiter(cfg *config.Config) *rateLimiter {
	rl := &rateLimiter{
		clients:  make(map[string]*clientInfo),
		enabled:  cfg.RateLimit.Enabled,
		maxReqs:  cfg.RateLimit.MaxRequests,
		window:   time.Duration(cfg.RateLimit.WindowSeconds) * time.Second,
		stopChan: make(chan struct{}),
	}
	// Start cleanup goroutine
	go rl.cleanup()
	return rl
}

// cleanup removes inactive clients every 10 minutes
func (rl *rateLimiter) cleanup() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-rl.stopChan:
			return
		case <-ticker.C:
			rl.mu.Lock()
			cutoff := time.Now().Add(-10 * time.Minute)
			for ip, client := range rl.clients {
				if client.lastAccess.Before(cutoff) && len(client.requests) == 0 {
					delete(rl.clients, ip)
				}
			}
			rl.mu.Unlock()
		}
	}
}

// Stop stops the cleanup goroutine
func (rl *rateLimiter) Stop() {
	close(rl.stopChan)
}

func (rl *rateLimiter) isAllowed(ip string) bool {
	if !rl.enabled {
		return true
	}

	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-rl.window)

	client, exists := rl.clients[ip]
	if !exists {
		client = &clientInfo{requests: make([]time.Time, 0)}
		rl.clients[ip] = client
	}

	client.lastAccess = now

	// Remove old requests outside the window
	validRequests := make([]time.Time, 0)
	for _, t := range client.requests {
		if t.After(cutoff) {
			validRequests = append(validRequests, t)
		}
	}

	if len(validRequests) >= rl.maxReqs {
		return false
	}

	client.requests = append(validRequests, now)
	return true
}

// rateLimitMiddleware returns a gin.HandlerFunc that limits requests per IP
func rateLimitMiddleware(rl *rateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		if !rl.isAllowed(ip) {
			retryAfter := int(rl.window.Seconds())
			c.Header("Retry-After", fmt.Sprintf("%d", retryAfter))
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":               "rate limit exceeded",
				"retry_after_seconds": retryAfter,
			})
			c.Abort()
			return
		}
		c.Next()
	}
}

// privacyLogFormatter formats access logs WITHOUT any personal data:
// no client IP, no query string (method, path, status, latency only).
func privacyLogFormatter(p gin.LogFormatterParams) string {
	path := p.Path
	if i := strings.IndexByte(path, '?'); i >= 0 {
		path = path[:i]
	}
	return fmt.Sprintf("%s %s %d %s\n", p.Method, path, p.StatusCode, p.Latency)
}

func setupRouter(cfg *config.Config) *gin.Engine {
	// Debug mode only with debug logging (release mode avoids route dumps
	// and debug warnings in production logs)
	if cfg.Logging.Level != "debug" {
		gin.SetMode(gin.ReleaseMode)
	}

	// gin.New() instead of gin.Default(): custom privacy-preserving logger
	// (no client IP, no query string - the default logger would log both)
	r := gin.New()
	r.Use(gin.LoggerWithFormatter(privacyLogFormatter), gin.Recovery())

	// Security headers for all responses
	r.Use(securityHeaders())

	// Rate limiting for API endpoints (not WebSocket)
	rl := newRateLimiter(cfg)
	api := r.Group("/api")
	api.Use(rateLimitMiddleware(rl))
	{
		api.POST("/session", handlers.CreateSessionHandler)
		api.POST("/session/:id/verify-pin", handlers.VerifyPINHandler)
		api.GET("/session/:id/qrcode", handlers.QRCodeHandler)
		api.POST("/session/:id/upload", handlers.UploadHandler)
		api.GET("/session/:id/pdf", handlers.PDFHandler)
		api.POST("/session/:id/finalize", handlers.FinalizeHandler)
		api.DELETE("/session/:id", handlers.DeleteSessionHandler)
	}

	r.GET("/ws/session/:id", handlers.WebSocketHandler)

	r.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path

		file, err := frontendFS.Open(path[1:])
		if err == nil {
			defer file.Close()
			content, _ := io.ReadAll(file)

			switch {
			case strings.HasSuffix(path, ".js"):
				c.Header("Content-Type", "application/javascript")
			case strings.HasSuffix(path, ".css"):
				c.Header("Content-Type", "text/css")
			case strings.HasSuffix(path, ".svg"):
				c.Header("Content-Type", "image/svg+xml")
			case strings.HasSuffix(path, ".ico"):
				c.Header("Content-Type", "image/x-icon")
			case strings.HasSuffix(path, ".html"):
				c.Header("Content-Type", "text/html; charset=utf-8")
			default:
				c.Header("Content-Type", "application/octet-stream")
			}
			c.String(200, string(content))
			return
		}

		file, err = frontendFS.Open("index.html")
		if err != nil {
			c.String(500, "Frontend not embedded correctly")
			return
		}
		defer file.Close()

		content, err := io.ReadAll(file)
		if err != nil {
			c.String(500, "Failed to read embedded frontend")
			return
		}

		c.Header("Content-Type", "text/html; charset=utf-8")
		c.String(200, string(content))
	})

	r.GET("/", func(c *gin.Context) {
		file, err := frontendFS.Open("index.html")
		if err != nil {
			c.String(500, "Frontend not embedded correctly")
			return
		}
		defer file.Close()

		content, err := io.ReadAll(file)
		if err != nil {
			c.String(500, "Failed to read embedded frontend")
			return
		}

		c.Header("Content-Type", "text/html; charset=utf-8")
		c.String(200, string(content))
	})

	return r
}

func main() {
	// 1. Load configuration
	cfg, err := config.LoadConfig("")
	if err != nil {
		slog.Error("Failed to load config", "error", err)
		os.Exit(1)
	}

	// 2. Set up logger
	setupLogger(cfg)

	// 3. Initialize session store with config values
	sessionStore := session.NewStore()
	cleanupStop := make(chan struct{})
	sessionStore.StartCleanup(cfg.Session.CleanupInterval, cleanupStop)

	// 4. Initialize WebSocket hub
	hub := websocket.NewHub()
	go hub.Run()

	// 5. Initialize handlers with dependencies
	handlers.Init(sessionStore, hub, cfg)

	// 6. Set up router
	r := setupRouter(cfg)

	// 7. Configure TLS: config-based or temporary certificate
	var certFile, keyFile string
	if cfg.Server.TLSCertPath != "" && cfg.Server.TLSKeyPath != "" {
		// Production: use certificates from config
		certFile = cfg.Server.TLSCertPath
		keyFile = cfg.Server.TLSKeyPath
		slog.Info("Using TLS certificates from config", "cert", certFile)
	} else {
		// Development: generate a temporary self-signed certificate
		certFile, keyFile = generateSelfSignedCert()
		defer os.Remove(certFile)
		defer os.Remove(keyFile)
		slog.Info("Using auto-generated self-signed certificate")
	}

	// 8. Create HTTP server
	srv := &http.Server{
		Addr:    getPort(cfg),
		Handler: r,
		TLSConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
		},
		// Suppress net/http connection errors (TLS handshake failures etc.) -
		// they contain client addresses which must not appear in the logs
		ErrorLog: log.New(io.Discard, "", 0),
	}

	// 9. Start graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-quit
		slog.Info("Server shutting down...")
		close(cleanupStop)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			slog.Error("Server forced to shutdown", "error", err)
		}
		slog.Info("Server stopped")
	}()

	// 10. Start server
	slog.Info("Server starting", "addr", "0.0.0.0"+srv.Addr, "port", cfg.Server.Port)
	if err := srv.ListenAndServeTLS(certFile, keyFile); err != nil && err != http.ErrServerClosed {
		slog.Error("Server failed", "error", err)
		os.Exit(1)
	}
}

// generateSelfSignedCert creates a temporary self-signed certificate and key.
func generateSelfSignedCert() (certPath, keyPath string) {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		slog.Error("Failed to generate private key", "error", err)
		os.Exit(1)
	}

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{Organization: []string{"DocFlow"}},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &privateKey.PublicKey, privateKey)
	if err != nil {
		slog.Error("Failed to create certificate", "error", err)
		os.Exit(1)
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	keyDER, _ := x509.MarshalECPrivateKey(privateKey)
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})

	certPath = writeTempFile("cert.pem", certPEM)
	keyPath = writeTempFile("key.pem", keyPEM)
	return
}

func setupLogger(cfg *config.Config) {
	level := slog.LevelInfo
	switch cfg.Logging.Level {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	}

	opts := &slog.HandlerOptions{Level: level}
	var handler slog.Handler
	switch cfg.Logging.Format {
	case "text":
		handler = slog.NewTextHandler(os.Stdout, opts)
	case "json":
		handler = slog.NewJSONHandler(os.Stdout, opts)
	default:
		slog.Warn("Unknown logging format, falling back to json", "format", cfg.Logging.Format)
		handler = slog.NewJSONHandler(os.Stdout, opts)
	}

	slog.SetDefault(slog.New(handler))
}

func writeTempFile(name string, data []byte) string {
	f, err := os.CreateTemp("", name+"-*.pem")
	if err != nil {
		slog.Error("Failed to create temp file", "file", name, "error", err)
		os.Exit(1)
	}
	if _, err := f.Write(data); err != nil {
		slog.Error("Failed to write temp file", "file", name, "error", err)
		os.Exit(1)
	}
	f.Close()
	return f.Name()
}
