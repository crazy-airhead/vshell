# Wails 3 Desktop Development Skill

vShell 项目基于 **Wails 3** (Go 后端 + Vue 3 前端) 构建的桌面 SSH 客户端管理工具。本文档定义了该技术栈的开发范式、架构决策和编码规范。

---

## 1. 技术栈

| 层级 | 技术 | 版本/说明 |
|------|------|-----------|
| 框架 | Wails 3 | v3.0.0-alpha.92 |
| 后端语言 | Go | 1.25.0 |
| 前端框架 | Vue 3 | Composition API + `<script setup>` |
| UI 库 | Naive UI | **唯一允许的 UI 库** |
| CSS | UnoCSS | presetUno + 自定义主题 |
| 状态管理 | Pinia | Composition Store 模式 |
| 国际化 | vue-i18n v9 | Composition API |
| 终端 | xterm.js | @xterm/xterm v6 |
| 编辑器 | Monaco Editor | 远程文件编辑 |
| 图表 | ECharts | 服务器资源监控 |
| 图标 | Iconify + Lucide | unplugin-icons |
| 数据库 | SQLite | modernc.org/sqlite（纯 Go，无 CGO） |
| 加密 | AES-256-GCM | 敏感数据存储前加密 |
| 构建 | Vite | 默认端口 9245 |

---

## 2. 项目结构

```
vshell/
├── main.go                    # 入口：窗口配置、原生菜单、资源嵌入
├── build/
│   └── config.yml             # Wails 3 构建配置（非 wails.json）
├── internal/                  # Go 后端（私有包）
│   ├── app/
│   │   ├── app.go             # AppService：Wails 服务层，所有绑定方法
│   │   ├── appservice.go      # （自动生成的绑定 ID 映射）
│   │   ├── sshconfig.go       # SSH config 文件导入/导出
│   │   └── icons/             # 菜单图标（//go:embed）
│   ├── ssh/
│   │   ├── client.go          # Manager：SSH 客户端池、连接管理
│   │   ├── session.go         # Session：PTY 会话、flushingWriter
│   │   └── monitor.go         # Monitor：服务器资源监控
│   ├── sftp/
│   │   ├── manager.go         # SFTP 管理器、并发传输池
│   │   └── client.go          # SFTP 客户端操作
│   ├── portforward/
│   │   └── forward.go         # 本地端口转发
│   ├── db/
│   │   ├── db.go              # SQLite 初始化、单连接、WAL 模式
│   │   └── migrations.go      # 内联 SQL 迁移
│   ├── crypto/
│   │   └── crypto.go          # AES-256-GCM 加解密服务
│   └── models/                # 数据模型
│       ├── connection.go
│       ├── group.go
│       ├── quick_command.go
│       └── port_forward.go
├── frontend/
│   ├── src/
│   │   ├── App.vue            # 根组件：布局、主题、菜单事件
│   │   ├── main.ts            # 应用入口：Pinia、i18n、Monaco Workers
│   │   ├── components/        # 按领域组织的组件
│   │   │   ├── sidebar/       # 连接树、表单
│   │   │   ├── terminal/      # 终端、编辑器标签
│   │   │   ├── sftp/          # 文件传输
│   │   │   ├── monitor/       # 监控面板
│   │   │   ├── settings/      # 设置弹窗
│   │   │   ├── keys/          # SSH 密钥管理
│   │   │   ├── config/        # SSH 配置管理
│   │   │   ├── activity/      # 活动栏
│   │   │   ├── panels/        # 底部面板、端口转发面板
│   │   │   └── common/        # 共享组件（可拖拽分隔器）
│   │   ├── stores/            # Pinia 状态仓库
│   │   │   ├── connection.ts  # 连接 CRUD、连接/断开
│   │   │   ├── terminal.ts    # 终端会话、标签页
│   │   │   ├── sftp.ts        # SFTP 操作
│   │   │   ├── monitor.ts     # 监控数据
│   │   │   ├── settings.ts    # UI 偏好、主题、语言
│   │   │   ├── layout.ts      # 布局尺寸
│   │   │   ├── transfers.ts   # 文件传输队列
│   │   │   ├── sshkey.ts      # SSH 密钥
│   │   │   └── sshconfig.ts   # SSH 配置
│   │   ├── composables/       # Vue 3 组合式函数
│   │   │   ├── useTerminal.ts      # 终端初始化与事件
│   │   │   ├── useTerminalManager.ts
│   │   │   ├── useEvents.ts       # Wails 事件辅助
│   │   │   ├── useShortcuts.ts    # 快捷键
│   │   │   └── useDragTransfer.ts # 拖拽传输
│   │   ├── locales/           # 国际化文件
│   │   │   ├── zh-CN.ts
│   │   │   └── en.ts
│   │   ├── styles/
│   │   │   └── global.css     # CSS 变量、主题、全局重置
│   │   ├── types/             # TypeScript 类型定义
│   │   └── utils/             # 工具函数
│   ├── bindings/              # Wails 自动生成的绑定（勿手动编辑）
│   │   └── vshell/internal/
│   │       ├── app/appservice.ts
│   │       └── models/models.ts
│   ├── uno.config.ts          # UnoCSS 配置
│   ├── vite.config.ts
│   └── tsconfig.json
└── doc/                       # 文档
```

