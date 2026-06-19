# 项目结构

## 目录总览

```
vshell/
├── main.go                     # 入口：窗口配置、原生菜单、//go:embed 前端资源
├── build/
│   └── config.yml             # Wails 3 构建配置（非 wails.json）
├── internal/                   # Go 后端（私有包）
│   ├── app/
│   │   ├── app.go             # AppService — Wails 服务层，所有绑定方法
│   │   ├── sshconfig.go       # SSH config 文件导入/导出
│   │   └── icons/             # 菜单图标（//go:embed）
│   ├── ssh/
│   │   ├── client.go          # Manager — SSH 客户端池、连接管理
│   │   ├── session.go         # Session — PTY 会话、flushingWriter（50ms 缓冲）
│   │   └── monitor.go         # Monitor — 服务器资源监控
│   ├── sftp/
│   │   ├── manager.go         # SFTP 管理器、并发传输池（最多 3）
│   │   └── client.go          # SFTP 客户端操作
│   ├── portforward/
│   │   └── forward.go         # 本地端口转发
│   ├── db/
│   │   ├── db.go              # SQLite 初始化（单连接、WAL、foreign keys）
│   │   └── migrations.go      # 内联 SQL 迁移（migrations + additive 切片）
│   ├── crypto/
│   │   └── crypto.go          # AES-256-GCM 加解密服务
│   └── models/                # 数据模型
│       ├── connection.go      # Connection / ConnectionForm / AuthType
│       ├── group.go           # Group
│       ├── quick_command.go   # QuickCommand
│       └── port_forward.go    # PortForward / ForwardType
├── frontend/
│   ├── bindings/              # 【自动生成，勿手动编辑】
│   │   └── vshell/internal/
│   │       ├── app/appservice.ts    # 所有绑定函数
│   │       └── models/models.ts     # 类型定义
│   ├── src/
│   │   ├── App.vue            # 根组件：布局 + 主题 + 菜单事件
│   │   ├── main.ts            # Pinia + i18n + Monaco Workers
│   │   ├── components/        # 按领域组织的组件
│   │   │   ├── sidebar/       # 连接树、表单、导航
│   │   │   ├── terminal/      # 终端、编辑器标签、会话管理
│   │   │   ├── sftp/          # 文件传输操作
│   │   │   ├── monitor/       # 系统监控 & 图表
│   │   │   ├── settings/      # 应用设置
│   │   │   ├── keys/          # SSH 密钥管理
│   │   │   ├── config/        # SSH 配置管理
│   │   │   ├── activity/      # 活动栏
│   │   │   ├── panels/        # 底部面板、端口转发面板
│   │   │   └── common/        # 共享组件（可拖拽分隔器）
│   │   ├── stores/            # Pinia 状态仓库（Composition Store 模式）
│   │   │   ├── connection.ts  # 连接 CRUD、连接/断开
│   │   │   ├── terminal.ts    # 终端会话、标签页
│   │   │   ├── sftp.ts        # SFTP 操作
│   │   │   ├── monitor.ts     # 监控数据
│   │   │   ├── settings.ts    # UI 偏好、主题、语言
│   │   │   ├── layout.ts      # 布局尺寸（sidebarWidth、bottomPanelHeight）
│   │   │   ├── transfers.ts   # 文件传输队列
│   │   │   ├── sshkey.ts      # SSH 密钥
│   │   │   └── sshconfig.ts   # SSH 配置
│   │   ├── composables/       # Vue 3 组合式函数
│   │   │   ├── useTerminal.ts       # 终端初始化与事件
│   │   │   ├── useTerminalManager.ts
│   │   │   ├── useEvents.ts        # Wails 事件辅助
│   │   │   ├── useShortcuts.ts     # 快捷键
│   │   │   └── useDragTransfer.ts  # 拖拽传输
│   │   ├── locales/           # 国际化（zh-CN.ts + en.ts）
│   │   ├── styles/
│   │   │   └── global.css     # CSS 变量、双主题（dark/light）、全局重置
│   │   ├── types/             # TypeScript 类型定义
│   │   └── utils/             # 工具函数
│   ├── uno.config.ts          # UnoCSS 配置（presetUno + 自定义主题映射）
│   ├── vite.config.ts         # Vite 配置
│   └── tsconfig.json          # TypeScript 配置
└── doc/                       # 文档
```

## 技术栈版本

| 层级 | 技术 | 版本/说明 |
|------|------|-----------|
| 框架 | Wails 3 | v3.0.0-alpha.92 |
| 后端 | Go | 1.25.0 |
| 前端框架 | Vue 3 | Composition API + `<script setup lang="ts">` |
| UI 库 | Naive UI | **唯一允许** |
| CSS | UnoCSS | presetUno + 自定义主题 |
| 状态管理 | Pinia | Composition Store 模式 |
| 国际化 | vue-i18n v9 | Composition API |
| 终端 | xterm.js | @xterm/xterm v6（fit/search/serialize/webgl） |
| 编辑器 | Monaco Editor | 远程文件编辑 |
| 图表 | ECharts | 服务器资源监控 |
| 图标 | Iconify + Lucide | unplugin-icons（`~icons/lucide/xxx`） |
| 数据库 | SQLite | modernc.org/sqlite（纯 Go，无 CGO） |
| 加密 | AES-256-GCM | 敏感数据存储前加密 |
| 构建 | Vite | 默认端口 9245 |

## 命名约定

| 类型 | 约定 | 示例 |
|------|------|------|
| Go 文件 | snake_case | `ssh_config.go`, `client.go` |
| Go 导出方法 | PascalCase | `ConnectSSH`, `ListConnections` |
| Go 私有方法 | camelCase | `buildSSHConfig`, `getConnectionByID` |
| Go 常量 | PascalCase / UpperCase | `AuthPassword`, `flushInterval` |
| Vue 组件文件 | PascalCase | `TerminalPane.vue`, `ConnectionTree.vue` |
| Store 文件 | kebab-case | `connection.ts`, `ssh-key.ts` |
| Composable 文件 | camelCase | `useTerminal.ts`, `useShortcuts.ts` |
| CSS 类 | kebab-case | `hover-overlay`, `panel-bg` |
| CSS 变量 | `--category-variant` | `--bg-primary`, `--text-secondary` |
| 事件名 | `domain:action` | `terminal:stdin`, `sftp:upload` |
| i18n key | `domain.key` | `connections.title`, `common.confirm` |

## 关键配置文件

### build/config.yml

Wails 3 构建配置，包含：
- `info` — 产品名称、标识符、版本、版权
- `dev_mode` — 开发模式配置（监听文件、忽略目录、执行命令）
- `fileAssociations` — 文件关联（可选）

### main.go

应用入口，负责：
- `//go:embed all:frontend/dist` 嵌入前端构建产物
- `//go:embed internal/app/icons/` 嵌入菜单图标
- `application.New()` 创建应用并注册 Service
- 创建原生菜单（vShell / File / Edit / Window）
- 创建窗口（标题、尺寸、macOS 标题栏配置）
- 注册窗口事件（文件拖放）
