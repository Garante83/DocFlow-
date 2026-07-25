package session

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

// SessionStatus represents the current status of a session.
type SessionStatus string

const (
	StatusWaitingForPIN SessionStatus = "waiting_for_pin"
	StatusUploadAllowed SessionStatus = "upload_allowed"
	StatusUploading     SessionStatus = "uploading"
	StatusUploaded      SessionStatus = "uploaded"
	StatusReady         SessionStatus = "ready"
	StatusDownloaded    SessionStatus = "downloaded"
)

// Session represents a user session for document scanning.
type Session struct {
	ID             uuid.UUID
	PIN            string
	Images         [][]byte
	PDF            []byte
	Status         SessionStatus
	CreatedAt      time.Time
	ExpiresAt      time.Time
	FailedAttempts int
	LockedUntil    time.Time
}

// ImageCount returns the number of images in the session.
func (s *Session) ImageCount() int {
	return len(s.Images)
}

// Store manages sessions in memory with thread safety.
type Store struct {
	sessions map[uuid.UUID]*Session
	mu       sync.Mutex
}

// NewStore creates a new session store.
func NewStore() *Store {
	return &Store{
		sessions: make(map[uuid.UUID]*Session),
	}
}

// Create creates a new session and returns it.
func (s *Store) Create() (*Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := uuid.New()
	session := &Session{
		ID:        id,
		Status:    StatusWaitingForPIN,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}

	s.sessions[id] = session
	return session, nil
}

// Get retrieves a session by its ID.
func (s *Store) Get(id uuid.UUID) (*Session, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, exists := s.sessions[id]
	return session, exists
}

// Delete removes a session from the store.
func (s *Store) Delete(id uuid.UUID) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.sessions, id)
}

// Update updates a session in the store.
func (s *Store) Update(session *Session) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.sessions[session.ID] = session
}
