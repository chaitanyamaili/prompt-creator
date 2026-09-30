# prompt-creator Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a Go CLI (`prompt-creator`) that assembles well-structured, token-efficient LLM prompts via an interactive wizard or flags, using config-driven categories and an optional local-LLM (Ollama) polish pass.

**Architecture:** A small set of focused `internal/` packages (category loading, template rendering, wizard I/O, clipboard, history, local-LLM client) composed by a thin `cmd/prompt-creator/main.go` that dispatches to `generate` (flags), the bare wizard (no args), or `categories list`. Built-in categories are embedded via `go:embed`; user categories and history live under `~/.config/prompt-creator/`.

**Tech Stack:** Go (stdlib only — `flag`, `text/template`, `encoding/json`, `net/http`, `os/exec`, `embed`). No third-party modules.

**Spec:** `docs/superpowers/specs/2026-09-30-prompt-creator-design.md`

## Global Constraints

- Single static binary, cross-platform (macOS/Linux/Windows) — stdlib only, no third-party Go modules.
- Only `--goal` is required in `generate`; every other field is optional and omitted from the rendered prompt when blank.
- No LLM calls by default; `--polish` is the only path that calls out to a local Ollama server, and it must never hard-fail the run — always fall back to the unpolished prompt with a warning.
- History is written only on explicit request (`--save` flag, or "y" at the wizard's save prompt) — never automatically.
- Config and history live under `~/.config/prompt-creator/` on every OS (a fixed path, not `os.UserConfigDir()`).

## Review Focus

- Whitespace-only answer to a required wizard field (e.g. spacebar then Enter) — expected to be treated as empty and re-prompted, not accepted as a real answer. Pinned in Task 4.
- A stray non-JSON file in the user categories directory (e.g. macOS `.DS_Store`) — expected to be ignored, not fail category loading. Pinned in Task 2.
- A category JSON whose `template` references a field key absent from its own `fields` list (a config typo) — expected to render that section blank rather than crashing. Pinned in Task 3.
- An unreachable or malformed `--ollama-host` value with `--polish` set — expected to fall back gracefully to the unpolished prompt, not crash or hang. Pinned in Tasks 10 and 7/10 (main-level fallback).
- An unrecognized flag passed to `generate` (typo) — expected to produce a clear usage error and non-zero exit, not be silently ignored. Pinned in Task 7.

---

### Task 1: Module scaffold + config directory helper

**Files:**
- Create: `go.mod`
- Create: `.gitignore`
- Create: `internal/appdir/appdir.go`
- Test: `internal/appdir/appdir_test.go`

**Interfaces:**
- Produces: `appdir.ConfigDir() (string, error)` — returns `$HOME/.config/prompt-creator`, creating it if it doesn't exist.

- [ ] **Step 1: Initialize the Go module**

Run: `cd ~/workspaces/personal/prompt-creator && go mod init prompt-creator`
Expected: creates `go.mod` with `module prompt-creator` and a `go` version line.

- [ ] **Step 2: Add a .gitignore**

```
/prompt-creator
*.test
*.out
```

- [ ] **Step 3: Write the failing test**

```go
// internal/appdir/appdir_test.go
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
```

- [ ] **Step 4: Run test to verify it fails**

Run: `go test ./internal/appdir/...`
Expected: FAIL (package/function does not exist yet)

- [ ] **Step 5: Write minimal implementation**

```go
// internal/appdir/appdir.go
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
```

- [ ] **Step 6: Run test to verify it passes**

Run: `go test ./internal/appdir/...`
Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add go.mod .gitignore internal/appdir
git commit -m "feat: add appdir config directory helper"
```

---

### Task 2: Category model, embedded generic category, loader

**Files:**
- Create: `categories/generic.json`
- Create: `categories/embed.go`
- Create: `internal/category/category.go`
- Test: `internal/category/category_test.go`

**Interfaces:**
- Consumes: nothing new (stdlib `embed`, `encoding/json`, `os` only).
- Produces:
  - `category.Field{Key, Label, Question string; Required bool}`
  - `category.Category{Name, Description, Template string; Fields []Field}`
  - `category.Load(userDir string) (map[string]category.Category, error)`

- [ ] **Step 1: Write the failing tests**

```go
// internal/category/category_test.go
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
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/category/...`
Expected: FAIL (package `category` and embedded `categories` package do not exist yet)

- [ ] **Step 3: Add the embedded built-in category**

```json
// categories/generic.json
{
  "name": "generic",
  "description": "Generic task prompt",
  "fields": [
    { "key": "goal", "label": "Goal", "question": "What do you want the LLM to do?", "required": true },
    { "key": "context", "label": "Context", "question": "Relevant background?", "required": false },
    { "key": "constraints", "label": "Constraints", "question": "Format, length, tone, things to avoid?", "required": false },
    { "key": "output_format", "label": "Output format", "question": "How should the response be structured?", "required": false },
    { "key": "verbosity", "label": "Verbosity", "question": "How much explanation should the LLM include?", "required": false },
    { "key": "scope_boundary", "label": "Do not", "question": "Anything explicitly out of scope?", "required": false },
    { "key": "assumed_knowledge", "label": "Assume I already know", "question": "What should the LLM skip re-explaining?", "required": false }
  ],
  "template": "Goal: {{.goal}}\n{{if .context}}\nContext:\n{{.context}}\n{{end}}{{if .constraints}}\nConstraints: {{.constraints}}\n{{end}}{{if .output_format}}\nOutput format: {{.output_format}}\n{{end}}{{if .verbosity}}\nVerbosity: {{.verbosity}}\n{{end}}{{if .scope_boundary}}\nDo not: {{.scope_boundary}}\n{{end}}{{if .assumed_knowledge}}\nAssume I already know: {{.assumed_knowledge}}\n{{end}}"
}
```

```go
// categories/embed.go
// Package categories bundles the built-in category definitions into the
// binary via go:embed.
package categories

import "embed"

//go:embed *.json
var FS embed.FS
```

- [ ] **Step 4: Implement the category model and loader**

```go
// internal/category/category.go
package category

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"prompt-creator/categories"
)

