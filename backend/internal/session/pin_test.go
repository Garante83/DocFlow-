package session

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGeneratePIN(t *testing.T) {
	pin := GeneratePIN()
	assert.Len(t, pin, 6)
	assert.Regexp(t, `^[0-9]{6}$`, pin)
}

func TestVerifyPIN_CorrectPIN(t *testing.T) {
	store := NewStore()
	sess, err := store.Create()
	require.NoError(t, err)

	// Set a known PIN
	sess.PIN = "123456"
	store.Update(sess)

	// Verify correct PIN
	err = VerifyPIN(store, sess.ID, "123456", nil)
	assert.NoError(t, err)

	// Check session status
	retrieved, _ := store.Get(sess.ID)
	assert.Equal(t, StatusUploadAllowed, retrieved.Status)
	assert.Equal(t, 0, retrieved.FailedAttempts)
	assert.True(t, retrieved.LockedUntil.IsZero())
}

func TestVerifyPIN_IncorrectPIN(t *testing.T) {
	store := NewStore()
	sess, err := store.Create()
	require.NoError(t, err)

	// Set a known PIN
	sess.PIN = "123456"
	store.Update(sess)

	// Verify incorrect PIN
	err = VerifyPIN(store, sess.ID, "111111", nil)
	assert.ErrorIs(t, err, ErrInvalidPIN)

	// Check session status
	retrieved, _ := store.Get(sess.ID)
	assert.Equal(t, StatusWaitingForPIN, retrieved.Status)
	assert.Equal(t, 1, retrieved.FailedAttempts)
	assert.False(t, retrieved.LockedUntil.IsZero())
}

func TestVerifyPIN_MultipleFailures(t *testing.T) {
	store := NewStore()
	sess, err := store.Create()
	require.NoError(t, err)

	// Set a known PIN
	sess.PIN = "123456"
	store.Update(sess)

	// First failed attempt
	err = VerifyPIN(store, sess.ID, "111111", nil)
	assert.Error(t, err)
	retrieved, _ := store.Get(sess.ID)
	assert.Equal(t, 1, retrieved.FailedAttempts)

	// After 3 failed attempts, session should be locked
	_ = VerifyPIN(store, sess.ID, "111111", nil)
	_ = VerifyPIN(store, sess.ID, "111111", nil)

	// Now any attempt should be locked
	err = VerifyPIN(store, sess.ID, "123456", nil)
	assert.ErrorIs(t, err, ErrPINLocked)
}

func TestVerifyPIN_LockedAfterThreeFailures(t *testing.T) {
	store := NewStore()
	sess, err := store.Create()
	require.NoError(t, err)

	// Set a known PIN
	sess.PIN = "123456"
	store.Update(sess)

	// Three failed attempts - should lock the session
	_ = VerifyPIN(store, sess.ID, "111111", nil)
	_ = VerifyPIN(store, sess.ID, "111111", nil)
	_ = VerifyPIN(store, sess.ID, "111111", nil)

	// Now any attempt should be locked
	err = VerifyPIN(store, sess.ID, "123456", nil)
	assert.ErrorIs(t, err, ErrPINLocked)
}

func TestVerifyPIN_NonExistentSession(t *testing.T) {
	store := NewStore()
	err := VerifyPIN(store, uuid.New(), "123456", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "session not found")
}
