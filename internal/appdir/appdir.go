package appdir

import (
	"os"
	"path/filepath"
)

// ConfigDir returns the prompt-creator config directory, creating it if it
// does not already exist.
func ConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".config", "prompt-creator")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}
