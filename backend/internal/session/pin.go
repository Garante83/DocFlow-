package session

import (
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// PIN configuration constants
const (
	pinLength       = 6
	maxAttempts     = 3
	lockoutDuration = 5 * time.Minute
)

// ErrPINLocked is returned when a session is temporarily locked due to too many failed attempts.
var ErrPINLocked = errors.New("session is temporarily locked due to too many failed attempts")

// ErrInvalidPIN is returned when the provided PIN is incorrect.
var ErrInvalidPIN = errors.New("invalid PIN")

// GeneratePIN generates a 6-digit PIN.
func GeneratePIN() string {
	var n uint32
	if err := binary.Read(rand.Reader, binary.LittleEndian, &n); err != nil {
		// Fallback to a less secure method if crypto/rand fails
		n = uint32(time.Now().UnixNano() % 1000000)
	}
	return fmt.Sprintf("%0*d", pinLength, n%1000000)
}

// VerifyPIN verifies the provided PIN against the session's PIN.
// It enforces a maximum of 3 attempts and a 5-minute lockout after the first failed attempt.
func VerifyPIN(store *Store, sessionID uuid.UUID, attempt string) error {
	session, exists := store.Get(sessionID)
	if !exists {
		return fmt.Errorf("session not found")
	}

	// Check if the session is locked
	if !session.LockedUntil.IsZero() && time.Now().Before(session.LockedUntil) {
		return ErrPINLocked
	}

	// Lock after max attempts
	if session.FailedAttempts >= maxAttempts {
		return ErrPINLocked
	}

	// Check if the PIN is correct
	if session.PIN != attempt {
		session.FailedAttempts++

		// Lock the session after the first failed attempt
		if session.FailedAttempts == 1 {
			session.LockedUntil = time.Now().Add(lockoutDuration)
		}

		return ErrInvalidPIN
	}

	// Reset failed attempts and lockout on successful verification
	session.FailedAttempts = 0
	session.LockedUntil = time.Time{}
	session.Status = StatusUploadAllowed

	return nil
}
