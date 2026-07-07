<script setup lang="ts">
import { ref, computed, watch, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  NModal, NTabs, NTabPane, NFormItem, NSlider, NSpace,
  NButton, NSelect, NInput, useMessage,
} from 'naive-ui'
import { Dialogs } from '@wailsio/runtime'
import { useSettingsStore } from '../../stores/settings'
import { terminalThemes } from '../../constants/terminalThemes'
import { comboFromKeyboardEvent, formatShortcutCombo } from '../../composables/useShortcuts'
import {
  GetGeoIPDownloadURL,
  ReadLocalFileContent,
  SetGeoIPDownloadURL,
  UpdateGeoIPDatabase,
  WriteLocalFileContent,
} from '../../../bindings/vshell/internal/app/appservice'
import type { ClientSettingsData, ShortcutMap } from '../../stores/settings'

const props = defineProps<{ show: boolean }>()
const emit = defineEmits<{ (e: 'update:show', val: boolean): void }>()

const { t, locale } = useI18n()
const settings = useSettingsStore()
const message = useMessage()
const geoIPDownloadURL = ref('')
const updatingGeoIP = ref(false)
const transferringSettings = ref(false)
const lastSavedGeoIPDownloadURL = ref('')
let geoIPSaveTimer: ReturnType<typeof setTimeout> | null = null

const clientSettingsExportType = 'vshell-client-settings'
const clientSettingsExportVersion = 1
const appVersion = 'v1.2.2'

interface ClientSettingsExportFile {
  type: typeof clientSettingsExportType
  version: number
  exported_at: string
  settings: ClientSettingsData
  geoip?: {
    download_url?: string
  }
}

const showModal = computed({
  get: () => props.show,
  set: (v: boolean) => emit('update:show', v),
})

const accentColors = [
  '#0078d4', '#0dbf8a', '#e040fb', '#ff6d00',
  '#e53935', '#00acc1', '#7cb342', '#fdd835',
]

const uiFonts = [
  { label: 'System', value: 'system-ui' },
  { label: 'PingFang SC', value: '"PingFang SC", "Helvetica Neue", sans-serif' },
  { label: 'Microsoft YaHei', value: '"Microsoft YaHei", sans-serif' },
  { label: 'Noto Sans SC', value: '"Noto Sans SC", sans-serif' },
  { label: 'Source Han Sans', value: '"Source Han Sans SC", sans-serif' },
  { label: 'Helvetica', value: 'Helvetica, Arial, sans-serif' },
  { label: 'Arial', value: 'Arial, sans-serif' },
]

const terminalFonts = [
  { label: 'Menlo', value: 'Menlo, Monaco, "Courier New", monospace' },
  { label: 'Monaco', value: 'Monaco, Menlo, "Courier New", monospace' },
  { label: 'Courier New', value: '"Courier New", Courier, monospace' },
  { label: 'Source Code Pro', value: '"Source Code Pro", Menlo, monospace' },
  { label: 'JetBrains Mono', value: '"JetBrains Mono", Menlo, monospace' },
  { label: 'Fira Code', value: '"Fira Code", Menlo, monospace' },
  { label: 'Cascadia Code', value: '"Cascadia Code", Menlo, monospace' },
  { label: 'Noto Sans Mono', value: '"Noto Sans Mono", Menlo, monospace' },
]

const shortcutActions: { key: keyof ShortcutMap; label: string }[] = [
  { key: 'newConnection', label: 'settings.shortcutNewConnection' },
  { key: 'newWindow', label: 'settings.shortcutNewWindow' },
  { key: 'closeTab', label: 'settings.shortcutCloseTab' },
  { key: 'toggleTheme', label: 'settings.shortcutToggleTheme' },
  { key: 'toggleSidebar', label: 'settings.shortcutToggleSidebar' },
  { key: 'focusTerminal', label: 'settings.shortcutFocusTerminal' },
]

const capturingKey = ref<keyof ShortcutMap | null>(null)
let stopShortcutCapture: (() => void) | null = null

function startCapture(action: keyof ShortcutMap) {
  stopCapture()
  capturingKey.value = action
  document.documentElement.setAttribute('data-shortcut-capturing', 'true')
  const handler = (e: KeyboardEvent) => {
    onKeyCapture(e)
  }
  window.addEventListener('keydown', handler, true)
  stopShortcutCapture = () => {
    window.removeEventListener('keydown', handler, true)
    document.documentElement.removeAttribute('data-shortcut-capturing')
  }
}

function stopCapture() {
  stopShortcutCapture?.()
  stopShortcutCapture = null
  capturingKey.value = null
}

