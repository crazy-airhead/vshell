# 架构核心：双通道通信

## 服务绑定模式（Wails 3）

后端通过 `application.NewService()` 将 Go 结构体方法暴露给前端：

```go
// main.go
svc := app.New()
wailsApp := application.New(application.Options{
    Services: []application.Service{
        application.NewService(svc),
    },
    Assets: application.AssetOptions{
        Handler: application.AssetFileServerFS(assets),
    },
})
svc.SetApp(wailsApp) // 后注入应用引用
```

### AppService 生命周期

```go
type AppService struct {
    wailsApp    *application.App
    db          *db.DB
    sshManager  *vshellssh.Manager
    sftpManager *sftp.Manager
    fwdManager  *portforward.Manager
    monitors    map[string]*vshellssh.Monitor
}

// 应用启动时调用 — 初始化数据库、管理器、事件监听
func (a *AppService) ServiceStartup(ctx context.Context, options application.ServiceOptions) error

// 应用关闭时调用 — 清理资源
func (a *AppService) ServiceShutdown() error
```

### 绑定方法签名要求

- 方法必须是导出的（首字母大写）
- 参数和返回值必须可 JSON 序列化
- 敏感字段使用 `json:"-"` 标签阻止泄露
- 错误通过 `error` 返回，前端以 Promise reject 接收

```go
type Connection struct {
    ID       string `json:"id"`
    Name     string `json:"name"`
    Password string `json:"-"` // 永不暴露到前端
}
```

## 双通道通信

项目使用两种前后端通信方式，根据场景选择：

### 通道 1：同步调用 — CRUD 操作

适用于请求-响应模式的操作（列表、创建、更新、删除）。

```typescript
// 前端调用自动生成的绑定（返回 Promise）
import { ListConnections, CreateConnection } from '../../bindings/vshell/internal/app/appservice'

const connections = await ListConnections()
await CreateConnection(form)
```

```go
// 后端对应导出方法
func (a *AppService) ListConnections() ([]models.Connection, error) { ... }
func (a *AppService) CreateConnection(form models.ConnectionForm) error { ... }
```

### 通道 2：事件系统 — 终端 I/O 和实时数据

**关键规则：终端数据必须使用事件系统，绝不能用同步调用。**

```
终端输入:   xterm.onData → Events.Emit("terminal:stdin")  → Go WriteStdin
终端输出:   goroutine → flushingWriter(50ms) → Events.Emit("terminal:stdout") → xterm.write
终端错误:   goroutine → flushingWriter(50ms) → Events.Emit("terminal:stderr") → xterm.write
终端resize: xterm.onResize → Events.Emit("terminal:resize") → Go WindowChange
终端关闭:   sess.Wait() → Events.Emit("terminal:closed") → 前端清理
```

#### 前端 → 后端

```typescript
import { Events } from '@wailsio/runtime'

// 终端输入
Events.Emit('terminal:stdin', { sessionID, data })

// 终端窗口大小
Events.Emit('terminal:resize', { sessionID, rows, cols })
```

#### 后端 → 前端

```go
// 在 ServiceStartup 中注册事件监听
a.wailsApp.Event.On("terminal:stdin", func(e *application.CustomEvent) {
    payload, _ := json.Marshal(e.Data)
    var msg struct {
        SessionID string `json:"sessionID"`
        Data      string `json:"data"`
    }
    if err := json.Unmarshal(payload, &msg); err != nil {
        return
    }
    a.sshManager.WriteStdin(msg.SessionID, []byte(msg.Data))
})

// 向前端发送事件
a.wailsApp.Event.Emit("terminal:stdout", map[string]any{
    "sessionID": sessionID,
    "data":      string(data),
})
```

#### 前端接收

```typescript
Events.On('terminal:stdout', (ev: any) => {
    const d = ev?.data
    if (d?.sessionID === sessionID && d?.data) {
        term.write(d.data)
    }
})

// 必须清理（否则内存泄漏）
onUnmounted(() => {
    Events.Off('terminal:stdout')
    term.dispose()
})
```

## flushingWriter 模式

终端输出使用缓冲写入器，避免逐字节发送事件导致性能问题：

```go
type flushingWriter struct {
    sessionID   string
    eventName   string
    onEvent     func(string, any)
    mu          sync.Mutex
    pending     []byte
    flushTicker *time.Ticker  // 50ms 定时刷新
}

func newFlushingWriter(sessionID, eventName string, onEvent func(string, any)) *flushingWriter {
    fw := &flushingWriter{
        sessionID:   sessionID,
        eventName:   eventName,
        onEvent:     onEvent,
        flushTicker: time.NewTicker(50 * time.Millisecond),
        done:        make(chan struct{}),
    }
    go fw.tickFlush()
    return fw
}

func (fw *flushingWriter) Write(p []byte) (int, error) {
    fw.mu.Lock()
    fw.pending = append(fw.pending, p...)
    fw.mu.Unlock()
    return len(p), nil
}

func (fw *flushingWriter) tickFlush() {
    for {
        select {
        case <-fw.flushTicker.C:
            fw.mu.Lock()
            fw.doFlush() // 将 pending 数据一次性发送
            fw.mu.Unlock()
        case <-fw.done:
            fw.flushTicker.Stop()
            return
        }
    }
}
```

## 事件命名约定

| 模式 | 方向 | 用途 | Payload 结构 |
|------|------|------|-------------|
| `terminal:stdin` | 前端→后端 | 终端输入 | `{ sessionID, data }` |
| `terminal:stdout` | 后端→前端 | 终端输出 | `{ sessionID, data }` |
| `terminal:stderr` | 后端→前端 | 终端错误 | `{ sessionID, data }` |
| `terminal:resize` | 前端→后端 | 窗口大小 | `{ sessionID, rows, cols }` |
| `terminal:closed` | 后端→前端 | 会话关闭 | `{ sessionID }` |
| `menu:settings` | 后端→前端 | 设置菜单 | `nil` |
| `menu:save` | 后端→前端 | 保存菜单 | `nil` |
| `menu:close-tab` | 后端→前端 | 关闭标签 | `nil` |
| `localfs:homedir` | 双向 | 获取主目录 | request: `nil` / result: `string` |
| `localfs:listdir` | 双向 | 列出目录 | `{ path }` / `[]LocalFileInfo` |
| `sftp:upload` | 双向 | 上传文件 | `{ connectionID, localPath, remotePath }` |
| `sftp:download` | 双向 | 下载文件 | `{ connectionID, remotePath, localPath }` |
| `sftp:transfer-done` | 后端→前端 | 传输完成 | `{ direction, connectionID }` |
| `sftp:*:error` | 后端→前端 | 传输错误 | `string (error message)` |
| `native:file-drop` | 后端→前端 | 文件拖放 | `{ files, targetId }` |

## SSH 客户端池设计

```
connectionID ──→ ssh.Client（一个连接一个客户端）
                    ├── Session 1 (sessionID) — 终端会话
                    ├── Session 2 (sessionID) — 终端会话
                    └── Session N (sessionID) — 可复用于 SFTP/Monitor
```

- 每个 `connectionID` 维护一个 `ssh.Client`
- 多个终端会话共享同一个客户端
- 最后一个会话关闭时自动清理客户端
- `EnsureClient()` 可在不创建终端会话的情况下建立连接（用于端口转发）
