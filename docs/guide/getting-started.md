# 快速开始

本章介绍如何下载安装 vShell，以及首次使用的最小流程：新建连接 → 打开终端 → 传输文件。想自己编译的看最后一节「从源码构建」。

---

## 1. 下载安装

从 [GitHub Releases](https://github.com/crazy-airhead/vshell/releases/latest) 下载对应平台的最新版本（自 v0.2.0 起提供三平台安装包）：

| 平台 | 文件 | 说明 |
|------|------|------|
| macOS | `vShell-<版本>-macos-universal.zip` | 通用二进制（Apple Silicon / Intel），解压后将 vShell.app 拖入「应用程序」 |
| Windows | `vShell-<版本>-windows-amd64-installer.exe` | 安装包（NSIS） |
| Windows | `vShell-<版本>-windows-amd64-portable.exe` | 便携版，免安装直接运行 |
| Linux | `vShell-<版本>-linux-amd64.deb` / `.rpm` | 用发行版包管理器安装 |
| Linux | `vShell-<版本>-linux-amd64.tar.gz` | 便携版，解压即用 |

> **macOS 未公证提示**：安装包未做 Apple 公证，首次打开若提示无法验证开发者，**右键 App →「打开」**，或执行 `xattr -d com.apple.quarantine /Applications/vshell.app` 后再打开。

## 2. 首次使用

1. **创建连接**：点击左条的「连接」图标，新建连接，填写主机、端口、用户名，选择认证方式（密码或私钥），详见[连接与分组管理](connections.md)
2. **打开终端**：双击连接即可打开 PTY 终端（`xterm-256color`），详见[终端使用](terminal.md)
3. **传输文件**：打开底部工具窗的 SFTP 标签浏览远程目录，支持拖拽传输，详见 [SFTP 文件管理](sftp.md)

## 3. 数据存储位置

| 平台 | 数据库路径 |
|------|-----------|
| macOS | `~/Library/Application Support/vshell/vshell.db` |
| Windows | `%AppData%\vshell\vshell.db` |
| Linux | `~/.config/vshell/vshell.db` |

加密密钥 `.enc_key` 与数据库同目录。数据库包含连接、分组、端口转发、证书任务等配置。密码、私钥、DNS API 凭据等敏感信息均以 AES-256-GCM 加密存储，详见[数据存储与加密](../dev/storage-crypto.md)。

---

## 4. 从源码构建（进阶）

克隆仓库后，准备好以下工具链：

| 工具 | 版本 | 安装 |
|------|------|------|
| Go | 1.25+ | [go.dev/dl](https://go.dev/dl/) |
| Node.js | 20+ | [nodejs.org](https://nodejs.org/)（或 nvm） |
| pnpm | 9+ | `npm i -g pnpm` 或 `corepack enable` |
| Wails 3 CLI | 与项目同版本 | `go install github.com/wailsapp/wails/v3/cmd/wails3@latest` |

> **版本对齐**：Wails 3 处于 beta 阶段，Go module、`@wailsio/runtime` 与 `wails3` CLI 需保持同一版本，否则可能出现绑定不匹配。当前项目使用 v3.0.0-beta.28。

### 开发模式（热重载）

```bash
git clone https://github.com/crazy-airhead/vshell.git
cd vshell
wails3 dev        # 前后端热重载，Vite 默认端口 9245（WAILS_VITE_PORT 可改）
```

> 源码位于 `artifacts` 分支（检出为同级 worktree `../vshell-artifacts`），构建命令在制品区执行，详见[构建与开发环境](../dev/development.md)。

### 生产构建

```bash
wails3 build      # 产出可执行文件（bin/ 目录）
```

### 仅前端 / 仅 Go

```bash
cd frontend && pnpm install && pnpm dev   # Vite dev server，端口 9245
go build .                                # 编译后端
go test ./...                             # 运行测试
```

---

## 5. 下一步

- 按功能模块阅读[使用指南总览](./)
- 了解内部实现请看[开发文档](../dev/)
