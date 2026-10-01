# Contributing

## Development

Common tasks are wrapped in a `Makefile` (run `make` to list them):

    make build    # build ./prompt-creator for the host
    make test     # go test ./...
    make check    # lint + tests — run this before opening a PR

`make` is the only tooling requirement. No third-party dependencies — keep it
that way unless there's a strong reason.

## Workflow

1. Write a failing test before writing the implementation (TDD).
2. Keep commits focused: one logical change per commit.
3. Run `make check` (lint + tests) before opening a PR.
4. Cross-platform builds should stay green:

       make cross-build   # darwin/linux/windows amd64 binaries into dist/

## Adding a category

Built-in categories live in `categories/*.json` and are embedded via
`go:embed`. Add a new field set/template there, or drop a JSON file in
`~/.config/prompt-creator/categories/` to test a category locally without
rebuilding — a user file with the same `name` overrides a built-in one.