function onKeyCapture(e: KeyboardEvent) {
  if (!capturingKey.value) return
  e.preventDefault()
  e.stopPropagation()

  if (e.key === 'Escape') {
    stopCapture()
    return
  }

  const combo = comboFromKeyboardEvent(e)
  if (combo) {
    settings.setShortcut(capturingKey.value, combo)
    stopCapture()
  }
}

function findFontValue(options: { value: string }[], current: string): string {
  return options.find(o => o.value === current)?.value || options[0].value
}

function formatCombo(combo: string): string {
  return formatShortcutCombo(combo)
}

function clearGeoIPSaveTimer() {
  if (geoIPSaveTimer) {
    clearTimeout(geoIPSaveTimer)
    geoIPSaveTimer = null
  }
}

onUnmounted(() => {
  stopCapture()
  clearGeoIPSaveTimer()
})

watch(
  () => props.show,
  async (visible) => {
    if (!visible) return
    try {
      const url = await GetGeoIPDownloadURL()
      lastSavedGeoIPDownloadURL.value = url
      geoIPDownloadURL.value = url
    } catch (e: any) {
      message.error(t('settings.geoipLoadFailed', { error: e }))
    }
  },
  { immediate: true },
)

watch(geoIPDownloadURL, (url) => {
  if (!props.show || url === lastSavedGeoIPDownloadURL.value) return

  clearGeoIPSaveTimer()
  geoIPSaveTimer = setTimeout(() => {
    geoIPSaveTimer = null
    void saveGeoIPDownloadURL()
  }, 600)
})

async function saveGeoIPDownloadURL() {
  try {
    await SetGeoIPDownloadURL(geoIPDownloadURL.value)
    const savedURL = await GetGeoIPDownloadURL()
    lastSavedGeoIPDownloadURL.value = savedURL
    geoIPDownloadURL.value = savedURL
  } catch (e: any) {
    message.error(t('settings.geoipSaveFailed', { error: e }))
  }
}

async function updateGeoIPDatabase() {
  clearGeoIPSaveTimer()
  updatingGeoIP.value = true
  try {
    await SetGeoIPDownloadURL(geoIPDownloadURL.value)
    await UpdateGeoIPDatabase()
    const savedURL = await GetGeoIPDownloadURL()
    lastSavedGeoIPDownloadURL.value = savedURL
    geoIPDownloadURL.value = savedURL
    message.success(t('settings.geoipUpdated'))
  } catch (e: any) {
    message.error(t('settings.geoipUpdateFailed', { error: e }))
  } finally {
    updatingGeoIP.value = false
  }
}

async function exportClientSettings() {
  const filePath = await Dialogs.SaveFile({
    Title: t('settings.exportClientSettings'),
    Filename: 'vshell-client-settings.json',
    Filters: [{ DisplayName: 'JSON', Pattern: '*.json' }],
  })
  if (!filePath) return

  transferringSettings.value = true
  try {
    const exportFile: ClientSettingsExportFile = {
      type: clientSettingsExportType,
      version: clientSettingsExportVersion,
      exported_at: new Date().toISOString(),
      settings: settings.exportSettings(),
      geoip: {
        download_url: geoIPDownloadURL.value || await GetGeoIPDownloadURL(),
      },
    }
    await WriteLocalFileContent(filePath, JSON.stringify(exportFile, null, 2))
    message.success(t('settings.clientSettingsExported'))
  } catch (e: any) {
    message.error(t('settings.clientSettingsExportFailed', { error: e }))
  } finally {
    transferringSettings.value = false
  }
}

async function importClientSettings() {
  const filePath = await Dialogs.OpenFile({
    Title: t('settings.importClientSettings'),
    CanChooseFiles: true,
    CanChooseDirectories: false,
    AllowsMultipleSelection: false,
    Filters: [{ DisplayName: 'JSON', Pattern: '*.json' }],
  })
  if (!filePath || Array.isArray(filePath)) return

  transferringSettings.value = true
  try {
    const content = await ReadLocalFileContent(filePath)
    const parsed = JSON.parse(content) as Partial<ClientSettingsExportFile>
    if (parsed.type !== clientSettingsExportType || parsed.version !== clientSettingsExportVersion || !parsed.settings) {
      throw new Error(t('settings.clientSettingsInvalidFile'))
    }

    settings.importSettings(parsed.settings)
    locale.value = settings.localeCode

    const importedGeoIPURL = parsed.geoip?.download_url
    if (typeof importedGeoIPURL === 'string') {
      await SetGeoIPDownloadURL(importedGeoIPURL)
      const savedURL = await GetGeoIPDownloadURL()
      lastSavedGeoIPDownloadURL.value = savedURL
      geoIPDownloadURL.value = savedURL
    }

    message.success(t('settings.clientSettingsImported'))
  } catch (e: any) {
    message.error(t('settings.clientSettingsImportFailed', { error: e?.message || e }))
  } finally {
    transferringSettings.value = false
  }
}
</script>

