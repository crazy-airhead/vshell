# vShell 设计语言（Islands 岛式）

> **定位**：本文档是 vShell 自有设计语言的**落地规范**——以 IntelliJ IDEA
> 设计语言（见仓库内 `idea-ui-design-language.md`，下称「IDEA 参考」）为源头、
> GMark 岛式实现（见仓库内 `gmark-design-language.md`，下称「GMark 参考」）
> 为岛式落地参照，针对 SSH 客户端形态定制。数值与规则以本文档为准。
> **基线**：artifacts 分支界面重写（2026-10），vShell 1.x。
> **维护**：改 token / 组件规格 / 交互规则时同步更新本文档对应条目，
> 先改这里再改代码。

---

## 目录

1. [设计原则](#1-设计原则)
2. [主题与 Token](#2-主题与-token)
3. [布局系统（Islands 岛式）](#3-布局系统islands-岛式)
4. [组件规范](#4-组件规范)
5. [终端区（中心岛）](#5-终端区中心岛)
6. [SFTP（底部工具窗）](#6-sftp底部工具窗)
7. [服务器监控（底部工具窗）](#7-服务器监控底部工具窗)
8. [连接树（左工具窗）](#8-连接树左工具窗)
9. [设置](#9-设置)
10. [交互模型](#10-交互模型)
11. [状态设计](#11-状态设计)
12. [与 IDEA / GMark 参考的差异决策记录](#12-与-idea--gmark-参考的差异决策记录)
13. [代码索引](#13-代码索引)

---

## 1. 设计原则

**继承（IDEA 参考 §1 精选八条）**

1. 工具优先，内容至上——铬框低调，焦点留给终端会话与文件操作。
2. 一切皆 Token——颜色/度量出自单一事实源，组件零裸值。
3. 灰阶为主体，单一品牌蓝为强调——彩色只用于选中/焦点/链接/语义/
   连接状态/图表。
4. 分层背景表达层级，不靠阴影（阴影仅浮动层：弹窗/通知）。
5. 1px 边框 + 低透明度叠加做交互反馈。
6. 状态集完整：default / hover / pressed / selected / focused / disabled。
7. 键盘优先——快捷键完备，鼠标是补充。
8. 明暗同构——token 镜像命名，切主题零结构变化。

**vShell 特有四条**

9. **岛式布局**——窗口是画布，所有面板是浮其上的圆角岛（Islands 模型，
   同 GMark）。
10. **会话为中心**——终端会话是一等公民：标签即会话、保活切换不丢缓冲、
    状态点常显。
11. **连接状态可视化**——连接的 connecting / online / offline / error
    四态有专属色 token，在连接树、标签、状态栏三处同色出现。
12. **Naive UI 为基座**——表单/弹窗/下拉等交互控件用 Naive UI 经 token
    桥接驱动；仅壳层（条/工具窗/标签行/状态栏）自绘 V\* 组件。不做全自绘。

---

## 2. 主题与 Token

### 2.1 三层同源架构

```
frontend/src/styles/global.css        ← 唯一事实源（亮暗成对镜像命名）
   ├─ uno.config.ts theme 引用 var(--*)
   └─ App.vue 运行时 getComputedStyle 现读真实值 → NConfigProvider themeOverrides
```

- **Naive 联动纪律（GMark R1 教训）**：Naive 内部 JS 解析颜色，`var()`
  字符串会静默失效。themeOverrides 必须运行时现读 `getComputedStyle`
  的真实值注入，且 computed 显式依赖 `settings.themeMode`（否则主题切换
  后不重读）。
- **规则**：组件零裸色值、零裸字号；圆角只用 {4,8,12,20}；间距只用
  {2,4,6,8,12,16,20,24}；动效 ≤300ms；ECharts / xterm / Monaco 等
  canvas 渲染器不能消费 `var()` 字符串，须经 `cssVar()` 解析真实值。

### 2.2 Islands 背景层级

**层次只存在于「岛 vs 画布」之间——所有岛同层同色。**

| 层 | 亮 | 暗 | 用途 |
| --- | --- | --- | --- |
| 画布 canvas | `#E9EAEE` | `#2B2D30` | 窗口底、岛间留白（接缝） |
| 铬条 chrome | `#E9EAEE` | `#2B2D30` | 工具栏、左条、状态栏——**与画布同色
  （`--bg-chrome` = canvas，融画布）**，三者与画布构成**一体外框**，
  仅内容面板（左/底部工具窗、中心岛）为浮岛（同 GMark 决策） |
| 岛（内容面板） | `#FFFFFF` | `#1E1F22` | 左工具窗、中心岛、底部工具窗 |
| 嵌入控件 | `#FFFFFF` | `#2B2D30` | 输入框、代码片、嵌入面板底 |
| 反转气泡 | `#27282E` | `#393B40` | tooltip / 深色反转层 |

- 终端内容背景跟随**终端配色方案**（不强制岛色）；方案为 `default` 时
  从主题 token 派生（暗 `#1E1F22` / 亮 `#FFFFFF`），与岛同层。
- 原生窗口底色（`main.go` BackgroundColour）取暗画布值，避免启动闪白。

### 2.3 色彩 Token 总表

**强调与交互（IDEA 蓝，产品决策）**

| Token | 亮 | 暗 | 说明 |
| --- | --- | --- | --- |
| `--color-primary` | `#3871E1` | `#3574F0` | 品牌蓝：主按钮/焦点/条按钮激活 |
| `--selection` | `#D0DFFE` | `#2E436E` | 列表/树选中块 |
| `--tab-selected-bg` | `#E3EBFE` | `#2E436E` | 标签激活填充 |
| `--hover` | `#EDF3FF` | `#393B40` | 可选项悬停 |
| `--hover-gray` | `#EBECF0` | `#43454A` | 行/标签中性悬停 |
| `--hover-overlay` | `rgba(0,0,0,.04)` | `rgba(255,255,255,.10)` | 图标钮悬停叠加 |
| `--pressed-overlay` | `rgba(0,0,0,.08)` | `rgba(255,255,255,.15)` | 按下叠加 |
| `--link` | `#315FBD` | `#6B9BFA` | 链接 |

**文字**

| Token | 亮 | 暗 |
| --- | --- | --- |
| `--text-primary` | `#000000` | `#DFE1E5` |
| `--text-secondary` | `#494B57` | `#B4B8BF` |
| `--text-muted` | `#6C707E` | `#9DA0A8` |
| `--text-disabled` | `#A8ADBD` | `#5A5D63` |
| `--text-inverted` | `#F0F1F2` | `#CED0D6` |

**边框**（暗色决策偏差见 §12-7）

| Token | 亮 | 暗 |
| --- | --- | --- |
| `--border-color` | `#DFE1E5` | `#323538` |
| `--border-strong` | `#C9CCD6` | `#43454A` |
| `--border-input` | `#C9CCD6` | `#4E5157` |

**语义**

| Token | 亮 | 暗 |
| --- | --- | --- |
| `--color-success` | `#208A3C` | `#57965C` |
| `--color-warning` | `#A46704` | `#BA9752` |
| `--warning-accent` | `#FFAF0F` | `#F2C55C` |
| `--color-error` | `#DB3B4B` | `#DB5C5C` |
| `--color-info` | `#3369D6` | `#6B9BFA` |

**连接状态（vShell 特有）**——树节点圆点、终端标签圆点、状态栏共用：

| Token | 亮 | 暗 | 语义 |
| --- | --- | --- | --- |
| `--status-connecting` | `#C77D0B` | `#F2C55C` | 连接建立中 |
| `--status-online` | `#208A3C` | `#57965C` | 会话在线 |
| `--status-offline` | `#A8ADBD` | `#6F737A` | 已断开 |
| `--status-error` | `#DB3B4B` | `#DB5C5C` | 连接失败 |

**监控图表**（ECharts 经 `cssVar()` 解析）

| Token | 亮 | 暗 |
| --- | --- | --- |
| `--chart-cpu` | `#3871E1` | `#6E9FF7` |
| `--chart-mem` | `#208A3C` | `#5CA56B` |
| `--chart-disk` | `#8C5CE0` | `#A98BEB` |
| `--chart-net-rx` | `#0E9CB8` | `#4FC3F7` |
| `--chart-net-tx` | `#D9632E` | `#FF8A65` |
| `--stat-bar-track` | `#EBECF0` | `#393B40` |

**传输方向**

| Token | 亮 | 暗 |
| --- | --- | --- |
| `--transfer-up` | `#3871E1` | `#3574F0` |
| `--transfer-down` | `#208A3C` | `#57965C` |

**阴影**（仅浮动层）：`--shadow-elevated` 亮 `0 8px 24px rgba(0,0,0,.16)`
/ 暗 `0 8px 24px rgba(0,0,0,.5)`。

### 2.4 岛度量 Token（主题无关，`:root` 定义）

| Token | 值 | 用途 |
| --- | --- | --- |
| `--island-arc` / `-compact` | `12px` / `12px` | 岛圆角（工具窗/中心岛，产品
  决策：12px） |
| `--island-bw` / `-compact` | `6px` / `4px` | 岛间画布留白 |
| `--bar-arc` | `12px` | 工具栏/左条/状态栏圆角（铬条融画布后视觉不可见） |
| `--traffic-light-w` | `76px` | 工具栏红绿灯预留 |

### 2.5 控件度量（沿 IDEA 参考 §5）

控件 28（紧凑 24）· 行 24 · 工具窗头 41 · 左条钮 40×40（图标 20）·
工具栏高 40 · 状态栏 28 · 标签行 40 · 标签药丸 32 · 焦点环 2px accent ·
圆角阶梯 4/8/12/20 · 间距阶梯 2/4/6/8/12/16/20/24（`--space-1..8`）。

### 2.6 双轨主题

- **通用主题**（UI 铬框）：亮/暗两套，`data-theme` 属性驱动 CSS 变量；
  `⌘⇧T` / 工具栏按钮 / 设置页切换；localStorage 持久化，默认暗色。
- **终端配色方案**：注册表独立（默认/Solarized 暗/亮/Dracula/Monokai/
  One Dark）；`default` 跟随通用主题，其余固定；修改对已开会话实时生效。
- Monaco 主题跟随通用主题（`vs` / `vs-dark`，watcher 实时切换）。
- 字体：UI 跟随设置（系统栈为默认，可换）；终端等宽字体独立设置；
  编辑器用 `--font-mono` 栈。

---

## 3. 布局系统（Islands 岛式）

### 3.1 骨架

```
┌──────────────────────────────────────────────────────────┐
│ 画布（--bg-canvas）                                        │
│ ┌────────────────────────────────────────────────────┐   │
│ │ 主工具栏岛（40，bar-arc 12，左 76px 红绿灯留位，拖拽区）│   │
│ └────────────────────────────────────────────────────┘   │
│ ┌──┐ ┌──────────┐ ┌──────────────────────┐               │
│ │左│ │左工具窗岛 │ │ 中心岛                 │               │
│ │条│ │(连接/密钥/ │ │ 会话标签 40            │               │
│ │通│ │ 配置/转发/ │ │ xterm / Monaco        │               │
│ │高│ │ 证书)      │ │                      │               │
│ │  │ └──────────┘ └──────────────────────┘               │
│ │  │ ┌──────────────────────────────────┐               │
│ │  │ │ 底部工具窗岛（监控 | SFTP，头 41）  │               │
│ │  │ └──────────────────────────────────┘               │
│ └──┘                                                    │
│ ┌────────────────────────────────────────────────────┐   │
│ │ 状态栏岛（28，bar-arc 12）                           │   │
│ └────────────────────────────────────────────────────┘   │
└──────────────────────────────────────────────────────────┘
```

- **左条通高**：自主工具栏岛下缘到状态栏岛上缘，高度固定；底部
  工具窗岛展开只压缩中心区（中央列内部），不压缩左条。
- 监控与 SFTP **同岛不同标签**（产品决策：底部工具窗聚合）；两面板
  同时挂载、`v-show` 切换（保监控选择态与 SFTP 浏览状态）。
- 中心岛**永不收起**——终端是一等公民，无会话时显示富空态。

### 3.2 间隙三规则（沿 GMark 定稿）

1. **岛间留白 = `--island-bw`（6px）**，由 flex `gap` 提供；画布四周
   padding 同值；所有接缝等宽。
2. **左条常驻**——隐藏的是左工具窗岛，不是条；条是唤回入口。
3. **隐藏面板不得叠加间隙**——0 尺寸的工具窗以负 margin 吃掉自身占位
   的容器 gap（`margin-right: -island-bw`），margin 与尺寸同步过渡
   （300ms），滑入滑出无跳变。不做则接缝翻倍（6→12px）。

### 3.3 拖拽热区（分割条）

- 热区 = 岛间画布留白上的独立元素：**可见宽度 6px**（负 margin 抵消
  相邻 gap，与非拖拽接缝视觉一致），命中区经透明伪元素两侧各扩 4px
  （总 14px）。
- 悬停/按住显示 `--selection` 色提示。
- 左工具窗宽 150–600；底部工具窗高 80–600；尺寸持久化
  （localStorage `vshell:layout`），加载时钳制。

### 3.4 面板显隐

- 条按钮点击 = toggle（同视图再点收起）；重开恢复「最后视图」
  （activeSidebar 记忆）。
- 显隐动画 300ms（宽/高 + 负 margin 过渡）。
- `⌘B` 翻转左工具窗；状态栏右侧有三个面板快速开关（active 高亮）。

### 3.5 红绿灯（macOS 原生）

- 无边框窗口（`MacTitleBarHidden` + `InvisibleTitleBarHeight 48`，
  对齐 GMark——红绿灯纵向位置与 40px 工具栏视觉居中），主工具栏
  岛左侧预留 `--traffic-light-w`（76px），区内不放置任何交互元素。
- 主工具栏岛为**唯一窗口拖拽区**（`-webkit-app-region: drag`，双击
  最大化）；工具栏内按钮 `no-drag`。

---

## 4. 组件规范

> 壳层自绘组件位于 `components/shell/`，前缀 V\*；表单/弹窗/下拉/
> 消息等交互控件一律 Naive UI（经 §2.1 桥接），不自绘。

### 4.1 VIconButton

- 尺寸 sm=24 / md=28 / lg=40（左条用 lg，图标 20）；悬停
  `--hover-overlay`、按下 `--pressed-overlay`、active = accent 实底
  白图标。
- 工具窗头动作一律用它（刷新/新建/折叠/更多…），不做面板内宽按钮。

### 4.2 VStripe（左条）

- 宽 40 常驻，**通高**（工具栏岛下缘到状态栏岛上缘，不随底部工具窗
  展开而压缩）；40×40 按钮；顶部 = 五个左工具窗入口（连接/密钥/SSH
  配置/端口转发/证书），底部 = 监控、SFTP（激活高亮）+ 设置齿轮。
- 激活态 = accent 实底圆角 8 + 白图标（Islands 语言；**不是**左侧
  2px 指示条）。
- tooltip 全部走 i18n。

### 4.3 VToolWindow（工具窗壳）

- 圆角 `--island-arc`（12）；头 41：标题（**基准 −1 = 12px**，IDEA
  medium 档，secondary，常规字重——相对偏移 `calc(var(--font-size-base)
  - 1px)`，跟随用户字号缩放）+
  `#tabs` 槽（底部工具窗的 监控|SFTP 药丸标签，靠左紧跟标题，间距
  `--space-3`）+ `#actions` 槽（VIconButton 24px）+ 隐藏钮（**短横线
  minus 图标**，Hide 语义，非关闭 X）。
- 收起 = 尺寸到 0 + 负 margin（§3.2-3），300ms，overflow hidden，
  禁 display:none / v-if 卸载。
- 面板自身的历史「头部行」上移为 `#actions` 动作钮（消灭各面板重复
  的头部条与 `.panel-action-btn`）。

### 4.4 标签语言（中心岛会话标签）

- 行 `--tab-row-h`（40）不横滚；标签 = **32px 高、四角 8px 圆角矩形
  填充块**，垂直居中浮于行内（Islands 激活语言；**不是**下划线）。
- 状态：默认透明、文字 0.75 透明度 → 悬停 `--hover-gray` → 激活
  `--tab-selected-bg`。
- **标签间留间距 6px**（`--space-3`）——圆角填充块贴死会糊成一条。
- 单标签 ≤300（截断 + tooltip）。
- **超宽折叠（GMark GEditorTabs 同款，IDEA「放不下即隐藏」）**：标签
  **保自然宽度不缩水**（`flex:0 0 auto`；极窄行经 `--v-tab-max` 随行
  收缩，下限 100）；放不下的按 **LRU（激活 → 最近使用 → 原始序）折叠
  隐藏**（激活标签恒可见）；被折叠标签脱离流但仍可测量（隐藏于
  -9999px）。行右端出现「全部标签」下拉钮（列表项含状态点 / 前缀
  色标 / 脏点，选中即激活并回带显示）；行内**滚轮 = 按最近使用序平移
  可见窗口**（skip ±1）。面板宽度 / 标签数 / 标题 / 字体变化时重算
  （ResizeObserver 逐标签 + 行观察）。
- 终端标签：7px 状态点（`--status-online` / `--status-offline`）+
  标题。
- 编辑器标签：`[远程]`/`[本地]` 前缀色标（`--color-info` /
  `--color-success`）+ **脏点与关闭钮同位叠放**：脏且未悬停显小点
  （纯指示器不可点），悬停小点让位给关闭钮；关闭钮 `visibility`
  切换（禁 display:none），20px 命中 / 16px 图标 / `padding: 0`。
- 右键菜单：复制会话 / 重连 / 断开 / 关闭 / 关闭其它 / 关闭全部
  （NDropdown）。
- 无「+」新建钮——终端会话从连接树双击创建。

### 4.5 VStatusBar（状态栏岛）

- 高 28，`bar-arc` 圆角；左中右三区：
  - **左**：活动连接名 + 状态点 + 状态文字（四态文案）+ 会话计数
    `{n} 个会话`。纯显示。
  - **中**：传输摘要（文件名 · 百分比 · 速度，tabular-nums；来自
    transfers store），点击直达底部 SFTP 标签；空闲时隐藏。
  - **右**：左面板 / 监控 / SFTP 三个开关钮（active 高亮）。
- 部件间距 `--space-3`；文字 `--font-size-xs`。

### 4.6 Naive 基座约定

- 弹窗（NModal）圆角 12、内边距沿 Naive；下拉/菜单圆角 8；
  tooltip 深色反转（`--bg-inverted`）。
- 表单控件高度 28（Naive medium）；密度紧凑处用 small（24）。
- 禁止浏览器原生 `alert/confirm/prompt`——一律 `useMessage` /
  `useDialog`。

---

## 5. 终端区（中心岛）

- 中心岛 = 会话标签行（§4.4）+ 内容堆叠 + 富空态（无会话时）。
- **保活契约**：所有会话视图同时挂载于 absolute 堆叠
  （`absolute inset-0 invisible pointer-events-none` → 激活
  `!visible !pointer-events-auto`），**禁 v-if / display:none**——
  切换标签不丢滚动缓冲与 WebGL 上下文。
- 内容视图必须 `flex:1; min-width:0`——否则按内容收缩、xterm fit 按
  窄宽算列数、内容折成窄列。
- xterm 初始化管线不动：ResizeObserver → fit + rAF 延迟首绘 +
  `term.refresh`（WebGL 0 高度重绘 bug 规避）。
- 终端 I/O 走 Wails Events（`terminal:stdin/stdout/resize`），
  红线不变。
- 断开会话：状态点转 `--status-offline`，终端内黄字提示 + 回车重连。
- Monaco 编辑器标签（远程文件 / 本地文件 / ssh config 原文）与终端
  标签同款视觉；`⌘S` 保存；脏点语义见 §4.4。

---

## 6. SFTP（底部工具窗）

- 双栏 commander：左远程（弹性 6）/ 右本地（弹性 4），中缝可拖。
- 每栏：面包屑路径（双击进入编辑）+ 目录树 + 可排序文件表；行 24
  节奏；选中 `--selection`；悬停 `--hover`。
- 传输条（底部）：方向色 `--transfer-up` / `--transfer-down`、
  百分比 tabular-nums；全部完成 2s 后自动收起。
- 拖拽语义：远程↔本地互拖、OS 文件拖入上传（`native:file-drop`）、
  ghost 用 `--bg-inverted` + 圆角 8。
- 打开远程文件 → 中心岛编辑器标签（`[远程]` 前缀）。
- 无活动连接时显示轻空态。

## 7. 服务器监控（底部工具窗）

- 左列 280px：主机信息、运行时长、CPU/内存/磁盘状态条（轨道
  `--stat-bar-track`、条高 6 圆角 3、阈值色
  success/warning-accent/error）、Top5 进程（可点选）。
- 右侧详情：CPU 核心、内存、磁盘、网络（ECharts 折线 + 接口/进程
  表）。
- **ECharts 纪律**：色值经 `cssVar()` 解析（canvas 不认 `var()`）；
  主题切换 watch 重渲；容器 ResizeObserver 里补 `chartInstance.resize()`；
  面板隐藏（offsetParent === null）时跳过渲染，显现后 nextTick 重渲。

## 8. 连接树（左工具窗）

- 行高：分组行 24（单行）；连接行**双行**自适应（约 34，`node-props`
  挂 `.conn-node` 类区分）；层级缩进 8px（`NTree :indent`，紧凑树）；
  分组可折叠（点箭头或双击分组行切换）；拖拽调整归属；双击连接即连接。
- 全行选中圆角 8、`--selection`；悬停 `--hover`。
- 连接行双行内容：状态点 + 名称（主色，截断）⏎ host:port
  （`--text-muted`，截断）+ 悬停动作钮（连接/编辑/删除，24px 命中）。
- 状态点**只报在线**：在线 = `--status-online` 绿点；未连接不显点
  （保留占位）。**前缀占位统一**：分组图标 14px / 70% 透明；状态点
  7px 居中于 14px 槽——两者间距均 4px，分组行与连接行文字起点对齐。
- 已连接连接用 `--status-online` 点标识；连接中 `--status-connecting`。
- 新建/重命名分组走行内输入，不弹窗。
- 其余左工具窗（密钥/SSH 配置/端口转发/证书）同规范：行 24、动作
  归工具窗头 `#actions`、状态用 `--status-*` / 语义色。

## 9. 设置

- 保留模态形态（单窗口工具，非 IDE 多窗口）：NModal 520，三标签
  「界面 / 终端 / 快捷键」，改动即时保存 localStorage。
- 界面：字体、字号（11–18）。**无主题色选择器**（单一品牌蓝，见
  §12-6）。
- 终端：字体、字号（10–24）、配色方案（7 项注册表）。
- 快捷键：7 项可自定义（按键捕获 + 重置）。
- 主题与语言切换在主工具栏岛右侧常驻。

## 10. 交互模型

### 10.1 键位表

| 动作 | 键 |
| --- | --- |
| 切换主题 | `⌘/Ctrl+Shift+T` |
| 切换左工具窗 | `⌘/Ctrl+B` |
| 新建连接（占位待接线） | `⌘/Ctrl+Shift+N` |
| 关闭标签 | `⌘/Ctrl+W`（原生菜单） |
| 设置 | `⌘,`（原生菜单） |
| 保存（编辑器） | `⌘/Ctrl+S`（原生菜单） |
| 复制/粘贴（终端内） | macOS `⌘C/⌘V`；Win/Linux `Ctrl+Shift+C/V` |

- 输入框/文本域聚焦时全局快捷键自动跳过；终端内按键透传远端。
- 原生菜单动作经 Wails Events（`menu:settings/save/close-tab`）。

### 10.2 Esc 语义

弹窗（Naive）内 Esc = 取消/关闭；终端聚焦时 Esc 透传远端，不被
UI 捕获。

## 11. 状态设计

- **连接四态**：connecting（黄点+「连接中…」）/ online（绿点+「在线」）/
  offline（灰点+「已断开」）/ error（红点+「错误」）。三处同色：树、
  标签、状态栏。
- **富空态**（中心岛无会话）：标题 + 操作提示 + 快捷键提示（「在连接
  面板中双击连接即可开始」）。
- **轻空态**（SFTP 无连接 / 监控无数据 / 列表无匹配）：一行文字 +
  图标。
- 通知走 Naive message/dialog；进度反馈优先内联（传输条/状态栏摘要），
  短任务不弹模态。

## 12. 与 IDEA / GMark 参考的差异决策记录

| # | 参考 | vShell 决策 | 原因 |
| --- | --- | --- | --- |
| 1 | GMark 全自绘 G\* 组件家族 | **Naive UI 基座 + V\* 壳**（仅条/工具窗/标签/状态栏自绘） | vShell 表单密集（连接/密钥/证书/转发/设置），Naive 已有基建与主题桥接；自绘壳只做岛式语言 |
| 2 | GMark 双侧对称（工作区/制品区） | **单左条 + 底部聚合** | SSH 客户端单工作区；监控与 SFTP 共岛标签（产品决策） |
| 3 | GMark 两个独立终端面板 | **单中心岛多标签** | 终端是主体而非附件；FinalShell 用户习惯 |
| 4 | GMark/IDEA 设置独立窗口 | **设置保留模态** | 单窗口工具形态；避免多窗口管理成本 |
| 5 | IDEA 状态栏 Git 部件 | **连接状态 + 传输摘要 + 面板开关** | 领域不同：连接状态即「分支」 |
| 6 | vShell 旧版 8 色主题色板（占位） | **移除**，单一品牌蓝 | 岛式 accent 是联动色组（primary/selection/tab-selected/hover × 亮暗），单 hex 无法派生六个伴生值 |
| 7 | IDEA 暗色边框 `#1E1F22` | **提亮为 `#323538`** | vShell 在岛内画 1px 分隔线，`#1E1F22` 在 `#1E1F22` 岛上不可见 |
| 8 | GMark 编辑器双标签系统（文件+AI 会话） | **单标签系统**（终端+编辑器混排） | 无 AI 会话域；编辑器标签即临时文档 |
| 9 | IDEA 键位体系（速度搜索/双 Shift） | **暂不引入**，保留现有 7 项 | 工具体量小，先行保持简单；列为后续候选 |
| 10 | GMark 标签「+」新建 | **无「+」** | 会话从连接树创建，语义不同 |

## 13. 代码索引

| 领域 | 位置（artifacts 分支 `frontend/src/`） |
| --- | --- |
| Token 单源 | `styles/global.css` |
| UnoCSS 映射 | `uno.config.ts` |
| Naive 桥接 | `App.vue`（getComputedStyle 现读注入） |
| 应用壳 | `components/shell/`（AppShell · ToolbarIsland · LeftStripe · ToolWindow · EditorIsland · EditorTabs · StatusBarIsland） |
| 布局状态 | `stores/layout.ts`（持久化 `vshell:layout`） |
| 终端 | `components/terminal/`（XTerminal · EditorTab）+ `composables/useTerminalManager.ts` |
| SFTP | `components/sftp/`（SFTPPanel · SFTPArea）+ `composables/useDragTransfer.ts` |
| 监控 | `components/monitor/MonitorPanel.vue` |
| 连接树 | `components/sidebar/`（ConnectionTree · ConnectionFormModal） |
| 其余左工具窗 | `components/keys/` · `components/config/` · `components/panels/PortForwardPanel.vue` · `components/cert/` |
| 设置 | `components/settings/SettingsModal.vue` |
| 分割条 | `components/common/DraggableDivider.vue` |

---

*本文档与 artifacts 分支界面重写同步定稿；后续修改设计时先改这里再改代码。*
