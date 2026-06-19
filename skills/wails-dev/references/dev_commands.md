# 开发命令与排错

## 开发命令

```bash
# 完整开发（Go 后端 + 前端热重载）
wails3 dev

# 仅前端开发
cd frontend && npm run dev

# 仅前端构建
cd frontend && npm run build

# 前端开发构建（无压缩）
cd frontend && npm run build:dev

# Go 后端构建
go build .

# Go 测试
go test ./...

# 重新生成 Wails 绑定（修改 Go 方法后必须运行）
wails3 generate bindings

# 生产构建
wails3 build
```

## 构建配置

### build/config.yml

```yaml
version: '3'
info:
  companyName: "vShell"
  productName: "vShell"
  productIdentifier: "dev.vshell.app"
  description: "SSH Client Management Tool"
  version: "0.0.1"

dev_mode:
  root_path: .
  log_level: warn
  debounce: 1000
  ignore:
    dir: [.git, node_modules, frontend, bin]
    file: [.DS_Store, "*_test.go"]
    watched_extension: ["*.go", "*.js", "*.ts"]
```

### Vite 配置

- 默认端口：9245（可通过 `WAILS_VITE_PORT` 环境变量修改）
- Monaco Editor Web Workers 在 `main.ts` 中配置

### Go 版本

- Go 1.25.0
- Wails v3.0.0-alpha.92

## Wails Runtime API 速查

```typescript
// 事件系统
import { Events } from '@wailsio/runtime'
Events.Emit('event:name', payload)
Events.On('event:name', handler)
Events.Off('event:name')

// 窗口操作
import { Window } from '@wailsio/runtime'
Window.ToggleMaximise()

// 原生文件对话框（如果需要）
import { Dialogs } from '@wailsio/runtime'
const path = await Dialogs.OpenFile({
  Title: 'Select Directory',
  CanChooseDirectories: true,
  CanChooseFiles: false,
})
```

## 端到端添加新功能

以"笔记功能"为例：

### Step 1 — 模型

创建 `internal/models/note.go`：

```go
package models

type Note struct {
    ID        string `json:"id"`
    Content   string `json:"content"`
    CreatedAt string `json:"created_at"`
}
```

### Step 2 — 数据库迁移

在 `internal/db/migrations.go` 添加：

```go
`CREATE TABLE IF NOT EXISTS notes (
    id TEXT PRIMARY KEY,
    content TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
)`,
```

### Step 3 — 绑定方法

在 `internal/app/app.go` 添加：

```go
func (a *AppService) ListNotes() ([]models.Note, error) {
    rows, err := a.db.Query("SELECT id, content, created_at FROM notes ORDER BY created_at DESC")
    if err != nil { return nil, err }
    defer rows.Close()
    var notes []models.Note
    for rows.Next() {
        var n models.Note
        if err := rows.Scan(&n.ID, &n.Content, &n.CreatedAt); err != nil { return nil, err }
        notes = append(notes, n)
    }
    return notes, nil
}
```

### Step 4 — 生成绑定

```bash
wails3 generate bindings
```

### Step 5 — Store

创建 `frontend/src/stores/note.ts`：

```typescript
import { defineStore } from 'pinia'
import { ref } from 'vue'
import { ListNotes } from '../../bindings/vshell/internal/app/appservice'

export const useNoteStore = defineStore('note', () => {
  const notes = ref<Note[]>([])

  async function loadNotes() {
    const result = await ListNotes()
    notes.value = (result || []).map((n: any) => ({
      id: n.id || '',
      content: n.content || '',
    }))
  }

  return { notes, loadNotes }
})
```

### Step 6 — 组件

创建 `frontend/src/components/notes/NotePanel.vue`（Naive UI + UnoCSS + i18n）

### Step 7 — 集成

在 App.vue 或对应面板中引入

## 常见陷阱排错

| 问题 | 原因 | 解决 |
|------|------|------|
| 终端 UI 冻结 | 终端数据走了同步调用 | 终端 I/O 必须用 Events |
| 内存泄漏 | 忘记清理事件监听 | `onUnmounted` 中 `Events.Off()` |
| 数据不响应 | 直接使用 Wails 类实例 | `map()` 转为普通对象 |
| 绑定函数找不到 | 未生成绑定或 Go 方法非公开 | `wails3 generate bindings` + 确保首字母大写 |
| 修改 bindings/ 被覆盖 | 手动编辑自动生成文件 | 该目录由工具生成，勿手动编辑 |
| 面板溢出/滚动异常 | flex 子元素溢出 | 添加 `min-height: 0; overflow: auto` |
| CGO 编译失败 | 使用了需要 CGO 的依赖 | 必须用 `modernc.org/sqlite` |
| 对话框样式异常 | 使用了 alert/confirm | 必须用 Naive UI 的 useDialog/useMessage |
| 敏感数据泄露到前端 | 模型未用 json:"-" | 写入前 `Crypto().Encrypt()`，模型 `json:"-"` |
| 前端找不到绑定类型 | 修改 Go 后未重新生成 | 运行 `wails3 generate bindings` |
| macOS 标题栏重叠 | 未设置隐藏标题栏 | 窗口选项设置 `InvisibleTitleBarHeight: 30` + `MacTitleBarHidden` |
| Monaco 不随容器调整 | 缺少 automaticLayout | 创建时设置 `automaticLayout: true` |
| 主题不生效 | CSS 变量未定义或未切换 | 检查 `global.css` 和 `data-theme` 属性 |
| Go 修改后前端无变化 | dev 模式未重新编译 | 修改 Go 代码后 `wails3 dev` 自动重新编译 |

## 性能策略

| 场景 | 策略 |
|------|------|
| 终端输出 | flushingWriter 50ms 缓冲，避免逐字节事件 |
| SSH 连接 | 客户端池复用，多会话共享同一连接 |
| SFTP 传输 | 并发池（最多 3 个并发传输） |
| 数据库 | WAL 模式、单连接 (`SetMaxOpenConns(1)`) |
| 前端渲染 | Vue 响应性 + 虚拟滚动（大文件列表） |
| CSS | UnoCSS 按需生成、CSS 变量运行时切换主题 |
