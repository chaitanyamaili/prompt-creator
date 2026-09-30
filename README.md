# prompt-creator

A CLI that helps build well-structured, token-efficient LLM prompts —
either through a guided interactive wizard or a single flag-based command.

## Install

Build from source (Go 1.26+, no third-party dependencies):

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

Run the prompt through a local Ollama model before printing it. If Ollama is
unreachable or doesn't respond within `--ollama-timeout` (default `30s`), the
unpolished prompt is printed instead and a warning goes to stderr:

    ./prompt-creator generate --goal "..." --polish --polish-model llama3 \
      --ollama-timeout 45s

## Categories

Built-in categories live in `categories/*.json` (embedded in the binary).
Add your own under `~/.config/prompt-creator/categories/*.json` — a file
with the same `name` as a built-in overrides it.

## Development

    go test ./...
