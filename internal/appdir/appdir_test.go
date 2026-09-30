package appdir

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfigDirCreatesAndReturnsPath(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("USERPROFILE", tmp)

	dir, err := ConfigDir()
	if err != nil {
		t.Fatalf("ConfigDir returned error: %v", err)
	}

	want := filepath.Join(tmp, ".config", "prompt-creator")
	if dir != want {
		t.Fatalf("ConfigDir = %q, want %q", dir, want)
	}

	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("expected dir to exist: %v", err)
	}
	if !info.IsDir() {
		t.Fatalf("expected %q to be a directory", dir)
	}
}
