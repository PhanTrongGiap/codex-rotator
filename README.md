# codex-rotator

[![CI](https://github.com/PhanTrongGiap/codex-rotator/actions/workflows/ci.yml/badge.svg)](https://github.com/PhanTrongGiap/codex-rotator/actions/workflows/ci.yml)

**Rotate Codex accounts on quota — not on suspicion.**

When your ChatGPT account hits its usage limit, `codex-rotator` quietly swaps to the next working account in your local pool. It makes one minimal probe request per interval, switches only when the current token is actually exhausted, and never touches your credentials outside your own machine.

> For developers who own multiple ChatGPT accounts and want uninterrupted Codex sessions — without the paranoia of getting flagged.

## Is it safe?

**Will I get banned for using this?**

The short answer is: this tool is designed specifically to avoid the behavior that gets accounts flagged.

Here is what it actually does. Once per interval (default: 60 seconds), it sends a single minimal request to check whether your current token is still valid. If it is, nothing happens. If the quota is exceeded, it swaps the active auth file to the next working account in your pool. That is the entire runtime behavior.

Compare that to what gets accounts flagged: rapid repeated requests, token sharing across IPs, credential stuffing, or scripted login loops. `codex-rotator` does none of those things.

**What about the OAuth flow?**

Each account is added through the official Codex OAuth browser flow — the same one you would use to log in manually. No credentials are ever entered into this tool. It receives a callback token from OpenAI's own auth system and saves it locally, exactly as the Codex CLI does.

**Who can see my tokens?**

Nobody but you. Tokens are stored in `~/.codex-rotator/auth/` on your local machine with `0600` file permissions. There is no backend, no telemetry, no sync. The tool does not make network requests other than the quota probe and the OAuth callback.

**The honest caveat**

This tool manages local auth files. It does not change OpenAI's quota limits, terms of service, or enforcement policies. Using multiple ChatGPT accounts is permitted for individuals under current terms, but read OpenAI's policies for your specific plan and use case. This tool will not protect you from policy violations unrelated to rotation behavior.

If you own the accounts, use them normally, and just want to stop manually swapping auth files — this tool is safe to run.

## Features

- **Your credentials never leave your machine.** All tokens are stored under `~/.codex-rotator/auth/` with restrictive permissions. No proxy, no relay, no cloud.
- **Rotation only happens when it needs to.** One probe per interval. If the token is valid, the tool does nothing. No unnecessary API calls.
- **No suspicious API bursts.** The tool does not poll aggressively or test accounts in rapid succession. One check, one switch if needed, then silence.
- **Accounts are added through the official OAuth flow.** No stolen credentials, no shared tokens — just the same browser-based login Codex itself uses.
- **Zero-friction install.** `make install` builds the binary, places it in `/usr/local/bin`, and enables a systemd user service. Done.
- **Works over SSH.** Prints the auth URL directly to the terminal so you can complete OAuth from any machine with a browser.
- **Codex-compatible by design.** Writes exactly the `~/.codex/auth.json` format Codex CLI expects. No adapter, no shim.
- **Migrate from CLIProxyAPI.** One `import` command pulls in your existing account files if you are moving from a CLIProxyAPI setup.

## Installation

Requirements: Go 1.24+

```bash
git clone https://github.com/PhanTrongGiap/codex-rotator
cd codex-rotator
make install
```

`make install` builds the binary, copies it to `/usr/local/bin`, and enables a
**systemd user service** that starts automatically and checks your token every minute.

```bash
# Check it's running
systemctl --user status codex-rotator

# Follow logs
journalctl --user -u codex-rotator -f
```

To remove:

```bash
make uninstall
```

See [docs/service.md](docs/service.md) for changing the interval, nohup, and cron alternatives.

## Quick start

```bash
# 1. Add one or more accounts
codex-rotator login

# 2. Confirm the pool
codex-rotator list --check

# 3. Done — the daemon rotates automatically from here
```

## Remote server login

When running `codex-rotator login` over SSH, the OAuth callback URL lands on
`localhost:1455` on your **local** machine, not the remote server. Use either method:

**Method A — paste the callback URL (interactive)**

1. Copy the auth URL printed by the CLI and open it in a browser on any machine.
2. After authenticating, copy the full redirect URL from the browser address bar and paste it back into the waiting CLI.

**Method B — `--callback-url` flag (non-interactive / scripted)**

```bash
codex-rotator login --callback-url "http://localhost:1455/auth/callback?code=...&state=..."
```

> Callback URLs are **one-time use and sensitive**. Do not log or share them.

## How it stores files

```text
~/.codex-rotator/auth/          account pool
  codex-you@gmail.com-plus.json
  codex-other@gmail.com.json

~/.codex/auth.json              active auth file read by Codex CLI
```

All files are created with `0600` permissions. Treat every JSON file in these locations as a secret.

## Commands

### `login`

Add a new account through the official Codex OAuth flow.

```bash
codex-rotator login
codex-rotator login --open-browser   # auto-open browser on desktop
```

### `list`

Show accounts in the pool.

```bash
codex-rotator list
codex-rotator list --check   # probe each token (slower)
```

```text
Pool: /root/.codex-rotator/auth (2 accounts)

  you@gmail.com                           valid
  other@gmail.com                         quota_exceeded
```

### `rotate`

Check the active token and swap to the first working account if needed.

```bash
codex-rotator rotate
```

### `daemon`

Run rotation checks continuously at a fixed interval.

```bash
codex-rotator daemon --interval 60s
```

When the daemon rotates to a new account, your current Codex session is still
using the old (exhausted) token in memory. **Exit the stalled session and relaunch
Codex** — it will pick up the new auth file automatically.

```bash
# Codex stalls → exit it → relaunch
codex
```

This is intentional: the tool rotates the auth file safely in the background, and
you resume when ready rather than having your session restarted without warning.

### `run`

Rotate once, then exec the real `codex` binary.

```bash
codex-rotator run
codex-rotator run -- chat
```

### `import`

Import accounts from a CLIProxyAPI-compatible auth directory.

```bash
codex-rotator import --from /app/CLIProxyAPI/.auth
```

Files without an `access_token` are skipped with a warning. Tokens and account IDs are never printed. Useful when migrating from CLIProxyAPI — just point `--from` at the existing `.auth` directory.

## Global flags

| Flag | Default | Description |
| --- | --- | --- |
| `--pool` | `~/.codex-rotator/auth` | Directory containing account JSON files |
| `--codex-auth` | `~/.codex/auth.json` | Active auth file used by Codex CLI |

## Development

```bash
make fmt    # format
make lint   # check formatting
make test   # run tests
make build  # build binary
```

See [SECURITY.md](SECURITY.md) for vulnerability reporting and token handling guidelines.

## Contributing

Issues and pull requests are welcome. Please read [CONTRIBUTING.md](CONTRIBUTING.md) before proposing changes, especially for auth, token storage, and rotation behavior.

## License

Released under the [MIT License](LICENSE).
