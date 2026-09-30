# prompt-creator design

## Purpose

A Go CLI tool, distributed as a single static binary (macOS/Linux/Windows),
that helps build well-structured, token-efficient LLM prompts. Serves two
audiences: the author (fast flag-based drafting) and less-technical users
(guided interactive wizard).

## Scope (v1)

- Interactive wizard (default) and flag-based `generate` command (power users).
- Category config files (JSON), each defining a field set and a template.
  Ships with one built-in `generic` category; users can add their own under
  `~/.config/prompt-creator/categories/`.
- Pure templating by default — deterministic, offline, no LLM calls required.
- Optional local-LLM polish pass via Ollama's local HTTP API (`--polish`).
- Clipboard copy (cross-platform, shelling out to OS tools) and stdout print
  by default.
- History saved only when explicitly requested (`--save` flag, or a
  save-prompt at the end of the interactive wizard).

## CLI interface

```
prompt-creator                          # interactive wizard, default category "generic"
prompt-creator generate --goal "..." [--context "..."] [--constraints "..."]
    [--output-format "..."] [--verbosity "..."] [--scope-boundary "..."]
    [--assumed-knowledge "..."] [--category NAME] [--save] [--polish]
    [--polish-model NAME] [--ollama-host URL] [--no-clipboard]

prompt-creator categories list          # list built-in + user-defined categories
```

- `generate`: only `--goal` is required. Other fields are optional and
  omitted from the rendered prompt if blank — no padding with "N/A".
- Interactive wizard: asks every field for the selected category; optional
  fields can be skipped with empty Enter; required fields re-prompt on
  empty input.
- `--category`: selects field set/template; defaults to `generic`.
- `--save`: persists the result. In wizard mode, if `--save` wasn't passed,
  the wizard asks "Save this? (y/n)" at the end. In `generate` mode
  (non-interactive), saving only happens via `--save`.
- `--polish`: runs the Ollama pass (see below); without it, output is the
  raw templated prompt.

## Category & field model

Each category is a JSON file (built-in ones embedded via `go:embed`, user
ones loaded from `~/.config/prompt-creator/categories/*.json` — a user file
overrides a built-in of the same name):

```json
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

Go's `text/template` renders it, conditionally omitting any field left
blank — keeping the rendered prompt itself token-lean, not just the process
of building it.

### Field selection rationale (token economy)

Beyond goal/context/constraints/output-format, three fields specifically
target reducing LLM token burn:

- **Verbosity**: default LLM behavior pads with explanation unless told not
  to; this is the single biggest lever for shorter responses.
- **Scope boundary (do-not-do list)**: prevents over-generation and the
  corrective back-and-forth that costs far more tokens than a slightly
  longer prompt.
- **Assumed knowledge**: stops the model from restating background the user
  already has.

## Clipboard

`internal/clipboard` shells out via `os/exec`, chosen by `runtime.GOOS`:
`pbcopy` (darwin), `xclip -selection clipboard` or `wl-copy` (linux,
whichever is found on `$PATH`), `clip` (windows). No third-party Go module.
If no clipboard utility is found, print a warning to stderr and continue —
the prompt was already printed to stdout, so the run still succeeds.

## History

Only written when saving is triggered (`--save` flag, or "y" at the
wizard's save prompt). Stored as one JSON file per prompt under
`~/.config/prompt-creator/history/<timestamp>-<category>.json`, containing
the field answers, category name, and rendered prompt.

No history-browsing subcommand in v1 — files are plain JSON, browsable with
`ls`/`cat`/`jq`. A `history list` command can be added later if it proves
useful.

## Local LLM polish pass (Ollama)

`internal/llm` defines:

```go
type Polisher interface {
    Polish(ctx context.Context, rawPrompt string) (string, error)
}
```

`OllamaPolisher` implements it with `net/http` (stdlib, no dependency) —
POSTs to `{ollama-host}/api/generate` (default `http://localhost:11434`)
with the chosen model and the raw templated prompt, asking it to
tighten/polish the prompt itself (not answer it).

When `--polish` is set and Ollama is unreachable or errors: print a warning
to stderr ("could not reach local LLM at <host>: is Ollama running?") and
fall back to the raw templated prompt rather than failing the whole run.

## Package structure

```
prompt-creator/
  cmd/prompt-creator/main.go   # arg parsing, dispatch to wizard or generate
  internal/category/           # Field/Category types, embedded + user-dir loading, merge/override logic
  internal/prompt/              # text/template rendering from field values
  internal/wizard/              # interactive Q&A over io.Reader/io.Writer (testable without real stdin)
  internal/clipboard/           # OS-dispatched copy, via a small interface so tests can fake exec
  internal/history/             # save rendered prompt + answers to ~/.config/prompt-creator/history/
  internal/llm/                 # Polisher interface + OllamaPolisher
  categories/generic.json       # embedded default, via go:embed
  go.mod
```

## Error handling

- Malformed user category JSON: named file + parse error, exits non-zero;
  doesn't prevent other valid categories from loading.
- `generate` without `--goal`: clear error, non-zero exit.
- Clipboard tool missing: warning to stderr, run still succeeds (prompt
  already on stdout).
- Ollama unreachable with `--polish`: warning to stderr, falls back to raw
  prompt, exit 0.

## Testing

- `internal/prompt`: table tests over field combinations, including
  all-optional-fields-blank, verifying no stray blank lines/labels.
- `internal/category`: loading embedded + user dirs, override-by-name,
  malformed JSON handling.
- `internal/wizard`: fake `io.Reader`/`io.Writer`, assert questions asked
  match category fields and required-field re-prompting on empty input.
- `internal/clipboard`: fake command runner interface, assert correct
  command chosen per `GOOS`, and graceful handling when the binary isn't
  found.
- `internal/llm`: `httptest.Server` standing in for Ollama — success path
  and unreachable/error fallback path.
- `cmd/prompt-creator`: a couple of end-to-end tests using `generate`
  flags, golden-file comparison of rendered output.
