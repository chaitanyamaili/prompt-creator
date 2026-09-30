package category

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadIncludesBuiltinGeneric(t *testing.T) {
	cats, err := Load(t.TempDir())
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	generic, ok := cats["generic"]
	if !ok {
		t.Fatalf("expected built-in %q category, got %v", "generic", cats)
	}
	if len(generic.Fields) != 7 {
		t.Fatalf("expected 7 fields, got %d", len(generic.Fields))
	}
	if generic.Fields[0].Key != "goal" || !generic.Fields[0].Required {
		t.Fatalf("expected first field to be required 'goal', got %+v", generic.Fields[0])
	}
}

func TestLoadMissingUserDirIsNotAnError(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "does-not-exist")
	cats, err := Load(missing)
	if err != nil {
		t.Fatalf("Load returned error for missing dir: %v", err)
	}
	if _, ok := cats["generic"]; !ok {
		t.Fatalf("expected built-in generic category even without a user dir")
	}
}

func TestLoadUserCategoryOverridesBuiltinByName(t *testing.T) {
	dir := t.TempDir()
	override := `{
		"name": "generic",
		"description": "overridden",
		"fields": [{"key": "goal", "label": "Goal", "question": "Goal?", "required": true}],
		"template": "{{.goal}}"
	}`
	if err := os.WriteFile(filepath.Join(dir, "generic.json"), []byte(override), 0o644); err != nil {
		t.Fatalf("writing override category: %v", err)
	}

	cats, err := Load(dir)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cats["generic"].Description != "overridden" {
		t.Fatalf("expected user category to override built-in, got %+v", cats["generic"])
	}
	if len(cats["generic"].Fields) != 1 {
		t.Fatalf("expected overridden field count 1, got %d", len(cats["generic"].Fields))
	}
}

func TestLoadUserCategoryAddsNewCategory(t *testing.T) {
	dir := t.TempDir()
	coding := `{
		"name": "coding",
		"description": "Coding task prompt",
		"fields": [{"key": "goal", "label": "Goal", "question": "Goal?", "required": true}],
		"template": "{{.goal}}"
	}`
	if err := os.WriteFile(filepath.Join(dir, "coding.json"), []byte(coding), 0o644); err != nil {
		t.Fatalf("writing new category: %v", err)
	}

	cats, err := Load(dir)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if _, ok := cats["generic"]; !ok {
		t.Fatalf("expected built-in generic category to remain present")
	}
	if _, ok := cats["coding"]; !ok {
		t.Fatalf("expected new user category %q to be loaded", "coding")
	}
}

func TestLoadMalformedUserCategoryReturnsNamedError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "broken.json")
	if err := os.WriteFile(path, []byte("{not valid json"), 0o644); err != nil {
		t.Fatalf("writing malformed category: %v", err)
	}

	_, err := Load(dir)
	if err == nil {
		t.Fatalf("expected error for malformed category file")
	}
	if !strings.Contains(err.Error(), path) {
		t.Fatalf("expected error to name file %q, got: %v", path, err)
	}
}

func TestLoadIgnoresNonJSONFilesInUserDir(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".DS_Store"), []byte("binary junk"), 0o644); err != nil {
		t.Fatalf("writing stray file: %v", err)
	}
	if err := os.Mkdir(filepath.Join(dir, "a-directory.json"), 0o755); err != nil {
		t.Fatalf("creating stray directory: %v", err)
	}

	cats, err := Load(dir)
	if err != nil {
		t.Fatalf("Load returned error for stray non-category files: %v", err)
	}
	if _, ok := cats["generic"]; !ok {
		t.Fatalf("expected built-in generic category to still load")
	}
}
