---
name: wails-dev
description: "Wails v3 + Vue 3 全栈桌面应用开发技能。覆盖 vShell 项目（SSH 客户端管理工具）的完整开发流程：Wails 3 服务绑定、双通道通信（同步调用 + 事件系统）、Go 后端 AppService 模式、Vue 3 Composition API + Naive UI + UnoCSS 前端、SQLite 数据库迁移、AES-256-GCM 加密、xterm.js 终端集成。前后端通过 Wails 绑定和事件系统无缝集成，采用桌面原生应用架构而非 SPA。"
---

# Wails v3 Desktop Dev Skill

为 **vShell 项目**提供 Wails 3 桌面应用全栈开发能力。后端 Go 通过 AppService 模式暴露绑定方法，前端 Vue 3 + Naive UI + UnoCSS 构建界面，终端 I/O 通过事件系统实现实时通信，SQLite + AES-256-GCM 处理数据持久化与安全。

**框架**: Wails 3 (v3.0.0-alpha.92)
**后端**: Go 1.25
**前端**: Vue 3 + TypeScript (Composition API + `<script setup>`)
**数据库**: SQLite (modernc.org/sqlite, 纯 Go 无 CGO)
**许可证**: 本地桌面应用，无云/服务端组件

**前端技术栈版本：**
- Naive UI（唯一允许的 UI 库）
- UnoCSS (presetUno + 自定义主题)
- Pinia (Composition Store)
- vue-i18n v9 (zh-CN / en)
- xterm.js (@xterm/xterm v6)
- Monaco Editor（远程文件编辑）
- ECharts（服务器资源监控）
- Iconify + Lucide (unplugin-icons)

## Critical Rules

1. **Terminal I/O 必须使用 Wails Events.** 终端 stdin/stdout/resize 数据必须通过 `Events.Emit` / `Events.On` 传输，禁止使用同步 `Call/Bind`，否则会冻结 UI。
2. **Naive UI 唯一允许.** 禁止使用 Element Plus、Ant Design 或其他 UI 库。禁止使用浏览器原生 `alert()`/`confirm()`/`prompt()`，必须使用 Naive UI 的 `useDialog()`/`useMessage()`。
3. **敏感数据必须加密.** 密码、私钥、密码短语必须通过 AES-256-GCM (`Crypto().Encrypt()`) 加密后存入数据库。模型中使用 `json:"-"` 阻止泄露到前端。
4. **纯 Go，无 CGO.** 必须使用 `modernc.org/sqlite`，禁止使用 `mattn/go-sqlite3` 或任何需要 CGO 的依赖。
5. **bindings/ 目录自动生成.** 禁止手动编辑 `frontend/bindings/` 下的文件。修改 Go 方法后必须运行 `wails3 generate bindings`。
6. **Wails 绑定返回类实例需转换.** 前端接收到的绑定结果是类实例，必须通过 `.map()` 转为普通对象以保持 Vue 响应性。
7. **事件监听必须清理.** 在 `onUnmounted` 中必须调用 `Events.Off()` 清理事件监听，否则会内存泄漏。
8. **SSH 终端必须用 PTY.** 必须使用 `RequestPty("xterm-256color", ...) + Shell()`，禁止 exec 单命令作为交互式终端。
9. **SQL 参数化查询.** 所有 SQL 必须使用 `?` 占位符，禁止字符串拼接 SQL。
10. **无云功能.** 这是纯本地桌面应用，禁止添加云同步、服务端代码、注册/登录等功能。
11. **Wails 3 配置是 build/config.yml.** 不是 `wails.json`，不是 `wails.v2.json`。
12. **Composition Store 模式.** Pinia Store 必须使用 Composition API 风格（`defineStore('name', () => { ... })`），禁止 Options API 风格。
13. **组件使用 `<script setup lang="ts">`.** 禁止 Options API 组件和 `defineComponent()` 写法。
14. **中文支持.** 当用户使用中文沟通时，所有回复和代码注释使用中文。

## Scene Navigation

> 根据用户场景，读取对应的 reference 文件获取详细信息。

### 项目与架构

| Scenario | Reference File | Grep Keywords |
|---|---|---|
| 项目结构 / 目录说明 / 文件组织 / 命名约定 / 技术栈版本 | `references/project_structure.md` | `internal/`, `frontend/`, `bindings/`, `models/`, `stores/` |
| 双通道通信 / 服务绑定 / 事件系统 / AppService 生命周期 / 事件命名约定 | `references/architecture.md` | `Events.Emit`, `Events.On`, `NewService`, `ServiceStartup`, `terminal:stdin` |

### 后端开发

| Scenario | Reference File | Grep Keywords |
|---|---|---|
| 添加绑定方法 / 数据库迁移 / 模型定义 / 敏感数据加密 / SSH 管理 / 事件注册 / SQLite CRUD | `references/backend_patterns.md` | `func (a *AppService)`, `migrations`, `Crypto().Encrypt`, `sshManager`, `json:"-"` |

### 前端开发

| Scenario | Reference File | Grep Keywords |
|---|---|---|
| 组件规范 / Naive UI 规则 / Pinia Store / 终端集成 / 国际化 / 图标使用 | `references/frontend_patterns.md` | `<script setup`, `useDialog`, `defineStore`, `useTerminal`, `useI18n`, `~icons/lucide` |

### 样式与布局

| Scenario | Reference File | Grep Keywords |
|---|---|---|
| CSS 变量主题 / UnoCSS 配置 / Naive UI 主题桥接 / 暗色/亮色切换 | `references/theme_system.md` | `data-theme`, `--bg-primary`, `uno.config.ts`, `naiveThemeOverrides` |
| 主布局结构 / ActivityBar / Sidebar / Terminal / Bottom Panel / 可拖拽分隔器 / 尺寸管理 | `references/layout_system.md` | `DraggableDivider`, `sidebarWidth`, `layout.ts`, `min-height: 0` |

### 开发参考

| Scenario | Reference File | Grep Keywords |
|---|---|---|
| 开发命令 / 构建配置 / 绑定生成 / Wails Runtime API / 常见陷阱排错 | `references/dev_commands.md` | `wails3 dev`, `wails3 build`, `generate bindings`, `Events`, `Window`, `Dialogs` |

## 开发工作流

典型的 vShell 功能开发流程：

```
1. 后端模型与数据库
   ├── 在 internal/models/ 创建模型（json tag，敏感字段 json:"-"）
   ├── 在 internal/db/migrations.go 添加 CREATE TABLE
   └── 增量迁移放入 additive 切片（ALTER TABLE，忽略错误）

2. 后端服务方法
   ├── 在 internal/app/app.go 添加导出方法（首字母大写）
   ├── 敏感字段使用 Crypto().Encrypt() 加密
   └── 运行 wails3 generate bindings 生成前端绑定

3. 前端 Store
   ├── 在 frontend/src/stores/ 创建 Composition Store
   ├── 调用绑定方法，map() 转换类实例为普通对象
   └── try/finally 管理 loading 状态

4. 前端组件
   ├── 在 components/ 对应领域目录创建 .vue 文件
   ├── 使用 <script setup lang="ts"> + Naive UI 组件
   ├── 使用 UnoCSS 工具类（bg-[var(--xxx)]）
   └── 国际化：t('key')，同步更新 zh-CN.ts 和 en.ts

5. 集成到布局
   ├── 在 App.vue 或对应面板引入组件
   ├── 如需事件通信：Events.Emit/On + onUnmounted 清理
   └── wails3 dev 验证
```

只需 `wails3 dev` 即可启动完整开发环境（Go 后端 + 前端热重载）。
