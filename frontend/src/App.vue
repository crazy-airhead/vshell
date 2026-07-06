<script setup lang="ts">
import { ref, computed, watch, onMounted, onUnmounted, nextTick } from 'vue'
import { useI18n } from 'vue-i18n'
import { NConfigProvider, darkTheme, NMessageProvider, NDialogProvider, type GlobalThemeOverrides } from 'naive-ui'
import { Events } from '@wailsio/runtime'
// eslint-disable-next-line @typescript-eslint/ban-ts-comment
// @ts-ignore: Window is exported at runtime
import { Window } from '@wailsio/runtime'
import IconSun from '~icons/lucide/sun'
import IconMoon from '~icons/lucide/moon'
import ActivityBar from './components/activity/ActivityBar.vue'
import ConnectionTree from './components/sidebar/ConnectionTree.vue'
import SnippetsPanel from './components/snippets/SnippetsPanel.vue'
import ProxyPanel from './components/proxy/ProxyPanel.vue'
import KeyManagementPanel from './components/keys/KeyManagementPanel.vue'
import SSHConfigPanel from './components/config/SSHConfigPanel.vue'
import PortForwardPanel from './components/panels/PortForwardPanel.vue'
import TerminalPane from './components/terminal/TerminalPane.vue'
import BottomPanel from './components/panels/BottomPanel.vue'
import DraggableDivider from './components/common/DraggableDivider.vue'
import SettingsModal from './components/settings/SettingsModal.vue'
import { useSettingsStore } from './stores/settings'
import { useLayoutStore } from './stores/layout'
import { useTerminalStore } from './stores/terminal'
import { useConnectionStore } from './stores/connection'
import { useShortcuts } from './composables/useShortcuts'
import { useTerminalManager } from './composables/useTerminalManager'
import { createAppThemeVars, resolveTerminalTheme } from './constants/terminalThemes'
import { shortcutDigitIndex } from './composables/useShortcuts'
import type { LocaleCode } from './stores/settings'
import { SaveWindowSize } from '../bindings/vshell/internal/app/appservice'

const { locale, t } = useI18n()
const settings = useSettingsStore()
const layout = useLayoutStore()
const terminalStore = useTerminalStore()
const connectionStore = useConnectionStore()
const { focusTerminal } = useTerminalManager()

const sidebarVisible = ref(true)
const showSettings = ref(false)
const geoIPDownloading = ref(false)

const selectedTerminalTheme = computed(() => resolveTerminalTheme(settings.terminalColorScheme, settings.isDark))
const appThemeVars = computed(() => createAppThemeVars(selectedTerminalTheme.value))
const naiveTheme = computed(() => appThemeVars.value.isDark ? darkTheme : null)
const themeIcon = computed(() => appThemeVars.value.isDark ? IconMoon : IconSun)
const localeLabel = computed(() => settings.localeCode === 'zh-CN' ? 'EN' : '中')

const naiveThemeOverrides = computed<GlobalThemeOverrides>(() => {
  const vars = appThemeVars.value

  return {
    common: {
      primaryColor: vars.colorPrimary,
      primaryColorHover: vars.colorPrimary + 'cc',
      primaryColorPressed: vars.colorPrimary + 'aa',
      infoColor: vars.colorInfo,
      successColor: vars.colorSuccess,
      warningColor: vars.colorWarning,
      errorColor: vars.colorError,
      borderRadius: '6px',
      borderColor: vars.borderColor,
      bodyColor: vars.bgPrimary,
      cardColor: vars.bgSecondary,
      modalColor: vars.bgSecondary,
      popoverColor: vars.bgSecondary,
      textColorBase: vars.textPrimary,
      textColor1: vars.textPrimary,
      textColor2: vars.textSecondary,
      textColor3: vars.textSecondary,
    },
  }
})

function handleLocaleSelect(key: string) {
  settings.setLocale(key as LocaleCode)
  locale.value = key
}

