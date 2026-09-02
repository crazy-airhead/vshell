# 开发文档总览

面向参与 vShell 开发的读者：整体架构、关键数据流与存储设计。使用类文档见[使用指南](../guide/)。

## 文档索引

| 文档 | 内容 |
|------|------|
| [架构总览](architecture.md) | Wails 3 前后端分层、服务绑定与事件通信全景、目录结构 |
| [终端 I/O 数据流](terminal-io.md) | 终端输入 / 输出 / 调整大小的事件链路与缓冲机制 |
| [数据存储与加密](storage-crypto.md) | SQLite 表结构、迁移机制、AES-256-GCM 凭证加密 |
| [构建与开发环境](development.md) | 工具链、Taskfile 任务、pnpm 项目隔离、文档站部署 |

## 核心约束（开发红线）

1. **终端数据走 Wails Events**，绝不使用同步 `Call/Bind`（会阻塞每字节的 I/O）
2. **敏感数据加密**：密码、私钥、口令必须 AES-256-GCM 加密后入库，禁止明文
3. **UI 一律 Naive UI**：禁止浏览器原生 `alert/confirm`，禁止混用其他 UI 库
4. **PTY 规范**：SSH 会话必须 `RequestPty("xterm-256color", ...)` + `Shell()`，不得用 exec 模拟交互终端
5. **纯本地**：无任何服务端代码、云同步、注册登录
6. **无 CGO**：SQLite 用 `modernc.org/sqlite`（纯 Go），不用 `mattn/go-sqlite3`

## 开发流程约定

- 问题登记：`docs/issues/`，编号四位零填充递增，模板见 `_TEMPLATE.md`（该目录不发布到文档站）
- 提交信息：中文描述，`fix:` / `feat:` 前缀
- 前端改动需通过 `pnpm typecheck`，后端改动需通过 `go build ./... && go vet ./...`
