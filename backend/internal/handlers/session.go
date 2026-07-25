package handlers

import (
	"net/http"

	"dokumentenscanner/internal/session"
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
		"session_id": sess.ID,
		"pin":        pin,
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

	err = session.VerifyPIN(deps.SessionStore, sessionID, request.PIN)
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
