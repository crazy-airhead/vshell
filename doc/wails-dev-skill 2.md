# Wails v3 + Vue 3 全栈开发技能

基于 GMark 项目提炼的 Wails v3 桌面应用开发模式与规范。

---

## 一、技术栈速查

| 层级 | 技术 | 版本 |
|------|------|------|
| 框架 | Wails v3 | alpha.92 |
| 后端 | Go | 1.26 |
| 前端 | Vue 3 + TypeScript | 3.2+ |
| 构建 | Vite | 8.x |
| 样式 | UnoCSS | 66.x |
| 状态 | Pinia | 3.x |
| 编辑器 | Monaco Editor | 0.55 |
| 终端 | xterm.js | 6.x |
| 图标 | @iconify/vue + lucide | 5.x |

---

## 二、项目结构约定

```
gmark/
├── main.go                     # 应用入口：注册 Service + 创建窗口
├── internal/                   # Go 后端（每个子包一个领域 Service）
│   ├── config/config.go        # 配置管理（TOML，~/.gmark/）
│   ├── project/service.go      # 项目管理 Service
│   ├── fs/service.go           # 文件系统 Service
│   ├── git/service.go          # Git 操作 Service
│   └── markdown/service.go     # Markdown 渲染 Service
├── frontend/
│   ├── bindings/               # 【自动生成】勿手动编辑
│   ├── src/
│   │   ├── utils/wails.ts      # 手工封装的服务调用层
│   │   ├── stores/             # Pinia Store（Composition API）
│   │   ├── views/              # 页面级组件
│   │   ├── editors/            # 编辑器组件
│   │   ├── panels/             # 面板组件（每个面板一个子目录）
│   │   ├── components/         # 通用组件
│   │   └── styles/variables.css # CSS 变量主题
│   ├── vite.config.ts          # Vite + Wails 插件 + UnoCSS
│   └── uno.config.ts           # UnoCSS 配置
└── Taskfile.yml                # Task 构建任务
```

---

## 三、后端 Service 开发模式

### 3.1 Service 注册

在 `main.go` 中通过 `application.NewService()` 注册：

```go
func main() {
    app := application.New(application.Options{
        Name: "GMark",
        Services: []application.Service{
            application.NewService(project.NewService()),
            application.NewService(fs.NewService()),
            application.NewService(git.NewService()),
            application.NewService(markdown.NewService()),
            // 新 Service 在此添加
        },
        Assets: application.AssetOptions{
            Handler: application.AssetFileServerFS(assets),
        },
        Mac: application.MacOptions{
            ApplicationShouldTerminateAfterLastWindowClosed: true,
        },
    })
    // ...
}
```

### 3.2 Service 编写规范

每个 Service 遵循固定结构：

```go
package mydomain

// 1. 数据模型（JSON tag 必写，Wails 用 JSON 序列化）
type MyModel struct {
    ID   string `json:"id"`
    Name string `json:"name"`
}

// 2. Service 结构体（无需嵌入 base，无 ctx 字段）
type Service struct {
    // 可包含内部状态（map、config 等）
    // 不可包含 Wails context（Wails v3 不需要）
}

// 3. 构造函数
func NewService() *Service {
    return &Service{}
}

// 4. 公开方法（首字母大写 = 暴露给前端）
// 参数和返回值必须可 JSON 序列化
func (s *Service) DoSomething(param1 string, param2 int) (*MyModel, error) {
    // ...
}
```

**规则：**
- 公开方法自动暴露给前端，无需额外注解
- 参数/返回值仅使用基本类型或 JSON-tagged struct
- 错误通过 `error` 返回，前端以 Promise reject 接收
- 需持久化的配置统一使用 `internal/config` 包

### 3.3 新增 Service 完整示例

以新增 `search` 服务为例：

**Step 1 — 创建 `internal/search/service.go`**

