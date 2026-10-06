# GMark 设计语言（ui-redesign 实现版）

> **定位**：本文档是 GMark 自有设计语言的**落地规范**——以 IntelliJ IDEA
> 设计语言（见 [idea-ui-design-language.md](./idea-ui-design-language.md)，
> 下称「参考文档」）为源头，经 `ui-redesign` 分支实现与多轮定稿审核修订后的
> **实际生效版本**。数值与规则以本文档为准；与参考文档冲突处，本文档代表
> GMark 的显式决策（差异清单见 §13）。
> **基线**：`ui-redesign` 分支（基于 artifacts 40df13b），2026-09-10 定稿审核期。
> **维护**：改 token / 组件规格 / 交互规则时同步更新本文档对应条目。

---

## 目录

1. [设计原则](#1-设计原则)
2. [主题与 Token](#2-主题与-token)
3. [布局系统（Islands 岛式）](#3-布局系统islands-岛式)
4. [组件规范（G\* 家族）](#4-组件规范g-家族)
5. [编辑器区](#5-编辑器区)
6. [终端](#6-终端)
7. [Git 视觉与交互](#7-git-视觉与交互)
8. [AI 会话区](#8-ai-会话区)
9. [设置窗口](#9-设置窗口)
10. [交互模型](#10-交互模型)
11. [状态设计](#11-状态设计)
12. [数据与 Mock 纪律](#12-数据与-mock-纪律)
13. [与 IDEA 参考的差异决策记录](#13-与-idea-参考的差异决策记录)
14. [代码索引](#14-代码索引)
15. [M6 真实数据接入点](#15-m6-真实数据接入点)

---

## 1. 设计原则

从 IDEA 继承并经 GMark 审核确认的十条，另加 GMark 特有四条：

**继承（参考文档 §1 精选）**

1. 工具优先，内容至上——铬框低调，焦点留给编辑器与对话。
2. 一切皆 Token——颜色/度量出自单一事实源，组件零裸值。
3. 灰阶为主体，单一品牌蓝为强调——彩色只用于选中/焦点/链接/语义/VCS。
4. 分层背景表达层级，不靠阴影（阴影仅浮动层）。
5. 1px 边框 + 低透明度叠加做交互反馈。
6. 状态集完整：default/hover/pressed/selected/focused/disabled。
7. 键盘优先——速度搜索、双 Shift、命令注册表。
8. 明暗同构——token 镜像命名，切主题零结构变化。

**GMark 特有**

9. **岛式布局**——窗口是画布，所有面板是浮其上的圆角岛（Islands 模型）。
10. **命令即交互**——菜单/按钮/快捷键/搜索面板共用同一命令注册表。
11. **Mock 纪律**——UI 消费 facade 接口，mock 与 bindings 双实现可切换。
12. **双侧对称**——工作区与制品区是平级概念，视觉/交互规则两侧同构。

---

## 2. 主题与 Token

### 2.1 三层同源架构

```
frontend/src/design-system/tokens.ts     ← 唯一事实源（亮/暗成对镜像命名）
   ├─ theme-engine 注入 CSS 变量（:root 亮 + [data-theme=dark] 暗）
   ├─ uno.config.ts theme 引用 var(--*)
   └─ naive.ts 现读 token 值喂 NConfigProvider themeOverrides
```

- **Naive 联动注意（R1 教训）**：Naive 内部 JS 解析颜色，`var()` 字符串会
  静默失效（曾致主按钮回退绿色）。themeOverrides 必须**运行时现读**
  `getComputedStyle` 的真实值注入，不能传 `var()` 字符串。
- **规则**：组件零裸色值、零裸字号；圆角只用 {4,8,12,20}；间距只用
  {2,4,6,8,12,16,20,24}；动效 ≤300ms。

### 2.2 Islands 背景层级（两套同构）

**层次只存在于「岛 vs 画布」之间——所有岛（工具窗/编辑器/终端）同层同色。**

| 层 | 亮 | 暗 | 用途 |
| --- | --- | --- | --- |
| 画布 canvas | `#E9EAEE` | `#2B2D30` | 窗口底、岛间留白 |
| 岛（全部） | `#FFFFFF` | `#1E1F22` | 工具窗条、工具窗、编辑器、终端、状态栏 |
| 弹出/编辑器内容 | `#FFFFFF` | `#1E1F22` | 同岛层（弹层浮于岛上） |

- 暗**编辑器内容最深**（JetBrains「内容下沉」语义，参考文档 §8.1）。
- 深色反转气泡/Tooltip：`--bg-inverted`（亮 `#27282E` / 暗 `#393B40`）。

### 2.3 品牌色与选中

| Token | 亮 | 暗 | 说明 |
| --- | --- | --- | --- |
| `--focus` | `#3871E1` | `#3574F0` | Islands 品牌蓝（亮微调） |
| `--selection` | `#D0DFFE` | `#2E436E` | 列表/树选中块 |
| `--tab-selected-bg` | `#E3EBFE` | `#2E436E` | 标签激活填充 |
| `--hover` / `--hover-gray` | `#EDF3FF` / `#EBECF0` | `#393B40` / `#43454A` | 悬停 |

### 2.4 岛度量 Token

| Token | 值 |
| --- | --- |
| `--island-arc` / compact | `20px` / `16px`（岛圆角） |
| `--island-bw` / compact | `6px` / `4px`（岛间画布留白） |
| `--bar-arc` | `12px`（工具栏/状态栏/工具窗条圆角） |

### 2.5 常规度量（沿参考文档）

控件 28（紧凑 24）· 行 24 · 标签行 40 · 工具窗头 41 · 工具窗条 40×40 ·
主工具栏 40 · 状态栏 28 · 焦点环 2px accent。

### 2.6 双轨主题

- **通用主题**（UI 铬框）：亮/暗两套；`⌘⇧T` / 命令面板 / 跟随系统。
- **编辑器方案**：注册表 `design-system/editorScheme.ts`，首批 IDEA
  Light/Dark；默认跟随通用主题、可手动打破（设置页/命令面板）。
  Monaco / Muya+Prism / mermaid 三管线同一 token 源（`--ed-*` 组）。
- 全局等宽：Droid Sans Mono Slashed（woff2 内置
  `src/design-system/styles/fonts/`，零为本体刻零；仅 Regular 字重，
  粗/斜体由浏览器合成）；CJK 回退 PingFang SC。
- 切换心跳：`theme-engine.themeStamp`，一切主题消费方 watch 它重染
  （Muya forceRender、Monaco redefine、xterm 重赋 theme）。

---

## 3. 布局系统（Islands 岛式）

### 3.1 骨架

```
┌──────────────────────────────────────────────────────────┐
│ 画布（--bg-canvas）                                        │
│ ┌────────────────────────────────────────────────────┐   │
│ │ 主工具栏岛（40，bar-arc 12）                          │   │
│ └────────────────────────────────────────────────────┘   │
│ ┌──┐ ┌──────────┐ ┌──────────────────┐ ┌──┐             │
│ │左│ │左工具窗岛 │ │ 编辑器岛           │ │右│  ← 工具窗条 │
│ │条│ │（可选）    │ │（标签+分屏+内容）  │ │条│    常驻 40  │
│ └──┘ └──────────┘ └──────────────────┘ └──┘             │
│ ┌────────────────────────────────────────────────────┐   │
│ │ 底部工具窗岛（可选：终端×2 / Git 历史 / 输出）        │   │
│ └────────────────────────────────────────────────────┘   │
│ ┌────────────────────────────────────────────────────┐   │
│ │ 状态栏岛（28）                                        │   │
│ └────────────────────────────────────────────────────┘   │
└──────────────────────────────────────────────────────────┘
```

### 3.2 间隙三规则（多次返工后的定稿）

1. **岛间留白 = `--island-bw`（6px）**，由 flex `gap` 提供；所有接缝等宽。
2. **工具窗条常驻**——隐藏的是面板（工具窗），不是条；条是唤回入口。
3. **隐藏面板不得叠加间隙**——0 尺寸的工具窗元素以负 margin 吃掉自身
   占位的容器 gap（left: margin-right / right: margin-left / bottom:
   margin-top = `-island-bw`），margin 与尺寸同步过渡，滑入滑出无跳变。
   若不做，面板隐藏后接缝翻倍（6→12px）。

### 3.3 拖拽热区（分割条）

- 热区 = 岛间画布留白上的独立元素：**可见宽度 6px**（负 margin 抵消相邻
  gap，与非拖拽接缝视觉一致），命中区经透明伪元素两侧各扩 4px（总 14px）。
- 悬停/按住显示 `--selection` 色提示。
- 左/右面板宽 150–600，底部 ≥80，编辑器分屏 ≥200，AI 列 200–900。

### 3.4 面板显隐

- 条按钮点击 = toggle 该面板（同视图再点收起）；重开恢复「最后视图」。
- 显隐动画 300ms（宽/高 + 负 margin 过渡）。
- ⌘⇧F12 隐藏全部；状态栏右侧有三个面板快速开关（active 高亮）。

### 3.5 红绿灯（macOS 原生）

- 原生窗口（wails）下主工具栏/设置标题栏左侧预留 76px。
- **环境判定必须在 main.ts 首位求值**（`nativeEnv.ts`）：
  `@wailsio/runtime` 静态引用链会伪造 `window._wails`，运行时探测不可靠；
  wails 注入脚本先于应用 bundle 执行，首位捕获才可靠。

---

## 4. 组件规范（G\* 家族）

> 全部位于 `src/design-system/components/`；颜色/度量一律引用 token。

### 4.1 GIconButton

- 尺寸 sm=24 / md=30 / lg=40（工具窗条用 lg，图标 20）；悬停
  `--hover-overlay`、按下 `--pressed-overlay`、active = accent 实底白图标。
- 角标：右上 14px 胶囊（`--badge-blue-*`）。
- **工具窗头动作一律用它**（新建/刷新/折叠/更多…），不做面板内宽按钮。

### 4.2 GStripe（工具窗条）

- 宽 40 常驻；40×40 按钮；顶部 = 侧面板按钮，`#bottom` 槽 = 底部面板按钮。
- 左条：资源管理器/结构/提交/记忆/技能/AI；底部：工作区终端/Git 历史/输出。
- 右条：制品区/制品提交/Worktrees；底部：**制品区终端**（默认制品路径）。

### 4.3 GToolWindow（工具窗壳）

- 头 41：标题 + `#header-tabs` 槽（终端会话标签 / Git 标签）+ `#default` 槽动作按钮
  （`action` 事件透传 MouseEvent 供菜单定位）+ 隐藏钮（**短横线 icon**，
  Hide 语义，非关闭 X）。标签**靠左紧跟标题**（标题自留 `--space-5` 间距，
  标签间 `--space-4` 间距，不居中——G24 用户指正）。
- 四角全圆 `--island-arc`；收起 = 尺寸到 0 + 负 margin（§3.2-3）。
- 动作定义在使用方（AppShell `leftActions`/`rightActions` computed），
  组件保持领域无关。

### 4.4 GEditorTabs（标签）

- 行 40；标签 = **32px 高、四角 8px 圆角矩形填充块**，垂直居中浮于行内
  （Islands 激活语言；**不是**下划线，也**不是**与内容连体的文件夹页）。
- 状态：默认透明 0.75 → 悬停中性圆角背景 → 激活 `--tab-selected-bg`。
- **标签间留间距**：`.g-tabs-row` `gap: var(--space-3)`（6px）——圆角填充块
  贴死会糊成一条（G41）。
- **关闭钮常驻占位**：`visibility` 切换（禁 display:none），显隐零抖动；
  20px 命中 / 16px 图标 / `padding:0`（默认 padding 会致图标偏右）。
- **修改态小点与关闭钮同位叠放**（G28）：修改且未悬停显小点（纯指示器，
  不可点），悬停小点让位给关闭钮；激活未修改常显关闭钮——关闭路径任何
  状态下悬停可达。
- `placement="top|bottom"`：bottom 时分隔线在上（AI 标签栏置底）；
  `#append` 槽放「+」新建。
- 单标签 ≤300、上限 30、空列表整行隐藏。

### 4.5 GPopupMenu / GFileTree / GSearchEverywhere / GNotification / GTooltip / GEmptyState / GRecentFiles

- **GPopupMenu**：圆角 8、1px 边框、行 24、选中圆角 8 内缩 8；全局单例
  （`popup-menu.ts` 的 `openContextMenu`）；Esc 经级联栈。
- **GFileTree**：行 24、根行 26 加粗、缩进 12/级、根行入缩进体系（12 基准 +
  折叠 chevron），子行自 depth=1 起——层级关系可见（G29）；选中全行圆角 8、
  Git 着色（`--git-*`）；内置速度搜索（直接输入过滤，命中 mark 高亮、
  无匹配红字、2s 空闲重置、反色指示条）。
- **GSearchEverywhere**（双 Shift）：670×600 顶部居中；分组权重排序
  （最近文件→文件→动作→Git 分支）+ `@file/@action/@branch/@recent`
  前缀过滤。
- **GNotification**：右上深色反转、圆角 12、宽 350、10s 自动消失、
  同组 4s 窗口合并「还有 N 条」、>120 字截断可展开、粘性通道；
  提交/推送/拉取/锚点复制结果都走它（`composables/notify.ts`）。
- **GTooltip**：深色反转、延迟 1200ms。
- **GEmptyState**：h3 + 说明 + 主操作 + 快捷键提示；可聚焦（Esc 可达）。
- **GRecentFiles**（⌘E）：打开文档 + 最近文件，再按 ⌘E 切「仅已修改」。

---

## 5. 编辑器区

### 5.1 中心区布局（互斥完整占用）

| 文档 | AI 会话 | 中心区 |
| --- | --- | --- |
| 有 | 有 | 左右分屏（拖拽 200–900 + 状态栏比例预设） |
| 有 | 无 | 文档独占 |
| 无 | 有 | **AI 独占整区**（文件列整体隐藏，含空态） |
| 无 | 无 | 文档富空状态 |

- 文件列（含标签）与 AI 列是 EditorArea 的**横向**子布局；外层
  `.editor-col` 只做尺寸约束，**不得**干预内部 flex 方向（曾致堆叠错乱）。

### 5.2 双标签系统【决策】

- 文件标签（每窗格顶部）与会话标签（AI 面板底部）**并存、视觉同款**，
  仅以位置与图标区分——方向区分即语义区分。
- AI 底部标签栏带 `#append` 的「+」新建；新会话**追加到末尾**（标签
  顺序 = 时间正序）并激活。

### 5.3 分屏（GroupNode 递归布局树）

- 布局树：leaf = 窗格；split = 方向 + 子 A 尺寸（拖拽）。
- **每个窗格有自己独立的标签栏**（select/close/右键作用于本组）——
  拆分后新窗格即有标签。
- **叶子固定纵向**（标签在上、编辑器在下）；方向类只作用于 split
  节点（曾因叶子误继承 horizontal 致编辑器挤到标签右侧）。
- 标签右键：关闭/关闭其它/关闭全部/关闭未修改 + **向右/向下拆分** +
  复制路径 + 在 Finder 中打开（wails 走 RevealInFolder 绑定）。

### 5.4 Muya 集成契约（浮层定位，多轮返工定稿）

1. **段落前置按钮（🔗把手）挂在 `muya.domNode`（滚动内容根）内**，
   随内容滚动/重排，天然无漂移；`domNode` 内联 `position:relative`。
2. **容器相对坐标定位**：沿 offsetParent 链累加 offsetLeft/Top 到
   domNode——不经视口换算（视口/容器两套参照系混用会产生漂移）。
3. **滚出可视区隐藏**：参考块与「最近可滚动祖先」（通用查找，不依赖
   应用类名）的 rect 相交判断；不满足则移屏 + opacity 0。
4. **不越界**：浮层是容器子元素，`overflow` 裁剪兜底；`hide()` 中间件
   仅保留给 BaseFloat 系（其它浮层）。
5. **左 gutter 预留 48px**：标题锚点图标以 `left:-40px` 悬挂，容器
   padding 不足会伸出岛外、视觉落入标签栏。
6. **host 高度用 `min-height:100%`**（`height:100%` 在滚动容器内塌陷，
   会致内容穿透到标签栏）。
7. **标题锚点复制（🔗）闭环**：muya 发 `heading-copy-link {key}` →
   宿主经 getTOC 解析 `#githubSlug` → 剪贴板 + 通知；同 key 300ms 去抖。
8. **githubSlug 保留 unicode**：`[^\p{L}\p{N}\s_-]gu`（CJK 标题不再剥成
   `gmark-`；与 HTML 导出标题 id 同生成器，锚点可解析）。

### 5.5 查找栏

- ⌘F 编辑器顶部滑入；Enter/⇧Enter 逐个跳；Monaco 真实跳转，打字机
  计数（M5 补块内跳）；Esc 关闭（级联第一层）；切文档自动收起。

---

## 6. 终端

**两个完全独立的终端面板，互不共用**（定稿；曾试过「双列同屏」与
「单面板切换」均被否）：

| | 工作区终端 | 制品区终端 |
| --- | --- | --- |
| 入口 | 左条底部 | 右条底部 |
| 默认 cwd | 工作区路径 | 制品区路径 |
| 会话 | 独立计数/激活 | 独立计数/激活 |

- 每面板：整幅显示当前激活会话；**标题栏** = 该侧会话标签 +「+」新建。
- 会话命名：该侧首个无序号（`工作区` / `制品区`），第二个起
  `工作区 - 2`（短横线分隔）。
- 会话 v-show 保活，切换不丢缓冲区；标签悬停 × 关闭；空侧显示轻空态。
- 会话视图在 flex 容器内必须 `flex:1; min-width:0`——否则按内容收缩、
  xterm fit 按窄宽算列数、内容折成窄列。
- M6：mock shell → terminal/service PTY（CreateTerminal/WriteTerminal/
  terminal:data），会话模型已对齐。

---

## 7. Git 视觉与交互

- **分支状态只在三处**：状态栏左侧双分支部件（folders/package 图标，
  **纯显示无操作**，工作区带 ▲▼）；提交面板头；分支弹窗。主工具栏不放
  Git。
- **Git 操作归 Git 面板头**：工作区/制品区提交面板头部 = 分支弹窗（⌘⌥B）/
  推送（⌘⇧K）/ 拉取（⌘T）。
- 非模态提交（13.6）：状态分组复选树 + 行右键族 + 信息历史 + CharCount；
  **Commit 默认主按钮**（Enter 提交）+ Amend + 提交并推送；结果走通知。
- 历史（底部工具窗）：车道图（HEAD 黄/本地绿/远程紫、当前分支行背景）、
  行 26、refs 徽标、详情 + 变更文件（点击 diff）、过滤。
- gutter 修改条 4px 圆角 3 三色；单击迷你 diff 卡（词级开关 + F7 导航 +
  回滚此行）；blame 行尾注入注释（命令切换）。
- 通知：提交/推送/拉取结果按 `git:{target}` 分组。

### 7.1 差异窗口（双栏 diff，IDEA SimpleDiffViewer/DiffDividerDrawUtil 复刻）

**窗口与工具栏**：点变更文件弹独立窗口（`#/diff` 路由，命名窗口
`gmark-diff` 单例复用，localStorage 载荷交接，storage 事件原地更新）；
主题跟随（monaco gmark 主题自注册 + storage 重读设置）。工具栏（IDEA
导航块顺序）：在编辑器中打开 │ 上一处/下一处变更 + N/M 计数 │ 保存(⌘S，
可编辑差异) …… 双栏/统一 │ ⚙ 设置（软换行/字符级高亮）。

**可编辑差异**（提交面板本地变更）：左=基线只读，右=工作区可编辑；
⌘S 保存写回工作区（mock `saveFile` 状态机 → 重推差异载荷收敛）；历史
面板差异只读。逐块「>>」还原钮在左行号列旁。

**中心对称布局**：左编辑器行号自绘于**右缘**（monaco 行号关）、滚动条
翻**左外缘**；右编辑器行号保持左缘；行号/操作/蓝条聚拢在中间分隔带
两侧。左编辑器 gutter 空 margin 收零、右 gutter 背景透明（消除竖线）。
monaco 装饰（renderIndicators 箭头、角落工具栏、+/- 符号、折叠控件）
全部关闭/隐藏——中间列只有自绘内容。

**无虚拟空行**：一侧缺行（对侧插入/删除）时 monaco 塞 diagonal-fill
spacer——`collapseSpacers` 把两侧 spacer 视觉归零并按「行索引 × 行高」
重写 `.view-lines` 行内联 top；MutationObserver 持续校正（monaco 重渲染
会重写行 top）。滚动同步基于编辑器总高度，不受影响。

**连线规则**（三段式：左高亮区 + 中间贝塞尔连接带 + 右高亮区；颜色
修改=蓝/插入=绿/删除=灰 = `--git-g-*`）：

| 场景 | 左侧 | 右侧 | 连接带锚点 |
| --- | --- | --- | --- |
| 修改 | 块：首行顶→尾行底 | 块：首行顶→尾行底 | 左首行顶↔右首行顶（前向曲线），左尾行底↔右尾行底（回程） |
| 插入 | **线**（2px）：左 `oS` 行底部 | 块：首行顶→尾行底 | 线连右首行顶 + 线连右尾行底（一线展开成块） |
| 删除 | 块：首行顶→尾行底 | **线**（2px）：右 `mS` 行底部 | 左首行顶连到线 + 左尾行底连到线（块两边汇到线） |

- **线 y = 变更发生处上一行的底部**（「从此行之后插入/删除」的语义）：
  插入线取左侧 `oS` 行底（monaco 插入场景 oS 指变更点行）；删除线取右侧
  `mS` 行底（monaco 删除场景 mS 语义即「从此行后删除」，mS 行就是上一行，
  不再 -1）。
- **线取线所在侧的坐标**：两侧行布局因增删不同步（左侧含被删行），
  跨侧取值会错一行——插入线用左编辑器坐标、删除线用右编辑器坐标。
- **锚点一律视觉 DOM 坐标**：`yOf` 按行序查 `.view-lines` 子元素
  `getBoundingClientRect`（spacer 压缩后 monaco 内部坐标会漂移一行，
  不可用 `getTopForLineNumber`）；`lineBottom = yOf + 行高`。
- **曲线形态**：三次贝塞尔，控制点在分隔带宽度 0.45 处（IDEA
  `CTRL_PROXIMITY_X`），端点切线水平平滑衔接；线侧两点重合时保留 1px
  起止差保证细三角可渲染；连接带点击跳转变更、悬停加亮。

---

## 8. AI 会话区

- 会话列表（左工具窗）+ 中心区会话视图；**无文档且 AI 开启时 AI 独占
  中心区**（§5.1）。
- 会话视图：消息流（用户/AI 气泡 token 染色 + markdown-it 渲染）+ HITL
  审批卡（**自动滚动监听必须覆盖审批卡/todo/子代理出现**——只听流式文本
  会在流式暂停时把审批卡留在可视区外）+ todo/子代理/成本 + 输入区
  （模型选择 + ⌘⏎ 发送 + 中断）；自动滚动可被上滑打断。
- 流式纪律：mock 字符泵 + 事件族（ai:dsl/usage/todos/subagents/approval/
  turn-end），形状对齐现役 bindings 消费面；恢复续播不二次审批。

---

## 9. 设置窗口

IDEA 版式（独立原生窗口，浏览器测试开子窗口）：

```
┌────────────────────────────────────────┐
│ 标题栏（40）：[红绿灯 76] 设置           │ ← 原生留位；浏览器同显标题条
├──────────┬─────────────────────────────┤
│ 搜索      │ 面包屑（AI › AI 提供者）      │
│ 顶层分类  │ …────────────────────────── │
│ ▸ AI(组) │ 设置表单（Naive 控件 token 化）│
│   缩进项  │                             │
└──────────┴─────────────────────────────┘
     ↑ 竖线分隔（--border-strong）
```

- **搜索在分类侧**（不占标题栏下整行）；按名称/分组过滤，当前项被过滤
  掉时自动切首个可见项。
- **分类可折叠**（组头 chevron 旋转 + 加粗，点击收起/展开，组内项缩进）；
  搜索时分组自动展开。
- **无右上关闭钮**：原生有红绿灯，浏览器测试关标签页即可（Esc 绑定一并
  移除）。
- 外观/编辑器/行为三节真绑定 settings store（mock 持久化）；AI 各节占位。

---

## 10. 交互模型

### 10.1 命令注册表

`app/commands.ts`：一切交互 = 稳定 id + 标题 + 描述 + 图标 + 同义词；
菜单/工具栏/快捷键/Search Everywhere 共用；`runCommand(id)` 派发。

### 10.2 键位（ui-redesign 生效集）

| 动作 | 键 |
| --- | --- |
| Search Everywhere | 双 Shift |
| 命令面板（动作） | ⌘⇧P |
| Recent Files（再按=仅已修改） | ⌘E |
| 文件内查找 | ⌘F |
| 编辑模式（打字机/源码） | ⌘/ |
| 主题亮暗 | ⌘⇧T |
| 分支弹窗 | ⌘⌥B |
| 提交面板 | ⌘⌥K |
| 推送 / 拉取 | ⌘⇧K / ⌘T |
| 下一处/上一处变更 | F7 / ⇧F7 |
| 隐藏全部工具窗 | ⌘⇧F12 |
| 工具窗槽位 | ⌘⌥1 资源管理器 · ⌘⌥2 结构 · ⌘⌥3 提交 · ⌘⌥4 AI · ⌘⌥5 工作区终端 |
| AI 发送 | ⌘⏎ |

- 标题 ⌘1–6 / 正文 ⌘0 不变（编辑器优先）；Go 原生菜单回写随 M6。

### 10.3 Esc 级联

`escStack`（LIFO）：一次 Esc 只退一层——速度搜索 → 浮层（菜单/弹窗/
迷你卡/查找栏）→ 更外层。组件 onMounted 注册 / onUnmount 注销。

### 10.4 速度搜索

`composables/speedSearch.ts`：非连续子串匹配、命中高亮、无匹配红字、
↑↓ 导航、Enter 提交、2s 空闲重置。已接入 GFileTree；一切列表型 UI 目标。

### 10.5 状态栏布局（定稿）

- **左**：工作区分支（folders 图标 + ▲▼）· 制品区分支（package 图标）。
  **纯显示，无操作**。dev 场景切换器与假进度条已移除（场景经命令面板）。
- **中**：Ln:Col · 行·词 · UTF-8（产品固定编码，常显）· 文件格式
  （动态分类，无文档隐藏）。
- **右**：模式切换 · **比例预设 1:1/2:1/3:1/1:2/1:3**（仅 AI 开启可用，
  灰置 + tooltip；下限钳 200——钳 380 会让 2:1/3:1 与默认无差）·
  左/底/右面板开关（active 高亮）· 通知。

---

## 11. 状态设计

- 富空状态（编辑器/无最近项目）：h3 + 说明 + 主操作 + 快捷键提示，可聚焦。
- 轻空态（Git 无变更 / 终端空侧 / 列表无匹配）：一行文字 + 图标。
- 通知分级见 §4.4 GNotification。
- 演示场景（命令面板「演示场景：…」）：normal/clean/conflict/
  ai-streaming/slow/empty/fatal，驱动 mock 数据形态。

---

## 12. 数据与 Mock 纪律

- UI 只消费 `data/facade.ts` 接口（Settings/Workspace/Fs/Git/Ai 终端将补）；
  形状对齐现役 bindings 消费面（M6 零阻抗接入）。
- mock 可变状态真实流转（stage/commit/新建文件等），交互必真。
- 事件经 `data/bus.ts`（浏览器态）；M6 同名转发 WailsEvents，UI 无感。

---

## 13. 与 IDEA 参考的差异决策记录

| # | 参考 | GMark 决策 | 原因 |
| --- | --- | --- | --- |
| 1 | expUI 平面 + 1px 分割线 | **Islands 岛式**（画布+圆角岛） | 对齐最新 IDEA；层级由岛/画布表达 |
| 2 | 亮色岛分三层 | **所有岛同层**（亮全白/暗全 #1E1F22） | 岛是同一层容器；审核确认 |
| 3 | 编辑器标签下划线 4px | **圆角矩形填充块**（32px 行内居中） | Islands 激活语言；审核两轮修正 |
| 4 | 工具窗关闭 X | **短横线（Hide）** | 隐藏非销毁语义 |
| 5 | 终端双列同屏 | **两个独立面板**（整幅互斥） | 双列过窄影响操作 |
| 6 | 状态栏 Git 可点 | 分支**纯显示**；操作归 Git 面板头 | 职责归位 |
| 7 | 全局搜索左条入口 | 移除（双 Shift/命令面板覆盖） | 降冗余 |
| 8 | 设置：搜索整行+平铺分类 | 搜索在**分类侧**+**折叠分组**+面包屑 | IDEA 设置树版式 |
| 9 | 窗口右上关闭钮 | 移除（原生红绿灯/浏览器关标签） | 审核确认 |
| 10 | Tab 会话序号 `（2）` | 短横线 `工作区 - 2` | 审核偏好 |
| 11 | 工具窗条随面板隐藏（曾改） | **条常驻**；负 margin 修间隙叠加 | 条是唤回入口 |

---

## 14. 代码索引

| 领域 | 位置 |
| --- | --- |
| Token 单源 | `src/design-system/tokens.ts` |
| 主题引擎/心跳 | `src/design-system/theme-engine.ts` |
| 编辑器方案注册表 | `src/design-system/editorScheme.ts` |
| Naive 联动 | `src/design-system/naive.ts` |
| G\* 组件 | `src/design-system/components/` |
| Esc 级联/速度搜索/通知 | `src/design-system/composables/` |
| 应用壳/命令/快捷键 | `src/app/`（AppShell.vue · commands.ts · useAppShortcuts.ts） |
| 布局状态 | `src/app/stores/layout.ts` |
| 编辑器分屏/文档 | `src/features/editor/`（GroupNode.vue · documents.ts） |
| muya 集成 | `src/editor/useMuya.ts` + `packages/muya/src/ui/paragraphFrontButton/` |
| 终端 | `src/features/terminal/`（terminalStore · TerminalTabs · TerminalPanel · TerminalSessionView） |
| Git | `src/features/git/`（gitStore · CommitPanel · LogPanel · BranchPopup · MiniDiffCard） |
| AI | `src/features/ai/`（aiStore · AiArea · SessionsPanel） |
| 设置窗口 | `src/app/windows/SettingsWindow.vue` |
| 数据层 | `src/data/`（facade.ts · mock/ · bus.ts） |

---

## 15. M6 真实数据接入点

| mock | 真实 |
| --- | --- |
| `data/mock/*` facade 实现 | bindings 适配器（同接口注册切换） |
| `data/bus.ts` 事件 | WailsEvents 同名转发 |
| 终端 mock shell | terminal/service PTY |
| 场景切换器 | dev-only 开关 |
| 设置 localStorage | 后端 Settings（Get/Save） |
| 新建文件 mock fs | fs/service CreateFile |
| Worktrees 新建/清理 mock | git/service AddWorktree/PruneWorktrees |
| RevealInFolder（已接） | 保持 |
| Go 原生菜单键位回写 | internal/menu（以 §10.2 为事实源） |

---

*本文档由 ui-redesign 定稿审核过程沉淀；后续修改设计时先改这里再改代码。*
