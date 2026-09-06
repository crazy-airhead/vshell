# 构建与开发环境

本文记录 vShell 的工具链、构建任务与开发工作流。仓库采用**工作区/制品区分离**：`main` 分支（工作区）只含文档站与设计文档；应用源码在 `artifacts` 分支，检出为同级 worktree `../vshell-artifacts`。项目结构分三层：**Go 后端**（`internal/`）、**Vue 3 前端**（`frontend/`，独立 pnpm 项目）、**文档站**（`docs/`，独立 pnpm 项目）。

---

## 1. 工具链版本

| 工具 | 版本 | 说明 |
|------|------|------|
| Go | 1.25.0 | 见 `go.mod` |
| Wails | v3.0.0-beta.16 | Go module、`@wailsio/runtime`、`wails3` CLI 三者需同版本 |
| Node.js | 20+ | 前端与文档站 |
| pnpm | 9+ | `frontend/package.json` 与 `docs/package.json` 各自声明 `packageManager` |
| SQLite | modernc.org/sqlite | 纯 Go 驱动，**无 CGO** |
| Task | [go-task/task](https://taskfile.dev) | 构建任务入口（制品区 `Taskfile.yml`） |

> **为什么是 pnpm 而不是 npm**：`wails3 build` 依据 lockfile 自动选择前端包管理器，仓库必须携带 `pnpm-lock.yaml`（见 `frontend/`）。文档站同样使用 pnpm。

## 2. 常用命令

> 应用开发、前端、Go 后端的命令在**制品区** `../vshell-artifacts`（`artifacts` 分支）执行；文档站命令在**工作区**（本目录）执行。

### 应用开发

```bash
wails3 dev               # 开发模式：Go + Vue 热重载
wails3 build             # 生产构建（自动安装前端依赖并构建）
wails3 task run          # 运行最近构建的二进制
```

### 前端（frontend/，独立 pnpm 项目）

```bash
cd frontend
pnpm install             # 安装依赖
pnpm dev                 # Vite dev server（默认端口 9245，WAILS_VITE_PORT 可覆盖）
pnpm build               # vue-tsc 类型检查 + vite 生产构建
pnpm build:dev           # 不压缩的开发构建
pnpm typecheck           # 仅类型检查（vue-tsc --noEmit）
```

### Go 后端

```bash
go build ./...
go test ./...
go vet ./...
```

### 文档站（docs/，独立 pnpm 项目，工作区执行）

```bash
cd docs
pnpm install             # 安装依赖
pnpm dev                 # VitePress 开发服务器
pnpm build               # 构建到 docs/.vitepress/dist
pnpm preview             # 本地预览构建产物
```

## 3. 构建配置文件

| 文件 | 职责 |
|------|------|
| `Taskfile.yml` | 制品区顶层任务（build / dev / run / package 等），include `build/Taskfile.yml` 等 |
| `build/Taskfile.yml` | 通用任务：前端依赖安装与构建（`dir: frontend`）、bindings 生成 |
| `build/config.yml` | Wails 3 项目信息（productName: vShell、identifier: `dev.vshell.app`）与 dev_mode 配置 |
| `main.go` | 窗口选项（`application.Options`）与服务注册 |

前端依赖安装由 `build/Taskfile.yml` 的 `install:frontend:deps` 完成：`dir: frontend` + `pnpm install`，并以 `pnpm-lock.yaml` / `node_modules` 做增量缓存。

## 4. pnpm 项目隔离

仓库内有两个相互独立的 pnpm 项目，**均不在仓库根**：

```text
vshell/                # 工作区（main）：根目录无 package.json / pnpm-workspace.yaml
└── docs/              # 文档站：package.json + pnpm-lock.yaml + pnpm-workspace.yaml

vshell-artifacts/      # 制品区（artifacts 分支）
└── frontend/          # 前端：package.json + pnpm-lock.yaml
```

- 根目录**不放** `pnpm-workspace.yaml`：一旦存在，`frontend/` 内执行的 `pnpm install` 会被重定向到 workspace 根，破坏 `wails3 build` 的前端构建
- `docs/pnpm-workspace.yaml` 仅用于 pnpm 11 的 `allowBuilds` 设置（允许 esbuild 安装脚本），不是包集合声明

## 5. 文档站部署

文档站由 GitHub Actions 自动部署（`.github/workflows/docs.yml`）：

- 触发条件：`main` 分支上 `docs/**` 或 workflow 文件变更，或手动触发
- 流程：`pnpm install --frozen-lockfile` → `vitepress build` → 发布到 `gh-pages` 分支（孤儿分支，删除的页面会同步消失）
- 访问地址：<https://crazy-airhead.github.io/vshell/>

---

## 延伸阅读

- [架构总览](architecture.md)
- [快速开始](../guide/getting-started.md)
