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
	"io"
	"log/slog"
	"math/big"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"dokumentenscanner/internal/config"
	"dokumentenscanner/internal/handlers"
	"dokumentenscanner/internal/session"
	"dokumentenscanner/internal/websocket"
	"github.com/gin-gonic/gin"
)

//go:embed index.html favicon.ico favicon.svg assets
var frontendFS embed.FS

func getPort(cfg *config.Config) string {
	return ":" + cfg.Server.Port
}

func readEmbeddedFile(name string) []byte {
	f, err := frontendFS.Open(name)
	if err != nil {
		slog.Error("Failed to read embedded file", "file", name, "error", err)
		os.Exit(1)
	}
	defer f.Close()
	data, err := io.ReadAll(f)
	if err != nil {
		slog.Error("Failed to read embedded file", "file", name, "error", err)
		os.Exit(1)
	}
	return data
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

func setupRouter() *gin.Engine {
	r := gin.Default()

	// Security headers for all responses
	r.Use(securityHeaders())

	api := r.Group("/api")
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
	// 1. Konfiguration laden
	cfg, err := config.LoadConfig("")
	if err != nil {
		slog.Error("Failed to load config", "error", err)
		os.Exit(1)
	}

	// 2. Logger einrichten
	setupLogger(cfg)

	// 3. Session-Store mit Config-Werten initialisieren
	sessionStore := session.NewStore()
	cleanupStop := make(chan struct{})
	sessionStore.StartCleanup(cfg.Session.CleanupInterval, cleanupStop)

	// 4. WebSocket-Hub initialisieren
	hub := websocket.NewHub()
	go hub.Run()

	// 5. Handler mit Abhaengigkeiten initialisieren
	handlers.Init(sessionStore, hub, cfg)

	// 6. Router einrichten
	r := setupRouter()

	// 7. TLS konfigurieren: Config-basiert oder temporaeres Cert
	var certFile, keyFile string
	if cfg.Server.TLSCertPath != "" && cfg.Server.TLSKeyPath != "" {
		// Production: Nutze config-basierte Zertifikate
		certFile = cfg.Server.TLSCertPath
		keyFile = cfg.Server.TLSKeyPath
		slog.Info("Using TLS certificates from config", "cert", certFile)
	} else {
		// Development: Generiere temporaeres self-signed Cert
		certFile, keyFile = generateSelfSignedCert()
		defer os.Remove(certFile)
		defer os.Remove(keyFile)
		slog.Info("Using auto-generated self-signed certificate")
	}

	// 8. HTTP-Server erstellen
	srv := &http.Server{
		Addr:    getPort(cfg),
		Handler: r,
		TLSConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
		},
	}

	// 9. Graceful Shutdown starten
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

	// 10. Server starten
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
		Subject:      pkix.Name{Organization: []string{"Dokumentenscanner"}},
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

	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	})

	logger := slog.New(handler)
	slog.SetDefault(logger)
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
