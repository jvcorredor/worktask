# List every recipe in this justfile
default:
    @just --list

# Compile the worktask CLI to ./bin/worktask
build:
    go build -o ./bin/worktask .

# Run the Go test suite
test:
    go test ./...

# Run the docs/public/install.sh test harness (POSIX shell, mocked curl)
test-install:
    sh tests/install/run.sh

# Run the post-release smoke-test assertion script's harness (POSIX shell, mocked worktask)
test-smoke-version:
    sh tests/smoke-test-version/run.sh

# Run go vet across all packages
vet:
    go vet ./...

# Format every Go file in place
fmt:
    gofmt -w .

# Fail if any Go file is not gofmt-clean (used by `ci`)
fmt-check:
    @out=$(gofmt -l .); if [ -n "$out" ]; then echo "Unformatted Go files:"; echo "$out"; exit 1; fi

# Remove build artifacts under ./bin
clean:
    rm -rf ./bin

# Boot the Astro dev server with docs/ as the working directory
docs-dev:
    cd docs && yarn dev

# Install (immutable) and build the Astro docs site, producing docs/dist/
docs-build:
    cd docs && yarn install --immutable && yarn build

# Placeholder for the lint gate; wired up in #10
lint:
    @echo "lint is not yet implemented; see https://github.com/jvcorredor/worktask/issues/10" >&2; exit 1

# Run the full local CI gate (fmt-check, vet, test, test-install, test-smoke-version) in workflow order
ci: fmt-check vet test test-install test-smoke-version

# Build a local snapshot release with goreleaser into ./dist (no publish)
release-snapshot:
    goreleaser release --snapshot --clean
