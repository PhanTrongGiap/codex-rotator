# Contributing

Thanks for helping improve `codex-rotator`. This project is intentionally small, so contributions should stay focused and easy to review.

## Development setup

```bash
git clone https://github.com/PhanTrongGiap/codex-rotator
cd codex-rotator
go mod download
make test
make build
```

Use temporary paths when testing commands that read or write auth files:

```bash
go run ./cmd/codex-rotator --pool /tmp/rotator-auth --codex-auth /tmp/codex-auth.json list
```

## Code style

- Run `go fmt ./...` before opening a pull request.
- Keep package boundaries aligned with `internal/oauth`, `internal/store`, `internal/probe`, and `internal/rotate`.
- Wrap errors with useful context, for example `fmt.Errorf("listing pool: %w", err)`.
- Avoid adding broad abstractions unless they simplify a real repeated pattern.

## Tests

Add `*_test.go` files beside the package you change. Prefer table-driven tests for filename generation, token parsing, status handling, and rotation decisions. Run `make test` to verify — it includes the race detector.

Do not use real `~/.codex` or `~/.codex-rotator` paths in tests. Use `t.TempDir()` and local fixture data.

## Pull requests

Please include:

- what changed and why
- commands you ran, especially `go test -race -cover ./...`
- any auth-file or token-handling risk
- screenshots or terminal output if CLI behavior changed

Commit messages should follow the current short conventional style, for example `feat: add daemon interval flag`, `fix: preserve auth file permissions`, or `docs: improve setup guide`.
