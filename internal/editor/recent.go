package editor

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const maxRecentFiles = 50

// RecentFiles tracks recently opened file paths.
type RecentFiles struct {
	Files []string `json:"files"`
	path  string   // path to the JSON file
}

// LoadRecentFiles loads recent files from ~/.config/ted/recent.json.
func LoadRecentFiles() *RecentFiles {
	configDir := os.Getenv("XDG_CONFIG_HOME")
	if configDir == "" {
		home, _ := os.UserHomeDir()
		configDir = filepath.Join(home, ".config")
	}
	p := filepath.Join(configDir, "ted", "recent.json")

	rf := &RecentFiles{path: p}

	data, err := os.ReadFile(p)
	if err != nil {
		return rf
	}
	json.Unmarshal(data, rf)
	return rf
}

// Add adds a file path to the top of the recent list.
func (rf *RecentFiles) Add(path string) {
	abs, err := filepath.Abs(path)
	if err == nil {
		path = abs
	}

	// Remove if already exists
	filtered := make([]string, 0, len(rf.Files))
	for _, f := range rf.Files {
		if f != path {
			filtered = append(filtered, f)
		}
	}

	// Prepend
	rf.Files = append([]string{path}, filtered...)

	// Trim
	if len(rf.Files) > maxRecentFiles {
		rf.Files = rf.Files[:maxRecentFiles]
	}

	rf.save()
}

func (rf *RecentFiles) save() {
	dir := filepath.Dir(rf.path)
	os.MkdirAll(dir, 0755)

	data, err := json.MarshalIndent(rf, "", "  ")
	if err != nil {
		return
	}
	os.WriteFile(rf.path, data, 0644)
}
