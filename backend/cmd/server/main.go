package main

import (
	"embed"
	"io"
	"log"
	"os"
	"strings"

	"dokumentenscanner/internal/handlers"
	"dokumentenscanner/internal/session"
	"dokumentenscanner/internal/websocket"
	"github.com/gin-gonic/gin"
)

//go:embed *
var frontendFS embed.FS

const (
	DefaultServerPort = ":8082"
)

func getPort() string {
	if port := os.Getenv("PORT"); port != "" {
		return ":" + port
	}
	return DefaultServerPort
}

func readEmbeddedFile(name string) []byte {
	f, err := frontendFS.Open(name)
	if err != nil {
		log.Fatal("Failed to read embedded file:", name, err)
	}
	defer f.Close()
	data, err := io.ReadAll(f)
	if err != nil {
		log.Fatal("Failed to read embedded file:", name, err)
	}
	return data
}

func main() {
	sessionStore := session.NewStore()
	handlers.SessionStore = sessionStore
	go sessionStore.StartCleanup(0)

	hub := websocket.NewHub()
	handlers.WebSocketHub = hub
	go hub.Run()

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

	port := getPort()
	log.Printf("Server started on https://0.0.0.0%s", port)
	log.Printf("Frontend available at https://localhost%s", port)
	if err := r.RunTLS(port, certFile, keyFile); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}

func writeTempFile(name string, data []byte) string {
	f, err := os.CreateTemp("", name+"-*.pem")
	if err != nil {
		log.Fatal("Failed to create temp file:", err)
	}
	if _, err := f.Write(data); err != nil {
		log.Fatal("Failed to write temp file:", err)
	}
	f.Close()
	return f.Name()
}
