package txadmin

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Session represents a cached txAdmin session
type Session struct {
	BaseURL    string    `json:"baseUrl"`
	Cookie     string    `json:"cookie"`
	CookieName string    `json:"cookieName"`
	CSRFToken  string    `json:"csrfToken"`
	ExpiresAt  time.Time `json:"expiresAt"`
}

// SessionManager handles session persistence
type SessionManager struct {
	sessionPath string
}

// NewSessionManager creates a new session manager
func NewSessionManager() (*SessionManager, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	sessionDir := filepath.Join(homeDir, ".opencore")
	if err := os.MkdirAll(sessionDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create session directory: %w", err)
	}

	return &SessionManager{
		sessionPath: filepath.Join(sessionDir, "txadmin-session.json"),
	}, nil
}

// Load loads a cached session from disk
func (sm *SessionManager) Load() (*Session, error) {
	data, err := os.ReadFile(sm.sessionPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // No cached session
		}
		return nil, fmt.Errorf("failed to read session file: %w", err)
	}

	var session Session
	if err := json.Unmarshal(data, &session); err != nil {
		// Corrupted file, delete it
		os.Remove(sm.sessionPath)
		return nil, nil
	}

	// Check if session is expired
	if time.Now().After(session.ExpiresAt) {
		os.Remove(sm.sessionPath)
		return nil, nil
	}

	return &session, nil
}

// Save saves a session to disk
func (sm *SessionManager) Save(session *Session) error {
	if session == nil {
		return nil
	}

	data, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}

	// Write with restricted permissions (only owner can read/write)
	if err := os.WriteFile(sm.sessionPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write session file: %w", err)
	}

	return nil
}

// Clear removes the cached session
func (sm *SessionManager) Clear() error {
	if err := os.Remove(sm.sessionPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove session file: %w", err)
	}
	return nil
}

// IsValid checks if a session is still valid
func (s *Session) IsValid() bool {
	if s == nil {
		return false
	}
	return time.Now().Before(s.ExpiresAt)
}
