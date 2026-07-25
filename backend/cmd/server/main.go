package main

import (
	"context"
	"embed"
	"io"
	"log/slog"
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

//go:embed *
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
	go sessionStore.StartCleanup(cfg.Session.CleanupInterval)

	// 4. WebSocket-Hub initialisieren
	hub := websocket.NewHub()
	go hub.Run()

	// 5. Handler mit Abhaengigkeiten initialisieren
	handlers.Init(sessionStore, hub, cfg)

	// 6. Router einrichten
	r := gin.Default()

	api := r.Group("/api")
	{
		api.POST("/session", handlers.CreateSessionHandler)
		api.POST("/session/:id/verify-pin", handlers.VerifyPINHandler)
		api.GET("/session/:id/qrcode", handlers.QRCodeHandler)
		api.POST("/session/:id/upload", handlers.UploadHandler)
		api.GET("/session/:id/pdf", handlers.PDFHandler)
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
			c.String(500, "Frontend not embedded correctly: "+err.Error())
			return
		}
		defer file.Close()

		content, err := io.ReadAll(file)
		if err != nil {
			c.String(500, "Failed to read embedded frontend: "+err.Error())
			return
		}

		c.Header("Content-Type", "text/html; charset=utf-8")
		c.String(200, string(content))
	})

	r.GET("/", func(c *gin.Context) {
		file, err := frontendFS.Open("index.html")
		if err != nil {
			c.String(500, "Frontend not embedded correctly: "+err.Error())
			return
		}
		defer file.Close()

		content, err := io.ReadAll(file)
		if err != nil {
			c.String(500, "Failed to read embedded frontend: "+err.Error())
			return
		}

		c.Header("Content-Type", "text/html; charset=utf-8")
		c.String(200, string(content))
	})

	// Write embedded cert/key to temp files for TLS
	certFile := writeTempFile("cert.pem", readEmbeddedFile("cert.pem"))
	keyFile := writeTempFile("key.pem", readEmbeddedFile("key.pem"))
	defer os.Remove(certFile)
	defer os.Remove(keyFile)

	// 7. HTTP-Server erstellen
	srv := &http.Server{
		Addr:    getPort(cfg),
		Handler: r,
	}

	// 8. Graceful Shutdown starten
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

	// 9. Server starten
	slog.Info("Server starting", "addr", "0.0.0.0"+srv.Addr, "port", cfg.Server.Port)
	slog.Info("Frontend available at https://localhost" + srv.Addr)
	if err := srv.ListenAndServeTLS(certFile, keyFile); err != nil && err != http.ErrServerClosed {
		slog.Error("Server failed", "error", err)
		os.Exit(1)
	}
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
