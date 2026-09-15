package session

import (
	crypto_rand "crypto/rand"
	"crypto/subtle"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/google/uuid"
)

// PIN configuration constants
const (
	pinLength = 6
)

// PINConfig holds configurable PIN verification settings.
type PINConfig struct {
	MaxAttempts     int
	LockoutDuration time.Duration
}

// DefaultPINConfig returns default PIN configuration.
func DefaultPINConfig() *PINConfig {
	return &PINConfig{
		MaxAttempts:     3,
		LockoutDuration: 5 * time.Minute,
	}
}

// ErrPINLocked is returned when a session is temporarily locked due to too many failed attempts.
var ErrPINLocked = errors.New("session is temporarily locked due to too many failed attempts")

// ErrInvalidPIN is returned when the provided PIN is incorrect.
var ErrInvalidPIN = errors.New("invalid PIN")

// randReader is the randomness source for PIN generation.
// It is a package variable so tests can inject a deterministic reader.
var randReader io.Reader = crypto_rand.Reader

// GeneratePIN generates a 6-digit PIN.
func GeneratePIN() string {
	var n uint32
	if err := binary.Read(randReader, binary.LittleEndian, &n); err != nil {
		// Fallback to a less secure method if crypto/rand fails
		n = uint32(time.Now().UnixNano() % 1000000)
	}
	return fmt.Sprintf("%0*d", pinLength, n%1000000)
}

// VerifyPIN verifies the provided PIN against the session's PIN.
// It enforces a configurable maximum of attempts and lockout duration.
func VerifyPIN(store *Store, sessionID uuid.UUID, attempt string, cfg *PINConfig) error {
	if cfg == nil {
		cfg = DefaultPINConfig()
	}

	session, exists := store.Get(sessionID)
	if !exists {
		return fmt.Errorf("session not found")
	}

	// Check if the session is locked
	if !session.LockedUntil.IsZero() && time.Now().Before(session.LockedUntil) {
		return ErrPINLocked
	}

	// Lock after max attempts
	if session.FailedAttempts >= cfg.MaxAttempts {
		return ErrPINLocked
	}

	// Check if the PIN is correct (constant-time comparison to prevent timing attacks)
	if subtle.ConstantTimeCompare([]byte(session.PIN), []byte(attempt)) != 1 {
		session.FailedAttempts++

		// Lock the session after the first failed attempt
		if session.FailedAttempts == 1 {
			session.LockedUntil = time.Now().Add(cfg.LockoutDuration)
		}

		return ErrInvalidPIN
	}

	// Reset failed attempts and lockout on successful verification
	session.FailedAttempts = 0
	session.LockedUntil = time.Time{}
	session.Status = StatusUploadAllowed

	return nil
}
