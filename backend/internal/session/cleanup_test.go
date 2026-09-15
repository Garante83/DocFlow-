package session

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStartCleanup_RemovesExpiredSession(t *testing.T) {
	store := NewStore()

	sess, err := store.Create()
	require.NoError(t, err)

	// Simulate an expired session
	store.mu.Lock()
	sess.ExpiresAt = time.Now().Add(-1 * time.Minute)
	store.mu.Unlock()

	stop := make(chan struct{})
	store.StartCleanup(10*time.Millisecond, stop)
	defer close(stop)

	// Wait for at least one cleanup tick
	assert.Eventually(t, func() bool {
		_, exists := store.Get(sess.ID)
		return !exists
	}, 2*time.Second, 10*time.Millisecond, "expired session should be removed by cleanup")
}

func TestStartCleanup_RemovesDownloadedSession(t *testing.T) {
	store := NewStore()

	sess, err := store.Create()
	require.NoError(t, err)

	store.mu.Lock()
	sess.Status = StatusDownloaded
	store.mu.Unlock()

	stop := make(chan struct{})
	store.StartCleanup(10*time.Millisecond, stop)
	defer close(stop)

	assert.Eventually(t, func() bool {
		_, exists := store.Get(sess.ID)
		return !exists
	}, 2*time.Second, 10*time.Millisecond, "downloaded session should be removed by cleanup")
}

func TestStartCleanup_KeepsActiveSession(t *testing.T) {
	store := NewStore()

	sess, err := store.Create()
	require.NoError(t, err)

	stop := make(chan struct{})
	store.StartCleanup(10*time.Millisecond, stop)
	defer close(stop)

	// Active session must survive several cleanup ticks
	time.Sleep(50 * time.Millisecond)

	retrieved, exists := store.Get(sess.ID)
	assert.True(t, exists, "active session must not be removed")
	assert.Equal(t, sess.ID, retrieved.ID)
}

func TestStartCleanup_StopChannelTerminatesGoroutine(t *testing.T) {
	store := NewStore()

	_, err := store.Create()
	require.NoError(t, err)

	stop := make(chan struct{})
	store.StartCleanup(10*time.Millisecond, stop)

	// Terminate the goroutine; it must stop without touching the store afterwards
	close(stop)
	time.Sleep(50 * time.Millisecond)

	// The goroutine exiting is observed indirectly: no panic, store still usable.
	// Use a new session to confirm the store remains functional.
	sess, err := store.Create()
	require.NoError(t, err)
	_, exists := store.Get(sess.ID)
	assert.True(t, exists)
}

func TestStartCleanup_DefaultIntervalUsed(t *testing.T) {
	store := NewStore()

	sess, err := store.Create()
	require.NoError(t, err)

	// interval 0 must fall back to DefaultCleanupInterval; the goroutine runs
	// but does not tick within the test window. Terminate via stop channel.
	stop := make(chan struct{})
	store.StartCleanup(0, stop)
	defer close(stop)

	time.Sleep(20 * time.Millisecond)

	_, exists := store.Get(sess.ID)
	assert.True(t, exists, "session must survive with default interval (5 min tick)")
}