---

## 3. 架构核心模式

### 3.1 服务绑定模式（Wails 3）

后端通过 `application.NewService()` 将 Go 结构体方法暴露给前端：

```go
// main.go
svc := app.New()
wailsApp := application.New(application.Options{
    Services: []application.Service{
        application.NewService(svc),
    },
})
svc.SetApp(wailsApp) // 后注入应用引用
```

**AppService 生命周期**：
- `ServiceStartup(ctx, options)` — 应用启动时初始化数据库、管理器、事件监听
- `ServiceShutdown()` — 应用关闭时清理资源

**绑定方法签名要求**：
- 方法必须是导出的（大写字母开头）
- 参数和返回值必须可 JSON 序列化
- 敏感字段使用 `json:"-"` 标签防止泄露到前端

```go
type Connection struct {
    Password string `json:"-"` // 永远不会发送到前端
    // ...
}
```

### 3.2 双通道通信模式

项目使用两种前后端通信方式，根据场景选择：

#### 同步调用 — CRUD 操作

适用于请求-响应模式的操作：

```typescript
// 前端调用自动生成的绑定
import { CreateConnection, ListConnections } from '../../bindings/vshell/internal/app/appservice'

// 调用返回 Promise
const connections = await ListConnections()
await CreateConnection(form)
```

```go
// 后端对应方法
func (a *AppService) ListConnections() ([]models.Connection, error) { ... }
func (a *AppService) CreateConnection(form models.ConnectionForm) error { ... }
```

#### 事件系统 — 终端 I/O 和实时数据

**关键规则：终端数据必须使用事件系统，永远不要用同步调用。**

前端 → 后端：
```typescript
import { Events } from '@wailsio/runtime'
Events.Emit('terminal:stdin', { sessionID, data })
Events.Emit('terminal:resize', { sessionID, rows, cols })
```

后端 → 前端：
```go
// 在 ServiceStartup 中注册事件监听
a.wailsApp.Event.On("terminal:stdin", func(e *application.CustomEvent) {
    // 解析并处理
})

// 向前端发送事件
a.wailsApp.Event.Emit("terminal:stdout", map[string]any{
    "sessionID": sessionID,
    "data":      string(data),
})
```

前端接收：
```typescript
Events.On('terminal:stdout', (ev: any) => {
    const d = ev?.data
    if (d?.sessionID === sessionID && d?.data) {
        term.write(d.data)
    }
})

// 清理（重要：避免内存泄漏）
onUnmounted(() => {
    Events.Off('terminal:stdout')
})
```

#### 事件命名约定