```go
package search

type SearchResult struct {
    Path    string `json:"path"`
    Line    int    `json:"line"`
    Content string `json:"content"`
}

type Service struct{}

func NewService() *Service {
    return &Service{}
}

func (s *Service) SearchFiles(query string, rootPath string) ([]SearchResult, error) {
    // 实现搜索逻辑
    return results, nil
}
```

**Step 2 — 在 `main.go` 注册**

```go
application.NewService(search.NewService()),
```

**Step 3 — 重新生成绑定**

```bash
wails3 generate bindings
```

绑定文件自动生成到 `frontend/bindings/github.com/airhead/gmark/internal/search/`。

### 3.4 配置管理

配置文件路径：`~/.gmark/config.toml`，使用 `pelletier/go-toml/v2`。

```go
// 读取配置
cfg, err := config.Load()

// 修改并保存
cfg.AddProject(entry)
cfg.Save()
```

配置结构使用 `toml` tag：

```go
type ProjectEntry struct {
    ID            string    `json:"id" toml:"id"`
    WorkspacePath string    `json:"workspacePath" toml:"workspace_path"`
    LastOpenedAt  time.Time `json:"lastOpenedAt" toml:"last_opened_at"`
}
```

---

## 四、前端服务调用层

### 4.1 wails.ts 封装模式

自动生成的绑定直接使用较冗长，需在 `utils/wails.ts` 中封装：

```typescript
import * as Search from '../../bindings/github.com/airhead/gmark/internal/search/service'

export const services = {
  // ... 已有服务

  search: {
    searchFiles: (query: string, rootPath: string) =>
      Search.SearchFiles(query, rootPath),
  },
}
```

**封装原则：**
- 统一入口 `services.{domain}.{method}`
- 转换枚举参数（如 `toTarget(source)` 将字符串转为 Target 枚举）
- 导出类型供 Store 和组件使用：
  ```typescript
  export type { SearchResult } from '../../bindings/github.com/airhead/gmark/internal/search/models'
  ```

### 4.2 类型导入

Wails 自动生成的类型从 `bindings/` 导入：

```typescript
// 正确
import type { FileEntry } from '../../bindings/github.com/airhead/gmark/internal/fs/models'
// 或通过 wails.ts 中转导出
export type { FileEntry } from '../../bindings/github.com/airhead/gmark/internal/fs/models'
```

---

## 五、Pinia Store 模式

### 5.1 Store 编写规范

统一使用 Composition API 风格：

```typescript
import { ref, computed } from 'vue'
import { defineStore } from 'pinia'
import { services } from '../utils/wails'

export const useXxxStore = defineStore('xxx', () => {
  // ─── State ───────────────────
  const items = ref<Item[]>([])
  const loading = ref(false)

  // ─── Getters ─────────────────
  const activeItem = computed(() => items.value.find(i => i.active))

  // ─── Actions ─────────────────
  async function loadItems(): Promise<void> {
    loading.value = true
    try {
      items.value = await services.xxx.list()
    } finally {
      loading.value = false
    }
  }

  // ─── Return ──────────────────
  return { items, loading, activeItem, loadItems }
})
```

**规则：**
- State 用 `ref()`，Getters 用 `computed()`
- 异步操作统一 `try/finally` 管理 loading 状态
- 不在 Store 中直接操作 DOM
- 返回对象明确列出所有暴露的属性和方法

### 5.2 在组件中使用

```typescript
const store = useXxxStore()

// 读取状态（自动响应式）
const items = store.items

// 调用 action
await store.loadItems()
```

---

## 六、组件编写规范

### 6.1 组件分类

| 类型 | 目录 | 示例 |
|------|------|------|
| 页面视图 | `views/` | MainLayout.vue, WelcomeView.vue |
| 编辑器 | `editors/` | MonacoWrapper.vue, MarkdownEditor.vue |
| 面板 | `panels/{name}/` | panels/git-panel/GitPanel.vue |
| 通用组件 | `components/` | StatusBar.vue |

### 6.2 SFC 结构

