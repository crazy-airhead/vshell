# 后端开发模式

## 添加绑定方法

### 步骤

1. 在 `internal/app/app.go` 的 `AppService` 上添加导出方法
2. 运行 `wails3 generate bindings`
3. 前端绑定自动更新到 `frontend/bindings/vshell/internal/app/appservice.ts`
4. 前端使用：`import { MyMethod } from '../../bindings/vshell/internal/app/appservice'`

### 方法签名要求

```go
// 导出方法 — 首字母大写，参数/返回值可 JSON 序列化
func (a *AppService) ListConnections() ([]models.Connection, error) { ... }
func (a *AppService) CreateConnection(form models.ConnectionForm) error { ... }
func (a *AppService) ConnectSSH(connectionID string) (string, error) { ... }
func (a *AppService) MoveConnection(connectionID string, groupID *string) error { ... }
```

### 不暴露到前端的方法

以小写字母开头的方法不会生成绑定：

```go
// 私有方法，不暴露到前端
func (a *AppService) getConnectionByID(id string) (*models.Connection, error) { ... }
func (a *AppService) listLocalDir(dirPath string) ([]LocalFileInfo, error) { ... }
```

### 内嵌类型定义

仅在 AppService 内使用的响应类型可直接定义在 `app.go` 中：

```go
type LocalFileInfo struct {
    Name    string `json:"name"`
    Path    string `json:"path"`
    Size    int64  `json:"size"`
    IsDir   bool   `json:"is_dir"`
}
```

## 数据库操作

### 数据库初始化

```go
// db/db.go — 单例模式
func New() (*DB, error) {
    sqlDB, _ := sql.Open("sqlite", dbPath+"?_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)")
    sqlDB.SetMaxOpenConns(1) // 单连接
    return &DB{DB: sqlDB, crypto: crypto.New()}, nil
}
```

数据库路径：`~/Library/Application Support/vshell/vshell.db`（macOS）

### 添加新表

1. 在 `internal/models/` 创建模型（加 `json` tag，敏感字段用 `json:"-"`）

```go
package models

type Note struct {
    ID        string `json:"id"`
    Content   string `json:"content"`
    CreatedAt string `json:"created_at"`
}
```

2. 在 `internal/db/migrations.go` 的 `migrations` 切片添加：

```go
`CREATE TABLE IF NOT EXISTS notes (
    id TEXT PRIMARY KEY,
    content TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
)`,
```

3. 增量迁移（已发布后加字段）放入 `additive` 切片（忽略错误）：

```go
`ALTER TABLE notes ADD COLUMN title TEXT`,
```

### 标准 CRUD 模式

```go
// List
func (a *AppService) ListNotes() ([]models.Note, error) {
    rows, err := a.db.Query("SELECT id, content, created_at FROM notes ORDER BY created_at DESC")
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var notes []models.Note
    for rows.Next() {
        var n models.Note
        if err := rows.Scan(&n.ID, &n.Content, &n.CreatedAt); err != nil {
            return nil, err
        }
        notes = append(notes, n)
    }
    return notes, nil
}

// Create
func (a *AppService) CreateNote(n models.Note) error {
    _, err := a.db.Exec("INSERT INTO notes (id, content) VALUES (?, ?)", n.ID, n.Content)
    return err
}

// Delete
func (a *AppService) DeleteNote(id string) error {
    _, err := a.db.Exec("DELETE FROM notes WHERE id = ?", id)
    return err
}
```

## 敏感数据处理

### 加密/解密

```go
// 加密 — 写入数据库前
encryptedPW, err := a.db.Crypto().Encrypt(form.Password)

// 解密 — 仅在后端使用时
password, err := a.db.Crypto().Decrypt(encrypted)
```

### 模型隔离

```go
type Connection struct {
    ID       string `json:"id"`
    Name     string `json:"name"`
    Password string `json:"-"`      // AES 加密，永不暴露
    PrivateKey string `json:"-"`    // AES 加密，永不暴露
    KeyPassphrase string `json:"-"` // AES 加密，永不暴露
}
```

### 更新时保留敏感字段

更新连接时，如果前端传空值，保留数据库中已有的加密值：

