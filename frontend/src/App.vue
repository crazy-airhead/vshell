<script setup lang="ts">
import { ref, computed, watch, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { NConfigProvider, darkTheme, NMessageProvider, NDialogProvider } from 'naive-ui'
import { Events } from '@wailsio/runtime'
import ConnectionTree from './components/sidebar/ConnectionTree.vue'
import MonitorPanel from './components/monitor/MonitorPanel.vue'
import TerminalPane from './components/terminal/TerminalPane.vue'
import SFTPArea from './components/sftp/SFTPArea.vue'
import DraggableDivider from './components/common/DraggableDivider.vue'
import SettingsModal from './components/settings/SettingsModal.vue'
import { useSettingsStore } from './stores/settings'
import type { LocaleCode } from './stores/settings'
import { useShortcuts } from './composables/useShortcuts'

const { locale, t } = useI18n()
const settings = useSettingsStore()

const sidebarWidth = ref(280)
const sidebarTreeHeight = ref(560)
const sftpPanelHeight = ref(220)
const sidebarVisible = ref(true)
const showSettings = ref(false)

const naiveTheme = computed(() => settings.isDark ? darkTheme : null)
const themeIcon = computed(() => settings.isDark ? '☾' : '☀')
const localeLabel = computed(() => settings.localeCode === 'zh-CN' ? 'EN' : '中')

function handleLocaleSelect(key: string) {
  settings.setLocale(key as LocaleCode)
  locale.value = key
}

// Sync CSS variables from settings store to document root
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

// Listen for menu events from native menu bar
onMounted(() => {
  Events.On('menu:settings', () => {
    showSettings.value = true
  })
})
</script>

<template>
  <NConfigProvider :theme="naiveTheme">
    <NMessageProvider>
      <NDialogProvider>
        <div class="app-layout">
          <!-- Title bar (overlaps with macOS traffic lights) -->
          <div class="title-bar">
            <span class="app-title">vShell</span>
            <div class="title-bar-right">
              <button class="title-bar-btn theme-btn" :title="settings.isDark ? t('settings.light') : t('settings.dark')" @click="settings.toggleTheme()">{{ themeIcon }}</button>
              <button class="title-bar-btn" :title="t('settings.language')" @click="handleLocaleSelect(settings.localeCode === 'zh-CN' ? 'en' : 'zh-CN')">{{ localeLabel }}</button>
              <button class="title-bar-btn" :title="t('settings.title')" @click="showSettings = true">&#9881;</button>
            </div>
          </div>

          <div class="app-body">
            <!-- Left Column: Connection Tree + Monitor -->
            <template v-if="sidebarVisible">
              <div class="left-column" :style="{ width: sidebarWidth + 'px' }">
                <div class="tree-zone" :style="{ height: sidebarTreeHeight + 'px', flexShrink: 0 }">
                  <ConnectionTree />
                </div>
                <DraggableDivider
                  direction="horizontal"
                  :modelValue="sidebarTreeHeight"
                  @update:modelValue="(v: number) => sidebarTreeHeight = v"
                  :min="100"
                  :max="800"
                />
                <div class="monitor-zone">
                  <MonitorPanel />
                </div>
              </div>

              <DraggableDivider
                direction="vertical"
                :modelValue="sidebarWidth"
                @update:modelValue="(v: number) => sidebarWidth = v"
                :min="200"
                :max="500"
              />
            </template>

            <!-- Right Column: Terminal + SFTP -->
            <div class="right-column">
              <div class="terminal-zone">
                <TerminalPane />
              </div>
              <DraggableDivider
                direction="horizontal"
                :modelValue="sftpPanelHeight"
                @update:modelValue="(v: number) => sftpPanelHeight = v"
                :min="80"
                :max="600"
              />
              <div class="sftp-zone" :style="{ height: sftpPanelHeight + 'px', flexShrink: 0 }">
                <SFTPArea />
              </div>
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
  border-bottom: 1px solid var(--border-color);
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
  padding: 6px;
  gap: 0;
}

.left-column {
  display: flex;
  flex-direction: column;
  flex-shrink: 0;
  min-width: 0;
  overflow: hidden;
}

.tree-zone {
  overflow: hidden;
  border-radius: 8px;
  background: var(--bg-secondary);
}

.monitor-zone {
  flex: 1;
  min-height: 0;
  overflow: hidden;
  border-radius: 8px;
  background: var(--bg-secondary);
}

.right-column {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-width: 0;
  overflow: hidden;
}

.terminal-zone {
  flex: 1;
  min-height: 0;
  overflow: hidden;
  border-radius: 8px;
  background: var(--bg-secondary);
}

.sftp-zone {
  overflow: hidden;
  border-radius: 8px;
  background: var(--bg-secondary);
}
</style>