<template>
  <NModal v-model:show="showModal" preset="card" :title="t('settings.title')" style="width: 720px" :mask-closable="true">
    <NTabs type="line" animated>
      <!-- General Tab -->
      <NTabPane :name="t('settings.general')">
        <NSpace vertical :size="16" style="padding: 8px 0">
          <NFormItem :label="t('settings.uiFontFamily')" label-placement="left" :show-feedback="false">
            <NSelect
              :value="findFontValue(uiFonts, settings.uiFontFamily)"
              :options="uiFonts"
              style="width: 260px"
              @update:value="settings.setUIFontFamily"
            />
          </NFormItem>

          <NFormItem :label="t('settings.uiFontSize')" label-placement="left" :show-feedback="false">
            <NSlider v-model:value="settings.uiFontSize" :min="11" :max="18" :step="1" :marks="{ 11: '11', 13: '13', 16: '16', 18: '18' }" style="flex: 1" @update:value="settings.setUIFontSize" />
            <span class="w-8 text-right" style="font-variant-numeric: tabular-nums">{{ settings.uiFontSize }}</span>
          </NFormItem>

          <NFormItem :label="t('settings.accentColor')" label-placement="left" :show-feedback="false">
            <div class="flex gap-2 flex-wrap">
              <button
                v-for="c in accentColors" :key="c"
                class="w-6 h-6 rounded-full border-2 border-transparent cursor-pointer transition-all duration-150 hover:scale-115"
                :class="{ '!border-[var(--text-primary)]': settings.accentColor === c }"
                :style="{ background: c }"
                @click="settings.setAccentColor(c)"
              />
            </div>
          </NFormItem>

          <NFormItem :label="t('settings.geoipDownloadURL')" label-placement="top" :show-feedback="false">
            <div class="flex gap-2 w-full">
              <NInput
                v-model:value="geoIPDownloadURL"
                :placeholder="t('settings.geoipDownloadURLPlaceholder')"
              />
              <NButton type="primary" :loading="updatingGeoIP" @click="updateGeoIPDatabase">
                {{ t('settings.geoipUpdate') }}
              </NButton>
            </div>
          </NFormItem>
          <NFormItem :label="t('settings.clientSettingsTransfer')" label-placement="left" :show-feedback="false">
            <div class="flex gap-2">
              <NButton :loading="transferringSettings" @click="exportClientSettings">
                {{ t('settings.exportClientSettings') }}
              </NButton>
              <NButton :loading="transferringSettings" @click="importClientSettings">
                {{ t('settings.importClientSettings') }}
              </NButton>
            </div>
          </NFormItem>

          <NFormItem :label="t('settings.appVersion')" label-placement="left" :show-feedback="false">
            <span class="settings-version">{{ appVersion }}</span>
          </NFormItem>
        </NSpace>
      </NTabPane>

      <!-- Terminal Tab -->
      <NTabPane :name="t('settings.terminal')">
        <NSpace vertical :size="16" style="padding: 8px 0">
          <NFormItem :label="t('settings.terminalFontFamily')" label-placement="left" :show-feedback="false">
            <NSelect
              :value="findFontValue(terminalFonts, settings.terminalFontFamily)"
              :options="terminalFonts"
              style="width: 260px"
              @update:value="settings.setTerminalFontFamily"
            />
          </NFormItem>

          <NFormItem :label="t('settings.terminalFontSize')" label-placement="left" :show-feedback="false">
            <NSlider v-model:value="settings.terminalFontSize" :min="10" :max="24" :step="1" :marks="{ 10: '10', 14: '14', 18: '18', 24: '24' }" style="flex: 1" @update:value="settings.setTerminalFontSize" />
            <span class="w-8 text-right" style="font-variant-numeric: tabular-nums">{{ settings.terminalFontSize }}</span>
          </NFormItem>

          <NFormItem :label="t('settings.colorScheme')" label-placement="top" :show-feedback="false">
            <div class="terminal-theme-grid">
              <button
                v-for="theme in terminalThemes"
                :key="theme.key"
                type="button"
                class="terminal-theme-card"
                :class="{ 'terminal-theme-card--active': settings.terminalColorScheme === theme.key }"
                @click="settings.setTerminalColorScheme(theme.key)"
              >
                <span
                  class="terminal-theme-preview"
                  :style="{ backgroundColor: theme.background, color: theme.foreground, borderColor: theme.brightBlack }"
                >
                  <span class="terminal-theme-preview__bar" :style="{ backgroundColor: theme.brightGreen }"></span>
                  <span class="terminal-theme-preview__bar terminal-theme-preview__bar--short" :style="{ backgroundColor: theme.green }"></span>
                  <span class="terminal-theme-preview__row">
                    <span :style="{ backgroundColor: theme.cyan }"></span>
                    <span :style="{ backgroundColor: theme.blue }"></span>
                    <span :style="{ backgroundColor: theme.yellow }"></span>
                  </span>
                  <span class="terminal-theme-preview__row terminal-theme-preview__row--small">
                    <span :style="{ backgroundColor: theme.magenta }"></span>
                    <span :style="{ backgroundColor: theme.foreground }"></span>
                    <span :style="{ backgroundColor: theme.cursor }"></span>
                  </span>
                </span>
                <span class="terminal-theme-name">{{ theme.name }}</span>
              </button>
            </div>
          </NFormItem>
        </NSpace>
      </NTabPane>

      <!-- Shortcuts Tab -->
      <NTabPane :name="t('settings.shortcuts')">
        <NSpace vertical :size="10" style="padding: 8px 0" data-shortcut-scope="settings" @keydown="onKeyCapture">
          <div v-for="s in shortcutActions" :key="s.key" class="flex items-center justify-between">
            <span class="text-[var(--font-size-base)] text-[var(--text-primary)]">{{ t(s.label) }}</span>
            <button
              class="bg-[var(--bg-tertiary)] border border-solid border-[var(--border-color)] text-[var(--text-primary)] text-[var(--font-size-sm)] px-3 py-1 rounded-[4px] cursor-pointer min-w-[120px] text-center font-mono transition-colors duration-150 hover:border-[var(--color-primary)]"
              :class="{ '!border-[var(--color-primary)] text-[var(--text-secondary)] !font-sans': capturingKey === s.key }"
              @click="startCapture(s.key)"
            >
              <template v-if="capturingKey === s.key">{{ t('settings.shortcutCaptureHint') }}</template>
              <template v-else>{{ formatCombo(settings.shortcuts[s.key]) }}</template>
            </button>
          </div>
          <NButton size="tiny" @click="settings.resetShortcuts">{{ t('settings.resetShortcuts') }}</NButton>
        </NSpace>
      </NTabPane>
    </NTabs>
  </NModal>