统一顺序：`<template>` → `<script setup>` → `<style scoped>`

```vue
<template>
  <div class="my-panel">
    <!-- 使用 Icon 组件 -->
    <Icon icon="lucide:search" :width="16" />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { Icon } from '@iconify/vue'
import { useXxxStore } from '../stores/xxx'

const store = useXxxStore()
</script>

<style scoped>
.my-panel {
  width: 100%;
  height: 100%;
  display: flex;
  flex-direction: column;
}
</style>
```

### 6.3 面板组件模板

面板组件统一使用 flex column 布局：

```vue
<template>
  <div class="xxx-panel">
    <!-- 工具栏（可选） -->
    <div class="toolbar">
      <button @click="refresh">
        <Icon icon="lucide:refresh-cw" :width="14" />
        Refresh
      </button>
    </div>
    <!-- 内容区 -->
    <div class="content">
      <!-- ... -->
    </div>
  </div>
</template>

<style scoped>
.xxx-panel {
  width: 100%;
  height: 100%;
  display: flex;
  flex-direction: column;
}
.toolbar {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 4px 8px;
}
.content {
  flex: 1;
  overflow: auto;
  padding: 8px;
  min-height: 0;  /* 关键：防止 flex 子元素溢出 */
}
</style>
```

### 6.4 图标使用

使用 `@iconify/vue` + `lucide` 图标集：

```vue
<script setup>
import { Icon } from '@iconify/vue'
</script>

<template>
  <Icon icon="lucide:folder" :width="22" />
  <Icon icon="lucide:git-branch" :width="14" />
</template>
```

常用图标：`folder`, `folder-open`, `file`, `file-code`, `file-text`, `search`, `bot`, `terminal`, `git-branch`, `send`, `refresh-cw`, `plus`, `x`, `bookmark`

---

## 七、布局集成

### 7.0 整体布局结构

```
┌─────────────────────────────────────────────────────────────────────────┐
│  TitleBar (30px) — 项目名称 | -webkit-app-region: drag (双击最大化)     │
├────┬───────────────────────────────────────────────────────────────┬────┤
│    │                                                               │    │
│ L  │  MiddleArea                                                   │ R  │
│ e  │  ┌──────────┐         ┌─────────────────────┐  ┌──────────┐  │ i  │
│ f  │  │          │ resize  │                     │  │          │  │ g  │
│ t  │  │  Left    │ handle  │                     │  │  Right   │  │ h  │
│    │  │  Panel   │   (V)   │    EditorArea       │  │  Panel   │  │ t  │
│ A  │  │          │         │                     │  │          │  │    │
│ c  │  │ Explorer │         │  ┌─ TabBar ──────┐ │  │ Artifacts│  │ A  │
│ t  │  │ Search   │         │  │ [W]file.md ●× │ │  │          │  │ c  │
│ i  │  │          │         │  ├───────────────┤ │  │          │  │ t  │
│ v  │  │ 250px    │         │  │               │ │  │  250px   │  │ i  │
│ i  │  │ default  │         │  │  Monaco /     │ │  │ default  │  │ v  │
│ t  │  │ min:150  │         │  │  Markdown /   │ │  │ min:150  │  │ i  │
│ y  │  │          │         │  │  Image        │ │  │          │  │ t  │
│    │  │          │         │  │               │ │  │          │  │ y  │
│ B  │  │          │         │  ├───────────────┤ │  │          │  │ B  │
│ a  │  │          │         │  │ Breadcrumb    │ │  │          │  │ a  │
│ r  │  └──────────┘         │  └───────────────┘ │  └──────────┘  │ r  │
│    │                      │                     │               │    │
│ ┌──┤                      └─────────────────────┘               │ ┌──┤
│ │  ├─ spacer ──────────────┬───────────────────────────────────┤ │  │
│ │  │                       │                                   │ │  │
│ AI│  resize handle (H)     │                                   │ │ 📦│
│ │  ├───────────────────────┤                                   │ │  │
│ TM│  Bottom Panel          │                                   │ │  │
│ │  │ ┌───────────────────┐ │                                   │ │  │
│ GT│  │ panel-header      │ │                                   │ │  │
│ │  │ ├───────────────────┤ │                                   │ │  │
│ OT│  │                   │ │                                   │ │  │
│ │  │  AI / Terminal /   │ │                                   │ │  │
│ └──┤  Git / Output      │ │                                   │ └──┤
│    │  200px default      │ │                                       │
│    │  min:100            │ │                                       │
│    │  └───────────────────┘ │                                       │
│ 42 ├────────────────────────┘                                       │
│ px │                                                                  │38p
│    │  StatusBar (24px)                                               │x │
│  📁 │  GMark │ W:main │ A:main │          │ Ln 12, Col 5 │ UTF-8 │ MD│  📦 │
│  🔍 ├──────────────────────────────────────────────────────────────────┤    │
│    │                                                                  │    │
├────┴───────────────────────────────────────────────────────────────┴────┤
│  --bg-base (#1e1e1e) 间距背景，面板使用 --bg-primary (#2b2b2b) 圆角卡片│
└─────────────────────────────────────────────────────────────────────────┘
```

