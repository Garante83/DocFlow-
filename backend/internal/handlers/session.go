package handlers

import (
	"net/http"
	"time"

	"docflow/internal/session"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// CreateSessionHandler handles the creation of a new session.
func CreateSessionHandler(c *gin.Context) {
	deps := getDeps()
	sess, err := deps.SessionStore.Create()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Generate a PIN for the session
	pin := session.GeneratePIN()
	sess.PIN = pin
	deps.SessionStore.Update(sess)

	c.JSON(http.StatusOK, gin.H{
		"session_id":       sess.ID,
		"pin":              pin,
		"max_file_size_mb": deps.Config.Upload.MaxFileSizeMB,
		"max_pages":        deps.Config.PDF.MaxPages,
	})
}

// VerifyPINHandler handles the verification of a PIN for a session.
func VerifyPINHandler(c *gin.Context) {
	deps := getDeps()
	sessionIDStr := c.Param("id")
	sessionID, err := uuid.Parse(sessionIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid session ID"})
		return
	}

	var request struct {
		PIN string `json:"pin" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	// Create PIN config from application config
	pinCfg := &session.PINConfig{
		MaxAttempts:     deps.Config.Session.MaxFailedAttempts,
		LockoutDuration: deps.Config.Session.LockoutDuration,
	}
	if pinCfg.MaxAttempts <= 0 {
		pinCfg.MaxAttempts = 3
	}
	if pinCfg.LockoutDuration <= 0 {
		pinCfg.LockoutDuration = 5 * time.Minute
	}

	err = session.VerifyPIN(deps.SessionStore, sessionID, request.PIN, pinCfg)
	if err != nil {
		if err == session.ErrPINLocked {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": err.Error()})
		} else if err == session.ErrInvalidPIN {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"valid": true, "message": "PIN verified successfully"})
}

// DeleteSessionHandler handles the deletion of a session.
func DeleteSessionHandler(c *gin.Context) {
	deps := getDeps()
	sessionIDStr := c.Param("id")
	sessionID, err := uuid.Parse(sessionIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid session ID"})
		return
	}

	deps.SessionStore.Delete(sessionID)
	c.Status(http.StatusNoContent)
}