function handleThemeToggle() {
  if (settings.terminalColorScheme === 'default') {
    settings.toggleTheme()
    return
  }
  settings.setTerminalColorScheme(appThemeVars.value.isDark ? 'termius-light' : 'termius-dark')
}

function closeActiveTab() {
  const id = terminalStore.activeTabID
  if (!id) return
  const tab = terminalStore.tabs.find(t => t.id === id)
  if (tab && tab.type !== 'editor') {
    connectionStore.disconnectSession(tab.id, tab.connectionID)
  }
  terminalStore.removeTab(id)
}

async function openNewConnectionModal() {
  layout.setSidebar('connections')
  sidebarVisible.value = true
  await nextTick()
  window.dispatchEvent(new CustomEvent('vshell:new-connection'))
}

function focusActiveTerminal() {
  const id = terminalStore.activeTabID
  if (!id) return
  focusTerminal(id)
}

async function openNewLocalWindow() {
  await terminalStore.openLocalTerminal()
}

function shouldSkipGlobalTabShortcut(e: KeyboardEvent): boolean {
  if (document.documentElement.hasAttribute('data-shortcut-capturing')) return true
  const target = e.target as HTMLElement | null
  const tag = target?.tagName
  return (
    tag === 'INPUT' ||
    tag === 'TEXTAREA' ||
    tag === 'SELECT' ||
    target?.isContentEditable === true ||
    target?.closest('[data-shortcut-scope="settings"]') !== null
  )
}

async function activateTabAt(index: number) {
  const tab = terminalStore.tabs[index]
  if (!tab) return
  terminalStore.activeTabID = tab.id
  await nextTick()
  if (tab.type !== 'editor') {
    focusTerminal(tab.id)
  }
}

async function cycleActiveTab(direction: 1 | -1) {
  const tabs = terminalStore.tabs
  if (tabs.length === 0) return

  const currentIndex = Math.max(0, tabs.findIndex(tab => tab.id === terminalStore.activeTabID))
  const nextIndex = (currentIndex + direction + tabs.length) % tabs.length
  await activateTabAt(nextIndex)
}

function handleTabShortcut(e: KeyboardEvent) {
  if (shouldSkipGlobalTabShortcut(e)) return
  if (!(e.metaKey || e.ctrlKey) || e.altKey || e.shiftKey) return

  const index = shortcutDigitIndex(e)
  if (index === null) return

  e.preventDefault()
  e.stopPropagation()
  void activateTabAt(index)
}

function handleAppShortcut(action: string) {
  switch (action) {
    case 'newConnection':
      void openNewConnectionModal()
      break
    case 'newWindow':
      void openNewLocalWindow()
      break
    case 'closeTab':
      closeActiveTab()
      break
    case 'toggleTheme':
      handleThemeToggle()
      break
    case 'toggleSidebar':
      sidebarVisible.value = !sidebarVisible.value
      break
    case 'focusTerminal':
      focusActiveTerminal()
      break
  }
}

const handleActivateTabIndexEvent = (event: Event) => {
  const index = (event as CustomEvent<{ index: number }>).detail?.index
  if (typeof index === 'number') void activateTabAt(index)
}

const handleAppShortcutEvent = (event: Event) => {
  const action = (event as CustomEvent<{ action: string }>).detail?.action
  if (action) handleAppShortcut(action)
}

