package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRunGenerateRequiresGoal(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"generate"}, nil, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("expected exit code 2, got %d", code)
	}
	if !strings.Contains(stderr.String(), "--goal is required") {
		t.Fatalf("expected missing-goal error, got: %s", stderr.String())
	}
}

func TestRunGeneratePrintsRenderedPrompt(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("USERPROFILE", tmp)

	var stdout, stderr bytes.Buffer
	code := run([]string{"generate", "--goal", "Write a poem", "--no-clipboard"}, nil, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d (stderr: %s)", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Goal: Write a poem") {
		t.Fatalf("expected rendered prompt on stdout, got: %s", stdout.String())
	}
}

func TestRunGenerateSucceedsEvenIfClipboardUnavailable(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("USERPROFILE", tmp)

	var stdout, stderr bytes.Buffer
	code := run([]string{"generate", "--goal", "Write a poem"}, nil, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0 regardless of clipboard availability, got %d (stderr: %s)", code, stderr.String())
	}
}

func TestRunGenerateUnknownCategory(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("USERPROFILE", tmp)

	var stdout, stderr bytes.Buffer
	code := run([]string{"generate", "--goal", "x", "--category", "nope", "--no-clipboard"}, nil, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("expected exit code 2, got %d", code)
	}
	if !strings.Contains(stderr.String(), `unknown category "nope"`) {
		t.Fatalf("expected unknown category error, got: %s", stderr.String())
	}
}

func TestRunGenerateSaveWritesHistoryFile(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("USERPROFILE", tmp)

	var stdout, stderr bytes.Buffer
	code := run([]string{"generate", "--goal", "Write a poem", "--save", "--no-clipboard"}, nil, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d (stderr: %s)", code, stderr.String())
	}

	historyDir := filepath.Join(tmp, ".config", "prompt-creator", "history")
	entries, err := os.ReadDir(historyDir)
	if err != nil {
		t.Fatalf("reading history dir: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected exactly 1 history file, got %d", len(entries))
	}

	data, err := os.ReadFile(filepath.Join(historyDir, entries[0].Name()))
	if err != nil {
		t.Fatalf("reading history file: %v", err)
	}
	var saved struct {
		Prompt string `json:"prompt"`
	}
	if err := json.Unmarshal(data, &saved); err != nil {
		t.Fatalf("unmarshaling history file: %v", err)
	}
	if !strings.Contains(saved.Prompt, "Write a poem") {
		t.Fatalf("expected saved prompt to contain goal text, got: %s", saved.Prompt)
	}
}

func TestRunGenerateUnknownFlagReturnsUsageError(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"generate", "--nope", "x"}, nil, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("expected exit code 2 for unknown flag, got %d (stderr: %s)", code, stderr.String())
	}
}

func TestRunUnknownCommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"bogus"}, nil, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("expected exit code 2, got %d", code)
	}
}

func TestRunWizardHappyPath(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("USERPROFILE", tmp)

	input := strings.NewReader(strings.Join([]string{
		"Write a poem", // goal
		"",             // context
		"",             // constraints
		"",             // output_format
		"",             // verbosity
		"",             // scope_boundary
		"",             // assumed_knowledge
		"n",            // save confirmation
	}, "\n") + "\n")
	var stdout, stderr bytes.Buffer

	code := run(nil, input, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d (stderr: %s)", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Goal: Write a poem") {
		t.Fatalf("expected rendered prompt on stdout, got: %s", stdout.String())
	}

	historyDir := filepath.Join(tmp, ".config", "prompt-creator", "history")
	entries, _ := os.ReadDir(historyDir)
	if len(entries) != 0 {
		t.Fatalf("expected no history file when save answer is 'n', got %d files", len(entries))
	}
}

func TestRunWizardSavesWhenConfirmed(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("USERPROFILE", tmp)

	input := strings.NewReader(strings.Join([]string{
		"Write a poem", "", "", "", "", "", "", "y",
	}, "\n") + "\n")
	var stdout, stderr bytes.Buffer

	code := run(nil, input, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d (stderr: %s)", code, stderr.String())
	}

	historyDir := filepath.Join(tmp, ".config", "prompt-creator", "history")
	entries, err := os.ReadDir(historyDir)
	if err != nil {
		t.Fatalf("reading history dir: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected exactly 1 history file, got %d", len(entries))
	}
}

func TestRunCategoriesListShowsBuiltinGeneric(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("USERPROFILE", tmp)

	var stdout, stderr bytes.Buffer
	code := run([]string{"categories", "list"}, nil, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d (stderr: %s)", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "generic\t") {
		t.Fatalf("expected generic category listed, got: %s", stdout.String())
	}
}

func TestRunCategoriesUnknownSubcommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"categories", "bogus"}, nil, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("expected exit code 2, got %d", code)
	}
}

func TestRunGenerateWithPolishUsesPolishedOutput(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("USERPROFILE", tmp)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"response": "Polished: Write a poem"})
	}))
	defer server.Close()

	var stdout, stderr bytes.Buffer
	code := run([]string{
		"generate", "--goal", "Write a poem", "--no-clipboard",
		"--polish", "--ollama-host", server.URL,
	}, nil, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d (stderr: %s)", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Polished: Write a poem") {
		t.Fatalf("expected polished prompt on stdout, got: %s", stdout.String())
	}
}

func TestRunGenerateWithPolishFallsBackWhenUnreachable(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("USERPROFILE", tmp)

	var stdout, stderr bytes.Buffer
	code := run([]string{
		"generate", "--goal", "Write a poem", "--no-clipboard",
		"--polish", "--ollama-host", "http://127.0.0.1:1",
	}, nil, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0 even when Ollama is unreachable, got %d", code)
	}
	if !strings.Contains(stdout.String(), "Goal: Write a poem") {
		t.Fatalf("expected raw prompt fallback on stdout, got: %s", stdout.String())
	}
	if !strings.Contains(stderr.String(), "could not reach local LLM") {
		t.Fatalf("expected warning on stderr, got: %s", stderr.String())
	}
}

func TestRunGenerateRejectsCategoryWithUnsupportedRequiredField(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("USERPROFILE", tmp)

	userCatDir := filepath.Join(tmp, ".config", "prompt-creator", "categories")
	if err := os.MkdirAll(userCatDir, 0o755); err != nil {
		t.Fatalf("creating user category dir: %v", err)
	}
	catJSON := `{
		"name": "narrow",
		"description": "narrow test category",
		"fields": [
			{"key": "topic", "label": "Topic", "question": "Topic?", "required": true}
		],
		"template": "Topic: {{.topic}}\n"
	}`
	if err := os.WriteFile(filepath.Join(userCatDir, "narrow.json"), []byte(catJSON), 0o644); err != nil {
		t.Fatalf("writing user category: %v", err)
	}

	var stdout, stderr bytes.Buffer
	code := run([]string{"generate", "--goal", "x", "--category", "narrow", "--no-clipboard"}, nil, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("expected exit code 2, got %d (stdout: %s, stderr: %s)", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), `required field "topic"`) {
		t.Fatalf("expected error naming the unsupported required field, got: %s", stderr.String())
	}
}

func TestRunGenerateWithPolishTimesOutOnSlowServer(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("USERPROFILE", tmp)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		json.NewEncoder(w).Encode(map[string]string{"response": "Polished: Write a poem"})
	}))
	defer server.Close()

	var stdout, stderr bytes.Buffer
	start := time.Now()
	code := run([]string{
		"generate", "--goal", "Write a poem", "--no-clipboard",
		"--polish", "--ollama-host", server.URL, "--ollama-timeout", "20ms",
	}, nil, &stdout, &stderr)
	elapsed := time.Since(start)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d (stderr: %s)", code, stderr.String())
	}
	if elapsed > 150*time.Millisecond {
		t.Fatalf("expected the call to be bounded by the timeout, took %s", elapsed)
	}
	if !strings.Contains(stdout.String(), "Goal: Write a poem") {
		t.Fatalf("expected raw prompt fallback on stdout, got: %s", stdout.String())
	}
	if !strings.Contains(stderr.String(), "could not reach local LLM") {
		t.Fatalf("expected warning on stderr, got: %s", stderr.String())
	}
}

func TestRunGenerateAllFieldsMatchesGoldenFile(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("USERPROFILE", tmp)

	var stdout, stderr bytes.Buffer
	code := run([]string{
		"generate",
		"--goal", "Refactor the auth middleware",
		"--context", "Go service, uses JWT",
		"--constraints", "Keep the public API unchanged",
		"--output-format", "unified diff",
		"--verbosity", "no explanation, code only",
		"--scope-boundary", "do not touch the database layer",
		"--assumed-knowledge", "I know Go and JWT already",
		"--no-clipboard",
	}, nil, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d (stderr: %s)", code, stderr.String())
	}

	want, err := os.ReadFile("testdata/golden_full_prompt.txt")
	if err != nil {
		t.Fatalf("reading golden file: %v", err)
	}
	got := strings.TrimRight(stdout.String(), "\n")
	wantTrimmed := strings.TrimRight(string(want), "\n")
	if got != wantTrimmed {
		t.Fatalf("output does not match golden file.\ngot:\n%s\nwant:\n%s", got, wantTrimmed)
	}
}
