<script setup lang="ts">
import { ref, computed, watch, onMounted } from 'vue'
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
import type { LocaleCode } from './stores/settings'

const { locale, t } = useI18n()
const settings = useSettingsStore()
const layout = useLayoutStore()
const terminalStore = useTerminalStore()
const connectionStore = useConnectionStore()

const sidebarVisible = ref(true)
const showSettings = ref(false)

const naiveTheme = computed(() => settings.isDark ? darkTheme : null)
const themeIcon = computed(() => settings.isDark ? IconMoon : IconSun)
const localeLabel = computed(() => settings.localeCode === 'zh-CN' ? 'EN' : '中')

const naiveThemeOverrides = computed<GlobalThemeOverrides>(() => {
  const s = getComputedStyle(document.documentElement)
  const primary = s.getPropertyValue('--color-primary').trim() || '#646cff'
  const info = s.getPropertyValue('--color-info').trim() || '#2080f0'
  const success = s.getPropertyValue('--color-success').trim() || '#52c41a'
  const warning = s.getPropertyValue('--color-warning').trim() || '#faad14'
  const error = s.getPropertyValue('--color-error').trim() || '#f5222d'
  const borderRadius = s.getPropertyValue('--border-radius').trim() || '6px'
  const border = s.getPropertyValue('--border-color').trim()

  return {
    common: {
      primaryColor: primary,
      primaryColorHover: primary + 'cc',
      primaryColorPressed: primary + 'aa',
      infoColor: info,
      successColor: success,
      warningColor: warning,
      errorColor: error,
      borderRadius,
      borderColor: border || undefined,
    },
  }
})

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
                :title="settings.isDark ? t('settings.light') : t('settings.dark')"
                @click="settings.toggleTheme()"
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

          <div class="flex flex-1 min-h-0">
            <!-- Activity Bar -->
            <ActivityBar @open-settings="showSettings = true" />

            <!-- Main Area: Sidebar + Terminal + Bottom Panel -->
            <div class="flex-1 flex flex-col min-w-0 p-1.5">
              <div class="flex flex-1 min-h-0">
                <!-- Sidebar -->
                <template v-if="sidebarVisible">
                  <div class="shrink-0 overflow-hidden rounded-[var(--border-radius)] bg-[var(--bg-secondary)]" :style="{ width: layout.sidebarWidth + 'px' }">
                    <ConnectionTree v-if="layout.activeSidebar === 'connections'" />
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