function syncCSSVars() {
  const root = document.documentElement
  const vars = appThemeVars.value
  root.setAttribute('data-theme', vars.isDark ? 'dark' : 'light')
  root.style.setProperty('--font-size-base', settings.uiFontSize + 'px')
  root.style.setProperty('--font-size-sm', Math.max(9, settings.uiFontSize - 2) + 'px')
  root.style.setProperty('--font-family', settings.uiFontFamily)
  root.style.setProperty('--color-primary', vars.colorPrimary)
  root.style.setProperty('--color-info', vars.colorInfo)
  root.style.setProperty('--color-success', vars.colorSuccess)
  root.style.setProperty('--color-warning', vars.colorWarning)
  root.style.setProperty('--color-error', vars.colorError)
  root.style.setProperty('--bg-primary', vars.bgPrimary)
  root.style.setProperty('--bg-secondary', vars.bgSecondary)
  root.style.setProperty('--bg-tertiary', vars.bgTertiary)
  root.style.setProperty('--bg-hover', vars.bgHover)
  root.style.setProperty('--text-primary', vars.textPrimary)
  root.style.setProperty('--text-secondary', vars.textSecondary)
  root.style.setProperty('--border-color', vars.borderColor)
  root.style.setProperty('--shadow-header', vars.shadowHeader)
  root.style.setProperty('--shadow-sider', vars.shadowSider)
  root.style.setProperty('--shadow-tab', vars.shadowTab)
  root.style.setProperty('--shadow-elevated', vars.shadowElevated)
  root.style.setProperty('--hover-overlay', vars.hoverOverlay)
  root.style.setProperty('--hover-overlay-strong', vars.hoverOverlayStrong)
  root.style.setProperty('--action-hover-bg', vars.actionHoverBg)
  root.style.setProperty('--delete-hover-color', vars.deleteHoverColor)
  root.style.setProperty('--delete-hover-bg', vars.deleteHoverBg)
  root.style.setProperty('--stat-bar-bg', vars.statBarBg)
}

watch(
  () => [
    settings.themeMode,
    settings.terminalColorScheme,
    settings.uiFontSize,
    settings.uiFontFamily,
    settings.accentColor,
  ],
  syncCSSVars,
  { immediate: true, deep: true },
)

// Register global shortcuts
useShortcuts({
  newConnection: () => handleAppShortcut('newConnection'),
  newWindow: () => handleAppShortcut('newWindow'),
  closeTab: () => handleAppShortcut('closeTab'),
  toggleTheme: () => handleAppShortcut('toggleTheme'),
  toggleSidebar: () => handleAppShortcut('toggleSidebar'),
  focusTerminal: () => handleAppShortcut('focusTerminal'),
})

let saveWindowSizeTimer: ReturnType<typeof setTimeout> | null = null

async function saveCurrentWindowSize() {
  try {
    const [width, height] = await Promise.all([Window.Width(), Window.Height()])
    await SaveWindowSize(Math.round(width), Math.round(height))
  } catch (e) {
    console.error('Failed to save window size:', e)
  }
}

function scheduleWindowSizeSave() {
  if (saveWindowSizeTimer) {
    clearTimeout(saveWindowSizeTimer)
  }
  saveWindowSizeTimer = setTimeout(() => {
    saveWindowSizeTimer = null
    void saveCurrentWindowSize()
  }, 300)
}

onMounted(async () => {
  window.addEventListener('keydown', handleTabShortcut, true)
  window.addEventListener('vshell:activate-tab-index', handleActivateTabIndexEvent)
  window.addEventListener('vshell:app-shortcut', handleAppShortcutEvent)
  window.addEventListener('resize', scheduleWindowSizeSave)

  if (terminalStore.tabs.length === 0) {
    try {
      await terminalStore.openLocalTerminal()
    } catch (e) {
      console.error('Failed to start local terminal:', e)
    }
  }

  Events.On('menu:settings', () => {
    showSettings.value = true
  })
  Events.On('menu:close-tab', () => {
    closeActiveTab()
  })
  Events.On('geoip:download:start', () => {
    geoIPDownloading.value = true
  })
  Events.On('geoip:download:done', () => {
    geoIPDownloading.value = false
  })
  Events.On('geoip:download:error', () => {
    geoIPDownloading.value = false
  })
})

onUnmounted(() => {
  window.removeEventListener('keydown', handleTabShortcut, true)
  window.removeEventListener('vshell:activate-tab-index', handleActivateTabIndexEvent)
  window.removeEventListener('vshell:app-shortcut', handleAppShortcutEvent)
  window.removeEventListener('resize', scheduleWindowSizeSave)
  if (saveWindowSizeTimer) {
    clearTimeout(saveWindowSizeTimer)
    saveWindowSizeTimer = null
  }
  void saveCurrentWindowSize()
})
</script>

