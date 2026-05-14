## Summary

- 

## Test plan

- [ ] `go test ./...`
- [ ] Manual CLI check, if behavior changed:

```bash
go run ./cmd/codex-rotator --pool /tmp/rotator-auth --codex-auth /tmp/codex-auth.json --help
```

## Security checklist

- [ ] No real tokens, auth files, or account IDs are included.
- [ ] Auth-file writes use explicit test paths or documented user paths.
- [ ] File permission behavior is preserved or explained.

## Notes

Link related issues and add terminal output or screenshots when they clarify CLI behavior.
