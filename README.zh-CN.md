# vShell

基于 **Wails 3**（Go + Vue 3）构建的桌面 SSH 客户端管理工具。纯本地应用——无云服务、无账号注册、无遥测数据。

## 功能特性

- **SSH 连接管理** — 树形分组组织连接，支持颜色标记和标签
- **完整终端仿真** — 基于 xterm.js 的 PTY 终端，支持 xterm-256color
- **SFTP 文件管理** — 浏览、上传、下载、删除文件，支持拖拽传输和进度追踪
- **远程文件编辑** — 使用 Monaco Editor 直接编辑远程文件
- **服务器资源监控** — 通过 ECharts 实时展示 CPU 和内存图表
- **SSH 密钥管理** — 生成、导入和管理 SSH 密钥对
- **SSH 配置导入导出** — 从 `~/.ssh/config` 导入连接配置
- **端口转发** — 本地端口转发，支持自动启动
- **快捷命令** — 保存并执行常用命令
- **国际化** — 支持中文和英文

## 技术栈

| 层级 | 技术 |
|------|------|
| 框架 | Wails 3 (alpha) |
| 后端 | Go 1.25 |
| 前端 | Vue 3 + TypeScript |
| UI 组件库 | Naive UI |
| CSS | UnoCSS |
| 终端 | xterm.js v6 |
| 编辑器 | Monaco Editor |
| 图表 | ECharts |
| 数据库 | SQLite（纯 Go 实现，无需 CGO） |
| 加密 | AES-256-GCM |
| 构建 | Taskfile + Vite |

## 开发

```bash
# 安装 Wails 3 CLI
go install github.com/wailsapp/wails/v3/cmd/wails3@latest

# 开发模式运行（前后端热重载）
wails3 dev

# 构建生产版本
wails3 build
```

仅前端开发：

```bash
cd frontend
npm install
npm run dev          # 启动 Vite 开发服务器，端口 9245
```

## 项目结构

```
vshell/
├── main.go                  # 应用入口，窗口配置，原生菜单
├── build/
│   └── config.yml           # Wails 3 构建配置
├── internal/
│   ├── app/                 # Wails AppService（所有绑定到前端的方法）
│   ├── ssh/                 # SSH 客户端、会话（PTY）及服务器监控
│   ├── sftp/                # SFTP 客户端和传输管理器
│   ├── portforward/         # 本地端口转发
│   ├── zmodem/              # Zmodem 协议支持
│   ├── db/                  # SQLite 数据库及迁移
│   ├── crypto/              # AES-256-GCM 加密
│   └── models/              # 数据模型
├── frontend/
│   └── src/
│       ├── components/      # 按领域组织的 Vue 组件
│       ├── stores/          # Pinia 状态管理
│       ├── composables/     # 可复用组合式函数
│       ├── locales/         # 国际化翻译（zh-CN、en）
│       ├── styles/          # 全局样式及主题变量
│       ├── types/           # TypeScript 类型定义
│       └── utils/           # 工具函数
└── doc/                     # 架构文档
```

## 编译构建

### 前置条件

- [Go](https://go.dev/dl/) 1.25+
- [Node.js](https://nodejs.org/) 18+
- [Wails 3 CLI](https://v3.wails.io/)：`go install github.com/wailsapp/wails/v3/cmd/wails3@latest`

### macOS

```bash
# 快速构建（仅生成二进制文件）
wails3 build
# 输出：bin/vshell

# 构建为 .app 应用包
./scripts/build_macos.sh
# 输出：bin/vshell.app

# 启动应用
open bin/vshell.app
```

### Windows

```bash
# 快速构建（仅生成二进制文件，默认 amd64）
wails3 task windows:build
# 输出：bin/vshell.exe

# 指定架构构建
wails3 task windows:build ARCH=arm64

# 构建并打包为 NSIS 安装程序
wails3 task windows:package FORMAT=nsis

# 或使用构建脚本
./scripts/build_windows.sh
./scripts/build_windows.sh ARCH=arm64 PACKAGE=true FORMAT=nsis
```

### 交叉编译

在 macOS 上交叉编译 Windows 版本（需要对应的 C 交叉编译器工具链）：

```bash
GOOS=windows GOARCH=amd64 wails3 task windows:build
```

## 数据库

SQLite 数据库存储在 `~/Library/Application Support/vshell/vshell.db`（macOS）。所有敏感数据（密码、私钥、口令）在存储前均使用 AES-256-GCM 加密。

> **注意**：本项目为作者自用工具，主要在 macOS 上使用和测试。虽然支持编译为 Windows 版本，但未经实际测试。

## 联系与赞助

<table>
  <tr>
    <td align="center">
      <img src="doc/images/wechat-qr.jpg" alt="微信" width="200"/><br/>
      <b>微信联系</b>
    </td>
    <td align="center">
      <img src="doc/images/sponsor-qr.jpg" alt="打赏" width="200"/><br/>
      <b>请我喝杯咖啡</b>
    </td>
  </tr>
</table>