| 模式 | 方向 | 用途 |
|------|------|------|
| `terminal:stdin` | 前端→后端 | 终端输入 |
| `terminal:stdout` | 后端→前端 | 终端输出 |
| `terminal:stderr` | 后端→前端 | 终端错误输出 |
| `terminal:resize` | 前端→后端 | 终端窗口大小变化 |
| `terminal:closed` | 后端→前端 | 会话关闭通知 |
| `menu:settings` | 后端→前端 | 原生菜单操作 |
| `menu:close-tab` | 后端→前端 | 原生菜单操作 |
| `localfs:*` | 双向 | 本地文件系统操作 |
| `sftp:upload/download/delete` | 双向 | SFTP 传输操作 |
| `native:file-drop` | 后端→前端 | 文件拖放 |

### 3.3 flushingWriter 模式

终端输出使用缓冲写入器，避免逐字节发送事件：

```go
type flushingWriter struct {
    sessionID   string
    eventName   string
    onEvent     func(string, any)
    mu          sync.Mutex
    pending     []byte
    flushTicker *time.Ticker  // 50ms 定时刷新
}

// 写入时仅追加到缓冲区
func (fw *flushingWriter) Write(p []byte) (int, error) {
    fw.mu.Lock()
    fw.pending = append(fw.pending, p...)
    fw.mu.Unlock()
    return len(p), nil
}

// 定时器触发时批量发送
func (fw *flushingWriter) tickFlush() {
    // 每 50ms 将缓冲数据一次性发送到前端
}
```

---

## 4. 后端开发范式

### 4.1 添加新的绑定方法

**步骤**：

1. 在 `internal/app/app.go` 的 `AppService` 上添加导出方法：

```go
func (a *AppService) MyNewFeature(param string) (MyResult, error) {
    // 实现逻辑
    return MyResult{}, nil
}
```

2. 运行绑定生成：
```bash
wails3 generate bindings
```

3. 前端绑定自动更新到 `frontend/bindings/vshell/internal/app/appservice.ts`

4. 前端使用：
```typescript
import { MyNewFeature } from '../../bindings/vshell/internal/app/appservice'
const result = await MyNewFeature('param')
```

### 4.2 添加新的数据库表

1. 在 `internal/models/` 下创建模型文件：

```go
package models

type MyModel struct {
    ID   string `json:"id"`
    Name string `json:"name"`
}
```

2. 在 `internal/db/migrations.go` 的 `migrations` 切片中添加 CREATE TABLE：

```go
`CREATE TABLE IF NOT EXISTS my_table (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
)`,
```

3. 如果是增量迁移（已发布后加字段），放入 `additive` 切片：
```go
`ALTER TABLE my_table ADD COLUMN description TEXT`,
```

### 4.3 敏感数据处理

所有密码、私钥、密码短语在写入数据库前必须加密：

```go
// 加密
encrypted, err := a.db.Crypto().Encrypt(plaintext)

// 解密（仅在后端使用时解密）
decrypted, err := a.db.Crypto().Decrypt(encrypted)
```

模型中使用 `json:"-"` 阻止序列化到前端：
```go
Password string `json:"-"` // AES 加密，永不暴露到前端
```

### 4.4 SSH 会话管理

```go
// 连接（复用已有客户端）
sessionID := uuid.New().String()
session, err := a.sshManager.Connect(conn, sessionID)

// 断开单个会话
a.sshManager.DisconnectSession(sessionID)

// 断开整个连接（清理所有会话 + SFTP + 监控 + 端口转发）
a.DisconnectSSH(connectionID)
```

SSH 客户端池设计：
- 每个 `connectionID` 维护一个 `ssh.Client`
- 多个终端会话共享同一个客户端
- 最后一个会话关闭时自动清理客户端

### 4.5 事件监听注册

在 `ServiceStartup` 中注册新的事件监听：

```go
a.wailsApp.Event.On("my:event", func(e *application.CustomEvent) {
    payload, _ := json.Marshal(e.Data)
    var msg struct {
        Field string `json:"field"`
    }
    if err := json.Unmarshal(payload, &msg); err != nil {
        return
    }
    // 处理事件...
})
```

---

## 5. 前端开发范式

