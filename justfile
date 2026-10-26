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

prefix := env_var_or_default("PREFIX", env_var("HOME") / ".local")
pkg := "github.com/sebjacobs/jotter/cmd"

# Build a version-stamped binary and copy it into $PREFIX/bin (default ~/.local/bin).
install:
    go build -ldflags "-s -w -X {{pkg}}.version=$(git describe --tags --always --dirty) -X {{pkg}}.commit=$(git rev-parse HEAD) -X {{pkg}}.date=$(date -u +%Y-%m-%dT%H:%M:%SZ)" -o bin/jotter .
    mkdir -p {{prefix}}/bin
    install -m 755 bin/jotter {{prefix}}/bin/jotter
    {{prefix}}/bin/jotter --version

# Remove the binary that `just install` put in $PREFIX/bin.
uninstall:
    rm -f {{prefix}}/bin/jotter

# Remove build artefacts.
clean:
    rm -rf bin/ dist/