</template>

<style scoped>
.terminal-theme-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
  width: 100%;
}

.terminal-theme-card {
  display: flex;
  align-items: center;
  gap: 12px;
  min-width: 0;
  padding: 10px;
  border: 1px solid var(--border-color);
  border-radius: 8px;
  background: var(--bg-tertiary);
  color: var(--text-primary);
  cursor: pointer;
  text-align: left;
  transition: border-color 0.15s ease, background-color 0.15s ease, box-shadow 0.15s ease;
}

.terminal-theme-card:hover {
  border-color: var(--color-primary);
  background: var(--bg-secondary);
}

.terminal-theme-card--active {
  border-color: var(--color-primary);
  box-shadow: 0 0 0 1px var(--color-primary);
}

.terminal-theme-preview {
  display: flex;
  flex: 0 0 auto;
  flex-direction: column;
  justify-content: center;
  gap: 5px;
  width: 70px;
  height: 48px;
  padding: 7px;
  border: 1px solid;
  border-radius: 6px;
  overflow: hidden;
}

.terminal-theme-preview__bar {
  display: block;
  width: 48px;
  height: 4px;
  border-radius: 999px;
}

.terminal-theme-preview__bar--short {
  width: 35px;
}

.terminal-theme-preview__row {
  display: flex;
  gap: 5px;
}

.terminal-theme-preview__row span {
  display: block;
  width: 18px;
  height: 4px;
  border-radius: 999px;
}

.terminal-theme-preview__row--small span {
  width: 13px;
}

.terminal-theme-name {
  min-width: 0;
  overflow: hidden;
  font-size: var(--font-size-base);
  font-weight: 500;
  line-height: 1.2;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.settings-version {
  color: var(--text-secondary);
  font-size: var(--font-size-base);
  font-variant-numeric: tabular-nums;
}

@media (max-width: 720px) {
  .terminal-theme-grid {
    grid-template-columns: 1fr;
  }
}
</style>