### 5.1 组件编写规范

**必须使用** `<script setup lang="ts">` 语法：

```vue
<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { Events } from '@wailsio/runtime'

const { t } = useI18n()
const data = ref<string>('')

onMounted(() => {
  Events.On('my:event', handler)
})

onUnmounted(() => {
  Events.Off('my:event')
})
</script>

<template>
  <!-- 使用 Naive UI 组件 -->
  <NButton @click="doSomething">{{ t('key') }}</NButton>
</template>
```

### 5.2 Naive UI 使用规则

**严格规则：只使用 Naive UI，禁止以下操作**：
- 禁止使用 `alert()`、`confirm()`、`prompt()` 等浏览器原生对话框
- 禁止使用 Element Plus、Ant Design 等其他 UI 库
- 使用 Naive UI 的 `useDialog()`、`useMessage()` 替代原生对话框

```typescript
import { useDialog, useMessage } from 'naive-ui'

const dialog = useDialog()
const message = useMessage()

// 替代 confirm()
dialog.warning({
  title: t('common.confirm'),
  content: t('common.deleteConfirm'),
  positiveText: t('common.confirm'),
  negativeText: t('common.cancel'),
  onPositiveClick: () => { /* 执行 */ },
})

// 替代 alert()
message.success(t('common.saved'))
```

### 5.3 Pinia Store 编写

使用 Composition Store 模式（非 Options Store）：

```typescript
import { defineStore } from 'pinia'
import { ref } from 'vue'
import { MyAction } from '../../bindings/vshell/internal/app/appservice'

export const useMyStore = defineStore('myFeature', () => {
  const items = ref<MyType[]>([])
  const loading = ref(false)

  async function loadItems() {
    loading.value = true
    try {
      const result = await MyAction()
      // Wails 返回的是类实例，需转为普通对象以保持 Vue 响应性
      items.value = (result || []).map((item: any) => ({
        id: item.id || '',
        name: item.name || '',
      })) as MyType[]
    } catch (e) {
      console.error('Failed:', e)
    } finally {
      loading.value = false
    }
  }

  return { items, loading, loadItems }
})
```

**关键点**：
- Wails 绑定返回的是类实例，需通过 `map()` 转为普通对象
- 使用 `try/catch/finally` 处理错误
- 异步操作使用 `loading` 状态

### 5.4 终端集成（useTerminal Composable）

```typescript
import { useTerminal } from '../composables/useTerminal'

const { terminal, init, fit, write } = useTerminal(sessionID)

// 在组件挂载时初始化
onMounted(() => {
  const el = terminalRef.value
  if (el) init(el)
})

// resize 时自动适配
const observer = new ResizeObserver(() => fit())
observer.observe(el)
```

### 5.5 主题系统

**CSS 变量驱动**，通过 `data-theme` 属性切换：

```css
/* global.css 中定义 */
[data-theme="dark"] {
  --bg-primary: rgb(18, 18, 18);
  --bg-secondary: rgb(28, 28, 28);
  --text-primary: rgb(224, 224, 224);
  --text-secondary: #858585;
  --border-color: #3d3d3d;
  --hover-overlay: rgba(255, 255, 255, 0.06);
}

[data-theme="light"] {
  --bg-primary: rgb(247, 250, 252);
  --bg-secondary: rgb(255, 255, 255);
  --text-primary: rgb(31, 31, 31);
  --text-secondary: #666666;
  --border-color: #d9d9d9;
  --hover-overlay: rgba(0, 0, 0, 0.04);
}
```

**UnoCSS 中引用 CSS 变量**：

```typescript
// uno.config.ts 中映射
theme: {
  colors: {
    'bg-primary': 'var(--bg-primary)',
    'bg-secondary': 'var(--bg-secondary)',
    'text-primary': 'var(--text-primary)',
    'text-secondary': 'var(--text-secondary)',
  },
}
```

**模板中使用**：

