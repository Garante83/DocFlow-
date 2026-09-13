package handlers

import (
	"bytes"
	"fmt"
	"image"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"docflow/internal/session"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// formFieldName is the name of the form field for file uploads
const formFieldName = "image"

// UploadHandler handles the upload of an image for a session.
func UploadHandler(c *gin.Context) {
	deps := getDeps()
	if deps.Config == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server configuration error"})
		return
	}
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

	// Check if the session is allowed to upload
	if sess.Status != session.StatusUploadAllowed && sess.Status != session.StatusUploading {
		c.JSON(http.StatusForbidden, gin.H{"error": "session not allowed to upload"})
		return
	}

	// Get the file from the request
	file, header, err := c.Request.FormFile(formFieldName)
	if err != nil {
		slog.Warn("Invalid file upload", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid file upload"})
		return
	}
	defer file.Close()

	// Get allowed types from config, use defaults if not set
	allowedTypes := deps.Config.Upload.AllowedTypes
	if len(allowedTypes) == 0 {
		// Fallback to default allowed types
		allowedTypes = []string{"image/jpeg", "image/png", "image/webp"}
	}

	// Note: For multipart/form-data uploads, the Content-Type of the individual file part
	// is stored in the part header, not the request header. We validate the actual image
	// format after decoding, which is more reliable than checking the Content-Type header.

	// Calculate max file size from config (convert MB to bytes)
	maxFileSizeMB := deps.Config.Upload.MaxFileSizeMB
	if maxFileSizeMB <= 0 {
		maxFileSizeMB = 10 // Default 10MB
	}
	maxFileSize := maxFileSizeMB << 20

	// Check file size against config
	if header.Size > int64(maxFileSize) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("file size exceeds maximum of %dMB", maxFileSizeMB),
		})
		return
	}

	// Read the file into a byte slice (with size limit to prevent DoS)
	buf := bytes.NewBuffer(nil)
	limitedReader := io.LimitReader(file, int64(maxFileSize)+1)
	if _, err := io.Copy(buf, limitedReader); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read file"})
		return
	}
	if buf.Len() > maxFileSize {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("file size exceeds maximum of %dMB", maxFileSizeMB),
		})
		return
	}

	// Validate the image format
	_, format, err := image.Decode(bytes.NewReader(buf.Bytes()))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid image format: not a valid image file"})
		return
	}

	// Check if the MIME type based on the image format is allowed
	// Map image format to MIME type
	imageFormatToMIME := map[string]string{
		"jpeg": "image/jpeg",
		"png":  "image/png",
		"gif":  "image/gif",
		"webp": "image/webp",
	}
	imageMIMEType := imageFormatToMIME[format]

	if imageMIMEType == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("image format '%s' is not supported", format),
		})
		return
	}

	// Verify the detected MIME type is in the allowed list
	mimeTypeAllowed := false
	for _, allowedType := range deps.Config.Upload.AllowedTypes {
		if imageMIMEType == allowedType {
			mimeTypeAllowed = true
			break
		}
	}
	if !mimeTypeAllowed {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("image format '%s' (MIME: %s) is not allowed. Allowed types: %s",
				format, imageMIMEType, strings.Join(allowedTypes, ", ")),
		})
		return
	}

	// Check max pages limit
	maxPages := deps.Config.PDF.MaxPages
	if maxPages <= 0 {
		maxPages = 20
	}
	if len(sess.Images) >= maxPages {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("maximum %d pages allowed", maxPages)})
		return
	}

	// Append the image to the session
	sess.Images = append(sess.Images, buf.Bytes())
	sess.Status = session.StatusUploading
	deps.SessionStore.Update(sess)

	// Broadcast image_added event with page count
	pageCount := len(sess.Images)
	message := fmt.Sprintf(`{"event":"image_added","session_id":"%s","page_count":%d}`, sessionID, pageCount)
	deps.WebSocketHub.Broadcast(sessionID, []byte(message))

	c.JSON(http.StatusOK, gin.H{"message": "image uploaded successfully", "page_count": pageCount})
}