#### ActivityBar 结构

```
 Left ActivityBar (42px)        Right ActivityBar (38px)
 ┌────────────────────┐         ┌────────────────────┐
 │  .activity-top     │         │                    │
 │  ┌──────────────┐  │         │  ┌──────────────┐  │
 │  │ 📁 Workspace │  │         │  │ 📦 Artifacts │  │
 │  └──────────────┘  │         │  └──────────────┘  │
 │  ┌──────────────┐  │         │                    │
 │  │ 🔍 Search    │  │         │                    │
 │  └──────────────┘  │         │                    │
 │                    │         │                    │
 │  (space-between)   │         │                    │
 │                    │         │                    │
 │  .activity-bottom  │         │                    │
 │  ┌──────────────┐  │         │                    │
 │  │ 🤖 AI Chat   │  │         │                    │
 │  └──────────────┘  │         │                    │
 │  ┌──────────────┐  │         │                    │
 │  │ ⬛ Terminal  │  │         │                    │
 │  └──────────────┘  │         │                    │
 │  ┌──────────────┐  │         │                    │
 │  │ 🔀 Git       │  │         │                    │
 │  └──────────────┘  │         │                    │
 │  ┌──────────────┐  │         │                    │
 │  │ 📜 Output    │  │         │                    │
 │  └──────────────┘  │         │                    │
 └────────────────────┘         └────────────────────┘

 活跃按钮：
   左侧按钮 → 左边缘出现 2px accent 蓝色条
   右侧按钮 → 右边缘出现 2px accent 蓝色条
 点击同一按钮 → 折叠/展开 toggle
```

#### 面板卡片结构

```
 所有面板统一结构：

 ┌─────────────────────────┐  ← border-radius: var(--card-radius) 4px
 │ panel-header   (30px)   │  ← font-size:11px, uppercase, --text-secondary
 ├─────────────────────────┤
 │                         │
 │  panel-body  (flex:1)   │  ← overflow:auto, min-height:0
 │                         │
 │                         │
 └─────────────────────────┘

 面板间通过 var(--gap) 1px 间距分开
 间距处显示 --bg-base (#1e1e1e) 背景色
 形成视觉上的 1px 分隔线效果

 Resize Handle:
   垂直（V）: width = var(--gap), cursor: col-resize
   水平（H）: height = var(--gap), cursor: row-resize
   hover 时 background 变为 var(--accent) 蓝色高亮
```

#### 编辑器区域结构

