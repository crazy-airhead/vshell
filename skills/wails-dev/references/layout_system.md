# 布局系统

## 主布局结构

```
┌──────────────────────────────────────────────────┐
│                Title Bar (30px)                    │
│  [vShell]     -webkit-app-region: drag     [🌙][中]│
│                                          连接进度条 │
├────┬─────────┬───────────────────────────────────┤
│    │         │                                   │
│ A  │ Sidebar │         Terminal Pane              │
│ c  │         │                                   │
│ t  │ 200-    │  ┌─Tab Bar (36px)──────────────┐  │
│ i  │ 500px   │  │ [Tab1] [Tab2] [Tab3]       │  │
│ v  │ ←drag→  │  ├────────────────────────────┤  │
│ i  │         │  │                            │  │
│ t  │         │  │   xterm.js Terminal        │  │
│ y  │         │  │   (ResizeObserver → fit)   │  │
│    │         │  │                            │  │
│ B  │─────────│  └────────────────────────────┘  │
│ a  │ (drag)  │                                   │
│ r  ├─────────┴───────────────────────────────────┤
│    │           Bottom Panel                       │
│(48)│           ←drag (h)→                         │
│    │                                              │
└────┴──────────────────────────────────────────────┘
```

## 布局组件层次

```
App.vue
├── NConfigProvider (Naive UI 主题)
│   └── NMessageProvider + NDialogProvider
│       └── 主容器 div.flex.flex-col.w-screen.h-screen
│           ├── Title Bar (30px) — -webkit-app-region: drag
│           ├── 主区域 div.flex.flex-1
│           │   ├── ActivityBar (48px)
│           │   └── 内容区 div.flex-1.flex.flex-col.p-1.5
│           │       ├── 上半部 div.flex.flex-1
│           │       │   ├── Sidebar (可选，DraggableDivider)
│           │       │   └── TerminalPane
│           │       └── 下半部 (可选，DraggableDivider)
│           │           └── BottomPanel
│           └── SettingsModal
```

## ActivityBar

左侧 48px 宽的活动栏，点击按钮切换 Sidebar 内容：

```vue
<ActivityBar @open-settings="showSettings = true" />
```

Sidebar 内容根据 `layout.activeSidebar` 切换：

```vue
<ConnectionTree v-if="layout.activeSidebar === 'connections'" />
<KeyManagementPanel v-else-if="layout.activeSidebar === 'keys'" />
<SSHConfigPanel v-else-if="layout.activeSidebar === 'ssh-config'" />
<PortForwardPanel v-else-if="layout.activeSidebar === 'port-forward'" />
```

## Sidebar 宽度管理

```typescript
// stores/layout.ts
const sidebarWidth = ref(260)

function setSidebarWidth(v: number) {
  sidebarWidth.value = Math.min(500, Math.max(200, v))
}
```

```vue
<div :style="{ width: layout.sidebarWidth + 'px' }">
  <!-- Sidebar 内容 -->
</div>
<DraggableDivider
  direction="vertical"
  :modelValue="layout.sidebarWidth"
  @update:modelValue="(v: number) => layout.setSidebarWidth(v)"
  :min="200"
  :max="500"
/>
```

## Bottom Panel 高度管理

```typescript
// stores/layout.ts
const bottomPanelHeight = ref(200)

function setBottomPanelHeight(v: number) {
  bottomPanelHeight.value = Math.min(600, Math.max(80, v))
}
```

```vue
<DraggableDivider
  direction="horizontal"
  :modelValue="layout.bottomPanelHeight"
  @update:modelValue="(v: number) => layout.setBottomPanelHeight(v)"
  :min="80"
  :max="600"
  :invert="true"
/>
<div :style="{ height: layout.bottomPanelHeight + 'px', flexShrink: 0, minHeight: 0 }">
  <BottomPanel />
</div>
```

## DraggableDivider 组件

共享组件 `components/common/DraggableDivider.vue`：

```vue
<DraggableDivider
  direction="vertical | horizontal"
  :modelValue="currentSize"
  @update:modelValue="setSize"
  :min="80"
  :max="600"
  :invert="false"  // true 时反向拖拽
/>
```

## 添加新面板

### 添加 Sidebar 面板

1. 在 `stores/layout.ts` 的 `activeSidebar` 类型中添加新值
2. 在 `ActivityBar.vue` 添加新按钮
3. 在 `App.vue` 的 sidebar 区域添加条件渲染

```vue
<MyNewPanel v-else-if="layout.activeSidebar === 'my-panel'" />
```

### 添加 Bottom 面板

1. 在 `stores/layout.ts` 添加新的 activeView 类型
2. 在 BottomPanel 组件中添加新标签页/视图

## Flex 布局关键技巧

```css
/* 防止 flex 子元素溢出 — 所有面板容器必须加 */
.content {
  flex: 1;
  overflow: auto;
  min-height: 0;  /* 关键：防止 flex 子元素撑开父容器 */
}

/* 面板标准结构 */
.panel {
  width: 100%;
  height: 100%;
  display: flex;
  flex-direction: column;
}
.panel-header { flex-shrink: 0; }
.panel-body { flex: 1; overflow: auto; min-height: 0; }
```

## Title Bar

macOS 自定义标题栏：

```typescript
// main.go 窗口配置
Mac: application.MacWindow{
    InvisibleTitleBarHeight: 30,
    TitleBar:                application.MacTitleBarHidden,
},
```

```html
<!-- App.vue -->
<div class="h-[30px] flex items-center justify-center"
     style="-webkit-app-region: drag">
  <span class="text-xs font-semibold">vShell</span>
  <div style="-webkit-app-region: no-drag">
    <!-- 按钮 -->
  </div>
</div>
```

双击标题栏最大化：`@dblclick="Window.ToggleMaximise()"`

## 窗口配置

```go
// main.go
win := wailsApp.Window.NewWithOptions(application.WebviewWindowOptions{
    Title:          "vShell",
    EnableFileDrop: true,
    Mac: application.MacWindow{
        InvisibleTitleBarHeight: 30,
        TitleBar:                application.MacTitleBarHidden,
    },
    BackgroundColour: application.NewRGB(30, 30, 30),
    Width:            1280,
    Height:           800,
    MinWidth:         960,
    MinHeight:        600,
    URL:              "/",
})
```
