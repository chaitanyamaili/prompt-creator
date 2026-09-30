# Contributing

## Development

    go build -o prompt-creator ./cmd/prompt-creator
    go test ./...

No third-party dependencies — keep it that way unless there's a strong reason.

## Workflow

1. Write a failing test before writing the implementation (TDD).
2. Keep commits focused: one logical change per commit.
3. Run `go test ./...` and `go vet ./...` before opening a PR.
4. Cross-platform builds should stay green:

       GOOS=darwin GOARCH=amd64 go build -o /dev/null ./cmd/prompt-creator
       GOOS=linux GOARCH=amd64 go build -o /dev/null ./cmd/prompt-creator
       GOOS=windows GOARCH=amd64 go build -o /dev/null ./cmd/prompt-creator

## Adding a category

Built-in categories live in `categories/*.json` and are embedded via
`go:embed`. Add a new field set/template there, or drop a JSON file in
`~/.config/prompt-creator/categories/` to test a category locally without
rebuilding — a user file with the same `name` overrides a built-in one.
