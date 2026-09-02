# 快速开始

本章介绍如何从源码构建并运行 vShell，以及首次使用的最小流程：新建连接 → 打开终端 → 传输文件。

---

## 1. 获取与构建

vShell 目前从源码构建。克隆仓库后，准备好以下工具链：

| 工具 | 版本 | 安装 |
|------|------|------|
| Go | 1.25+ | [go.dev/dl](https://go.dev/dl/) |
| Node.js | 20+ | [nodejs.org](https://nodejs.org/)（或 nvm） |
| pnpm | 9+ | `npm i -g pnpm` 或 `corepack enable` |
| Wails 3 CLI | 与项目同版本 | `go install github.com/wailsapp/wails/v3/cmd/wails3@latest` |

> **版本对齐**：Wails 3 处于 beta 阶段，Go module、`@wailsio/runtime` 与 `wails3` CLI 需保持同一版本，否则可能出现绑定不匹配。

### 开发模式（热重载）

```bash
git clone https://github.com/crazy-airhead/vshell.git
cd vshell
wails3 dev        # 前后端热重载，Vite 默认端口 9245（WAILS_VITE_PORT 可改）
```

### 生产构建

```bash
wails3 build      # 产出可执行文件（bin/ 目录）
```

### 仅前端开发

```bash
cd frontend
pnpm install
pnpm dev          # Vite dev server，端口 9245
```

### 仅 Go 后端

```bash
go build .        # 编译后端
go test ./...     # 运行测试
```

---

## 2. 首次使用

1. **创建连接**：点击侧栏的连接管理，新建连接，填写主机、端口、用户名，选择认证方式（密码或私钥），详见[连接与分组管理](connections.md)
2. **打开终端**：双击连接即可打开 PTY 终端（`xterm-256color`），详见[终端使用](terminal.md)
3. **传输文件**：打开 SFTP 面板浏览远程目录，支持拖拽传输，详见 [SFTP 文件管理](sftp.md)

---

## 3. 数据存储位置

| 平台 | 数据库路径 |
|------|-----------|
| macOS | `~/Library/Application Support/vshell/vshell.db` |

数据库包含连接、分组、快捷命令、端口转发配置。密码与私钥均以 AES-256-GCM 加密存储，详见[数据存储与加密](../dev/storage-crypto.md)。

---

## 4. 下一步

- 按功能模块阅读[使用指南总览](./)
- 了解内部实现请看[开发文档](../dev/)
