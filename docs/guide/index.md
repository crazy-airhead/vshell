# 使用指南总览

vShell 是一个**纯本地**的桌面 SSH 客户端管理工具（类似 FinalShell），基于 Wails 3（Go 后端 + Vue 3 前端）构建。没有云端组件、没有账号体系、没有遥测——你的所有数据都保存在本机，敏感信息经 AES-256-GCM 加密后存储。

## 功能矩阵

| 功能 | 说明 | 文档 |
|------|------|------|
| 连接管理 | 分组树形组织、拖拽调整归属、密码 / 私钥 / 交互式认证 | [连接与分组管理](connections.md) |
| SSH 密钥 | 生成、导入、管理密钥对 | [SSH 密钥管理](keys.md) |
| ~/.ssh/config | 结构化编辑、导入为连接 | [导入导出](ssh-config.md) |
| 终端 | xterm.js PTY 终端、复制粘贴、断线重连 | [终端使用](terminal.md) |
| 远程编辑 | Monaco Editor 编辑远程文件 | [远程文件编辑](editor.md) |
| SFTP | 浏览、传输、拖拽、进度跟踪 | [SFTP 文件管理](sftp.md) |
| 服务器监控 | CPU / 内存 / 磁盘 / 进程 / 网络实时图表 | [服务器监控](monitor.md) |
| 端口转发 | 本地端口转发、服务预设、自动启动 | [端口转发](port-forwarding.md) |
| 个性化 | 中英双语、主题、终端配色等 | [设置与主题](settings.md) |

## 数据安全

- 所有数据存储在本地 SQLite 数据库（macOS 位于 `~/Library/Application Support/vshell/vshell.db`）
- 密码、私钥、口令等敏感字段使用 AES-256-GCM 加密后落库，绝不存明文
- 不联网上报任何信息

## 环境要求

| 项目 | 要求 |
|------|------|
| 操作系统 | macOS（Windows / Linux 支持随 Wails 3 跨平台能力提供） |
| Go | 1.25+ |
| Node.js | 20+ |
| 包管理 | pnpm（前端与文档站均使用） |

## 反馈与问题追踪

开发过程中的问题记录见仓库 `docs/issues/` 目录（含模板 `_TEMPLATE.md`），可在 [GitHub Issues](https://github.com/crazy-airhead/vshell/issues) 提交新问题。
