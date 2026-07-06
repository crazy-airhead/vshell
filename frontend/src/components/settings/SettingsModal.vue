<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  NModal, NTabs, NTabPane, NFormItem, NSlider, NSpace,
  NButton, NSelect, NInput, useMessage,
} from 'naive-ui'
import { useSettingsStore } from '../../stores/settings'
import {
  GetGeoIPDownloadURL,
  SetGeoIPDownloadURL,
  UpdateGeoIPDatabase,
} from '../../../bindings/vshell/internal/app/appservice'
import type { ShortcutMap } from '../../stores/settings'

const props = defineProps<{ show: boolean }>()
const emit = defineEmits<{ (e: 'update:show', val: boolean): void }>()

const { t } = useI18n()
const settings = useSettingsStore()
const message = useMessage()
const geoIPDownloadURL = ref('')
const updatingGeoIP = ref(false)

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

const colorSchemes = computed(() => [
  { label: t('settings.schemeDefault'), value: 'default' },
  { label: t('settings.schemeSolarizedDark'), value: 'solarized-dark' },
  { label: t('settings.schemeSolarizedLight'), value: 'solarized-light' },
  { label: t('settings.schemeDracula'), value: 'dracula' },
  { label: t('settings.schemeMonokai'), value: 'monokai' },
  { label: t('settings.schemeOneDark'), value: 'one-dark' },
])

const shortcutActions: { key: keyof ShortcutMap; label: string }[] = [
  { key: 'newConnection', label: 'settings.shortcutNewConnection' },
  { key: 'closeTab', label: 'settings.shortcutCloseTab' },
  { key: 'toggleTheme', label: 'settings.shortcutToggleTheme' },
  { key: 'toggleSidebar', label: 'settings.shortcutToggleSidebar' },
  { key: 'focusTerminal', label: 'settings.shortcutFocusTerminal' },
]

const capturingKey = ref<keyof ShortcutMap | null>(null)

function startCapture(action: keyof ShortcutMap) {
  capturingKey.value = action
}

function onKeyCapture(e: KeyboardEvent) {
  if (!capturingKey.value) return
  e.preventDefault()
  e.stopPropagation()

  const parts: string[] = []
  if (e.metaKey || e.ctrlKey) parts.push('CommandOrControl')
  if (e.shiftKey) parts.push('Shift')
  if (e.altKey) parts.push('Alt')

  const key = e.key
  if (!['Control', 'Shift', 'Alt', 'Meta'].includes(key)) {
    parts.push(key.length === 1 ? key.toUpperCase() : key)
    settings.setShortcut(capturingKey.value, parts.join('+'))
    capturingKey.value = null
  }
}

function formatCombo(combo: string): string {
  return combo.replace(/CommandOrControl/g, navigator.platform.includes('Mac') ? '⌘' : 'Ctrl')
    .replace(/\+/g, ' + ')
}

function findFontValue(options: { value: string }[], current: string): string {
  return options.find(o => o.value === current)?.value || options[0].value
}

watch(
  () => props.show,
  async (visible) => {
    if (!visible) return
    try {
      geoIPDownloadURL.value = await GetGeoIPDownloadURL()
    } catch (e: any) {
      message.error(t('settings.geoipLoadFailed', { error: e }))
    }
  },
  { immediate: true },
)

async function saveGeoIPDownloadURL() {
  try {
    await SetGeoIPDownloadURL(geoIPDownloadURL.value)
    geoIPDownloadURL.value = await GetGeoIPDownloadURL()
    message.success(t('settings.geoipSaved'))
  } catch (e: any) {
    message.error(t('settings.geoipSaveFailed', { error: e }))
  }
}

async function updateGeoIPDatabase() {
  updatingGeoIP.value = true
  try {
    await SetGeoIPDownloadURL(geoIPDownloadURL.value)
    await UpdateGeoIPDatabase()
    geoIPDownloadURL.value = await GetGeoIPDownloadURL()
    message.success(t('settings.geoipUpdated'))
  } catch (e: any) {
    message.error(t('settings.geoipUpdateFailed', { error: e }))
  } finally {
    updatingGeoIP.value = false
  }
}
</script>

<template>
  <NModal v-model:show="showModal" preset="card" :title="t('settings.title')" style="width: 520px" :mask-closable="true">
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
          <div class="flex justify-end gap-2">
            <NButton @click="saveGeoIPDownloadURL">{{ t('common.save') }}</NButton>
          </div>
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

          <NFormItem :label="t('settings.colorScheme')" label-placement="left" :show-feedback="false">
            <NSelect v-model:value="settings.terminalColorScheme" :options="colorSchemes" style="width: 200px" @update:value="settings.setTerminalColorScheme" />
          </NFormItem>
        </NSpace>
      </NTabPane>

      <!-- Shortcuts Tab -->
      <NTabPane :name="t('settings.shortcuts')">
        <NSpace vertical :size="10" style="padding: 8px 0" @keydown="onKeyCapture">
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
