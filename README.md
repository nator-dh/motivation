# motivation

Local Go daemon that fires curated motivational quotes as macOS notifications on cron schedules. Single binary + an embedded web dashboard, backed by a YAML quote library. Click a notification to open its linked picture or YouTube video.

## Requirements

- macOS 11+ (developed and tested on macOS 26 Tahoe)
- Go 1.26+
- `swiftc` (ships with Xcode Command Line Tools — `xcode-select --install`)

No third-party CLIs or Homebrew taps required.

## Install

```sh
git clone https://github.com/<you>/motivation.git
cd motivation
make build          # compiles the Swift helper + the Go binary
make install        # copies both to ~/.local/bin (override with PREFIX=)
```

`make build` produces:
- `./motivation` — the Go daemon
- `./bin/MotivationNotify.app` — a tiny Swift `.app` bundle used to deliver notifications via `UNUserNotificationCenter`. Required on modern macOS because Apple's notification API only accepts requests from registered app bundles.

`make install` (default `PREFIX=~/.local/bin`) installs both side-by-side so the daemon can find the helper at runtime.

## First run

```sh
./motivation -config ./quotes.yaml -addr 127.0.0.1:8765
```

Then open <http://127.0.0.1:8765> in your browser.

The first time a notification fires, macOS will pop **"MotivationNotify wants to send you notifications"** — click **Allow**. If you miss the prompt, enable it manually under **System Settings → Notifications → MotivationNotify** (set alert style to Banners or Alerts).

Trigger a test notification from another shell to force the prompt:

```sh
curl -X POST http://127.0.0.1:8765/api/test
```

## Usage

CLI flags:

| Flag      | Default                | Purpose                                |
|-----------|------------------------|----------------------------------------|
| `-config` | `./quotes.yaml`        | Path to the quote library              |
| `-addr`   | `127.0.0.1:8765`       | HTTP listen address for the dashboard  |

HTTP endpoints:

| Method | Path              | Effect                                            |
|--------|-------------------|---------------------------------------------------|
| GET    | `/`               | Dashboard                                         |
| GET    | `/api/state`      | JSON snapshot of quotes + next run times          |
| POST   | `/api/reload`     | Re-read `quotes.yaml` and rebuild the cron        |
| POST   | `/api/test`       | Fire a test notification through the helper       |
| POST   | `/api/fire/{id}`  | Fire a specific quote by `id`                     |

Edit `quotes.yaml` while the daemon is running, then either click **Reload** in the dashboard or:

```sh
curl -X POST http://127.0.0.1:8765/api/reload
```

### Quote schema

```yaml
defaults:
  timezone: ""          # IANA tz, empty = system local

general_schedules:      # pick a random matching quote at each tick
  - cron: "0 9,13,17 * * *"
    category: focus

quotes:
  - id: marcus-mind
    text: "You have power over your mind — not outside events."
    author: Marcus Aurelius
    categories: [stoic, focus]
    media:              # first URL is what clicking the notification opens
      - https://en.wikipedia.org/wiki/Meditations
    schedules:
      - "0 8 * * MON"   # standard 5-field cron
```

## Make targets

| Target           | Purpose                                                                  |
|------------------|--------------------------------------------------------------------------|
| `make build`     | Build the Swift helper `.app` + the Go binary                            |
| `make helper`    | Build only the Swift helper                                              |
| `make run`       | Build, then run the daemon (`CONFIG=...` / `ADDR=...` overridable)       |
| `make install`   | Install binary + helper to `$(PREFIX)` (default `~/.local/bin`)          |
| `make fmt`       | `go fmt ./...`                                                           |
| `make vet`       | `go vet ./...`                                                           |
| `make tidy`      | `go mod tidy`                                                            |
| `make clean`     | Remove `./motivation` and `./bin/`                                       |

## Helper override

The daemon searches for `MotivationNotify` in this order:

1. `$MOTIVATION_HELPER`
2. `./bin/MotivationNotify.app/Contents/MacOS/MotivationNotify`
3. Alongside the running `motivation` binary
4. `~/.local/bin/MotivationNotify.app/Contents/MacOS/MotivationNotify`

If none are found, the daemon falls back to `osascript display notification` (no click-to-open) and logs a one-shot warning.

## Troubleshooting

- **No notifications appear at all.** Open **System Settings → Notifications → MotivationNotify**. If it's missing, the consent prompt was dismissed or never appeared — run the helper directly once to force it:
  ```sh
  open -a "$(pwd)/bin/MotivationNotify.app" --args -title test -message hi
  ```
- **Notification appears but clicking does nothing.** Confirm the quote actually has a `media:` entry in `quotes.yaml`. The helper waits up to `-timeout` seconds (default 30) for the click; clicks after that are ignored.
- **"swiftc: command not found".** Install Xcode Command Line Tools: `xcode-select --install`.

## License

MIT (or whatever you prefer — set it here).