```
 EditorArea
 ┌─────────────────────────────────────────────────────────┐
 │ TabBar (34px)                                           │
 │ ┌──────────┐ ┌──────────┐ ┌──────────┐                │
 │ │[W]file.md│ │[A]code.go│ │[W]readme │  ← scroll →    │
 │ │    ●  ×  │ │       ×  │ │       ×  │                │
 │ └──────────┘ └──────────┘ └──────────┘                │
 │  ▬▬▬▬▬▬▬▬▬  ← active tab 底部 2px accent 下划线       │
 ├─────────────────────────────────────────────────────────┤
 │                                                         │
 │  EditorContent (flex:1)                                 │
 │  ┌───────────────────────────────────────────────────┐ │
 │  │                                                   │ │
 │  │  根据 Tab.mode 切换：                              │ │
 │  │    'code'  → MonacoWrapper   (通用代码编辑)        │ │
 │  │    'live'  → MarkdownEditor  (代码 + 预览分栏)     │ │
 │  │    'image' → ImagePreview    (图片居中预览)        │ │
 │  │                                                   │ │
 │  └───────────────────────────────────────────────────┘ │
 │                                                         │
 │  空状态（无 Tab 时）:                                   │
 │  ┌───────────────────────────────────────────────────┐ │
 │  │            </> (48px icon)                        │ │
 │  │    Open a file from the file tree to start editing│ │
 │  │               Cmd+P to quick open                │ │
 │  └───────────────────────────────────────────────────┘ │
 ├─────────────────────────────────────────────────────────┤
 │ Breadcrumb (22px)                                       │
 │ WORKSPACE  /path/to/file.md     ← source 徽章 + 路径   │
 └─────────────────────────────────────────────────────────┘

 Tab 结构:
 ┌────────────────────┐
 │ [W] filename.md ● ×│
 └────────────────────┘
  ↑       ↑        ↑ ↑
  来源   文件名  脏  关闭
  徽章            标记 按钮

 来源徽章: W=workspace(--warning 黄色), A=artifacts(--success 绿色)
 脏标记: ● (--warning 黄色), 仅 isDirty 时显示
 关闭按钮: × 仅 hover 时显示(opacity: 0→1)
```

#### Markdown 编辑器布局

```
 MarkdownEditor (mode === 'live')
 ┌─────────────────────────────────────────────────────────┐
 │ Toolbar (32px)                                          │
 │ [H1][H2][H3][B][I][</>][•][1.][🔗][🖼][⊞]             │
 │  ↑   ↑   ↑   ↑  ↑   ↑   ↑  ↑   ↑   ↑   ↑            │
 │  H1  H2  H3  Bld Ita Code UL OL Link Img Tbl           │
 ├──────────────────────────┬──────────────────────────────┤
 │                          │                              │
 │  Monaco Editor           │  HTML Preview               │
 │  (markdown language)     │  (goldmark 渲染结果)         │
 │                          │                              │
 │  flex:1                  │  flex:1                     │
 │  编辑内容变更时          │  实时更新                     │
 │  自动调用后端渲染        │                              │
 │                          │                              │
 └──────────────────────────┴──────────────────────────────┘
               ↕ border-left: 1px solid var(--border)
```

#### 欢迎页布局

```
 WelcomeView（无项目时显示）
 ┌─────────────────────────────────────────────────────────┐
 │                                                         │
 │                    (全屏居中, max-width:480px)           │
 │                                                         │
 │                      🔖 (48px)                          │
 │                       GMark                             │
 │              AI-driven creative editor                  │
 │                                                         │
 │                  ┌──────────────┐                       │
 │                  │ + New Project│                       │
 │                  └──────────────┘                       │
 │                                                         │
 │                  RECENT PROJECTS                        │
 │                  ────────────────                       │
 │                  📁 My Project                          │
 │                     /path/to/project                    │
 │                                                         │
 │                  ── 点击 New Project 后弹出 Modal ──    │
 │                  ┌──────────────────────┐               │
 │                  │ Create Project       │               │
 │                  │                      │               │
 │                  │ Project Name         │               │
 │                  │ ┌──────────────────┐ │               │
 │                  │ │                  │ │               │
 │                  │ └──────────────────┘ │               │
 │                  │                      │               │
 │                  │ Project Directory    │               │
 │                  │ ┌────────────┬─────┐ │               │
 │                  │ │ /path/...  │ 📂  │ │               │
 │                  │ └────────────┴─────┘ │               │
 │                  │ workspace/ &          │               │
 │                  │ artifacts/ auto-create│               │
 │                  │                      │               │
 │                  │         [Cancel][Create]│             │
 │                  └──────────────────────┘               │
 │                                                         │
 └─────────────────────────────────────────────────────────┘
```

