package session

import (
	"log/slog"
	"time"
)

// Session configuration constants
const (
	// DefaultSessionTimeout is the duration after which a session expires
	DefaultSessionTimeout = 1 * time.Hour
	// DefaultCleanupInterval is how often expired sessions are checked
	DefaultCleanupInterval = 5 * time.Minute
)

// StartCleanup starts a goroutine that periodically cleans up expired sessions.
// If interval is 0, DefaultCleanupInterval will be used.
func (s *Store) StartCleanup(interval time.Duration) {
	if interval == 0 {
		interval = DefaultCleanupInterval
	}

	go func() {
		for {
			time.Sleep(interval)
			s.cleanupExpiredSessions()
		}
	}()
}

// cleanupExpiredSessions removes sessions that have expired or are marked as downloaded.
func (s *Store) cleanupExpiredSessions() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	for id, session := range s.sessions {
		if session.Status == StatusDownloaded || now.After(session.ExpiresAt) {
			slog.Debug("Cleaning up session", "session_id", id, "status", session.Status, "expired", now.After(session.ExpiresAt))
			delete(s.sessions, id)
		}
	}
}
