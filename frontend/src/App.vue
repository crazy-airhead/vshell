<script setup lang="ts">
import { ref, computed, watch, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { NConfigProvider, darkTheme, NMessageProvider, NDialogProvider, type GlobalThemeOverrides } from 'naive-ui'
import { Events } from '@wailsio/runtime'
import AppShell from './components/shell/AppShell.vue'
import SettingsModal from './components/settings/SettingsModal.vue'
import { useSettingsStore } from './stores/settings'
import { useLayoutStore } from './stores/layout'
import { useTerminalStore } from './stores/terminal'
import { useConnectionStore } from './stores/connection'
import { useShortcuts } from './composables/useShortcuts'

const { locale } = useI18n()
const settings = useSettingsStore()
const layout = useLayoutStore()
const terminalStore = useTerminalStore()
const connectionStore = useConnectionStore()

const showSettings = ref(false)

const naiveTheme = computed(() => settings.isDark ? darkTheme : null)

/**
 * Naive 主题桥接（设计语言 §2.1）：CSS 变量单源，此处经 getComputedStyle
 * 现读真实值注入——禁传 var() 字符串（Naive 需可运算的颜色值）。
 * 显式依赖 settings.themeMode：主题切换后 data-theme 已由 syncCSSVars
 * 的 watcher（pre-flush，先于渲染）更新，此处重读才拿到新值。
 */
const naiveThemeOverrides = computed<GlobalThemeOverrides>(() => {
  void settings.themeMode
  const s = getComputedStyle(document.documentElement)
  const read = (name: string, fallback: string) => s.getPropertyValue(name).trim() || fallback
  const primary = read('--color-primary', '#3871E1')
  const island = read('--bg-island', '#FFFFFF')
  const component = read('--bg-component', '#FFFFFF')
  const inverted = read('--bg-inverted', '#27282E')
  const radiusM = read('--radius-m', '8px')

  return {
    common: {
      primaryColor: primary,
      primaryColorHover: primary + 'cc',
      primaryColorPressed: primary + 'aa',
      primaryColorSuppl: primary,
      infoColor: read('--color-info', '#3369D6'),
      successColor: read('--color-success', '#208A3C'),
      warningColor: read('--color-warning', '#A46704'),
      errorColor: read('--color-error', '#DB3B4B'),
      borderRadius: radiusM,
      borderRadiusSmall: read('--radius-s', '4px'),
      borderColor: read('--border-color', '#DFE1E5'),
      bodyColor: island,
      cardColor: island,
      popoverColor: island,
      tableColor: island,
      inputColor: component,
      actionColor: component,
      hoverColor: read('--hover-gray', '#EBECF0'),
    },
    Card: { borderRadius: read('--radius-l', '12px') },
    Dropdown: { borderRadius: radiusM },
    Tooltip: {
      color: inverted,
      borderRadius: radiusM,
    },
  }
})

/** CSS 变量用户覆盖层：data-theme 切换 + 字体跟随设置 */
function syncCSSVars() {
  const root = document.documentElement
  root.setAttribute('data-theme', settings.themeMode)
  root.style.setProperty('--font-size-base', settings.uiFontSize + 'px')
  root.style.setProperty('--font-size-sm', Math.max(9, settings.uiFontSize - 2) + 'px')
  root.style.setProperty('--font-size-xs', Math.max(8, settings.uiFontSize - 3) + 'px')
  root.style.setProperty('--font-family', settings.uiFontFamily)
}

watch(
  () => [settings.themeMode, settings.uiFontSize, settings.uiFontFamily],
  syncCSSVars,
  { immediate: true },
)

// Register global shortcuts
useShortcuts({
  toggleTheme: () => settings.toggleTheme(),
  toggleSidebar: () => layout.toggleSidebar(),
})

onMounted(() => {
  Events.On('menu:settings', () => {
    showSettings.value = true
  })
  Events.On('menu:close-tab', () => {
    const id = terminalStore.activeTabID
    if (!id) return
    const tab = terminalStore.tabs.find(t => t.id === id)
    if (tab && tab.type !== 'editor' && tab.connectionID) {
      connectionStore.disconnect(tab.connectionID)
    }
    terminalStore.removeTab(id)
  })
})
</script>

<template>
  <NConfigProvider :theme="naiveTheme" :theme-overrides="naiveThemeOverrides">
    <NMessageProvider>
      <NDialogProvider>
        <AppShell @open-settings="showSettings = true" />
        <SettingsModal v-model:show="showSettings" />
      </NDialogProvider>
    </NMessageProvider>
  </NConfigProvider>
</template>
