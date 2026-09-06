# vShell

A desktop SSH client management tool built with **Wails 3** (Go + Vue 3). A purely local application — no cloud, no accounts, no telemetry.

## Features

- **SSH Connection Management** — Organize connections in a tree hierarchy with groups, colors, and tags
- **Full Terminal Emulation** — xterm.js-based PTY terminal with xterm-256color support
- **SFTP File Management** — Browse, upload, download, delete, and drag-and-drop file transfers with progress tracking
- **Remote File Editing** — Edit remote files directly with Monaco Editor
- **Server Resource Monitoring** — Real-time CPU and memory charts via ECharts
- **SSH Key Management** — Generate, import, and manage SSH key pairs
- **SSH Config Import/Export** — Import connections from `~/.ssh/config`
- **Port Forwarding** — Local port forwarding with auto-start option
- **Quick Commands** — Save and execute frequently used commands
- **i18n** — Chinese and English language support

## Tech Stack

| Layer | Technology |
|-------|-----------|
| Framework | Wails 3 (alpha) |
| Backend | Go 1.25 |
| Frontend | Vue 3 + TypeScript |
| UI Library | Naive UI |
| CSS | UnoCSS |
| Terminal | xterm.js v6 |
| Editor | Monaco Editor |
| Charts | ECharts |
| Database | SQLite (pure Go, no CGO) |
| Encryption | AES-256-GCM |
| Build | Taskfile + Vite |

## Development

> **Workspace / artifacts split**: source code lives on the `artifacts` branch, checked out as the sibling worktree `../vshell-artifacts`. This `main` branch holds only docs and Claude/skills config — run the commands below in the artifacts worktree.

```bash
# Install Wails 3 CLI
go install github.com/wailsapp/wails/v3/cmd/wails3@latest

# Run in development mode (hot-reload for frontend and backend)
wails3 dev

# Build production executable
wails3 build
```

Frontend-only development:

```bash
cd frontend
npm install
npm run dev          # Start Vite dev server on port 9245
```

## Project Structure

The repo is split in two: `main` (this workspace) holds `docs/` and `skills/`; the `artifacts` branch (`../vshell-artifacts` worktree) holds the source:

```
vshell-artifacts/
├── main.go                  # Application entry point, window config, native menu
├── build/
│   └── config.yml           # Wails 3 build configuration
├── internal/
│   ├── app/                 # Wails AppService (all bound methods exposed to frontend)
│   ├── ssh/                 # SSH client, session (PTY), and server monitor
│   ├── sftp/                # SFTP client and transfer manager
│   ├── portforward/         # Local port forwarding
│   ├── zmodem/              # Zmodem protocol support
│   ├── db/                  # SQLite database and migrations
│   ├── crypto/              # AES-256-GCM encryption
│   └── models/              # Data models
├── frontend/
│   └── src/
│       ├── components/      # Vue components by domain
│       ├── stores/          # Pinia state stores
│       ├── composables/     # Reusable composables
│       ├── locales/         # i18n translations (zh-CN, en)
│       ├── styles/          # Global CSS with theme variables
│       ├── types/           # TypeScript type definitions
│       └── utils/           # Utility functions
```

## Database

SQLite database stored at `~/Library/Application Support/vshell/vshell.db` (macOS). All sensitive data (passwords, private keys, passphrases) is encrypted with AES-256-GCM before storage.
