package handlers

import (
	"fmt"
	"net/http"

	"dokumentenscanner/internal/session"
	"dokumentenscanner/pkg/utils"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// PDF handler configuration constants
const (
	// DefaultPDFFilename is the default filename for downloaded PDFs
	DefaultPDFFilename = "document.pdf"
)

// PDFHandler generates and returns a PDF for a session.
func PDFHandler(c *gin.Context) {
	sessionIDStr := c.Param("id")
	sessionID, err := uuid.Parse(sessionIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid session ID"})
		return
	}

	// Check if the session exists
	sess, exists := SessionStore.Get(sessionID)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
		return
	}

	// Check if the session has an uploaded image
	if len(sess.Image) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no image uploaded for this session"})
		return
	}

	// Generate the PDF
	pdfBytes, err := utils.GeneratePDF(sess.Image)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Store the PDF in the session
	sess.PDF = pdfBytes
	sess.Status = session.StatusReady
	SessionStore.Update(sess)

	// Broadcast pdf_ready event to desktop clients
	message := fmt.Sprintf(`{"event":"pdf_ready","session_id":"%s"}`, sessionID)
	WebSocketHub.Broadcast(sessionID, []byte(message))

	// Return the PDF
	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", DefaultPDFFilename))
	c.Header("Content-Length", fmt.Sprintf("%d", len(pdfBytes)))
	c.Status(http.StatusOK)
	_, _ = c.Writer.Write(pdfBytes)
}
