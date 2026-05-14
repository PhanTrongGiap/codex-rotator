# Security Policy

`codex-rotator` handles local authentication tokens. Treat bugs involving token exposure, unsafe permissions, or unintended auth-file writes as security-sensitive.

## Supported versions

This repository currently supports the latest commit on the default branch.

## Reporting a vulnerability

Please do not open a public issue for vulnerabilities.

Report privately through GitHub Security Advisories if available, or contact the maintainer directly through the repository owner's GitHub profile. Include:

- a clear description of the issue
- affected command or package
- reproduction steps using temporary auth paths
- whether tokens, account IDs, or auth files can be exposed or overwritten

## Handling secrets

- Never commit `~/.codex/auth.json` or files from `~/.codex-rotator/auth/`.
- Redact `access_token`, `id_token`, `refresh_token`, and account IDs from logs and screenshots.
- Prefer test fixtures with fake token values.
- Revoke or rotate any real token that may have been shared.
