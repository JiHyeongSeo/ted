package session

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Session stores the editor state to be restored on next launch.
type Session struct {
	Files       []string `json:"files"`       // absolute paths of open file tabs
	ActiveIndex int      `json:"activeIndex"` // which tab was active
	Directory   string   `json:"directory"`   // sidebar root directory
}

// sessionPath returns the path to the session file for the given project directory.
// Each project directory gets its own session file keyed by an md5 of its path.
// If projectDir is empty, falls back to the global session.json.
func sessionPath(configDir, projectDir string) string {
	if projectDir == "" {
		return filepath.Join(configDir, "session.json")
	}
	hash := fmt.Sprintf("%x", md5.Sum([]byte(projectDir)))
	return filepath.Join(configDir, "sessions", "session-"+hash+".json")
}

// Save writes the session to disk.
func Save(configDir string, s *Session) error {
	path := sessionPath(configDir, s.Directory)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// Load reads the session for the given project directory from disk.
// Returns an empty session if not found.
func Load(configDir, projectDir string) (*Session, error) {
	data, err := os.ReadFile(sessionPath(configDir, projectDir))
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