```html
<!-- 直接使用 UnoCSS 映射 -->
<div class="bg-bg-primary text-text-primary">

<!-- 或使用 CSS 变量语法 -->
<div class="bg-[var(--bg-primary)] text-[var(--text-primary)]">
```

**Naive UI 主题桥接**（在 App.vue 中）：

```typescript
const naiveTheme = computed(() => settings.isDark ? darkTheme : null)
const naiveThemeOverrides = computed<GlobalThemeOverrides>(() => {
  const s = getComputedStyle(document.documentElement)
  const primary = s.getPropertyValue('--color-primary').trim()
  return {
    common: {
      primaryColor: primary,
      borderRadius: s.getPropertyValue('--border-radius').trim(),
    },
  }
})
```

### 5.6 国际化

**使用模式**：

```typescript
import { useI18n } from 'vue-i18n'
const { t } = useI18n()

// 模板中
<span>{{ t('connections.title') }}</span>

// 代码中
message.success(t('connections.saved'))
```

**添加新翻译**：在 `locales/zh-CN.ts` 和 `locales/en.ts` 中同步添加键值。

### 5.7 图标使用

使用 Iconify Lucide 图标集：

```vue
<script setup>
import IconServer from '~icons/lucide/server'
import IconPlus from '~icons/lucide/plus'
</script>

<template>
  <IconServer :width="16" :height="16" />
  <IconPlus :width="14" :height="14" />
</template>
```

---

## 6. 添加新功能的标准流程

以"添加笔记功能"为例：

### 步骤 1：后端模型

创建 `internal/models/note.go`：

```go
package models

type Note struct {
    ID           string `json:"id"`
    ConnectionID string `json:"connection_id"`
    Content      string `json:"content"`
    CreatedAt    string `json:"created_at"`
}
```

### 步骤 2：数据库迁移

在 `internal/db/migrations.go` 添加：

```go
`CREATE TABLE IF NOT EXISTS notes (
    id TEXT PRIMARY KEY,
    connection_id TEXT NOT NULL,
    content TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
)`,
```

### 步骤 3：后端服务方法

在 `internal/app/app.go` 添加：

```go
func (a *AppService) ListNotes(connectionID string) ([]models.Note, error) {
    rows, err := a.db.Query("SELECT id, connection_id, content, created_at FROM notes WHERE connection_id = ?", connectionID)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    var notes []models.Note
    for rows.Next() {
        var n models.Note
        if err := rows.Scan(&n.ID, &n.ConnectionID, &n.Content, &n.CreatedAt); err != nil {
            return nil, err
        }
        notes = append(notes, n)
    }
    return notes, nil
}
```

### 步骤 4：生成绑定

```bash
wails3 generate bindings
```

### 步骤 5：前端 Store

创建 `frontend/src/stores/note.ts`：

```typescript
import { defineStore } from 'pinia'
import { ref } from 'vue'
import { ListNotes } from '../../bindings/vshell/internal/app/appservice'

export const useNoteStore = defineStore('note', () => {
  const notes = ref<Note[]>([])

  async function loadNotes(connectionID: string) {
    const result = await ListNotes(connectionID)
    notes.value = (result || []).map((n: any) => ({
      id: n.id || '',
      connectionId: n.connection_id || '',
      content: n.content || '',
    }))
  }

  return { notes, loadNotes }
})
```

### 步骤 6：前端组件

创建 `frontend/src/components/notes/NotePanel.vue`，使用 Naive UI + UnoCSS。

### 步骤 7：集成到布局

在 `App.vue` 或对应面板中引入新组件。

---

## 7. 布局系统

### 7.1 主布局结构

```
┌──────────────────────────────────────────────────┐
│                   Title Bar (30px)                │
│  [vShell]                        [主题] [语言]    │
├────┬─────────┬───────────────────────────────────┤
│    │         │                                   │
│ A  │ Side    │       Terminal Pane                │
│ c  │ bar     │                                   │
│ t  │ (200-   │                                   │
│ i  │  500px) │                                   │
│ v  │         │                                   │
│ i  │─────────│                                   │
│ t  │(dragger)│                                   │
│ y  │         │                                   │
│    │         │                                   │
│ B  ├─────────┴───────────────────────────────────┤
│ a  │              Bottom Panel                    │
│ r  │                                              │
│(48)│                                              │
└────┴──────────────────────────────────────────────┘
```