### 7.1 添加左侧面板

**Step 1 — 在 layout.ts 添加类型和状态**

```typescript
export type LeftView = 'explorer' | 'search' | 'newPanel'
```

**Step 2 — 在 MainLayout.vue 添加按钮**

```typescript
const leftItems = [
  { id: 'explorer', label: 'Workspace', icon: 'lucide:folder' },
  { id: 'search', label: 'Search', icon: 'lucide:search' },
  { id: 'newPanel', label: 'New Panel', icon: 'lucide:star' },
]
```

**Step 3 — 在 MainLayout.vue 添加面板内容**

```vue
<template v-if="layoutStore.leftActiveView">
  <div class="side-panel" :style="{ width: layoutStore.leftPanelWidth + 'px' }">
    <div class="panel-header">
      <span>{{ leftItems.find(i => i.id === layoutStore.leftActiveView)?.label }}</span>
    </div>
    <div class="panel-body">
      <NewPanel v-if="layoutStore.leftActiveView === 'newPanel'" />
    </div>
  </div>
</template>
```

### 7.2 添加底部面板

底部面板按钮在左侧 ActivityBar 的 `.activity-bottom` 区域：

```typescript
const bottomItems = [
  { id: 'ai', label: 'AI Chat', icon: 'lucide:bot' },
  { id: 'newBottom', label: 'New Bottom', icon: 'lucide:star' },
]
```

底部面板内容在 `main-column` 的底部：

```vue
<NewBottomPanel v-else-if="layoutStore.bottomActiveView === 'newBottom'" />
```

### 7.3 添加右侧面板

右侧面板按钮在右侧 ActivityBar：

```typescript
const rightItems = [
  { id: 'artifacts', label: 'Artifacts', icon: 'lucide:package' },
  { id: 'newRight', label: 'New Right', icon: 'lucide:star' },
]
```

---

## 八、样式系统

### 8.1 CSS 变量

所有颜色通过 CSS 变量引用，定义在 `styles/variables.css`：

```css
/* 使用方式 */
.my-element {
  background: var(--bg-primary);
  color: var(--text-primary);
  border: 1px solid var(--border);
  border-radius: var(--card-radius);
}
.my-element:hover {
  background: var(--bg-hover);
  color: var(--text-bright);
}
```

**关键变量：**
- 背景：`--bg-base`（最底） → `--bg-primary`（面板） → `--bg-secondary`（工具栏） → `--bg-hover`
- 文本：`--text-secondary`（标签） → `--text-primary`（正文） → `--text-bright`（标题）
- 强调：`--accent` / `--accent-hover`
- 状态：`--success` / `--warning` / `--danger`
- 间距：`--gap: 1px`，圆角：`--card-radius: 4px`

### 8.2 样式约定

- 所有组件使用 `<style scoped>` 防止样式泄漏
- 不使用传统 border 分隔面板，通过 `--bg-base` 间距背景自然分隔
- 面板圆角统一 `border-radius: var(--card-radius)`
- 滚动条样式全局定义在 `variables.css`，组件无需重复
- 深色/浅色切换：根元素添加/移除 `.light-theme` class

### 8.3 交互状态色

```css
/* 按钮/列表项 hover */
background: var(--bg-hover);

/* 按钮/列表项 active/selected */
background: var(--bg-active);

/* 输入框 focus */
border-color: var(--accent);

/* 文件状态 */
.status-added { color: var(--success); }
.status-modified { color: var(--warning); }
.status-deleted { color: var(--danger); }
```

