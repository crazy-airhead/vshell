# 设置与主题

设置入口：ActivityBar 底部齿轮图标，或原生菜单 **vShell → Settings...（`⌘,`）**。520px 模态，分「界面 / 终端 / 快捷键」三个标签页，全部改动即时保存到 `localStorage`。

应用主题（浅色 / 深色）与语言（中 / 英）在**标题栏右上角**切换，`⌘⇧T` 也可切换主题。

---

## 1. 界面

| 设置项 | 取值 |
|--------|------|
| 字体 | System / PingFang SC / Microsoft YaHei / Noto Sans SC / Source Han Sans / Helvetica / Arial |
| 字号 | 11–18（滑块，刻度 11 / 13 / 16 / 18） |
| 主题色 | 8 个预设色块 |

> 「主题色」当前为占位设置：已存储但尚未接入界面主色变量，暂不产生可见效果。

## 2. 终端

| 设置项 | 取值 |
|--------|------|
| 字体 | Menlo / Monaco / Courier New / Source Code Pro / JetBrains Mono / Fira Code / Cascadia Code / Noto Sans Mono |
| 字号 | 10–24（滑块） |
| 配色方案 | 默认 / Solarized 暗色 / Solarized 浅色 / Dracula / Monokai / One Dark |

字体、字号、配色修改对已打开的终端**实时生效**（无需重连）。终端行列数由窗口尺寸自动 fit，无需手工设置。

## 3. 快捷键

7 项可自定义，点击条目进入「按键捕获」状态后按下新组合即可；「重置」恢复默认。

| 动作 | 默认 | 状态 |
|------|------|------|
| 新建连接 | `⌘/Ctrl+Shift+N` | 设置项存在，**暂未接线** |
| 关闭标签 | `⌘/Ctrl+W` | 由原生菜单实现 |
| 切换主题 | `⌘/Ctrl+Shift+T` | ✅ |
| 切换侧栏 | `⌘/Ctrl+B` | ✅ |
| 聚焦终端 | `` ⌘/Ctrl+` `` | 设置项存在，**暂未接线** |
| 复制（终端） | macOS `⌘C`；Win/Linux `Ctrl+Shift+C` | ✅ 终端内拦截 |
| 粘贴（终端） | macOS `⌘V`；Win/Linux `Ctrl+Shift+V` | ✅ 终端内拦截 |

全局快捷键在焦点位于输入框 / 文本域 / 下拉时自动跳过；终端内的键由终端单独处理（`Ctrl+C` / `Ctrl+V` 始终透传远端）。

## 4. 原生菜单（macOS）

| 菜单 | 快捷键 | 效果 |
|------|--------|------|
| vShell → Settings... | `⌘,` | 打开设置 |
| File → Save | `⌘S` | 保存当前激活的编辑器标签 |
| File → Close | `⌘W` | 断开当前标签所属连接并关闭标签 |
| Edit | `⌘Z` 等 | 系统标准编辑角色 |

## 5. 窗口

- 默认 1280×800，最小 960×600
- macOS 隐藏原生标题栏，使用自绘标题栏（30px，可拖拽移动窗口）；双击标题栏最大化
- 连接建立过程中标题栏下方有细进度条动画

---

## 延伸阅读

- [终端使用](terminal.md)