// Field describes one question the wizard/flag interface asks for a
// category.
type Field struct {
	Key      string `json:"key"`
	Label    string `json:"label"`
	Question string `json:"question"`
	Required bool   `json:"required"`
}

// Category is a named field set plus the text/template used to assemble
// the final prompt from answers.
type Category struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Fields      []Field `json:"fields"`
	Template    string  `json:"template"`
}

// Load returns all built-in categories overlaid with any user-defined
// categories found in userDir (a user category with the same Name replaces
// the built-in one of that name). A missing userDir is not an error.
// Non-.json entries and subdirectories in userDir are ignored.
func Load(userDir string) (map[string]Category, error) {
	result := map[string]Category{}

	builtinEntries, err := categories.FS.ReadDir(".")
	if err != nil {
		return nil, fmt.Errorf("reading embedded categories: %w", err)
	}
	for _, entry := range builtinEntries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		data, err := categories.FS.ReadFile(entry.Name())
		if err != nil {
			return nil, fmt.Errorf("reading embedded category %q: %w", entry.Name(), err)
		}
		cat, err := parse(entry.Name(), data)
		if err != nil {
			return nil, err
		}
		result[cat.Name] = cat
	}

	userEntries, err := os.ReadDir(userDir)
	if err != nil {
		if os.IsNotExist(err) {
			return result, nil
		}
		return nil, fmt.Errorf("reading user category dir %q: %w", userDir, err)
	}
	for _, entry := range userEntries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		path := filepath.Join(userDir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("reading user category %q: %w", path, err)
		}
		cat, err := parse(path, data)
		if err != nil {
			return nil, err
		}
		result[cat.Name] = cat
	}

	return result, nil
}

