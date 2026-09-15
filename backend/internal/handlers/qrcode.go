package handlers

import (
	"net"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/skip2/go-qrcode"
)

// QR code configuration constants
const (
	// QRCodeSize is the size of the generated QR code in pixels
	QRCodeSize = 256
	// QRCodeLevel is the error correction level
	QRCodeLevel = qrcode.Medium
)

// getFrontendURL returns the base URL advertised in the QR code.
// Priority: config server.public_url > legacy FRONTEND_URL env > LAN IP auto-detect.
// A public URL is essential when the server runs behind a reverse proxy or in a
// container, because the auto-detected address is then not reachable from
// outside (reverse proxy address / Docker bridge IP).
func getFrontendURL() string {
	if deps := getDeps(); deps != nil && deps.Config != nil {
		if url := deps.Config.Server.PublicURL; url != "" {
			return strings.TrimSuffix(url, "/")
		}
	}

	if url := os.Getenv("FRONTEND_URL"); url != "" {
		return url
	}

	ip := getLocalIP()
	port := getDefaultPort()
	return "https://" + ip + port
}

// getDefaultPort returns the configured server port
func getDefaultPort() string {
	if deps := getDeps(); deps != nil && deps.Config != nil && deps.Config.Server.Port != "" {
		return ":" + deps.Config.Server.Port
	}
	return ":8082"
}

// getLocalIP detects the LAN IP by creating a UDP connection to an external address
func getLocalIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "localhost"
	}
	defer conn.Close()
	return conn.LocalAddr().(*net.UDPAddr).IP.String()
}

// QRCodeHandler generates a QR code for a session.
func QRCodeHandler(c *gin.Context) {
	deps := getDeps()
	sessionIDStr := c.Param("id")
	sessionID, err := uuid.Parse(sessionIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid session ID"})
		return
	}

	// Check if the session exists
	_, exists := deps.SessionStore.Get(sessionID)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
		return
	}

	// Generate QR code URL
	frontendURL := getFrontendURL()
	qrURL := frontendURL + "/#/mobile?session_id=" + sessionID.String()

	// Generate QR code
	qr, err := qrcode.New(qrURL, QRCodeLevel)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate QR code"})
		return
	}

	// Return QR code as PNG
	c.Header("Content-Type", "image/png")
	c.Status(http.StatusOK)
	pngData, _ := qr.PNG(QRCodeSize)
	_, _ = c.Writer.Write(pngData)
}
