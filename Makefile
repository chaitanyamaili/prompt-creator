# prompt-creator — task runner
#
# Run `make` (or `make help`) to see available targets.
# No third-party tools required: everything here is `go`, `gofmt`, and `git`.

BINARY  := prompt-creator
PKG     := ./cmd/prompt-creator
DISTDIR := dist

# Version string injected at build time; harmless when there are no tags.
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -ldflags "-X main.version=$(VERSION)"

# Go files tracked for formatting (excludes anything under dist/).
GOFILES := $(shell find . -type f -name '*.go' -not -path './$(DISTDIR)/*')

# Platforms built by `cross-build`, mirroring CONTRIBUTING.md.
PLATFORMS := darwin/amd64 linux/amd64 windows/amd64

# ARGS lets you pass flags through `make run`, e.g.:
#   make run ARGS="generate --goal 'Refactor auth'"
ARGS ?=

.DEFAULT_GOAL := help

.PHONY: help build run test test-race cover vet fmt fmt-check lint cross-build check clean tidy

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) \
		| sort \
		| awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-12s\033[0m %s\n", $$1, $$2}'

build: ## Build the binary for the host platform
	go build $(LDFLAGS) -o $(BINARY) $(PKG)

run: ## Run the CLI (pass flags via ARGS="...")
	go run $(PKG) $(ARGS)

test: ## Run all tests
	go test ./...

test-race: ## Run all tests with the race detector
	go test -race ./...

cover: ## Run tests with coverage and print a per-function summary
	go test -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

vet: ## Run go vet
	go vet ./...

fmt: ## Format all Go files in place
	gofmt -w $(GOFILES)

fmt-check: ## Fail if any Go file is not gofmt-clean
	@unformatted=$$(gofmt -l $(GOFILES)); \
	if [ -n "$$unformatted" ]; then \
		echo "These files are not gofmt-clean:"; \
		echo "$$unformatted"; \
		exit 1; \
	fi

lint: vet fmt-check ## Dependency-free lint: go vet + gofmt check

cross-build: ## Build release binaries for all target platforms into dist/
	@mkdir -p $(DISTDIR)
	@for platform in $(PLATFORMS); do \
		os=$${platform%/*}; arch=$${platform#*/}; \
		ext=""; [ "$$os" = "windows" ] && ext=".exe"; \
		out="$(DISTDIR)/$(BINARY)-$$os-$$arch$$ext"; \
		echo "building $$out"; \
		GOOS=$$os GOARCH=$$arch go build $(LDFLAGS) -o "$$out" $(PKG) || exit 1; \
	done

check: lint test ## Pre-PR gate: lint + tests

clean: ## Remove build artifacts
	rm -f $(BINARY) coverage.out
	rm -rf $(DISTDIR)

tidy: ## Tidy go.mod/go.sum
	go mod tidy
