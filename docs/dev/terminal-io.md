# 终端 I/O 数据流

终端是 vShell 中数据量最大、实时性最强的通道，其 I/O 全部走 **Wails Events**，并有专门的缓冲合并机制。

---

## 1. 链路全景

```text
输入（按键）                                    输出（回显 / 命令输出）
─────────────────────────────────────────────────────────────────────
xterm onData                                    Go goroutine
   │                                              │ io.Copy
   ▼                                              ▼
Events.Emit("terminal:stdin")              stdoutPipe ──► flushingWriter（50ms 缓冲）
   {sessionID, data}                             │ 每 50ms 合并 flush
   │                                              ▼
app.go 监听                                onEvent("terminal:stdout", {sessionID, data})
   │                                              │
   ▼                                              ▼
session.WriteStdin(data)                   useTerminalManager 按 sessionID 分发
   │                                              │
   ▼                                              ▼
stdinPipe ──► 远端 PTY ──► Shell            term.write(data) 渲染
```

调整大小：xterm `onResize` → `terminal:resize {sessionID, rows, cols}` → `session.WindowChange(rows, cols)`。

## 2. 会话建立（`ssh/session.go`）

每开一个终端标签：

1. 在复用的 `ssh.Client` 上 `NewSession()`
2. `RequestPty("xterm-256color", 24, 80, modes)` —— `ECHO: 1`，TTY 收发速率 14400
3. **先建管道再启 shell**：`StdinPipe` / `StdoutPipe` / `StderrPipe`（避免 Shell 启动后管道未就绪丢数据）
4. `sess.Shell()`
5. stdout / stderr 各接一个 `flushingWriter`，各起 goroutine `io.Copy`
6. 第三个 goroutine `sess.Wait()` 结束后发 `terminal:closed {sessionID}`，前端据此显示断连提示

## 3. flushingWriter：50ms 批量合并

PTY 输出常以字节级碎包到达（vim / top 等全屏程序尤其明显），逐字节 Emit 事件会造成大量跨语言序列化开销。`flushingWriter` 的做法：

| 机制 | 实现 |
|------|------|
| `Write(p)` | 只把 p 追加进 `pending` 缓冲，立即返回（不阻塞读取方） |
| `flushTicker` | 每 **50ms** 把整个 `pending` 作为一次事件载荷 Emit，复用底层数组（`pending[:0]`） |
| 结束 | `io.Copy` 返回后强制 `Flush()` 一次，保证尾包不丢 |

即：**按时间窗口合并，不设缓冲上限**。对终端场景，50ms 低于人眼感知阈值，既流畅又把事件量压缩了几个数量级。

## 4. 前端分发（`useTerminalManager.ts`）

- 全局单例监听 `terminal:stdout`，按 `sessionID` 写入对应 `terminals` Map 中的 xterm 实例 —— 多标签并存时各终端只收到自己的数据
- `terminal:closed` 把对应标签标记为断开，并写入黄色提示「--- 连接已断开，按回车键重连 ---」
- 按回车触发重连：断开旧会话 → 重新 `ConnectSSH` → 原标签接续新 sessionID

## 5. 渲染

- `XTerminal.vue`：xterm.js v6 + FitAddon（自动适配容器尺寸）+ WebglAddon（失败回退 canvas）
- 字体 / 字号 / 配色来自 settings store，watch 实时热更新 `term.options`
- `attachCustomKeyEventHandler` 拦截复制 / 粘贴组合键（命中返回 false 不进 PTY）；macOS 上 `Ctrl+C` / `Ctrl+V` 永不拦截，保留远端语义

## 6. 设计要点小结

1. 输入输出都是事件流，**没有任何同步 Call 参与 I/O**（红线 1）
2. PTY 必须 `xterm-256color` + `Shell()`（红线 4），不使用 exec 模拟
3. 输出方向做时间窗口合并（50ms），输入方向天然低频无需合并
4. stderr 独立管道与事件（`terminal:stderr`），当前前端未消费

---

## 延伸阅读

- [终端使用](../guide/terminal.md)
- [架构总览](architecture.md)
