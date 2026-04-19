# jotter — local dev tasks.
# Run `just` to list recipes, `just <name>` to invoke one.

default:
    @just --list

# Build the binary into bin/.
build:
    go build -o bin/jotter .

# Run all tests.
test:
    go test ./...

# Run the linter (same config as CI).
lint:
    golangci-lint run

# Validate the GoReleaser config against its JSON Schema and semantic rules.
release-check:
    goreleaser check

# Run every check CI runs — build, test, lint, release-check. Use before pushing.
check: build test lint release-check

# Run a GoReleaser dry-run to validate the release config.
release-snapshot:
    goreleaser release --snapshot --clean

# Remove build artefacts.
clean:
    rm -rf bin/ dist/
