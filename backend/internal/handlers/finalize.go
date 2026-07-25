package handlers

import (
	"fmt"
	"net/http"

	"dokumentenscanner/internal/session"
	"dokumentenscanner/pkg/utils"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// FinalizeHandler finalizes the upload and generates the PDF.
func FinalizeHandler(c *gin.Context) {
	deps := getDeps()
	sessionIDStr := c.Param("id")
	sessionID, err := uuid.Parse(sessionIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid session ID"})
		return
	}

	// Check if the session exists
	sess, exists := deps.SessionStore.Get(sessionID)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
		return
	}

	// Check if the session has at least one image
	if len(sess.Images) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no images uploaded for this session"})
		return
	}

	// Check if the session is in a valid state for finalization
	if sess.Status != session.StatusUploadAllowed && sess.Status != session.StatusUploading {
		c.JSON(http.StatusForbidden, gin.H{"error": "session cannot be finalized in current state"})
		return
	}

	// Get JPEG quality from config
	jpegQuality := deps.Config.PDF.JPEGQuality
	if jpegQuality <= 0 || jpegQuality > 100 {
		jpegQuality = 85
	}

	// Generate multi-page PDF
	pdfBytes, err := utils.GenerateMultiPagePDF(sess.Images, jpegQuality)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to generate PDF: %v", err)})
		return
	}

	// Update session with PDF and status
	sess.PDF = pdfBytes
	sess.Status = session.StatusReady
	deps.SessionStore.Update(sess)

	// Broadcast pdf_ready event to desktop clients
	message := fmt.Sprintf(`{"event":"pdf_ready","session_id":"%s","page_count":%d}`, sessionID, len(sess.Images))
	deps.WebSocketHub.Broadcast(sessionID, []byte(message))

	c.JSON(http.StatusOK, gin.H{
		"message":    "PDF generated successfully",
		"page_count": len(sess.Images),
		"pdf_size":   len(pdfBytes),
	})
}
