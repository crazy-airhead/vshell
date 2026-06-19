# 前端开发模式

## 组件编写规范

必须使用 `<script setup lang="ts">` 语法：

```vue
<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { Events } from '@wailsio/runtime'
import { useConnectionStore } from '../stores/connection'

const { t } = useI18n()
const store = useConnectionStore()
const data = ref<string>('')

onMounted(() => {
  Events.On('my:event', handler)
})

onUnmounted(() => {
  Events.Off('my:event')  // 必须清理
})
</script>

<template>
  <!-- 使用 Naive UI 组件 -->
  <NButton @click="doSomething">{{ t('common.confirm') }}</NButton>
</template>
```

## Naive UI 使用规则

### 严格规则

- **唯一允许的 UI 库**，禁止 Element Plus / Ant Design / Vuetify 等
- **禁止浏览器原生对话框**：`alert()` / `confirm()` / `prompt()`
- 使用 Naive UI 的 `useDialog()` / `useMessage()` 替代

### 替代原生对话框

```typescript
import { useDialog, useMessage } from 'naive-ui'

const dialog = useDialog()
const message = useMessage()

// 替代 confirm()
dialog.warning({
  title: t('common.confirm'),
  content: t('common.deleteConfirm'),
  positiveText: t('common.confirm'),
  negativeText: t('common.cancel'),
  onPositiveClick: () => {
    // 执行删除
  },
})

// 替代 alert()
message.success(t('common.saved'))
message.error(t('common.failed'))
```

### 常用 Naive UI 组件

```vue
<script setup>
import {
  NButton, NInput, NSelect, NDataTable, NModal,
  NForm, NFormItem, NTabs, NTabPane, NTooltip,
  NDropdown, NPopconfirm, NSwitch, NSlider,
  NConfigProvider, NMessageProvider, NDialogProvider,
  darkTheme
} from 'naive-ui'
</script>
```

### 根组件 Provider 包裹

App.vue 中必须包裹 Provider：

```vue
<template>
  <NConfigProvider :theme="naiveTheme" :theme-overrides="naiveThemeOverrides">
    <NMessageProvider>
      <NDialogProvider>
        <!-- 应用内容 -->
      </NDialogProvider>
    </NMessageProvider>
  </NConfigProvider>
</template>
```

## Pinia Store 模式（Composition Store）

### 标准 Store 模板

```typescript
import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { MyAction } from '../../bindings/vshell/internal/app/appservice'

export const useMyStore = defineStore('myFeature', () => {
  // ─── State ───────────────────
  const items = ref<MyType[]>([])
  const loading = ref(false)
  const error = ref<string | null>(null)

  // ─── Getters ─────────────────
  const activeItems = computed(() => items.value.filter(i => i.active))

  // ─── Actions ─────────────────
  async function loadItems() {
    loading.value = true
    error.value = null
    try {
      const result = await MyAction()
      // Wails 返回类实例，需转为普通对象
      items.value = (result || []).map((item: any) => ({
        id: item.id || '',
        name: item.name || '',
      })) as MyType[]
    } catch (e) {
      error.value = String(e)
      console.error('Failed:', e)
    } finally {
      loading.value = false
    }
  }

  // ─── Return ──────────────────
  return { items, loading, error, activeItems, loadItems }
})
```

### 关键点

- **Wails 返回值转换**：绑定返回类实例，必须 `map()` 转为普通对象
- **Loading 管理**：异步操作统一 `try/finally` 管理 loading 状态
- **返回对象**：明确列出所有暴露的属性和方法
- **禁止**：不在 Store 中直接操作 DOM

### 在组件中使用

```typescript
const store = useMyStore()

// 读取状态（自动响应式）
const items = store.items
const loading = store.loading

// 调用 action
await store.loadItems()
```

### Store 间依赖

```typescript
import { useMonitorStore } from './monitor'

async function connect(id: string) {
  const sessionID = await ConnectSSH(id)
  connectedIDs.value.add(id)

  // 调用其他 store
  const monitorStore = useMonitorStore()
  await monitorStore.startMonitoring(id)

  return sessionID
}
```

## 终端集成（useTerminal Composable）

### 使用模式

