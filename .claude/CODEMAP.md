# Code Map

Local Go daemon that fires curated motivational quotes as macOS notifications on per-quote and general cron schedules. Single binary, embedded HTML dashboard, YAML-driven library.

## Files

| File | Purpose |
|---|---|
| `main.go` | Entry point. Flag parsing (`-config`, `-addr`), wires `Notifier` + `Scheduler` + `Server`, runs HTTP listener, handles `SIGINT`/`SIGTERM` for clean shutdown. |
| `config.go` | YAML schema types (`Config`, `Defaults`, `GeneralSchedule`, `Quote`) and `LoadConfig(path)`. `Quote.Media` is `[]string` (list of picture/YouTube URLs). Validates unique non-empty quote IDs and non-empty text. `Quote.HasCategory(cat)` helper. |
| `scheduler.go` | `Scheduler` wraps `robfig/cron/v3`. `Reload()` (atomic swap behind RWMutex), `FireByID(id)`, `Snapshot()` for UI state, `fireRandom(category)` for general schedules. Each registered entry tracks its `cron.EntryID` for accurate next-run lookups. `fireQuote` passes `Media[0]` as the click target and `RandomEmoji()` as the icon. |
| `emojis.go` | Curated `motivationalEmojis` slice (~40 entries) and `RandomEmoji()` helper using `math/rand/v2`. Used by `scheduler.fireQuote` and `handlers.handleTest`; the chosen emoji is rendered to a PNG by the Swift helper and shown as the notification's icon. |
| `notifier.go` | `Notifier.Notify(title, body, openURL, emoji)`. Locates the `MotivationNotify` Swift helper inside `MotivationNotify.app` (search order: `$MOTIVATION_HELPER`, `./bin/...`, alongside the Go binary, `~/.local/bin/...`); falls back to `osascript` (no click, no icon) when missing. Helper is started detached (`cmd.Start()`) so it can outlive the call and wait for a click. Safe arg passing — never shells out with concatenated strings. AppleScript fallback uses `quoteAS` to escape. |
| `notify-helper/main.swift` | Tiny Swift CLI compiled into a `.app` bundle. Uses `UNUserNotificationCenter` to post one notification, then runs the main `RunLoop` waiting for the user to click (or `-timeout` seconds — default 30 — to elapse). On click, opens the `-open` URL via `NSWorkspace.shared.open` and exits. `-emoji E` renders E to a 512×512 transparent PNG via `NSAttributedString.draw` + `NSBitmapImageRep`, then does three things: (1) overwrites `Contents/Resources/AppIcon.png` and touches the bundle so LaunchServices/Notification Center re-reads the app icon, (2) sets `NSApp.applicationIconImage` on the running process as a backup, (3) attaches the PNG as a `UNNotificationAttachment` (right-side thumbnail). Required because macOS 13+/Tahoe silently drops notifications from unregistered/unsigned senders like terminal-notifier. |
| `notify-helper/Info.plist` | Bundle metadata for the helper: `CFBundleIdentifier=com.motivation.notifier`, `CFBundleIconFile=AppIcon` (resolved per-fire from `Contents/Resources/AppIcon.png`), `LSUIElement=true` (no Dock icon). Required for `UNUserNotificationCenter` to recognise the binary as a real app. |
| `handlers.go` | `Server` + routes. Embeds `web/index.html` via `embed.FS`. Endpoints: `GET /`, `GET /api/state`, `POST /api/reload`, `POST /api/test`, `POST /api/fire/{id}`. |
| `web/index.html` | Single-file dashboard. Vanilla JS, dark theme. Lists quotes + general schedules with next-run times; buttons for Fire/Reload/Test. Polls `/api/state` every 30s. |
| `quotes.yaml` | Default quote library. Contains `defaults.timezone`, `general_schedules[]`, and `quotes[]`. Cron cheat sheet at top. |
| `Makefile` | Targets: `build` (compiles Swift helper into `bin/MotivationNotify.app` via `swiftc` + ad-hoc `codesign`, then `go build`), `helper`, `run` (CONFIG/ADDR overridable), `tidy`, `fmt`, `vet`, `clean` (also removes `bin/`), `install` (installs both binary and `.app` bundle, PREFIX overridable, default `~/.local/bin`), `uninstall` (removes installed binary + `.app` and restarts `usernoted`/`NotificationCenter` to flush cached icon for `com.motivation.notifier`). `make` with no args → `help`. |
| `go.mod` / `go.sum` | Deps: `github.com/robfig/cron/v3`, `gopkg.in/yaml.v3`. Go 1.26.1. |
| `.claude/hooks/codemap-reminder.sh` | PostToolUse hook script — reminds Claude to refresh CODEMAP.md and ARC.md after source-file edits. |

## Module relationships

```
main ──▶ Notifier  (locates MotivationNotify.app on first Notify)
   │         │
   │         └─▶ exec MotivationNotify (detached) ──▶ UNUserNotificationCenter
   │                                                    │
   │                                                    └─▶ NSWorkspace.open(URL) on click
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
- `Notifier.once` ensures the helper path is resolved exactly once.
- Each fire spawns a detached helper process that owns its own `RunLoop` and exits on click or `-timeout`. Multiple notifications in flight run as independent helper processes.
