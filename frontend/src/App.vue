<script setup lang="ts">
import { ref, computed, watch, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { NConfigProvider, darkTheme, NMessageProvider, NDialogProvider } from 'naive-ui'
import { Events } from '@wailsio/runtime'
// eslint-disable-next-line @typescript-eslint/ban-ts-comment
// @ts-ignore: Window is exported at runtime
import { Window } from '@wailsio/runtime'
import ActivityBar from './components/activity/ActivityBar.vue'
import ConnectionTree from './components/sidebar/ConnectionTree.vue'
import KeyManagementPanel from './components/keys/KeyManagementPanel.vue'
import SSHConfigPanel from './components/config/SSHConfigPanel.vue'
import TerminalPane from './components/terminal/TerminalPane.vue'
import BottomPanel from './components/panels/BottomPanel.vue'
import DraggableDivider from './components/common/DraggableDivider.vue'
import SettingsModal from './components/settings/SettingsModal.vue'
import { useSettingsStore } from './stores/settings'
import { useLayoutStore } from './stores/layout'
import { useTerminalStore } from './stores/terminal'
import { useConnectionStore } from './stores/connection'
import { useShortcuts } from './composables/useShortcuts'
import type { LocaleCode } from './stores/settings'

const { locale, t } = useI18n()
const settings = useSettingsStore()
const layout = useLayoutStore()
const terminalStore = useTerminalStore()
const connectionStore = useConnectionStore()

const sidebarVisible = ref(true)
const showSettings = ref(false)

const naiveTheme = computed(() => settings.isDark ? darkTheme : null)
const themeIcon = computed(() => settings.isDark ? '☾' : '☀')
const localeLabel = computed(() => settings.localeCode === 'zh-CN' ? 'EN' : '中')

function handleLocaleSelect(key: string) {
  settings.setLocale(key as LocaleCode)
  locale.value = key
}

function syncCSSVars() {
  const root = document.documentElement
  root.setAttribute('data-theme', settings.themeMode)
  root.style.setProperty('--font-size-base', settings.uiFontSize + 'px')
  root.style.setProperty('--font-size-sm', Math.max(9, settings.uiFontSize - 2) + 'px')
  root.style.setProperty('--font-family', settings.uiFontFamily)
  root.style.setProperty('--accent-color', settings.accentColor)
  root.style.setProperty('--accent-hover', settings.accentColor + 'cc')
  root.style.setProperty('--action-hover-color', settings.accentColor)
  root.style.setProperty('--action-hover-bg', settings.accentColor + '1a')
}

watch(
  () => [settings.themeMode, settings.uiFontSize, settings.uiFontFamily, settings.accentColor],
  syncCSSVars,
  { immediate: true, deep: true },
)

// Register global shortcuts
useShortcuts({
  toggleTheme: () => settings.toggleTheme(),
  toggleSidebar: () => { sidebarVisible.value = !sidebarVisible.value },
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
  <NConfigProvider :theme="naiveTheme">
    <NMessageProvider>
      <NDialogProvider>
        <div class="app-layout">
          <!-- Title bar -->
          <div class="title-bar" @dblclick="Window.ToggleMaximise()">
            <span class="app-title">vShell</span>
            <div class="title-bar-right">
              <button class="title-bar-btn theme-btn" :title="settings.isDark ? t('settings.light') : t('settings.dark')" @click="settings.toggleTheme()">{{ themeIcon }}</button>
              <button class="title-bar-btn" :title="t('settings.language')" @click="handleLocaleSelect(settings.localeCode === 'zh-CN' ? 'en' : 'zh-CN')">{{ localeLabel }}</button>
            </div>
          </div>

          <div class="app-body">
            <!-- Activity Bar -->
            <ActivityBar @open-settings="showSettings = true" />

            <!-- Main Area: Sidebar + Terminal + Bottom Panel -->
            <div class="main-area">
              <div class="content-row">
                <!-- Sidebar -->
                <template v-if="sidebarVisible">
                  <div class="sidebar" :style="{ width: layout.sidebarWidth + 'px' }">
                    <ConnectionTree v-if="layout.activeSidebar === 'connections'" />
                    <KeyManagementPanel v-else-if="layout.activeSidebar === 'keys'" />
                    <SSHConfigPanel v-else-if="layout.activeSidebar === 'ssh-config'" />
                  </div>
                  <DraggableDivider
                    direction="vertical"
                    :modelValue="layout.sidebarWidth"
                    @update:modelValue="(v: number) => layout.setSidebarWidth(v)"
                    :min="200"
                    :max="500"
                  />
                </template>

                <!-- Terminal -->
                <div class="terminal-zone">
                  <TerminalPane />
                </div>
              </div>

              <!-- Bottom Panel -->
              <template v-if="layout.bottomAnyVisible">
                <DraggableDivider
                  direction="horizontal"
                  :modelValue="layout.bottomPanelHeight"
                  @update:modelValue="(v: number) => layout.setBottomPanelHeight(v)"
                  :min="80"
                  :max="600"
                  :invert="true"
                />
                <div class="bottom-zone" :style="{ height: layout.bottomPanelHeight + 'px', flexShrink: 0, minHeight: 0 }">
                  <BottomPanel />
                </div>
              </template>
            </div>
          </div>
        </div>

        <SettingsModal v-model:show="showSettings" />
      </NDialogProvider>
    </NMessageProvider>
  </NConfigProvider>
</template>

<style scoped>
.app-layout {
  display: flex;
  flex-direction: column;
  width: 100vw;
  height: 100vh;
  overflow: hidden;
  background: var(--bg-primary);
}

.title-bar {
  height: 30px;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 0 8px;
  background: var(--bg-tertiary);
  flex-shrink: 0;
  position: relative;
  -webkit-app-region: drag;
}

.app-title {
  font-size: 12px;
  font-weight: 600;
  color: var(--text-primary);
}

.title-bar-right {
  position: absolute;
  right: 8px;
  display: flex;
  align-items: center;
  gap: 2px;
  -webkit-app-region: no-drag;
}

.title-bar-btn {
  background: none;
  border: none;
  color: var(--text-secondary);
  font-size: 11px;
  cursor: pointer;
  padding: 2px 8px;
  border-radius: 3px;
  transition: color 0.15s, background 0.15s;
}

.title-bar-btn:hover {
  color: var(--text-primary);
  background: var(--hover-overlay);
}

.theme-btn {
  font-size: 14px;
  padding: 2px 6px;
}

.app-body {
  display: flex;
  flex: 1;
  min-height: 0;
}

.main-area {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-width: 0;
  padding: 6px;
  gap: 0;
}

.content-row {
  display: flex;
  flex: 1;
  min-height: 0;
}

.sidebar {
  flex-shrink: 0;
  overflow: hidden;
  border-radius: 8px;
  background: var(--bg-secondary);
}

.terminal-zone {
  flex: 1;
  min-width: 0;
  min-height: 0;
  overflow: hidden;
  border-radius: 8px;
  background: var(--bg-secondary);
}

.bottom-zone {
  overflow: hidden;
  background: var(--bg-secondary);
  border-radius: 8px 8px 0 0;
}
</style>
