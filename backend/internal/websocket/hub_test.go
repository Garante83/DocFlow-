package websocket

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewHub tests the creation of a new Hub
func TestNewHub(t *testing.T) {
	hub := NewHub()

	require.NotNil(t, hub)
	assert.NotNil(t, hub.clients)
	assert.NotNil(t, hub.register)
	assert.NotNil(t, hub.unregister)
	assert.NotNil(t, hub.broadcast)
	assert.NotNil(t, hub.stopChan)
}

// TestHubBroadcast tests the Broadcast method
func TestHubBroadcast(t *testing.T) {
	hub := NewHub()
	
	// Create a test session ID
	sessionID := uuid.New()
	
	// Test that Broadcast doesn't panic with a valid hub
	hub.Broadcast(sessionID, []byte("test message"))
	
	// Verify the message was sent to the broadcast channel
	select {
	case msg := <-hub.broadcast:
		assert.Equal(t, sessionID, msg.sessionID)
		assert.Equal(t, []byte("test message"), msg.data)
	default:
		t.Fatal("Expected message in broadcast channel")
	}
}

// TestHubRegister tests the Register method
func TestHubRegister(t *testing.T) {
	hub := NewHub()
	
	// Test that the channels exist
	assert.NotNil(t, hub.register)
	assert.NotNil(t, hub.unregister)
}

// TestHubStop tests the Stop method
func TestHubStop(t *testing.T) {
	hub := NewHub()
	
	// Test that Stop doesn't panic
	hub.Stop()
	
	// Verify stopChan is closed by trying to read from it
	select {
	case <-hub.stopChan:
		// Expected - channel is closed
	default:
		t.Fatal("Expected stopChan to be closed")
	}
}
