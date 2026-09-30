package history

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"
)

func TestSaveWritesJSONFileUnderHistoryDir(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("USERPROFILE", tmp)

	entry := Entry{
		Category: "generic",
		Answers:  map[string]string{"goal": "Write a poem"},
		Prompt:   "Goal: Write a poem",
		SavedAt:  time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC),
	}

	path, err := Save(entry)
	if err != nil {
		t.Fatalf("Save returned error: %v", err)
	}
	if !strings.Contains(path, "history") {
		t.Fatalf("expected path under history dir, got %q", path)
	}
	if !strings.HasSuffix(path, "20260930T120000Z-generic.json") {
		t.Fatalf("expected timestamped filename, got %q", path)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading saved file: %v", err)
	}
	var got Entry
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshaling saved file: %v", err)
	}
	if got.Prompt != entry.Prompt || got.Answers["goal"] != "Write a poem" {
		t.Fatalf("saved entry mismatch, got %+v", got)
	}
}

func TestSaveDefaultsSavedAtWhenZero(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("USERPROFILE", tmp)

	before := time.Now().UTC()
	path, err := Save(Entry{Category: "generic", Prompt: "x"})
	if err != nil {
		t.Fatalf("Save returned error: %v", err)
	}
	after := time.Now().UTC()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading saved file: %v", err)
	}
	var got Entry
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshaling saved file: %v", err)
	}
	if got.SavedAt.Before(before) || got.SavedAt.After(after) {
		t.Fatalf("expected SavedAt to default to now, got %v (want between %v and %v)", got.SavedAt, before, after)
	}
}
