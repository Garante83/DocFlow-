package session

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewStore(t *testing.T) {
	store := NewStore()
	assert.NotNil(t, store)
	assert.NotNil(t, store.sessions)
}

func TestCreateSession(t *testing.T) {
	store := NewStore()

	sess, err := store.Create()
	require.NoError(t, err)
	require.NotNil(t, sess)
	assert.NotEqual(t, uuid.Nil, sess.ID)
	assert.Equal(t, StatusWaitingForPIN, sess.Status)
	assert.WithinDuration(t, time.Now(), sess.CreatedAt, time.Second)
	assert.WithinDuration(t, time.Now().Add(time.Hour), sess.ExpiresAt, time.Second)
}

func TestGetSession(t *testing.T) {
	store := NewStore()

	// Create a session
	sess, err := store.Create()
	require.NoError(t, err)

	// Get existing session
	retrieved, exists := store.Get(sess.ID)
	assert.True(t, exists)
	assert.Equal(t, sess.ID, retrieved.ID)

	// Get non-existing session
	_, exists = store.Get(uuid.New())
	assert.False(t, exists)
}

func TestDeleteSession(t *testing.T) {
	store := NewStore()

	// Create a session
	sess, err := store.Create()
	require.NoError(t, err)

	// Delete session
	store.Delete(sess.ID)

	// Verify deletion
	_, exists := store.Get(sess.ID)
	assert.False(t, exists)
}

func TestUpdateSession(t *testing.T) {
	store := NewStore()

	// Create a session
	sess, err := store.Create()
	require.NoError(t, err)

	// Modify session
	sess.Status = StatusUploadAllowed
	sess.PIN = "123456"

	// Update session
	store.Update(sess)

	// Verify update
	retrieved, exists := store.Get(sess.ID)
	assert.True(t, exists)
	assert.Equal(t, StatusUploadAllowed, retrieved.Status)
	assert.Equal(t, "123456", retrieved.PIN)
}