---

## 九、编辑器集成

### 9.1 添加新编辑模式

在 `stores/editor.ts` 的 `resolveMode()` 中添加判定：

```typescript
function resolveMode(name: string): Tab['mode'] {
  const lower = name.toLowerCase()
  if (/\.(png|jpe?g|gif|svg|webp|bmp|ico)$/i.test(lower)) return 'image'
  if (/\.(md|markdown)$/i.test(lower)) return 'live'
  if (/\.(diff|patch)$/i.test(lower)) return 'diff'  // 新增
  return 'code'
}
```

在 `EditorArea.vue` 中添加组件：

```vue
<DiffViewer
  v-else-if="editorStore.activeTab.mode === 'diff'"
  :key="editorStore.activeTabId ?? ''"
  :tab="editorStore.activeTab"
/>
```

### 9.2 Monaco Editor 配置

Monaco 编辑器统一配置：

```typescript
monaco.editor.create(container, {
  language: getLanguage(filename),  // 根据扩展名映射
  theme: 'vs-dark',                // 深色主题
  fontSize: 13,
  fontFamily: "'JetBrains Mono', 'Fira Code', Menlo, Monaco, monospace",
  lineNumbers: 'on',
  minimap: { enabled: true },
  scrollBeyondLastLine: false,
  automaticLayout: true,           // 自动适应容器
  padding: { top: 8 },
})
```

语言映射表在 `MonacoWrapper.vue` 的 `languageMap` 中维护。

### 9.3 编辑器快捷键

在 Monaco 实例上注册：

```typescript
editor.addAction({
  id: 'save-file',
  label: 'Save File',
  keybindings: [monaco.KeyMod.CtrlCmd | monaco.KeyCode.KeyS],
  run: async (ed) => {
    await services.fs.writeFile(tab.path, ed.getValue())
    editorStore.setDirty(tab.id, false)
  },
})
```

---

## 十、端到端添加新功能

以添加「全局搜索」功能为例，完整流程：

### Step 1 — Go 后端

创建 `internal/search/service.go`：

```go
package search

type Match struct {
    Path    string `json:"path"`
    Line    int    `json:"line"`
    Content string `json:"content"`
}

type Service struct{}

func NewService() *Service { return &Service{} }

func (s *Service) SearchContent(query, rootPath string, caseSensitive bool) ([]Match, error) {
    // 实现搜索...
    return matches, nil
}
```

在 `main.go` 注册：

```go
application.NewService(search.NewService()),
```

生成绑定：

```bash
wails3 generate bindings
```

### Step 2 — 前端服务层

在 `utils/wails.ts` 添加：

```typescript
import * as Search from '../../bindings/github.com/airhead/gmark/internal/search/service'

export type { Match } from '../../bindings/github.com/airhead/gmark/internal/search/models'

export const services = {
  // ... 已有
  search: {
    searchContent: (query: string, rootPath: string, caseSensitive: boolean) =>
      Search.SearchContent(query, rootPath, caseSensitive),
  },
}
```

### Step 3 — Store

创建 `stores/search.ts`：

```typescript
import { ref } from 'vue'
import { defineStore } from 'pinia'
import { services, type Match } from '../utils/wails'

export const useSearchStore = defineStore('search', () => {
  const results = ref<Match[]>([])
  const loading = ref(false)

  async function search(query: string, rootPath: string): Promise<void> {
    loading.value = true
    try {
      results.value = await services.search.searchContent(query, rootPath, false)
    } finally {
      loading.value = false
    }
  }

  function clear(): void {
    results.value = []
  }

  return { results, loading, search, clear }
})
```

### Step 4 — UI 组件

创建 `panels/search/SearchPanel.vue`，遵循面板组件模板。

### Step 5 — 集成到布局

如果作为左侧面板：更新 `layout.ts` 类型 → 更新 `MainLayout.vue` 的 `leftItems` → 添加面板渲染。

