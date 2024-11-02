package sipnexus

import (
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/itzmanish/sipnexus/pkg/logger"
	"github.com/itzmanish/sipnexus/pkg/media"
)

type SessionStatus uint8

const (
	SessionStatus_New SessionStatus = iota
	SessionStatus_Ringing
	SessionStatus_Connected
	SessionStatus_Disconnected
	SessionStatus_Failed
)

type Session struct {
	ID     string
	CallID string
	Status SessionStatus
	rtc    media.MediaEngine

	CreatedAt time.Time
}

type SessionManager struct {
	sessions map[string]*Session
	mu       sync.RWMutex
}

func NewSessionManager() *SessionManager {
	return &SessionManager{
		sessions: make(map[string]*Session),
	}
}

func (sm *SessionManager) CreateSession(callID string) *Session {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	session := &Session{
		ID:        uuid.New().String(),
		CallID:    callID,
		CreatedAt: time.Now(),
		rtc:       media.NewUDPMediaEngine(logger.NewLogger()),
	}
	sm.sessions[session.ID] = session
	return session
}

func (sm *SessionManager) GetSession(sessionID string) (*Session, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	session, exists := sm.sessions[sessionID]
	return session, exists
}

func (sm *SessionManager) DeleteSession(sessionID string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	delete(sm.sessions, sessionID)
}

func (sm *SessionManager) GetOrCreateSession(callID string) *Session {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	for _, session := range sm.sessions {
		if session.CallID == callID {
			return session
		}
	}

	// If no existing session found, create a new one
	session := &Session{
		ID:        uuid.New().String(),
		CallID:    callID,
		CreatedAt: time.Now(),
	}
	sm.sessions[session.ID] = session
	return session
}

func (sm *SessionManager) CleanupSessions() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	now := time.Now()
	for id, session := range sm.sessions {
		if now.Sub(session.CreatedAt) > 24*time.Hour {
			delete(sm.sessions, id)
		}
	}
}
