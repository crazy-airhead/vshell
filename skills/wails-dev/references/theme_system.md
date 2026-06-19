# 主题系统

## CSS 变量驱动

所有颜色和视觉令牌通过 CSS 变量定义在 `styles/global.css`，通过 `data-theme` 属性切换：

```css
/* 暗色主题 */
[data-theme="dark"] {
  /* 语义颜色 */
  --color-primary: #646cff;
  --color-info: #2080f0;
  --color-success: #52c41a;
  --color-warning: #faad14;
  --color-error: #f5222d;

  /* 背景 */
  --bg-primary: rgb(18, 18, 18);       /* 最底层 */
  --bg-secondary: rgb(28, 28, 28);     /* 面板/卡片 */
  --bg-tertiary: #2d2d2d;              /* 标题栏/工具栏 */
  --bg-hover: #3d3d3d;                 /* 悬停背景 */

  /* 文本 */
  --text-primary: rgb(224, 224, 224);  /* 正文 */
  --text-secondary: #858585;           /* 次要/标签 */

  /* 边框 */
  --border-color: #3d3d3d;

  /* 阴影 */
  --shadow-header: 0 1px 2px rgba(0, 0, 0, 0.3);
  --shadow-sider: 2px 0 8px 0 rgba(0, 0, 0, 0.25);
  --shadow-elevated: 0 4px 12px rgba(0, 0, 0, 0.4);

  /* 交互 */
  --hover-overlay: rgba(255, 255, 255, 0.06);
  --action-hover-bg: rgba(100, 108, 255, 0.15);
  --delete-hover-color: #f5222d;
  --delete-hover-bg: rgba(245, 34, 45, 0.15);
}

/* 亮色主题 */
[data-theme="light"] {
  --color-primary: #646cff;
  --color-info: #2080f0;
  --color-success: #52c41a;
  --color-warning: #faad14;
  --color-error: #f5222d;

  --bg-primary: rgb(247, 250, 252);
  --bg-secondary: rgb(255, 255, 255);
  --bg-tertiary: #e8e8e8;
  --bg-hover: #e0e0e0;

  --text-primary: rgb(31, 31, 31);
  --text-secondary: #666666;

  --border-color: #d9d9d9;

  --shadow-header: 0 1px 2px rgb(0, 21, 41, 0.08);
  --shadow-elevated: 0 4px 12px rgba(0, 0, 0, 0.08);

  --hover-overlay: rgba(0, 0, 0, 0.04);
  --action-hover-bg: rgba(100, 108, 255, 0.1);
  --delete-hover-bg: rgba(245, 34, 45, 0.1);
}
```

## 全局重置与排版

```css
/* global.css */
*, *::before, *::after { margin: 0; padding: 0; box-sizing: border-box; }
html, body { width: 100%; height: 100%; overflow: hidden; font-family: var(--font-family); }
#app { width: 100%; height: 100%; }

/* 滚动条 */
::-webkit-scrollbar { width: 8px; height: 8px; }
::-webkit-scrollbar-track { background: transparent; }
::-webkit-scrollbar-thumb { background: var(--bg-hover); border-radius: 4px; }
::-webkit-scrollbar-thumb:hover { background: var(--text-secondary); }

/* 布局令牌 */
:root {
  --border-radius: 6px;
  --font-size-base: 13px;
  --font-size-sm: 11px;
  --terminal-bar-height: 36px;
  --title-bar-height: 30px;
  --activity-bar-width: 48px;
}
```

## UnoCSS 配置

`uno.config.ts` 映射 CSS 变量到 UnoCSS 主题色：

```typescript
import { defineConfig, presetUno } from 'unocss'

export default defineConfig({
  presets: [presetUno({ dark: 'class' })],
  theme: {
    colors: {
      primary: 'var(--color-primary)',
      info: 'var(--color-info)',
      success: 'var(--color-success)',
      warning: 'var(--color-warning)',
      error: 'var(--color-error)',
      'bg-primary': 'var(--bg-primary)',
      'bg-secondary': 'var(--bg-secondary)',
      'bg-tertiary': 'var(--bg-tertiary)',
      'bg-hover': 'var(--bg-hover)',
      'text-primary': 'var(--text-primary)',
      'text-secondary': 'var(--text-secondary)',
      'border': 'var(--border-color)',
    },
  },
  shortcuts: {
    'flex-center': 'flex items-center justify-center',
    'flex-col-center': 'flex flex-col items-center justify-center',
    'panel-bg': 'bg-[var(--bg-secondary)] rounded-[var(--border-radius)]',
    'hover-overlay': 'transition-colors duration-150 hover:bg-[var(--hover-overlay)] hover:text-[var(--text-primary)]',
    'text-muted': 'text-[var(--text-secondary)]',
    'text-active': 'text-[var(--text-primary)]',
  },
  rules: [
    [/^text-size-(\d+)$/, ([, d]) => ({ 'font-size': `${d}px` })],
  ],
})
```

### 模板中使用

```html
<!-- CSS 变量语法（推荐） -->
<div class="bg-[var(--bg-secondary)] text-[var(--text-primary)] rounded-[var(--border-radius)]">

<!-- UnoCSS 映射色（更简洁但语义较弱） -->
<div class="bg-bg-secondary text-text-primary">

<!-- 使用 shortcuts -->
<div class="panel-bg flex-center">
```

## Naive UI 主题桥接

在 `App.vue` 中桥接 CSS 变量到 Naive UI 主题：

```typescript
import { NConfigProvider, darkTheme, type GlobalThemeOverrides } from 'naive-ui'

const naiveTheme = computed(() => settings.isDark ? darkTheme : null)

const naiveThemeOverrides = computed<GlobalThemeOverrides>(() => {
  const s = getComputedStyle(document.documentElement)
  const primary = s.getPropertyValue('--color-primary').trim() || '#646cff'
  const borderRadius = s.getPropertyValue('--border-radius').trim() || '6px'
  const border = s.getPropertyValue('--border-color').trim()

  return {
    common: {
      primaryColor: primary,
      primaryColorHover: primary + 'cc',
      primaryColorPressed: primary + 'aa',
      infoColor: s.getPropertyValue('--color-info').trim(),
      successColor: s.getPropertyValue('--color-success').trim(),
      warningColor: s.getPropertyValue('--color-warning').trim(),
      errorColor: s.getPropertyValue('--color-error').trim(),
      borderRadius,
      borderColor: border || undefined,
    },
  }
})
```

```vue
<template>
  <NConfigProvider :theme="naiveTheme" :theme-overrides="naiveThemeOverrides">
    <!-- ... -->
  </NConfigProvider>
</template>
```

## 主题切换

```typescript
// stores/settings.ts
const themeMode = ref<'dark' | 'light'>(localStorage.getItem('theme') || 'dark')
const isDark = computed(() => themeMode.value === 'dark')

function toggleTheme() {
  themeMode.value = isDark.value ? 'light' : 'dark'
  localStorage.setItem('theme', themeMode.value)
}
```

```typescript
// App.vue — 同步 CSS 变量
function syncCSSVars() {
  const root = document.documentElement
  root.setAttribute('data-theme', settings.themeMode)
  root.style.setProperty('--font-size-base', settings.uiFontSize + 'px')
  root.style.setProperty('--font-size-sm', Math.max(9, settings.uiFontSize - 2) + 'px')
  root.style.setProperty('--font-family', settings.uiFontFamily)
}

watch(() => [settings.themeMode, settings.uiFontSize, settings.uiFontFamily], syncCSSVars, { immediate: true })
```
