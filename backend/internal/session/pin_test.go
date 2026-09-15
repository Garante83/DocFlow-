package session

import (
	"bytes"
	"errors"
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

func TestGeneratePIN_Randomness(t *testing.T) {
	// Multiple PINs must differ (6-digit space is large enough for 20 samples)
	seen := make(map[string]bool)
	for i := 0; i < 20; i++ {
		pin := GeneratePIN()
		assert.Regexp(t, `^[0-9]{6}$`, pin)
		seen[pin] = true
	}
	assert.Greater(t, len(seen), 1, "generated PINs must not be constant")
}

func TestGeneratePIN_FallbackOnReaderError(t *testing.T) {
	orig := randReader
	t.Cleanup(func() { randReader = orig })

	// Inject a failing reader to force the time-based fallback path
	randReader = errReader{}

	pin := GeneratePIN()
	assert.Len(t, pin, 6)
	assert.Regexp(t, `^[0-9]{6}$`, pin)
}

func TestGeneratePIN_DeterministicReader(t *testing.T) {
	orig := randReader
	t.Cleanup(func() { randReader = orig })

	// LittleEndian uint32 0x000F4240 = 1.000.000 -> mod 1.000.000 = 0 -> "000000"
	randReader = bytes.NewReader([]byte{0x40, 0x42, 0x0F, 0x00})
	assert.Equal(t, "000000", GeneratePIN())

	// LittleEndian uint32 0x0001869F = 99.999 -> "099999"
	randReader = bytes.NewReader([]byte{0x9F, 0x86, 0x01, 0x00})
	assert.Equal(t, "099999", GeneratePIN())

	// Leading zeros must be preserved ("000123")
	randReader = bytes.NewReader([]byte{0x7B, 0x00, 0x00, 0x00})
	assert.Equal(t, "000123", GeneratePIN())
}

type errReader struct{}

func (errReader) Read(p []byte) (int, error) { return 0, errors.New("no entropy") }

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
