# Code Map

Local Go daemon that fires curated motivational quotes as macOS notifications on per-quote and general cron schedules. Single binary, embedded HTML dashboard, YAML-driven library.

## Files

| File | Purpose |
|---|---|
| `main.go` | Entry point. Flag parsing (`-config`, `-addr`), wires `Notifier` + `Scheduler` + `Server`, runs HTTP listener, handles `SIGINT`/`SIGTERM` for clean shutdown. |
| `config.go` | YAML schema types (`Config`, `Defaults`, `GeneralSchedule`, `Quote`) and `LoadConfig(path)`. `Quote.Media` is `[]string` (list of picture/YouTube URLs). Validates unique non-empty quote IDs and non-empty text. `Quote.HasCategory(cat)` helper. |
| `scheduler.go` | `Scheduler` wraps `robfig/cron/v3`. `Reload()` (atomic swap behind RWMutex), `FireByID(id)`, `Snapshot()` for UI state, `fireRandom(category)` for general schedules. Each registered entry tracks its `cron.EntryID` for accurate next-run lookups. `fireQuote` passes `Media[0]` as the notification's click target. |
| `notifier.go` | `Notifier.Notify(title, body, openURL)`. Detects `terminal-notifier` once via `exec.LookPath`; falls back to `osascript` (no click) and warns once. Safe arg passing — never shells out with concatenated strings. AppleScript fallback uses `quoteAS` to escape. |
| `handlers.go` | `Server` + routes. Embeds `web/index.html` via `embed.FS`. Endpoints: `GET /`, `GET /api/state`, `POST /api/reload`, `POST /api/test`, `POST /api/fire/{id}`. |
| `web/index.html` | Single-file dashboard. Vanilla JS, dark theme. Lists quotes + general schedules with next-run times; buttons for Fire/Reload/Test. Polls `/api/state` every 30s. |
| `quotes.yaml` | Default quote library. Contains `defaults.timezone`, `general_schedules[]`, and `quotes[]`. Cron cheat sheet at top. |
| `Makefile` | Targets: `build`, `run` (CONFIG/ADDR overridable), `tidy`, `fmt`, `vet`, `clean`, `install` (PREFIX overridable, default `~/.local/bin`), `install-notifier`. `make` with no args → `help`. |
| `go.mod` / `go.sum` | Deps: `github.com/robfig/cron/v3`, `gopkg.in/yaml.v3`. Go 1.26.1. |
| `.claude/hooks/codemap-reminder.sh` | PostToolUse hook script — reminds Claude to refresh CODEMAP.md and ARC.md after source-file edits. |

## Module relationships

```
main ──▶ Notifier  (probes terminal-notifier on first Notify)
   │
   ├──▶ Scheduler ──▶ robfig/cron/v3
   │       │
   │       ├──▶ config.LoadConfig ──▶ yaml.v3
   │       │
   │       └──▶ Notifier.Notify (on every fire)
   │
   └──▶ Server
           ├──▶ embed web/index.html
           ├──▶ Scheduler.Snapshot / Reload / FireByID
           └──▶ Notifier.Notify (only for /api/test)
```

## Public API surface

- HTTP: `GET /`, `GET /api/state`, `POST /api/reload`, `POST /api/test`, `POST /api/fire/{id}`
- CLI flags: `-config` (default `./quotes.yaml`), `-addr` (default `127.0.0.1:8765`)
- YAML schema documented in `quotes.yaml` header

## Concurrency

- HTTP server runs in a goroutine; main blocks on signal channel.
- `Scheduler.mu` (RWMutex) guards `cfg`, `cron`, `loc`, `entries`. `Reload()` builds a new cron under the lock, swaps atomically, then stops the old cron outside the lock.
- `Scheduler.FireByID` dispatches the actual notification in a goroutine so the HTTP handler returns immediately.
- `Notifier.once` ensures `terminal-notifier` is probed exactly once.
