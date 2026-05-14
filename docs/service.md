# Running codex-rotator as a Service

The recommended way to install codex-rotator is `make install`, which builds the
binary, copies it to `/usr/local/bin`, and enables a **systemd user service** that
starts automatically on login and restarts on failure.

## Quick install (recommended)

```bash
git clone https://github.com/PhanTrongGiap/codex-rotator
cd codex-rotator
make install
```

That's it. The daemon starts immediately and checks your token every minute.

```bash
# Check status
systemctl --user status codex-rotator

# Follow logs
journalctl --user -u codex-rotator -f
```

## Uninstall

```bash
make uninstall
```

## Change the rotation interval

Edit `~/.config/systemd/user/codex-rotator.service`, update the `--interval` value
in `ExecStart`, then reload:

```bash
systemctl --user daemon-reload
systemctl --user restart codex-rotator
```

---

## Alternative: nohup

If systemd is not available (e.g. macOS, WSL without systemd):

```bash
nohup codex-rotator daemon --interval 1m >> ~/.codex-rotator/daemon.log 2>&1 &
echo $! > ~/.codex-rotator/daemon.pid
```

Stop:

```bash
kill $(cat ~/.codex-rotator/daemon.pid)
```

## Alternative: cron

For a minimal footprint without a long-running process:

```
*/1 * * * * /usr/local/bin/codex-rotator rotate >> ~/.codex-rotator/rotate.log 2>&1
```

Add with `crontab -e`.