func parse(source string, data []byte) (Category, error) {
	var cat Category
	if err := json.Unmarshal(data, &cat); err != nil {
		return Category{}, fmt.Errorf("parsing category %q: %w", source, err)
	}
	if cat.Name == "" {
		return Category{}, fmt.Errorf("category %q: missing required \"name\" field", source)
	}
	return cat, nil
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./internal/category/...`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add categories internal/category
git commit -m "feat: add category model and loader with user overrides"
```

---

### Task 3: Prompt template rendering

**Files:**
- Create: `internal/prompt/prompt.go`
- Test: `internal/prompt/prompt_test.go`

**Interfaces:**
- Consumes: `category.Category`, `category.Field` (Task 2).
- Produces: `prompt.Render(cat category.Category, answers map[string]string) (string, error)`

- [ ] **Step 1: Write the failing tests**

```go
// internal/prompt/prompt_test.go
package prompt

import (
	"strings"
	"testing"

	"prompt-creator/internal/category"
)

func testCategory() category.Category {
	return category.Category{
		Name: "test",
		Fields: []category.Field{
			{Key: "goal", Required: true},
			{Key: "context"},
			{Key: "constraints"},
		},
		Template: "Goal: {{.goal}}\n{{if .context}}\nContext:\n{{.context}}\n{{end}}{{if .constraints}}\nConstraints: {{.constraints}}\n{{end}}",
	}
}

func TestRenderIncludesOnlyAnsweredFields(t *testing.T) {
	out, err := Render(testCategory(), map[string]string{"goal": "Write a poem"})
	if err != nil {
		t.Fatalf("Render returned error: %v", err)
	}
	if !strings.Contains(out, "Goal: Write a poem") {
		t.Fatalf("expected goal line in output, got:\n%s", out)
	}
	if strings.Contains(out, "Context:") {
		t.Fatalf("expected blank optional field to be omitted, got:\n%s", out)
	}
	if strings.Contains(out, "Constraints:") {
		t.Fatalf("expected blank optional field to be omitted, got:\n%s", out)
	}
}

func TestRenderIncludesAllAnsweredFields(t *testing.T) {
	out, err := Render(testCategory(), map[string]string{
		"goal":        "Write a poem",
		"context":     "For a birthday card",
		"constraints": "Under 4 lines",
	})
	if err != nil {
		t.Fatalf("Render returned error: %v", err)
	}
	for _, want := range []string{"Goal: Write a poem", "Context:\nFor a birthday card", "Constraints: Under 4 lines"} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected output to contain %q, got:\n%s", want, out)
		}
	}
}

func TestRenderInvalidTemplateReturnsError(t *testing.T) {
	cat := testCategory()
	cat.Template = "{{.goal"
	_, err := Render(cat, map[string]string{"goal": "x"})
	if err == nil {
		t.Fatalf("expected error for invalid template")
	}
}

func TestRenderTemplateReferencingUnknownFieldKeyRendersBlank(t *testing.T) {
	cat := category.Category{
		Name: "test",
		Fields: []category.Field{
			{Key: "goal", Required: true},
		},
		// "extra" is not declared in Fields — a config typo.
		Template: "Goal: {{.goal}}\nExtra: {{.extra}}\n",
	}

	out, err := Render(cat, map[string]string{"goal": "Write a poem"})
	if err != nil {
		t.Fatalf("Render returned error for unknown template key: %v", err)
	}
	if !strings.Contains(out, "Goal: Write a poem") {
		t.Fatalf("expected goal line, got:\n%s", out)
	}
	if !strings.Contains(out, "Extra: \n") {
		t.Fatalf("expected unknown field to render blank rather than erroring, got:\n%s", out)
	}
}

func TestRenderRealGenericCategoryOmitsBlankOptionalFields(t *testing.T) {
	cats, err := category.Load(t.TempDir())
	if err != nil {
		t.Fatalf("category.Load returned error: %v", err)
	}
	generic, ok := cats["generic"]
	if !ok {
		t.Fatalf("expected built-in generic category")
	}

	out, err := Render(generic, map[string]string{"goal": "Refactor the parser"})
	if err != nil {
		t.Fatalf("Render returned error: %v", err)
	}
	if strings.TrimSpace(out) != "Goal: Refactor the parser" {
		t.Fatalf("expected only the goal line for an otherwise-blank generic prompt, got:\n%q", out)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/prompt/...`
Expected: FAIL (package `prompt` does not exist yet)

- [ ] **Step 3: Implement Render**

```go
// internal/prompt/prompt.go
package prompt

import (
	"bytes"
	"fmt"
	"text/template"

	"prompt-creator/internal/category"
)

// Render executes the category's template against the provided field
// answers (keyed by Field.Key) and returns the assembled prompt text.
func Render(cat category.Category, answers map[string]string) (string, error) {
	tmpl, err := template.New(cat.Name).Parse(cat.Template)
	if err != nil {
		return "", fmt.Errorf("parsing template for category %q: %w", cat.Name, err)
	}

	data := map[string]string{}
	for _, field := range cat.Fields {
		data[field.Key] = answers[field.Key]
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("rendering template for category %q: %w", cat.Name, err)
	}
	return buf.String(), nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/prompt/...`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/prompt
git commit -m "feat: render prompts from category templates"
```

---

### Task 4: Interactive wizard primitives

**Files:**
- Create: `internal/wizard/wizard.go`
- Test: `internal/wizard/wizard_test.go`

**Interfaces:**
- Consumes: `category.Category`, `category.Field` (Task 2).
- Produces:
  - `wizard.Run(w io.Writer, scanner *bufio.Scanner, cat category.Category) (map[string]string, error)`
  - `wizard.Confirm(w io.Writer, scanner *bufio.Scanner, question string) (bool, error)`

- [ ] **Step 1: Write the failing tests**

```go
// internal/wizard/wizard_test.go
package wizard

import (
	"bufio"
	"bytes"
	"strings"
	"testing"

	"prompt-creator/internal/category"
)

func testCategory() category.Category {
	return category.Category{
		Name: "test",
		Fields: []category.Field{
			{Key: "goal", Label: "Goal", Question: "What do you want?", Required: true},
			{Key: "context", Label: "Context", Question: "Any background?", Required: false},
		},
	}
}

func TestRunCollectsAnswersInOrder(t *testing.T) {
	input := strings.NewReader(strings.Join([]string{"Write a poem", "For a birthday"}, "\n") + "\n")
	var output bytes.Buffer

	answers, err := Run(&output, bufio.NewScanner(input), testCategory())
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if answers["goal"] != "Write a poem" {
		t.Fatalf("expected goal answer, got %q", answers["goal"])
	}
	if answers["context"] != "For a birthday" {
		t.Fatalf("expected context answer, got %q", answers["context"])
	}
	if !strings.Contains(output.String(), "What do you want?") {
		t.Fatalf("expected required question without optional hint, got:\n%s", output.String())
	}
	if !strings.Contains(output.String(), "Any background? (optional, press Enter to skip)") {
		t.Fatalf("expected optional question with hint, got:\n%s", output.String())
	}
}

func TestRunSkipsOptionalFieldOnEmptyInput(t *testing.T) {
	input := strings.NewReader(strings.Join([]string{"Write a poem", ""}, "\n") + "\n")
	var output bytes.Buffer

	answers, err := Run(&output, bufio.NewScanner(input), testCategory())
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if answers["context"] != "" {
		t.Fatalf("expected empty context answer, got %q", answers["context"])
	}
}

func TestRunRepromptsOnEmptyRequiredField(t *testing.T) {
	input := strings.NewReader(strings.Join([]string{"", "", "Write a poem", ""}, "\n") + "\n")
	var output bytes.Buffer

	answers, err := Run(&output, bufio.NewScanner(input), testCategory())
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if answers["goal"] != "Write a poem" {
		t.Fatalf("expected goal answer after reprompt, got %q", answers["goal"])
	}
	if !strings.Contains(output.String(), `"Goal" is required.`) {
		t.Fatalf("expected reprompt message, got:\n%s", output.String())
	}
}

func TestRunRepromptsOnWhitespaceOnlyRequiredField(t *testing.T) {
	input := strings.NewReader(strings.Join([]string{"   ", "Write a poem", ""}, "\n") + "\n")
	var output bytes.Buffer

	answers, err := Run(&output, bufio.NewScanner(input), testCategory())
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if answers["goal"] != "Write a poem" {
		t.Fatalf("expected whitespace-only answer to be rejected and re-prompted, got %q", answers["goal"])
	}
}

func TestRunReturnsErrorOnUnexpectedEOF(t *testing.T) {
	input := strings.NewReader("")
	var output bytes.Buffer

	_, err := Run(&output, bufio.NewScanner(input), testCategory())
	if err == nil {
		t.Fatalf("expected error when input ends before required field answered")
	}
}

func TestConfirmYes(t *testing.T) {
	scanner := bufio.NewScanner(strings.NewReader("y\n"))
	var output bytes.Buffer

	ok, err := Confirm(&output, scanner, "Save this?")
	if err != nil {
		t.Fatalf("Confirm returned error: %v", err)
	}
	if !ok {
		t.Fatalf("expected true for 'y' answer")
	}
	if !strings.Contains(output.String(), "Save this? (y/n): ") {
		t.Fatalf("expected question in output, got: %s", output.String())
	}
}

func TestConfirmNo(t *testing.T) {
	scanner := bufio.NewScanner(strings.NewReader("n\n"))
	var output bytes.Buffer

	ok, err := Confirm(&output, scanner, "Save this?")
	if err != nil {
		t.Fatalf("Confirm returned error: %v", err)
	}
	if ok {
		t.Fatalf("expected false for 'n' answer")
	}
}

func TestConfirmEmptyAnswerIsNo(t *testing.T) {
	scanner := bufio.NewScanner(strings.NewReader("\n"))
	var output bytes.Buffer

	ok, err := Confirm(&output, scanner, "Save this?")
	if err != nil {
		t.Fatalf("Confirm returned error: %v", err)
	}
	if ok {
		t.Fatalf("expected false for empty answer")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/wizard/...`
Expected: FAIL (package `wizard` does not exist yet)

- [ ] **Step 3: Implement Run and Confirm**

```go
// internal/wizard/wizard.go
package wizard

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"prompt-creator/internal/category"
)

// Run asks the user each field's question for the given category, writing
// prompts to w and reading answers from scanner. Required fields are
// re-asked until a non-empty (after trimming whitespace) answer is given.
// It returns the collected answers keyed by Field.Key.
func Run(w io.Writer, scanner *bufio.Scanner, cat category.Category) (map[string]string, error) {
	answers := map[string]string{}

	for _, field := range cat.Fields {
		for {
			hint := " (optional, press Enter to skip)"
			if field.Required {
				hint = ""
			}
			fmt.Fprintf(w, "%s%s: ", field.Question, hint)

			if !scanner.Scan() {
				if err := scanner.Err(); err != nil {
					return nil, fmt.Errorf("reading answer for %q: %w", field.Key, err)
				}
				return nil, fmt.Errorf("reading answer for %q: unexpected end of input", field.Key)
			}
			answer := strings.TrimSpace(scanner.Text())

			if answer == "" && field.Required {
				fmt.Fprintf(w, "%q is required.\n", field.Label)
				continue
			}

			answers[field.Key] = answer
			break
		}
	}

	return answers, nil
}

// Confirm asks a yes/no question, writing it to w and reading the answer
// from scanner. It returns true only for "y" or "yes" (case-insensitive).
func Confirm(w io.Writer, scanner *bufio.Scanner, question string) (bool, error) {
	fmt.Fprintf(w, "%s (y/n): ", question)

	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return false, fmt.Errorf("reading confirmation: %w", err)
		}
		return false, fmt.Errorf("reading confirmation: unexpected end of input")
	}
	answer := strings.ToLower(strings.TrimSpace(scanner.Text()))
	return answer == "y" || answer == "yes", nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/wizard/...`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/wizard
git commit -m "feat: add interactive wizard Q&A primitives"
```

---

### Task 5: Cross-platform clipboard

**Files:**
- Create: `internal/clipboard/clipboard.go`
- Test: `internal/clipboard/clipboard_test.go`

**Interfaces:**
- Consumes: nothing new (stdlib `os/exec`, `runtime` only).
- Produces:
  - `clipboard.Copy(text string) error`
  - `clipboard.ErrUnavailable error` (sentinel; check with `errors.Is`)

- [ ] **Step 1: Write the failing tests**

```go
// internal/clipboard/clipboard_test.go
package clipboard

import (
	"errors"
	"testing"
)

func TestCommandForOSDarwin(t *testing.T) {
	name, args, err := commandForOS("darwin", func(string) (string, error) { return "", errors.New("not found") })
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if name != "pbcopy" || len(args) != 0 {
		t.Fatalf("got name=%q args=%v, want pbcopy with no args", name, args)
	}
}

func TestCommandForOSWindows(t *testing.T) {
	name, _, err := commandForOS("windows", func(string) (string, error) { return "", errors.New("not found") })
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if name != "clip" {
		t.Fatalf("got name=%q, want clip", name)
	}
}

func TestCommandForOSLinuxPrefersXclip(t *testing.T) {
	lookPath := func(name string) (string, error) {
		if name == "xclip" {
			return "/usr/bin/xclip", nil
		}
		return "", errors.New("not found")
	}
	name, args, err := commandForOS("linux", lookPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if name != "xclip" || len(args) != 2 {
		t.Fatalf("got name=%q args=%v, want xclip with 2 args", name, args)
	}
}

func TestCommandForOSLinuxFallsBackToWlCopy(t *testing.T) {
	lookPath := func(name string) (string, error) {
		if name == "wl-copy" {
			return "/usr/bin/wl-copy", nil
		}
		return "", errors.New("not found")
	}
	name, _, err := commandForOS("linux", lookPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if name != "wl-copy" {
		t.Fatalf("got name=%q, want wl-copy", name)
	}
}

func TestCommandForOSLinuxNoToolFound(t *testing.T) {
	lookPath := func(string) (string, error) { return "", errors.New("not found") }
	_, _, err := commandForOS("linux", lookPath)
	if err == nil {
		t.Fatalf("expected error when no clipboard tool is found")
	}
}

func TestCommandForOSUnsupported(t *testing.T) {
	_, _, err := commandForOS("plan9", func(string) (string, error) { return "", errors.New("not found") })
	if err == nil {
		t.Fatalf("expected error for unsupported OS")
	}
}

func TestCopySucceedsOrReturnsUnavailable(t *testing.T) {
	err := Copy("hello from prompt-creator tests")
	if err != nil && !errors.Is(err, ErrUnavailable) {
		t.Fatalf("expected nil or ErrUnavailable, got: %v", err)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/clipboard/...`
Expected: FAIL (package `clipboard` does not exist yet)

- [ ] **Step 3: Implement Copy**

```go
// internal/clipboard/clipboard.go
package clipboard

import (
	"errors"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

// ErrUnavailable indicates the clipboard could not be reached. Callers
// should treat this as non-fatal — the prompt has already been printed.
var ErrUnavailable = errors.New("clipboard unavailable")

// commandForOS returns the executable name and args used to copy stdin to
// the system clipboard on the given GOOS, or an error if no suitable tool
// is available. lookPath is injected so tests can simulate any OS/PATH
// combination without real clipboard binaries.
func commandForOS(goos string, lookPath func(string) (string, error)) (string, []string, error) {
	switch goos {
	case "darwin":
		return "pbcopy", nil, nil
	case "windows":
		return "clip", nil, nil
	case "linux":
		if _, err := lookPath("xclip"); err == nil {
			return "xclip", []string{"-selection", "clipboard"}, nil
		}
		if _, err := lookPath("wl-copy"); err == nil {
			return "wl-copy", nil, nil
		}
		return "", nil, errors.New("no clipboard utility found (tried xclip, wl-copy)")
	default:
		return "", nil, fmt.Errorf("unsupported OS %q", goos)
	}
}

// Copy writes text to the system clipboard using an OS-appropriate external
// command. Any failure to find or run that command is wrapped in
// ErrUnavailable so callers can treat it as a warning, not a fatal error.
func Copy(text string) error {
	name, args, err := commandForOS(runtime.GOOS, exec.LookPath)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrUnavailable, err)
	}

	cmd := exec.Command(name, args...)
	cmd.Stdin = strings.NewReader(text)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%w: running %s: %v", ErrUnavailable, name, err)
	}
	return nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/clipboard/...`
Expected: PASS (on a machine without a clipboard tool, `TestCopySucceedsOrReturnsUnavailable` still passes because the failure is wrapped in `ErrUnavailable`)

- [ ] **Step 5: Commit**

```bash
git add internal/clipboard
git commit -m "feat: add cross-platform clipboard copy"
```

---

### Task 6: History persistence

**Files:**
- Create: `internal/history/history.go`
- Test: `internal/history/history_test.go`

**Interfaces:**
- Consumes: `appdir.ConfigDir() (string, error)` (Task 1).
- Produces:
  - `history.Entry{Category string; Answers map[string]string; Prompt string; SavedAt time.Time}`
  - `history.Save(entry Entry) (string, error)` — returns the path written.

- [ ] **Step 1: Write the failing tests**

```go
// internal/history/history_test.go
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
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/history/...`
Expected: FAIL (package `history` does not exist yet)

- [ ] **Step 3: Implement Save**

```go
// internal/history/history.go
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
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/history/...`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/history
git commit -m "feat: save prompts to timestamped history files"
```

---

### Task 7: CLI `generate` command

**Files:**
- Create: `cmd/prompt-creator/main.go`
- Test: `cmd/prompt-creator/main_test.go`

**Interfaces:**
- Consumes: `appdir.ConfigDir` (1), `category.Load`/`category.Category` (2), `prompt.Render` (3), `clipboard.Copy` (5), `history.Save`/`history.Entry` (6).
- Produces:
  - `run(args []string, stdin io.Reader, stdout, stderr io.Writer) int` (package `main`)
  - `runGenerate(args []string, stdout, stderr io.Writer) int` (package `main`)

- [ ] **Step 1: Write the failing tests**

```go
// cmd/prompt-creator/main_test.go
package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
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
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./cmd/prompt-creator/...`
Expected: FAIL (package `main`'s `run`/`runGenerate` do not exist yet)

- [ ] **Step 3: Implement main.go**

```go
// cmd/prompt-creator/main.go
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"prompt-creator/internal/appdir"
	"prompt-creator/internal/category"
	"prompt-creator/internal/clipboard"
	"prompt-creator/internal/history"
	"prompt-creator/internal/prompt"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "interactive wizard is not implemented yet; use the \"generate\" command")
		return 2
	}

	switch args[0] {
	case "generate":
		return runGenerate(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown command %q\n", args[0])
		return 2
	}
}

