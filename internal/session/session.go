package session

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Session stores the editor state to be restored on next launch.
type Session struct {
	Files       []string `json:"files"`       // absolute paths of open file tabs
	ActiveIndex int      `json:"activeIndex"` // which tab was active
	Directory   string   `json:"directory"`   // sidebar root directory
}

// sessionPath returns the path to the session file.
func sessionPath(configDir string) string {
	return filepath.Join(configDir, "session.json")
}

// Save writes the session to disk.
func Save(configDir string, s *Session) error {
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(sessionPath(configDir), data, 0644)
}

// Load reads the session from disk. Returns an empty session if not found.
func Load(configDir string) (*Session, error) {
	data, err := os.ReadFile(sessionPath(configDir))
	if err != nil {
		if os.IsNotExist(err) {
			return &Session{}, nil
		}
		return nil, err
	}
	var s Session
	if err := json.Unmarshal(data, &s); err != nil {
		return &Session{}, nil
	}
	return &s, nil
}