```go
func (a *AppService) UpdateConnection(form models.ConnectionForm) error {
    var encryptedPW string
    if form.Password != "" {
        encryptedPW, _ = a.db.Crypto().Encrypt(form.Password)
    } else {
        a.db.QueryRow("SELECT password FROM connections WHERE id = ?", form.ID).Scan(&encryptedPW)
    }
    // ... 类似处理 PrivateKey 和 KeyPassphrase
}
```

## SSH 会话管理

### 连接与断开

```go
// 连接（自动复用已有客户端）
sessionID := uuid.New().String()
session, err := a.sshManager.Connect(conn, sessionID)

// 断开单个会话（如果最后一个会话则清理整个连接）
a.sshManager.DisconnectSession(sessionID)

// 断开整个连接（清理所有会话 + SFTP + 监控 + 端口转发）
a.DisconnectSSH(connectionID)
```

### 完整断开清理流程

```go
func (a *AppService) DisconnectSSH(connectionID string) {
    a.StopMonitor(connectionID)
    a.sftpManager.CloseClient(connectionID)
    a.sshManager.Disconnect(connectionID)
    a.fwdManager.StopAllForConnection(connectionID)
}
```

### Monitor 管理

```go
// 启动监控
func (a *AppService) StartMonitor(connectionID string) error

// 停止监控
func (a *AppService) StopMonitor(connectionID string)
```

## 事件监听注册

在 `ServiceStartup` 中注册：

```go
func (a *AppService) ServiceStartup(ctx context.Context, options application.ServiceOptions) error {
    // ... 初始化数据库和管理器

    emit := func(event string, data any) {
        a.wailsApp.Event.Emit(event, data)
    }

    // 注册事件监听
    a.wailsApp.Event.On("domain:action", func(e *application.CustomEvent) {
        payload, _ := json.Marshal(e.Data)
        var msg struct {
            Field string `json:"field"`
        }
        if err := json.Unmarshal(payload, &msg); err != nil {
            return
        }
        // 处理事件...
    })

    return nil
}
```

### 异步事件处理

长时间运行的操作（如 SFTP 传输）使用 goroutine：

```go
a.wailsApp.Event.On("sftp:upload", func(e *application.CustomEvent) {
    // 解析参数...
    go func() {
        if err := a.sftpManager.UploadFile(msg.ConnectionID, msg.LocalPath, msg.RemotePath); err != nil {
            emit("sftp:upload:error", err.Error())
        }
    }()
})
```

## SFTP 操作

```go
// 同步方法（前端直接调用）
func (a *AppService) SFTPReadDir(connectionID, path string) ([]sftp.FileInfo, error)
func (a *AppService) SFTPReadFileContent(connectionID, remotePath string) (string, error)
func (a *AppService) SFTPWriteFileContent(connectionID, remotePath, content string) error

// 异步方法（内部用 goroutine，通过事件通知结果）
func (a *AppService) SFTPUpload(connectionID, localPath, remotePath string) error
func (a *AppService) SFTPDownload(connectionID, remotePath, localPath string) error
```

## 本地文件系统操作

```go
func (a *AppService) GetHomeDir() (string, error)
func (a *AppService) ListLocalDir(dirPath string) ([]LocalFileInfo, error)
func (a *AppService) ReadLocalFileContent(localPath string) (string, error) // 限制 5MB
func (a *AppService) WriteLocalFileContent(localPath, content string) error
func (a *AppService) DeleteLocalFile(localPath string) error
func (a *AppService) OpenInFileManager(path string) error // 跨平台
```

## SSH 密钥管理

```go
func (a *AppService) ListSSHKeys() ([]SSHKeyInfo, error)      // 扫描 ~/.ssh
func (a *AppService) SaveSSHKey(name, privateKey, publicKey string) error
func (a *AppService) RenameSSHKey(oldName, newName string) error
func (a *AppService) DeleteSSHKey(name string) error
func (a *AppService) ReadSSHKeyContent(name, kind string) (string, error)  // kind: "pub" / "private"
func (a *AppService) GenerateSSHKey(name, keyType string, bits int, comment, passphrase string) error
```