<template>
  <NConfigProvider :theme="naiveTheme" :theme-overrides="naiveThemeOverrides">
    <NMessageProvider>
      <NDialogProvider>
        <div class="flex flex-col w-screen h-screen overflow-hidden bg-[var(--bg-primary)]">
          <!-- Title bar -->
          <div
            class="h-[30px] flex items-center justify-center px-2 bg-[var(--bg-tertiary)] shrink-0 relative overflow-hidden"
            style="-webkit-app-region: drag"
            @dblclick="Window.ToggleMaximise()"
          >
            <span class="text-xs font-semibold text-[var(--text-primary)]">vShell</span>
            <div class="absolute right-2 flex items-center gap-[2px]" style="-webkit-app-region: no-drag">
              <button
                class="bg-transparent border-none text-[var(--text-secondary)] text-[11px] cursor-pointer px-2 py-[2px] rounded-[3px] transition-colors duration-150 hover:text-[var(--text-primary)] hover:bg-[var(--hover-overlay)]"
                :title="appThemeVars.isDark ? t('settings.light') : t('settings.dark')"
                @click="handleThemeToggle"
              ><component :is="themeIcon" :width="14" :height="14" /></button>
              <button
                class="bg-transparent border-none text-[var(--text-secondary)] text-[11px] cursor-pointer px-2 py-[2px] rounded-[3px] transition-colors duration-150 hover:text-[var(--text-primary)] hover:bg-[var(--hover-overlay)]"
                :title="t('settings.language')"
                @click="handleLocaleSelect(settings.localeCode === 'zh-CN' ? 'en' : 'zh-CN')"
              >{{ localeLabel }}</button>
            </div>
            <!-- Connection progress bar -->
            <div
              v-if="connectionStore.connecting"
              class="absolute bottom-0 h-[2px] animate-connecting-bar"
              style="left: 48px; right: 0; width: auto;"
            />
          </div>

          <div
            v-if="geoIPDownloading"
            class="h-[28px] shrink-0 flex items-center justify-center thin-border-b bg-[var(--bg-secondary)] text-[12px] text-[var(--text-secondary)]"
          >
            {{ t('geoip.downloading') }}
          </div>

          <div class="flex flex-1 min-h-0">
            <!-- Activity Bar -->
            <ActivityBar
              @open-settings="showSettings = true"
              @show-sidebar="sidebarVisible = true"
              @toggle-sidebar="sidebarVisible = !sidebarVisible"
            />

            <!-- Main Area: Sidebar + Terminal + Bottom Panel -->
            <div class="flex-1 flex flex-col min-w-0 p-1.5">
              <div class="flex flex-1 min-h-0">
                <!-- Sidebar -->
                <template v-if="sidebarVisible">
                  <div class="shrink-0 overflow-hidden rounded-[var(--border-radius)] bg-[var(--bg-secondary)]" :style="{ width: layout.sidebarWidth + 'px' }">
                    <ConnectionTree v-if="layout.activeSidebar === 'connections'" @collapse-sidebar="sidebarVisible = false" />
                    <SnippetsPanel v-else-if="layout.activeSidebar === 'snippets'" />
                    <ProxyPanel v-else-if="layout.activeSidebar === 'proxies'" />
                    <KeyManagementPanel v-else-if="layout.activeSidebar === 'keys'" />
                    <SSHConfigPanel v-else-if="layout.activeSidebar === 'ssh-config'" />
                    <PortForwardPanel v-else-if="layout.activeSidebar === 'port-forward'" />
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
                <div class="flex-1 min-w-0 min-h-0 overflow-hidden rounded-[var(--border-radius)] bg-[var(--bg-secondary)]">
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
                <div class="overflow-hidden bg-[var(--bg-secondary)] rounded-t-[var(--border-radius)]" :style="{ height: layout.bottomPanelHeight + 'px', flexShrink: 0, minHeight: 0 }">
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
