<script setup lang="ts">
import { ref, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  NModal, NTabs, NTabPane, NFormItem, NSlider, NSpace,
  NButton, NSelect,
} from 'naive-ui'
import { useSettingsStore } from '../../stores/settings'
import type { ShortcutMap } from '../../stores/settings'

const props = defineProps<{ show: boolean }>()
const emit = defineEmits<{ (e: 'update:show', val: boolean): void }>()

const { t } = useI18n()
const settings = useSettingsStore()

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
</script>

<template>
  <NModal v-model:show="showModal" preset="card" :title="t('settings.title')" style="width: 520px" :mask-closable="true">
    <NTabs type="line" animated>
      <!-- Interface Tab -->
      <NTabPane :name="t('settings.interface')">
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
            <span style="width:32px;text-align:right;font-variant-numeric:tabular-nums">{{ settings.uiFontSize }}</span>
          </NFormItem>

          <NFormItem :label="t('settings.accentColor')" label-placement="left" :show-feedback="false">
            <div class="color-swatches">
              <button
                v-for="c in accentColors" :key="c"
                class="swatch"
                :class="{ active: settings.accentColor === c }"
                :style="{ background: c }"
                @click="settings.setAccentColor(c)"
              />
            </div>
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
            <span style="width:32px;text-align:right;font-variant-numeric:tabular-nums">{{ settings.terminalFontSize }}</span>
          </NFormItem>

          <NFormItem :label="t('settings.colorScheme')" label-placement="left" :show-feedback="false">
            <NSelect v-model:value="settings.terminalColorScheme" :options="colorSchemes" style="width: 200px" @update:value="settings.setTerminalColorScheme" />
          </NFormItem>
        </NSpace>
      </NTabPane>

      <!-- Shortcuts Tab -->
      <NTabPane :name="t('settings.shortcuts')">
        <NSpace vertical :size="10" style="padding: 8px 0" @keydown="onKeyCapture">
          <div v-for="s in shortcutActions" :key="s.key" class="shortcut-row">
            <span class="shortcut-label">{{ t(s.label) }}</span>
            <button
              class="shortcut-key"
              :class="{ capturing: capturingKey === s.key }"
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
.color-swatches {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}

.swatch {
  width: 24px;
  height: 24px;
  border-radius: 50%;
  border: 2px solid transparent;
  cursor: pointer;
  transition: border-color 0.15s, transform 0.1s;
}

.swatch:hover {
  transform: scale(1.15);
}

.swatch.active {
  border-color: var(--text-primary);
}

.shortcut-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.shortcut-label {
  font-size: var(--font-size-base);
  color: var(--text-primary);
}

.shortcut-key {
  background: var(--bg-tertiary);
  border: 1px solid var(--border-color);
  color: var(--text-primary);
  font-size: var(--font-size-sm);
  padding: 4px 12px;
  border-radius: 4px;
  cursor: pointer;
  min-width: 120px;
  text-align: center;
  font-family: monospace;
  transition: border-color 0.15s;
}

.shortcut-key:hover {
  border-color: var(--accent-color);
}

.shortcut-key.capturing {
  border-color: var(--accent-color);
  color: var(--text-secondary);
  font-family: inherit;
}
</style>