```typescript
import { useTerminal } from '../composables/useTerminal'

const { terminal, init, fit, write } = useTerminal(sessionID)

const terminalRef = ref<HTMLElement>()

onMounted(() => {
  const el = terminalRef.value
  if (el) init(el)
})

// resize 时自动适配
const observer = new ResizeObserver(() => fit())
observer.observe(el)

onUnmounted(() => {
  observer.disconnect()
  // useTerminal 内部已处理 Events.Off 和 term.dispose()
})
```

### useTerminal 内部实现要点

```typescript
export function useTerminal(sessionID: string) {
  function init(el: HTMLElement) {
    const term = new Terminal({
      cursorBlink: true,
      fontSize: 14,
      fontFamily: 'Menlo, Monaco, "Courier New", monospace',
      theme: { background: '#1e1e1e', foreground: '#cccccc' },
      allowProposedApi: true,
    })

    // 加载插件
    const fit = new FitAddon()
    term.loadAddon(fit)
    term.loadAddon(new SearchAddon())
    term.loadAddon(new SerializeAddon())

    // WebGL 加速（失败则回退到 canvas）
    try { term.loadAddon(new WebglAddon()) } catch {}

    term.open(el)
    fit.fit()

    // 双向事件绑定
    term.onData((data) => Events.Emit('terminal:stdin', { sessionID, data }))
    Events.On('terminal:stdout', (ev) => {
      if (ev?.data?.sessionID === sessionID) term.write(ev.data.data)
    })
    term.onResize(({ rows, cols }) => {
      Events.Emit('terminal:resize', { sessionID, rows, cols })
    })

    onUnmounted(() => {
      Events.Off('terminal:stdout')
      term.dispose()
    })
  }

  return { terminal, init, fit, write }
}
```

## 调用后端绑定

### 自动生成的绑定

位置：`frontend/bindings/vshell/internal/app/appservice.ts`

```typescript
import {
  ListConnections,
  CreateConnection,
  ConnectSSH,
  DisconnectSSH,
  // ... 所有导出的 AppService 方法
} from '../../bindings/vshell/internal/app/appservice'

// 类型导入
import { ConnectionForm, AuthType } from '../../bindings/vshell/internal/models/models'
```

### 绑定调用返回 CancellablePromise

```typescript
const result = await ListConnections()      // CancellablePromise<Connection[]>
const sessionID = await ConnectSSH(connID)  // CancellablePromise<string>
await CreateConnection(form)                // CancellablePromise<void>
```

### 创建带类型的表单

```typescript
const form = new ConnectionForm({
  id: crypto.randomUUID(),
  name: 'My Server',
  host: '192.168.1.1',
  port: 22,
  username: 'root',
  auth_type: AuthType.AuthPassword,
  password: 'secret',
  private_key: '',
  key_passphrase: '',
  group_id: null,
  upload_path: '/',
  sort_order: 0,
})
await CreateConnection(form)
```

## 国际化

### 使用模式

```typescript
import { useI18n } from 'vue-i18n'
const { t } = useI18n()

// 模板中
<span>{{ t('connections.title') }}</span>

// 代码中
message.success(t('connections.saved'))
dialog.warning({ title: t('common.confirm'), content: t('common.deleteConfirm') })
```

### 添加新翻译

同步编辑 `locales/zh-CN.ts` 和 `locales/en.ts`：

```typescript
// zh-CN.ts
export default {
  notes: {
    title: '笔记',
    empty: '暂无笔记',
    saved: '笔记已保存',
  },
}

// en.ts
export default {
  notes: {
    title: 'Notes',
    empty: 'No notes yet',
    saved: 'Note saved',
  },
}
```

## 图标使用

使用 Iconify Lucide 图标集，通过 unplugin-icons 按需加载：

```vue
<script setup>
import IconServer from '~icons/lucide/server'
import IconPlus from '~icons/lucide/plus'
import IconTrash from '~icons/lucide/trash-2'
import IconSettings from '~icons/lucide/settings'
</script>

<template>
  <IconServer :width="16" :height="16" />
  <IconPlus :width="14" :height="14" />
</template>
```

### 常用图标

`server`, `folder`, `file`, `terminal`, `plus`, `trash-2`, `settings`, `edit`, `copy`, `download`, `upload`, `refresh-cw`, `search`, `x`, `check`, `chevron-right`, `chevron-down`, `monitor`, `key`, `globe`