### 7.2 布局尺寸管理

```typescript
// stores/layout.ts
const sidebarWidth = ref(260)
const bottomPanelHeight = ref(200)

function setSidebarWidth(v: number) {
  sidebarWidth.value = Math.min(500, Math.max(200, v))
}
```

### 7.3 可拖拽分隔器

使用 `components/common/DraggableDivider.vue`：

```vue
<DraggableDivider
  direction="vertical"
  :modelValue="layout.sidebarWidth"
  @update:modelValue="(v: number) => layout.setSidebarWidth(v)"
  :min="200"
  :max="500"
/>
```

---

## 8. 安全规范

1. **敏感数据加密**：密码、私钥、密码短语必须使用 AES-256-GCM 加密后存入数据库
2. **前端隔离**：模型使用 `json:"-"` 阻止敏感字段序列化到前端
3. **文件权限**：SSH 密钥文件权限 0600，目录权限 0700
4. **SQL 注入防护**：所有 SQL 查询使用参数化查询（`?` 占位符），禁止字符串拼接
5. **主机密钥**：当前使用 `InsecureIgnoreHostKey()`（开发阶段），生产需实现已知主机验证

---

## 9. 性能优化

| 场景 | 策略 |
|------|------|
| 终端输出 | `flushingWriter` 50ms 缓冲，避免逐字节事件 |
| SSH 连接 | 客户端池复用，多会话共享同一连接 |
| SFTP 传输 | 并发池（最多 3 个并发传输） |
| 数据库 | WAL 模式、单连接（`SetMaxOpenConns(1)`） |
| 前端渲染 | Vue 响应性 + 虚拟滚动（大文件列表） |
| CSS | UnoCSS 按需生成、CSS 变量运行时切换主题 |

---

## 10. 开发命令速查

```bash
# 完整开发（Go 后端 + 前端热重载）
wails3 dev

# 仅前端开发
cd frontend && npm run dev

# 仅前端构建
cd frontend && npm run build

# Go 后端构建
go build .

# Go 测试
go test ./...

# 重新生成 Wails 绑定
wails3 generate bindings

# 生产构建
wails3 build
```

---

## 11. 常见陷阱

1. **终端数据走同步调用**：会导致 UI 冻结。终端 I/O 必须用 Events。
2. **忘记清理事件监听**：在 `onUnmounted` 中必须调用 `Events.Off()`。
3. **直接使用 Wails 类实例**：绑定返回的是类实例，必须 `map()` 转为普通对象。
4. **修改 bindings/ 目录**：该目录是自动生成的，修改会被覆盖。
5. **使用 CGO 依赖**：项目要求纯 Go（`modernc.org/sqlite`），不能用 `mattn/go-sqlite3`。
6. **使用非 Naive UI 组件**：禁止混用其他 UI 库或原生对话框。
7. **SSH exec 模式**：交互式终端必须用 `RequestPty` + `Shell()`，不能 exec 单命令。
8. **忘记加密敏感数据**：写入数据库前必须加密。

---

## 12. 文件命名约定

| 类型 | 约定 | 示例 |
|------|------|------|
| Go 文件 | `snake_case` | `ssh_config.go`, `client.go` |
| Go 函数 | `PascalCase`（导出）/ `camelCase`（私有） | `ConnectSSH`, `buildSSHConfig` |
| TypeScript 文件 | `kebab-case` | `use-terminal.ts` |
| Vue 组件 | `PascalCase` | `TerminalPane.vue`, `ConnectionTree.vue` |
| Store 文件 | `kebab-case` | `connection.ts`, `ssh-key.ts` |
| CSS 类 | `kebab-case` | `hover-overlay`, `panel-bg` |
| 事件名 | `domain:action` | `terminal:stdin`, `sftp:upload` |