func runGenerate(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("generate", flag.ContinueOnError)
	fs.SetOutput(stderr)

	goal := fs.String("goal", "", "what you want the LLM to do (required)")
	contextAnswer := fs.String("context", "", "relevant background")
	constraints := fs.String("constraints", "", "format, length, tone, things to avoid")
	outputFormat := fs.String("output-format", "", "how the response should be structured")
	verbosity := fs.String("verbosity", "", "how much explanation the LLM should include")
	scopeBoundary := fs.String("scope-boundary", "", "anything explicitly out of scope")
	assumedKnowledge := fs.String("assumed-knowledge", "", "what the LLM should skip re-explaining")
	categoryName := fs.String("category", "generic", "category to use")
	save := fs.Bool("save", false, "save the generated prompt to history")
	noClipboard := fs.Bool("no-clipboard", false, "skip copying the prompt to the clipboard")

	if err := fs.Parse(args); err != nil {
		return 2
	}

	if *goal == "" {
		fmt.Fprintln(stderr, "--goal is required")
		return 2
	}

	userDir, err := appdir.ConfigDir()
	if err != nil {
		fmt.Fprintf(stderr, "resolving config dir: %v\n", err)
		return 1
	}

	cats, err := category.Load(filepath.Join(userDir, "categories"))
	if err != nil {
		fmt.Fprintf(stderr, "loading categories: %v\n", err)
		return 1
	}

	cat, ok := cats[*categoryName]
	if !ok {
		fmt.Fprintf(stderr, "unknown category %q\n", *categoryName)
		return 2
	}

	answers := map[string]string{
		"goal":              *goal,
		"context":           *contextAnswer,
		"constraints":       *constraints,
		"output_format":     *outputFormat,
		"verbosity":         *verbosity,
		"scope_boundary":    *scopeBoundary,
		"assumed_knowledge": *assumedKnowledge,
	}

	rendered, err := prompt.Render(cat, answers)
	if err != nil {
		fmt.Fprintf(stderr, "rendering prompt: %v\n", err)
		return 1
	}

	fmt.Fprintln(stdout, rendered)

	if !*noClipboard {
		if err := clipboard.Copy(rendered); err != nil {
			fmt.Fprintf(stderr, "warning: %v\n", err)
		}
	}

	if *save {
		path, err := history.Save(history.Entry{
			Category: *categoryName,
			Answers:  answers,
			Prompt:   rendered,
		})
		if err != nil {
			fmt.Fprintf(stderr, "warning: could not save history: %v\n", err)
		} else {
			fmt.Fprintf(stderr, "saved to %s\n", path)
		}
	}

	return 0
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./cmd/prompt-creator/...`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add cmd/prompt-creator
git commit -m "feat: add generate CLI command"
```

---

### Task 8: CLI default interactive wizard mode

**Files:**
- Modify: `cmd/prompt-creator/main.go` (replace the no-args stub with a real wizard flow)
- Modify: `cmd/prompt-creator/main_test.go` (add wizard tests)

**Interfaces:**
- Consumes: `wizard.Run`/`wizard.Confirm` (Task 4), everything Task 7 consumes.
- Produces: `runWizard(stdin io.Reader, stdout, stderr io.Writer) int` (package `main`)

- [ ] **Step 1: Write the failing tests**

Append to `cmd/prompt-creator/main_test.go`:

```go
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
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./cmd/prompt-creator/...`
Expected: FAIL (`runWizard` not wired in; no-args path still returns the Task 7 stub error)

- [ ] **Step 3: Implement the wizard flow**

Replace the full contents of `cmd/prompt-creator/main.go` with:

```go
// cmd/prompt-creator/main.go
package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"prompt-creator/internal/appdir"
	"prompt-creator/internal/category"
	"prompt-creator/internal/clipboard"
	"prompt-creator/internal/history"
	"prompt-creator/internal/prompt"
	"prompt-creator/internal/wizard"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		return runWizard(stdin, stdout, stderr)
	}

	switch args[0] {
	case "generate":
		return runGenerate(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown command %q\n", args[0])
		return 2
	}
}

func runWizard(stdin io.Reader, stdout, stderr io.Writer) int {
	userDir, err := appdir.ConfigDir()
	if err != nil {
		fmt.Fprintf(stderr, "resolving config dir: %v\n", err)
		return 1
	}

	cats, err := category.Load(filepath.Join(userDir, "categories"))
	if err != nil {
		fmt.Fprintf(stderr, "loading categories: %v\n", err)
		return 1
	}

	cat, ok := cats["generic"]
	if !ok {
		fmt.Fprintln(stderr, "built-in \"generic\" category is missing")
		return 1
	}

	scanner := bufio.NewScanner(stdin)

	answers, err := wizard.Run(stdout, scanner, cat)
	if err != nil {
		fmt.Fprintf(stderr, "%v\n", err)
		return 1
	}

	rendered, err := prompt.Render(cat, answers)
	if err != nil {
		fmt.Fprintf(stderr, "rendering prompt: %v\n", err)
		return 1
	}

	fmt.Fprintln(stdout, rendered)

	if err := clipboard.Copy(rendered); err != nil {
		fmt.Fprintf(stderr, "warning: %v\n", err)
	}

	save, err := wizard.Confirm(stdout, scanner, "Save this prompt to history?")
	if err != nil {
		fmt.Fprintf(stderr, "%v\n", err)
		return 1
	}
	if save {
		path, err := history.Save(history.Entry{
			Category: "generic",
			Answers:  answers,
			Prompt:   rendered,
		})
		if err != nil {
			fmt.Fprintf(stderr, "warning: could not save history: %v\n", err)
		} else {
			fmt.Fprintf(stdout, "saved to %s\n", path)
		}
	}

	return 0
}

func runGenerate(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("generate", flag.ContinueOnError)
	fs.SetOutput(stderr)

	goal := fs.String("goal", "", "what you want the LLM to do (required)")
	contextAnswer := fs.String("context", "", "relevant background")
	constraints := fs.String("constraints", "", "format, length, tone, things to avoid")
	outputFormat := fs.String("output-format", "", "how the response should be structured")
	verbosity := fs.String("verbosity", "", "how much explanation the LLM should include")
	scopeBoundary := fs.String("scope-boundary", "", "anything explicitly out of scope")
	assumedKnowledge := fs.String("assumed-knowledge", "", "what the LLM should skip re-explaining")
	categoryName := fs.String("category", "generic", "category to use")
	save := fs.Bool("save", false, "save the generated prompt to history")
	noClipboard := fs.Bool("no-clipboard", false, "skip copying the prompt to the clipboard")

	if err := fs.Parse(args); err != nil {
		return 2
	}

	if *goal == "" {
		fmt.Fprintln(stderr, "--goal is required")
		return 2
	}

	userDir, err := appdir.ConfigDir()
	if err != nil {
		fmt.Fprintf(stderr, "resolving config dir: %v\n", err)
		return 1
	}

	cats, err := category.Load(filepath.Join(userDir, "categories"))
	if err != nil {
		fmt.Fprintf(stderr, "loading categories: %v\n", err)
		return 1
	}

	cat, ok := cats[*categoryName]
	if !ok {
		fmt.Fprintf(stderr, "unknown category %q\n", *categoryName)
		return 2
	}

	answers := map[string]string{
		"goal":              *goal,
		"context":           *contextAnswer,
		"constraints":       *constraints,
		"output_format":     *outputFormat,
		"verbosity":         *verbosity,
		"scope_boundary":    *scopeBoundary,
		"assumed_knowledge": *assumedKnowledge,
	}

	rendered, err := prompt.Render(cat, answers)
	if err != nil {
		fmt.Fprintf(stderr, "rendering prompt: %v\n", err)
		return 1
	}

	fmt.Fprintln(stdout, rendered)

	if !*noClipboard {
		if err := clipboard.Copy(rendered); err != nil {
			fmt.Fprintf(stderr, "warning: %v\n", err)
		}
	}

	if *save {
		path, err := history.Save(history.Entry{
			Category: *categoryName,
			Answers:  answers,
			Prompt:   rendered,
		})
		if err != nil {
			fmt.Fprintf(stderr, "warning: could not save history: %v\n", err)
		} else {
			fmt.Fprintf(stderr, "saved to %s\n", path)
		}
	}

	return 0
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./cmd/prompt-creator/...`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add cmd/prompt-creator
git commit -m "feat: add interactive wizard as the default CLI mode"
```

---

### Task 9: CLI `categories list` subcommand

**Files:**
- Modify: `cmd/prompt-creator/main.go` (add the `categories` dispatch branch and `runCategories`)
- Modify: `cmd/prompt-creator/main_test.go` (add categories-list tests)

**Interfaces:**
- Consumes: `category.Load` (Task 2), `run`'s dispatch (Task 7/8).
- Produces: `runCategories(args []string, stdout, stderr io.Writer) int` (package `main`)

- [ ] **Step 1: Write the failing tests**

Append to `cmd/prompt-creator/main_test.go`:

```go
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
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./cmd/prompt-creator/...`
Expected: FAIL (`categories` command not recognized yet — falls into the `default` "unknown command" branch)

- [ ] **Step 3: Add the categories dispatch and implementation**

In `cmd/prompt-creator/main.go`, add `"sort"` to the import block, add a case to `run`'s switch, and add `runCategories`:

```go
	switch args[0] {
	case "generate":
		return runGenerate(args[1:], stdout, stderr)
	case "categories":
		return runCategories(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown command %q\n", args[0])
		return 2
	}
```

```go
func runCategories(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || args[0] != "list" {
		fmt.Fprintln(stderr, "usage: prompt-creator categories list")
		return 2
	}

	userDir, err := appdir.ConfigDir()
	if err != nil {
		fmt.Fprintf(stderr, "resolving config dir: %v\n", err)
		return 1
	}

	cats, err := category.Load(filepath.Join(userDir, "categories"))
	if err != nil {
		fmt.Fprintf(stderr, "loading categories: %v\n", err)
		return 1
	}

	names := make([]string, 0, len(cats))
	for name := range cats {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		fmt.Fprintf(stdout, "%s\t%s\n", name, cats[name].Description)
	}

	return 0
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./cmd/prompt-creator/...`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add cmd/prompt-creator
git commit -m "feat: add categories list CLI command"
```

---

### Task 10: Local LLM (Ollama) polish pass

**Files:**
- Create: `internal/llm/llm.go`
- Test: `internal/llm/llm_test.go`
- Modify: `cmd/prompt-creator/main.go` (wire `--polish`/`--polish-model`/`--ollama-host` into `runGenerate`)
- Modify: `cmd/prompt-creator/main_test.go` (add polish tests)

**Interfaces:**
- Consumes: `runGenerate` (Task 7/8).
- Produces:
  - `llm.Polisher` interface with `Polish(ctx context.Context, rawPrompt string) (string, error)`
  - `llm.OllamaPolisher{Host, Model string; Client *http.Client}` implementing `Polisher`

- [ ] **Step 1: Write the failing tests for OllamaPolisher**

```go
// internal/llm/llm_test.go
package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestOllamaPolisherSendsPromptAndReturnsPolishedText(t *testing.T) {
	var gotBody ollamaRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/generate" {
			t.Fatalf("expected path /api/generate, got %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decoding request body: %v", err)
		}
		json.NewEncoder(w).Encode(ollamaResponse{Response: "Polished prompt"})
	}))
	defer server.Close()

	p := OllamaPolisher{Host: server.URL, Model: "llama3"}
	got, err := p.Polish(context.Background(), "Goal: write a poem")
	if err != nil {
		t.Fatalf("Polish returned error: %v", err)
	}
	if got != "Polished prompt" {
		t.Fatalf("got %q, want %q", got, "Polished prompt")
	}
	if gotBody.Model != "llama3" {
		t.Fatalf("expected model %q sent, got %q", "llama3", gotBody.Model)
	}
	if !strings.Contains(gotBody.Prompt, "Goal: write a poem") {
		t.Fatalf("expected raw prompt embedded in request, got %q", gotBody.Prompt)
	}
}

func TestOllamaPolisherReturnsErrorOnNonOKStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("boom"))
	}))
	defer server.Close()

	p := OllamaPolisher{Host: server.URL, Model: "llama3"}
	_, err := p.Polish(context.Background(), "x")
	if err == nil {
		t.Fatalf("expected error for non-200 response")
	}
}

func TestOllamaPolisherReturnsErrorWhenUnreachable(t *testing.T) {
	p := OllamaPolisher{Host: "http://127.0.0.1:1", Model: "llama3"}
	_, err := p.Polish(context.Background(), "x")
	if err == nil {
		t.Fatalf("expected error when host is unreachable")
	}
}

func TestOllamaPolisherReturnsErrorOnMalformedHost(t *testing.T) {
	p := OllamaPolisher{Host: "not-a-url", Model: "llama3"}
	_, err := p.Polish(context.Background(), "x")
	if err == nil {
		t.Fatalf("expected error for malformed host")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/llm/...`
Expected: FAIL (package `llm` does not exist yet)

- [ ] **Step 3: Implement OllamaPolisher**

```go
// internal/llm/llm.go
package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// Polisher tightens a raw templated prompt using an LLM.
type Polisher interface {
	Polish(ctx context.Context, rawPrompt string) (string, error)
}

// OllamaPolisher polishes prompts using a local Ollama server's
// /api/generate endpoint.
type OllamaPolisher struct {
	Host   string // e.g. "http://localhost:11434"
	Model  string
	Client *http.Client
}

type ollamaRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

type ollamaResponse struct {
	Response string `json:"response"`
}

const polishInstruction = "Tighten and polish the following prompt for clarity and brevity. Return only the polished prompt text, with no commentary:\n\n"

// Polish sends rawPrompt to the configured Ollama server and returns the
// polished text.
func (p OllamaPolisher) Polish(ctx context.Context, rawPrompt string) (string, error) {
	client := p.Client
	if client == nil {
		client = http.DefaultClient
	}

	body, err := json.Marshal(ollamaRequest{
		Model:  p.Model,
		Prompt: polishInstruction + rawPrompt,
		Stream: false,
	})
	if err != nil {
		return "", fmt.Errorf("encoding ollama request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.Host+"/api/generate", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("building ollama request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("calling ollama at %s: %w", p.Host, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("ollama returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var parsed ollamaResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return "", fmt.Errorf("decoding ollama response: %w", err)
	}
	if parsed.Response == "" {
		return "", fmt.Errorf("ollama returned an empty response")
	}

	return parsed.Response, nil
}
```

- [ ] **Step 4: Run llm tests to verify they pass**

Run: `go test ./internal/llm/...`
Expected: PASS

- [ ] **Step 5: Write the failing CLI-level tests**

Append to `cmd/prompt-creator/main_test.go`:

```go
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
```

Add `"net/http"` and `"net/http/httptest"` to the import block of `cmd/prompt-creator/main_test.go`.

- [ ] **Step 6: Run tests to verify they fail**

Run: `go test ./cmd/prompt-creator/...`
Expected: FAIL (`--polish` flag not recognized yet)

- [ ] **Step 7: Wire polish into runGenerate**

In `cmd/prompt-creator/main.go`, add `"context"` and `"prompt-creator/internal/llm"` to the import block, add three flags and the polish call inside `runGenerate`, right after `rendered, err := prompt.Render(...)` succeeds and before `fmt.Fprintln(stdout, rendered)`:

```go
	polish := fs.Bool("polish", false, "run the prompt through a local LLM (Ollama) polish pass")
	polishModel := fs.String("polish-model", "llama3", "Ollama model to use for polishing")
	ollamaHost := fs.String("ollama-host", "http://localhost:11434", "Ollama server host")
```

(add these three lines alongside the other `fs.*` flag declarations, before `fs.Parse(args)`)

```go
	if *polish {
		polisher := llm.OllamaPolisher{Host: *ollamaHost, Model: *polishModel}
		polished, err := polisher.Polish(context.Background(), rendered)
		if err != nil {
			fmt.Fprintf(stderr, "warning: could not reach local LLM at %s: %v\n", *ollamaHost, err)
		} else {
			rendered = polished
		}
	}
```

(insert this block immediately after the `rendered, err := prompt.Render(cat, answers)` error check, before `fmt.Fprintln(stdout, rendered)`)

- [ ] **Step 8: Run tests to verify they pass**

Run: `go test ./... `
Expected: PASS across all packages

- [ ] **Step 9: Commit**

```bash
git add internal/llm cmd/prompt-creator
git commit -m "feat: add optional Ollama polish pass to generate"
```

---

### Task 11: README, golden end-to-end test, cross-platform build check

**Files:**
- Create: `README.md`
- Create: `cmd/prompt-creator/testdata/golden_full_prompt.txt`
- Modify: `cmd/prompt-creator/main_test.go` (add the golden-file test)

**Interfaces:**
- Consumes: all previous tasks (end-to-end).
- Produces: nothing new — documentation and verification only.

- [ ] **Step 1: Write the failing golden-file test**

Append to `cmd/prompt-creator/main_test.go`:

```go
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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./cmd/prompt-creator/...`
Expected: FAIL (`testdata/golden_full_prompt.txt` does not exist yet)

- [ ] **Step 3: Add the golden file**

```
Goal: Refactor the auth middleware

Context:
Go service, uses JWT

Constraints: Keep the public API unchanged

Output format: unified diff

Verbosity: no explanation, code only

Do not: do not touch the database layer

Assume I already know: I know Go and JWT already
```

Save as `cmd/prompt-creator/testdata/golden_full_prompt.txt`.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./cmd/prompt-creator/...`
Expected: PASS

- [ ] **Step 5: Write the README**

```markdown
# prompt-creator

A CLI that helps build well-structured, token-efficient LLM prompts —
either through a guided interactive wizard or a single flag-based command.

## Install

Build from source (Go 1.21+, no third-party dependencies):

    go build -o prompt-creator ./cmd/prompt-creator

## Usage

Interactive wizard (default):

    ./prompt-creator

Flag-based, for quick drafting:

    ./prompt-creator generate --goal "Refactor the auth middleware" \
      --context "Go service, uses JWT" \
      --constraints "Keep the public API unchanged" \
      --verbosity "no explanation, code only"

Only `--goal` is required; every other field is optional and omitted from
the rendered prompt when left blank.

List available categories:

    ./prompt-creator categories list

Save a generated prompt to `~/.config/prompt-creator/history/`:

    ./prompt-creator generate --goal "..." --save

Run the prompt through a local Ollama model before printing it:

    ./prompt-creator generate --goal "..." --polish --polish-model llama3

## Categories

Built-in categories live in `categories/*.json` (embedded in the binary).
Add your own under `~/.config/prompt-creator/categories/*.json` — a file
with the same `name` as a built-in overrides it.

## Development

    go test ./...
```

- [ ] **Step 6: Verify cross-platform builds**

Run each of these from the repo root and confirm all three succeed with no output (no errors):

```bash
GOOS=darwin GOARCH=amd64 go build -o /dev/null ./cmd/prompt-creator
GOOS=linux GOARCH=amd64 go build -o /dev/null ./cmd/prompt-creator
GOOS=windows GOARCH=amd64 go build -o /dev/null ./cmd/prompt-creator
```

- [ ] **Step 7: Run the full test suite**

Run: `go test ./...`
Expected: PASS across every package

- [ ] **Step 8: Commit**

```bash
git add README.md cmd/prompt-creator/testdata cmd/prompt-creator/main_test.go
git commit -m "docs: add README and golden end-to-end test"
```
