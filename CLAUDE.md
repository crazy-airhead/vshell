# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

vShell is a desktop SSH client management tool (similar to FinalShell) built with **Wails 3** (Go backend + Vue 3 frontend). It is a **purely local application** — no cloud/server components.

## Development Commands

This project uses **pnpm** for frontend package management (declared in `frontend/package.json` → `packageManager`). `wails3 dev` / `wails3 build` auto-detect the runner from the lockfile, so keep `pnpm-lock.yaml` (not `package-lock.json`).

```bash
wails3 dev              # Run in development mode with hot-reload
wails3 build            # Build production executable
cd frontend && pnpm run build   # Build frontend only
```

Frontend-only (from `frontend/`):
```bash
pnpm install            # Install dependencies (uses/generates pnpm-lock.yaml)
pnpm dev                # Vite dev server
pnpm build              # vue-tsc + vite production build
pnpm build:dev          # vue-tsc + vite dev build (no minify)
pnpm typecheck          # vue-tsc --noEmit (type check only)
```

Go-only:
```bash
go build .              # Build Go backend
go test ./...           # Run tests
```

## Architecture

### Backend (Go) — `internal/`

- **`app/app.go`** — `AppService` is the Wails service exposing all bound methods to the frontend (CRUD for connections/groups/quick-commands, SSH connect/disconnect, SFTP operations, SSH key management, local filesystem access, file editing). It orchestrates the managers.
- **`app/sshconfig.go`** — SSH config file (`~/.ssh/config`) import/export logic.
- **`app/icons/`** — PNG menu bar icons (save, close) embedded via `//go:embed`.
- **`ssh/client.go`** — `Manager` holds active SSH sessions, handles connect/disconnect, builds `ssh.ClientConfig` with encrypted credentials.
- **`ssh/session.go`** — `Session` wraps an SSH session with PTY. Uses a `flushingWriter` that buffers stdout/stderr and emits events every 50ms to avoid per-byte overhead.
- **`ssh/monitor.go`** — `Monitor` reuses the SSH client to periodically exec commands (`/proc/stat`, `/proc/meminfo`, etc.) for server resource monitoring.
- **`sftp/manager.go`** + **`sftp/client.go`** — SFTP operations with a concurrency pool (3 concurrent transfers). Wraps `io.Reader`/`io.Writer` for progress tracking via events.
- **`portforward/forward.go`** — Local port forwarding via `net.Listen` → `ssh.Client.Dial` → bidirectional `io.Copy`.
- **`zmodem/zmodem.go`** — Zmodem file transfer protocol support (minimal/stub).
- **`db/db.go`** — SQLite via `modernc.org/sqlite` (pure Go, no CGO). Stored at `~/Library/Application Support/vshell/vshell.db` (macOS). Single connection (`SetMaxOpenConns(1)`).
- **`db/migrations.go`** — Inline SQL migrations run on startup. Tables: `groups`, `connections`, `quick_commands`, `port_forwards`.
- **`crypto/crypto.go`** — AES-256-GCM encryption for passwords, private keys, and passphrases stored in the database.
- **`models/`** — Data models: `Connection`, `Group`, `QuickCommand`, `PortForward`.

### Frontend (Vue 3 + TypeScript) — `frontend/src/`

- **UI Library**: Naive UI (strict — never use Element Plus, Ant Design, or browser native dialogs).
- **CSS Framework**: UnoCSS (presetUno + custom theme colors/shortcuts in `uno.config.ts`).
- **State**: Pinia stores in `stores/` — `connection.ts`, `terminal.ts`, `sftp.ts`, `monitor.ts`, `settings.ts`, `layout.ts`, `transfers.ts`, `sshkey.ts`, `sshconfig.ts`.
- **Routing**: Vue Router.
- **i18n**: `vue-i18n` with `locales/zh-CN.ts` and `locales/en.ts`.
- **Terminal**: xterm.js (`@xterm/xterm` v6) with addons: fit, search, serialize, web-links, webgl.
- **Editor**: Monaco Editor for remote file editing (`components/terminal/EditorTab.vue`).
- **Charts**: ECharts for server resource monitoring.
- **Icons**: Iconify with Lucide icon set, loaded via `unplugin-icons`.
- **Components**: Organized by domain — `sidebar/`, `terminal/`, `sftp/`, `monitor/`, `settings/`, `keys/`, `config/`, `activity/`, `panels/`, `common/`.
- **Composables**: `useTerminal.ts`, `useTerminalManager.ts`, `useEvents.ts`, `useShortcuts.ts`, `useDragTransfer.ts`.

### Critical Data Flow: Terminal I/O

Terminal data **must** use Wails Events system, **never** `Call/Bind`:

- **Input**: xterm `onData` → `Events.Emit("terminal:stdin", {sessionID, data})` → Go `session.StdinPipe.Write(data)`
- **Output**: Go goroutine reads stdout → `flushingWriter` buffers and emits → `Events.Emit("terminal:stdout", {sessionID, data})` → frontend `xterm.write(data)`
- **Resize**: xterm `onResize` → `Events.Emit("terminal:resize", {sessionID, rows, cols})` → Go `session.WindowChange(rows, cols)`

### Event-Based Communication Pattern

Backend-to-frontend uses Wails `Event.Emit`. Frontend-to-backend for terminal/streaming uses `Events.Emit`. CRUD operations use direct Wails service method calls (the `application.NewService` binding pattern).

Native menu actions (Settings, Save, Close Tab) are emitted from Go to frontend via Wails Events: `menu:settings`, `menu:save`, `menu:close-tab`.

## Critical Rules

1. **Terminal data**: Always use Wails Events for terminal I/O — never use synchronous `Call/Bind`.
2. **Sensitive data encryption**: Passwords, private keys, and passphrases must be encrypted with AES-256-GCM before database storage. Never store plaintext.
3. **UI components**: Always use Naive UI components — no browser native `alert/confirm`, no other UI libraries.
4. **PTY requirement**: SSH sessions must call `RequestPty("xterm-256color", ...)` then `Shell()`, never exec single commands as interactive terminals.
5. **No cloud features**: This is a purely local desktop app. No server-side code, no cloud sync, no registration/login.
6. **No CGO**: Use `modernc.org/sqlite` (pure Go SQLite driver), not `mattn/go-sqlite3`.

## Key Configuration

- **Wails config**: Wails 3 uses `build/config.yml` (not `wails.json`). App window options are in `main.go` via `application.Options`.
- **Package manager**: pnpm (`frontend/package.json` → `packageManager`). Never use npm/yarn — `wails3 build` selects the frontend runner from the lockfile, so the repo must carry `pnpm-lock.yaml`, not `package-lock.json`.
- **Vite port**: Default `9245`, configurable via `WAILS_VITE_PORT` env var.
- **Go version**: 1.25.0
- **Wails version**: v3.0.0-alpha.92 (alpha API may change)
