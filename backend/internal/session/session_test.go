package session

import (
	"sync"
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

// Test helper: creates a store with one session containing n images.
func storeWithImages(t *testing.T, n int) (*Store, *Session) {
	t.Helper()
	store := NewStore()
	sess, err := store.Create()
	require.NoError(t, err)
	for i := 0; i < n; i++ {
		sess.Images = append(sess.Images, []byte{0xFF, 0xD8})
	}
	return store, sess
}

func TestImageCount_Empty(t *testing.T) {
	store, sess := storeWithImages(t, 0)
	assert.Equal(t, 0, sess.ImageCount())
	assert.NotNil(t, store)
}

func TestImageCount_WithImages(t *testing.T) {
	_, sess := storeWithImages(t, 3)
	assert.Equal(t, 3, sess.ImageCount())
}

func TestUpdateFunc_ModifiesSession(t *testing.T) {
	store := NewStore()
	sess, err := store.Create()
	require.NoError(t, err)

	called := false
	err = store.UpdateFunc(sess.ID, func(s *Session) error {
		called = true
		s.Status = StatusUploadAllowed
		s.PIN = "654321"
		return nil
	})

	require.NoError(t, err)
	assert.True(t, called, "function must be invoked for existing session")

	retrieved, exists := store.Get(sess.ID)
	require.True(t, exists)
	assert.Equal(t, StatusUploadAllowed, retrieved.Status)
	assert.Equal(t, "654321", retrieved.PIN)
}

func TestUpdateFunc_SessionNotFound(t *testing.T) {
	store := NewStore()

	err := store.UpdateFunc(uuid.New(), func(s *Session) error {
		t.Error("function must not be invoked for non-existing session")
		return nil
	})

	assert.Error(t, err)
}

func TestUpdateFunc_PropagatesFunctionError(t *testing.T) {
	store := NewStore()
	sess, err := store.Create()
	require.NoError(t, err)

	testErr := assert.AnError
	err = store.UpdateFunc(sess.ID, func(s *Session) error {
		return testErr
	})

	assert.ErrorIs(t, err, testErr)

	// The session must still exist after a failed update
	_, exists := store.Get(sess.ID)
	assert.True(t, exists)
}

func TestUpdateFunc_ConcurrentUpdatesAreAtomic(t *testing.T) {
	store := NewStore()
	sess, err := store.Create()
	require.NoError(t, err)

	// Concurrent increments of FailedAttempts must not lose updates
	const goroutines = 50
	var wg sync.WaitGroup
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = store.UpdateFunc(sess.ID, func(s *Session) error {
				s.FailedAttempts++
				return nil
			})
		}()
	}
	wg.Wait()

	retrieved, _ := store.Get(sess.ID)
	assert.Equal(t, goroutines, retrieved.FailedAttempts,
		"all concurrent updates must be applied under the lock (no TOCTOU)")
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
