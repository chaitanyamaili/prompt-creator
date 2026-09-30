package history

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"prompt-creator/internal/appdir"
)

// Entry is what gets persisted for a single saved prompt.
type Entry struct {
	Category string            `json:"category"`
	Answers  map[string]string `json:"answers"`
	Prompt   string            `json:"prompt"`
	SavedAt  time.Time         `json:"saved_at"`
}

// Save writes entry to a new timestamped file under the history directory
// and returns the path written. If entry.SavedAt is zero, it defaults to
// the current time.
func Save(entry Entry) (string, error) {
	dir, err := appdir.ConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolving config dir: %w", err)
	}
	historyDir := filepath.Join(dir, "history")
	if err := os.MkdirAll(historyDir, 0o755); err != nil {
		return "", fmt.Errorf("creating history dir: %w", err)
	}

	if entry.SavedAt.IsZero() {
		entry.SavedAt = time.Now().UTC()
	}
	name := fmt.Sprintf("%s-%s.json", entry.SavedAt.Format("20060102T150405Z"), entry.Category)
	path := filepath.Join(historyDir, name)

	data, err := json.MarshalIndent(entry, "", "  ")
	if err != nil {
		return "", fmt.Errorf("encoding history entry: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", fmt.Errorf("writing history entry: %w", err)
	}
	return path, nil
}
