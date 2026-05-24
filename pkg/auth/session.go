package auth

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

const SessionTTL = 30 * 24 * time.Hour

type Session struct {
	ID        string
	Type      Type
	CreatedAt time.Time
}

type SessionStore struct {
	mu       sync.RWMutex
	sessions map[string]*Session
}

func NewSessionStore() *SessionStore {
	return &SessionStore{
		sessions: make(map[string]*Session),
	}
}

func (s *SessionStore) Create(t Type) *Session {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic("auth: failed to generate session ID")
	}

	sess := &Session{
		ID:        hex.EncodeToString(b),
		Type:      t,
		CreatedAt: time.Now(),
	}

	s.mu.Lock()
	s.sessions[sess.ID] = sess
	s.mu.Unlock()

	return sess
}

func (s *SessionStore) Get(id string) *Session {
	if id == "" {
		return nil
	}
	s.mu.RLock()
	sess, ok := s.sessions[id]
	s.mu.RUnlock()

	if !ok {
		return nil
	}

	if time.Since(sess.CreatedAt) > SessionTTL {
		s.Delete(id)
		return nil
	}

	return sess
}

func (s *SessionStore) Delete(id string) {
	s.mu.Lock()
	delete(s.sessions, id)
	s.mu.Unlock()
}
