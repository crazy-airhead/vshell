# 架构总览

vShell 是 Wails 3 桌面应用：**Go 后端**承担全部系统能力（SSH / SFTP / 文件 / 数据库），**Vue 3 前端**纯 UI。两者通过「服务绑定（同步调用）+ Wails Events（流式推送）」双通道通信。

---

## 1. 总体结构

```text
┌──────────────────────────────── main.go ────────────────────────────────┐
│  embed frontend/dist · 原生菜单（menu:settings/save/close-tab）          │
│  窗口：1280×800 · macOS 隐藏标题栏(30px) · EnableFileDrop                │
│  Services: [AppService]                                                  │
└───────────────────────────────────┬──────────────────────────────────────┘
                                    │
┌──────────────────────────── internal/app.AppService ────────────────────┐
│  wailsApp · db · sshManager · sftpManager · fwdManager · monitors{}    │
│                                                                         │
│  服务绑定方法（81 个）：连接/分组/快捷命令/端口转发/SFTP/密钥/sshconfig/   │
│  本地FS/证书 · 生命周期：ServiceStartup / ServiceShutdown               │
│  连接管理：ConnectSSH / DisconnectSSH / DisconnectSession               │
└───┬──────────┬──────────────┬──────────────┬──────────┬────────────────┘
    │          │              │              │          │
┌───▼───┐  ┌───▼───────┐  ┌───▼─────────┐  ┌─▼────────┐ │
│ ssh   │  │ sftp      │  │ portforward │  │ db       │ │
│ 会话/ │  │ 传输并发池 │  │ 本地转发     │  │ SQLite   │ │
│ 监控  │  │ (cap 3)   │  │             │  │ WAL 单连接│ │
└───────┘  └───────────┘  └─────────────┘  └──────────┘ │
┌───────────┐  ┌───────────┐                             │
│ crypto    │  │ cert      │◄────────────────────────────┘
│ AES-GCM   │  │ acme.sh   │  经 SSH/SFTP 驱动远程服务器上的
│ .enc_key  │  │ 远程编排  │  acme.sh（检测/安装/签发/续签/部署）
└───────────┘  └───────────┘
```

## 2. 前端结构（frontend/src）

无路由的单页面组合（vue-router 依赖存在但未使用路由表）。界面为**岛屿（Islands）设计语言**——画布 + 浮动圆角岛，岛间统一 6px 接缝（规范见[岛屿设计语言](../ui/vshell-design-language.md)），自上而下：

| 层 | 组件 / 模块 |
|----|-------------|
| 岛屿壳 | `shell/AppShell.vue` —— 画布级布局，集中持有各面板头部动作 |
| 顶部工具栏岛 | `shell/ToolbarIsland.vue` —— 唯一窗口拖拽区、主题 / 语言切换、连接进度条 |
| 左条 + 左工具窗岛 | `shell/LeftStripe.vue`（40px 通高图标条：连接 / 密钥 / SSH 配置 / 端口转发 / 证书 + SFTP / 监控 / 设置）；`shell/ToolWindow.vue`（可收起）承载 `sidebar/`（连接树、连接表单）、`keys/`、`config/`、`panels/`（端口转发）、`cert/`（证书管理） |
| 中心岛（永不收起） | `shell/EditorIsland.vue` + `shell/EditorTabs.vue`（超宽标签按 IDEA 方式折叠进下拉）→ `terminal/XTerminal.vue` / `EditorTab.vue`（Monaco）；所有会话保活堆叠（visibility 切换，不销毁） |
| 底部工具窗岛 | `monitor/MonitorPanel.vue`（ECharts）、`sftp/SFTPArea.vue` |
| 状态栏岛 | `shell/StatusBarIsland.vue` —— 连接四态与会话数、传输摘要、面板快速开关 |
| 状态 | Pinia：connection / terminal / sftp / monitor / settings / layout / transfers / sshkey / sshconfig / cert（10 个） |
| 组合式 | `useTerminalManager.ts`（全局 stdout 分发）、`useShortcuts.ts`、`useDragTransfer.ts`（应用内拖拽） |

## 3. 双通道通信

### 3.1 服务绑定（同步，请求 / 响应）

CRUD 与一次性操作走 `application.NewService(AppService)` 绑定，前端调用生成于 `frontend/bindings/vshell/internal/app/appservice.ts`（81 个函数，分布在 `app.go` / `cert.go` / `sshconfig.go`）。典型分组：

| 分组 | 代表方法 |
|------|----------|
| 连接 / 分组 | `ListConnections` `CreateConnection` `MoveConnection` `GetPassword` … |
| SSH 生命周期 | `ConnectSSH` `DisconnectSSH` `DisconnectSession` `StartMonitor` `StopMonitor` |
| SFTP | `SFTPReadDir` `SFTPUpload` `SFTPDownload` `SFTPDelete` `SFTPReadFileContent` `SFTPCancelTransfers` |
| 密钥 | `ListSSHKeys` `GenerateSSHKey` `SaveSSHKey` `GetSSHKeyUsage` … |
| sshconfig | `ReadSSHConfig` `WriteSSHConfig` `GetSSHConfigImportCandidates` `ImportSSHConfigHosts` … |
| 本地 FS | `GetHomeDir` `ListLocalDir` `ReadLocalFileContent` `OpenInFileManager` … |
| 端口转发 | `CreatePortForward` `StartPortForward` `StopPortForward` `ListRunningPortForwards` … |
| 证书 | `ListCertTasks` `CreateCertTask` `StartCertIssue` `StartCertRenew` `ListDNSProviders` `DetectCertEnvironment` `ListRemoteCerts` `GetCertServerLog` `CancelCertOp` … |

