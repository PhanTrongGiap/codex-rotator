# Repository Guidelines

## Project Structure & Module Organization

This is a Go CLI module for `codex-rotator`.

- `cmd/codex-rotator/main.go` contains the Cobra CLI entrypoint and command wiring.
- `internal/oauth/` handles OAuth, PKCE, JWT parsing, and the local callback server.
- `internal/store/` reads and writes account-pool token files.
- `internal/probe/` checks token status.
- `internal/rotate/` implements token selection and auth-file replacement.
- `README.md` documents user-facing installation and command usage.

The root `codex-rotator` binary is a build artifact. Rebuild from source during development.

## Build, Test, and Development Commands

- `go build -o codex-rotator ./cmd/codex-rotator` builds the CLI binary.
- `go run ./cmd/codex-rotator --help` runs the CLI without installing it.
- `go test ./...` runs all package tests.
- `go test ./... -race` runs tests with the race detector for concurrency-sensitive changes.
- `go fmt ./...` formats all Go source files.
- `go mod tidy` updates `go.mod` and `go.sum` after dependency changes.

For manual checks, use a temporary pool and auth path:

```bash
go run ./cmd/codex-rotator --pool /tmp/rotator-auth --codex-auth /tmp/codex-auth.json list
```

## Coding Style & Naming Conventions

Use standard Go formatting and idioms. Keep packages small and aligned with the current `internal/<domain>` layout. Prefer explicit error wrapping with context, for example `fmt.Errorf("listing pool: %w", err)`.

Export only APIs used across packages. Use mixedCaps for Go identifiers, short receiver names, and descriptive command/helper names such as `cmdRotate`, `ReadCurrentToken`, and `WriteCodexAuth`.

## Testing Guidelines

Tests live in `*_test.go` files beside each package. All packages have existing coverage — add tests alongside any package you change. Prefer table-driven tests for parsing, filename generation, token serialization, and rotation decisions.

Run `make test` before submitting. Use `go test ./internal/rotate -run TestName` for narrow iteration. Never touch real `~/.codex` data in tests — use `t.TempDir()` for all auth paths.

## Commit & Pull Request Guidelines

Existing history uses short conventional-style commits such as `feat: initial codex-rotator CLI` and `docs: add English README`. Continue that pattern with prefixes like `feat:`, `fix:`, `docs:`, `test:`, and `refactor:`.

Pull requests should include a concise description, user-visible behavior changes, test results, and any risks around auth files or token handling. Include terminal output or screenshots only when they clarify CLI behavior.

## Security & Configuration Tips

Treat token files as secrets. Do not commit real files from `~/.codex/auth.json` or `~/.codex-rotator/auth/`. Preserve restrictive file permissions when writing credentials (`0600` for auth files, `0700` for directories), and use temporary paths when developing or testing.
