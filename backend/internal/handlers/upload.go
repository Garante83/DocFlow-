package handlers

import (
	"bytes"
	"fmt"
	"image"
	"io"
	"net/http"

	"dokumentenscanner/internal/session"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Upload configuration constants
const (
	maxFileSizeMB = 10
	maxFileSize   = maxFileSizeMB << 20 // 10MB in bytes
	formFieldName = "image"
)

// allowedImageFormats defines the supported image formats
var allowedImageFormats = map[string]bool{
	"jpeg": true,
	"png":  true,
}

// UploadHandler handles the upload of an image for a session.
func UploadHandler(c *gin.Context) {
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

	// Check if the session is allowed to upload
	if sess.Status != session.StatusUploadAllowed {
		c.JSON(http.StatusForbidden, gin.H{"error": "session not allowed to upload"})
		return
	}

	// Get the file from the request
	file, header, err := c.Request.FormFile(formFieldName)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid file upload"})
		return
	}
	defer file.Close()

	// Check file size
	if header.Size > maxFileSize {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file size exceeds 10MB"})
		return
	}

	// Read the file into a byte slice
	buf := bytes.NewBuffer(nil)
	if _, err := io.Copy(buf, file); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read file"})
		return
	}

	// Validate the image format
	_, format, err := image.Decode(bytes.NewReader(buf.Bytes()))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid image format"})
		return
	}

	// Check if the format is allowed
	if !allowedImageFormats[format] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "only JPEG and PNG formats are allowed"})
		return
	}

	// Store the image in the session
	sess.Image = buf.Bytes()
	sess.Status = session.StatusUploaded
	SessionStore.Update(sess)

	// Broadcast image_uploaded event to desktop clients
	message := fmt.Sprintf(`{"event":"image_uploaded","session_id":"%s"}`, sessionID)
	WebSocketHub.Broadcast(sessionID, []byte(message))

	c.JSON(http.StatusOK, gin.H{"message": "image uploaded successfully"})
}