如果作为底部面板：更新 `bottomItems` → 添加面板渲染。

---

## 十一、事件通信

### 11.1 Go → 前端（Event 推送）

```go
// Go 端发送
import "github.com/wailsapp/wails/v3/pkg/application"

// 在 Service 中无法直接访问 app，需通过窗口或其他方式
// Wails v3 事件模式：
runtime.EventsEmit(ctx, "event:name", payload)
```

```typescript
// 前端监听
import { EventsOn } from '@wailsio/runtime'

EventsOn('event:name', (data) => {
  store.handleEvent(data)
})
```

### 11.2 前端 → Go（Binding 调用）

标准调用模式，无需事件：

```typescript
const result = await services.xxx.method(param)
```

---

## 十二、开发与构建

### 12.1 日常开发

```bash
# 安装前端依赖
cd frontend && npm install

# 开发模式（热重载）
wails3 dev
# 或使用 Task
task dev
```

### 12.2 绑定生成

修改 Go Service 的公开方法后，必须重新生成绑定：

```bash
wails3 generate bindings
```

生成位置：`frontend/bindings/github.com/airhead/gmark/internal/{domain}/`

### 12.3 构建发布

```bash
# 当前平台构建
wails3 build

# 指定平台
wails3 build -platform darwin/arm64
wails3 build -platform darwin/amd64
wails3 build -platform windows/amd64
wails3 build -platform linux/amd64
```

### 12.4 注意事项

- 修改 Go 代码后需要重启 `wails3 dev`
- 前端代码修改自动热重载
- `bindings/` 目录由工具生成，勿手动编辑
- `//go:embed all:frontend/dist` 要求构建前先 `npm run build`
- macOS 窗口使用 `InvisibleTitleBarHeight: 28` + `MacTitleBarHidden` 实现自定义标题栏

---

## 十三、常见模式速查

### 有状态 Service（如 Git）

```go
type Service struct {
    repos map[Target]*git.Repository  // 内部状态
    paths map[Target]string
}

func NewService() *Service {
    return &Service{
        repos: make(map[Target]*git.Repository),
        paths: make(map[Target]string),
    }
}
```

### 无状态 Service（如 Markdown）

```go
type Service struct {
    md goldmark.Markdown  // 初始化一次的引擎
}

func NewService() *Service {
    md := goldmark.New(goldmark.WithExtensions(...))
    return &Service{md: md}
}
```

### 对话框（系统原生）

```typescript
import { Dialogs } from '@wailsio/runtime'

// 选择目录
const path = await Dialogs.OpenFile({
  Title: 'Select Directory',
  CanChooseDirectories: true,
  CanChooseFiles: false,
})
```

### 窗口操作

```typescript
import { Window } from '@wailsio/runtime'

Window.ToggleMaximise()
```

### 双区域模式

后端使用 Target 枚举区分 workspace/artifacts：

```go
type Target string
const (
    Workspace Target = "workspace"
    Artifacts Target = "artifacts"
)
```

前端统一转换：

```typescript
function toTarget(source: string): Target {
  return source === 'workspace' ? Target.Workspace : Target.Artifacts
}
```

---

## 十四、排错指南

| 问题 | 原因 | 解决 |
|------|------|------|
| 前端找不到 binding 类型 | 未生成绑定 | `wails3 generate bindings` |
| Go 方法前端不可调用 | 方法非公开或参数不可序列化 | 确保首字母大写 + JSON tag |
| 面板溢出/滚动异常 | flex 子元素缺少 `min-height: 0` | 添加 `min-height: 0; overflow: auto` |
| Monaco 不随容器调整 | 缺少 `automaticLayout: true` | 创建时设置该选项 |
| 主题不生效 | CSS 变量未定义 | 检查 `variables.css` 是否被导入 |
| macOS 标题栏重叠 | 未设置 `InvisibleTitleBarHeight` | 窗口选项中设置 28px + Hidden |
