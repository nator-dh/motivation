# Architecture

A single Go process hosts three collaborating pieces: a **Scheduler** running cron jobs, a **Notifier** that delegates to macOS notification tooling, and an **HTTP Server** that exposes a small embedded dashboard. State lives entirely in memory; the YAML file is the only persistence.

## Component diagram

```mermaid
flowchart LR
    subgraph Process["motivation (single Go binary)"]
        Main[main.go<br/>flags + signals]
        Sched[Scheduler<br/>robfig/cron/v3]
        Notif[Notifier]
        HTTP[HTTP Server<br/>net/http]
        UI[(embedded<br/>web/index.html)]
    end

    YAML[(quotes.yaml)]
    Browser[Browser dashboard]
    TN[terminal-notifier]
    OSA[osascript]
    NC[macOS Notification Center]

    Main --> Sched
    Main --> Notif
    Main --> HTTP

    HTTP --- UI
    HTTP <-->|JSON| Browser

    Sched -->|LoadConfig| YAML
    Sched -->|Notify| Notif
    HTTP -->|Snapshot, Reload,<br/>FireByID| Sched
    HTTP -->|/api/test| Notif

    Notif -->|preferred| TN
    Notif -->|fallback| OSA
    TN --> NC
    OSA --> NC
```

## Fire-a-quote sequence

```mermaid
sequenceDiagram
    autonumber
    participant Cron as robfig/cron
    participant Sched as Scheduler
    participant Notif as Notifier
    participant TN as terminal-notifier
    participant NC as Notification Center
    participant User

    Cron->>Sched: tick (quote or general)
    alt per-quote schedule
        Sched->>Sched: fireQuote(q)
    else general schedule
        Sched->>Sched: pick random quote in category
        Sched->>Sched: fireQuote(q)
    end
    Sched->>Notif: Notify(author, text, media)
    Notif->>TN: exec terminal-notifier -open <media>
    TN->>NC: post toast
    NC-->>User: notification
    User->>NC: click
    NC->>TN: open <media> in default browser
```

## Reload sequence

```mermaid
sequenceDiagram
    participant Browser
    participant HTTP
    participant Sched as Scheduler
    participant Cron as robfig/cron

    Browser->>HTTP: POST /api/reload
    HTTP->>Sched: Reload()
    Sched->>Sched: LoadConfig(path)
    Sched->>Cron: new cron, register all entries
    Sched->>Cron: Start()
    Note over Sched: atomic swap under mu.Lock()
    Sched->>Cron: old.Stop() (outside lock)
    HTTP-->>Browser: {"status":"reloaded"}
```

## Data flow at startup

```mermaid
flowchart TD
    A[main: parse flags] --> B[NewScheduler<br/>configPath, notifier]
    B --> C[Scheduler.Start → Reload]
    C --> D[LoadConfig parses YAML]
    D --> E[Build cron entries<br/>per-quote + general]
    E --> F[cron.Start]
    F --> G[NewServer wraps Scheduler+Notifier]
    G --> H[http.ListenAndServe :8765]
    H --> I[block on SIGINT/SIGTERM]
    I --> J[httpSrv.Shutdown + Scheduler.Stop]
```

## Design choices

- **In-process scheduler.** Avoids external dependencies; trivially restartable.
- **Atomic Reload.** A new `cron.Cron` is built and swapped under a mutex so in-flight HTTP requests never see a half-loaded state.
- **`cron.EntryID` lookup, not slice index.** `cron.Entries()` returns sorted by next-run, so we store IDs and look up `Next` per entry to keep UI mapping correct.
- **terminal-notifier preferred.** It supports `-open URL` for click-to-open; `osascript display notification` does not. Detection cached via `sync.Once`. When a quote has multiple `media` URLs, the notification opens the first; the dashboard lists all as numbered links.
- **Loopback only.** HTTP binds `127.0.0.1` by default — no auth, single-user laptop tool.
- **YAML is the only state.** Reload re-reads the file; no internal DB.