### 3.2 Wails Events（流式 / 推送）

**后端 → 前端**（`wailsApp.Event.Emit`）：

| 事件 | 载荷 | 用途 |
|------|------|------|
| `terminal:stdout` / `terminal:stderr` | `{sessionID, data}` | 终端输出流 |
| `terminal:closed` | `{sessionID}` | 会话结束 |
| `monitor:stats` / `:disk` / `:load` / `:processes` / `:netconns` | `{connectionID, …}` | 监控指标（2s/10s/5s/5s/10s） |
| `sftp:progress` | `TransferProgress` | 传输进度 |
| `sftp:transfer-done` | `{direction, connectionID}` | 传输 / 删除完成 |
| `sftp:upload:error` / `sftp:download:error` | string | 传输失败 |
| `cert:log` | `{taskID?/opID?, connectionID, stream, line, ts}` | acme.sh 操作日志逐行流式回传 |
| `cert:stage` | `{taskID?/opID?, connectionID, stage, status, error?}` | 证书编排阶段推进（start / ok / fail） |
| `menu:settings` / `menu:save` / `menu:close-tab` | nil | 原生菜单动作 |
| `native:file-drop` | `{files, targetId}` | 系统文件拖入 |

**前端 → 后端**（`Events.Emit`，`app.ServiceStartup` 注册监听）：

| 事件 | 载荷 | 处理 |
|------|------|------|
| `terminal:stdin` | `{sessionID, data}` | `WriteStdin` |
| `terminal:resize` | `{sessionID, rows, cols}` | `WindowChange` |

> 红线：终端数据**必须**走 Events，同步 Call 会把每字节 I/O 变成跨语言往返。`localfs:*` / `sftp:upload` / `sftp:download` 等旧事件通道已废弃改绑定调用，后端监听仍在（遗留双通道）。

## 4. 关键连接管理

`ssh.Manager` 维护三层映射：`clients[connectionID]*ssh.Client`、`sessions[sessionID]*Session`、`connSess[connectionID][]sessionID`。

- 同一连接的**终端、SFTP、监控、端口转发复用同一条 SSH TCP 连接**
- 连接级断开（`Disconnect`）依次：停监控 → 关 SFTP → 关全部会话与 client → 停全部转发
- 端口转发可在无终端时按需 `EnsureClient` 建立纯转发连接，空闲后 `RemoveClient` 回收
- 会话通道数受服务端 `MaxSessions`（OpenSSH 默认 10）限制：超限时 `NewSession()` 报错经 `friendlySessionError` 转译为可操作提示（关闭未用标签或调大 sshd_config），**不再误关共享连接**（问题 0010，含回归测试）

认证构建（`buildSSHConfig`）：密码 / 私钥（含口令）/ Keyboard-Interactive 三种已实现；Agent 与跳板机未实现。`HostKeyCallback` 当前为 `ssh.InsecureIgnoreHostKey()`（不校验主机指纹），超时 10s。

## 5. 证书管理（`internal/cert`）

通过 SSH 在远程服务器上驱动 acme.sh，编排「检测 → 安装 → 签发 / 续签 → 部署证书 → 检查 cron → 完成」的阶段流：

- **编排器**（`issue.go` / `manager.go`）：每阶段经 `cert:stage` 事件汇报进度，远程命令输出逐行经 `cert:log` 流式回传，并持久化到 `cert_tasks.last_log`
- **续签刻意走 `--issue --force`**：`--renew` 会重放服务器旧 conf 里的插件配置，vShell 侧更换 DNS 提供商 / 凭据不会生效
- **DNS 凭据**：AES 加密存储、`json:"-"` 永不出后端；签发时经 SFTP 写临时 env 文件（0600，命令结束自删）
- **DNS 提供商注册表**（`dnsproviders.go`）：aliyun / dnspod / tencent / cloudflare / custom，含凭据字段校验
- 应用启动时 `resetInterruptedCertTasks` 把上次被中断的「执行中」任务复位

## 6. 已知实现边界

| 项 | 状态 |
|----|------|
| SSH Agent 认证、跳板机 | 未实现（连接时报错） |
| 端口转发 | 仅 `local`；`remote` / `dynamic` 启动即报错 |
| 端口转发「自动启动」 | 仅存储勾选，无自动拉起路径 |
| ZMODEM | 纯桩（`zmodem.Detector`，无调用方） |
| 终端 stderr 事件 | 后端发射、前端未消费 |
| SFTP 重命名 / chmod | 未实现 |
| 快捷命令 | 后端 API + 表齐备，前端无 UI |
| DB 迁移 | 无版本表，`CREATE TABLE IF NOT EXISTS` + 容错 `ALTER` |

---

## 延伸阅读

- [终端 I/O 数据流](terminal-io.md)
- [数据存储与加密](storage-crypto.md)
- [构建与开发环境](development.md)
