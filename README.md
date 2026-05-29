# vShell

A desktop SSH client management tool built with **Wails 3** (Go + Vue 3). A purely local application — no cloud, no accounts, no telemetry.

[中文文档](README.zh-CN.md)

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

```
vshell/
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
└── doc/                     # Architecture documentation
```

## Building

### Prerequisites

- [Go](https://go.dev/dl/) 1.25+
- [Node.js](https://nodejs.org/) 18+
- [Wails 3 CLI](https://v3.wails.io/): `go install github.com/wailsapp/wails/v3/cmd/wails3@latest`

### macOS

```bash
# Quick build (binary only)
wails3 build
# Output: bin/vshell

# Build as .app bundle
./scripts/build_macos.sh
# Output: bin/vshell.app

# Open the app
open bin/vshell.app
```

### Windows

```bash
# Quick build (binary only, default amd64)
wails3 task windows:build
# Output: bin/vshell.exe

# Build for specific architecture
wails3 task windows:build ARCH=arm64

# Build with NSIS installer
wails3 task windows:package FORMAT=nsis

# Or use the build script
./scripts/build_windows.sh
./scripts/build_windows.sh ARCH=arm64 PACKAGE=true FORMAT=nsis
```

### Cross-compilation

To cross-compile from macOS for Windows (requires appropriate C cross-compiler toolchain):

```bash
# Build Windows binary on macOS
GOOS=windows GOARCH=amd64 wails3 task windows:build
```

## Database

SQLite database stored at `~/Library/Application Support/vshell/vshell.db` (macOS). All sensitive data (passwords, private keys, passphrases) is encrypted with AES-256-GCM before storage.

> **Note**: This is a personal tool primarily used and tested on macOS. Windows builds are available but untested.

## Contact & Sponsor

<table>
  <tr>
    <td align="center">
      <img src="doc/images/wechat-qr.jpg" alt="WeChat" width="200"/><br/>
      <b>WeChat</b>
    </td>
    <td align="center">
      <img src="doc/images/sponsor-qr.jpg" alt="Sponsor" width="200"/><br/>
      <b>Buy me a coffee</b>
    </td>
  </tr>
</table>
