<script setup lang="ts">
import { reactive, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { NInput, NInputNumber, NButton, NPopconfirm, NSpace } from 'naive-ui'
import type { SSHConfigEntry, SSHConfigDirective } from '../../types'

const props = defineProps<{
  entry: SSHConfigEntry
  index: number
  expanded: boolean
}>()

const emit = defineEmits<{
  (e: 'save', index: number, entry: SSHConfigEntry): void
  (e: 'delete', index: number): void
  (e: 'update:expanded', val: boolean): void
}>()

const { t } = useI18n()

const form = reactive<SSHConfigEntry>({ type: 'Host', pattern: '', directives: [] })

watch(() => props.expanded, (val) => {
  if (val) {
    form.type = props.entry.type
    form.pattern = props.entry.pattern
    form.directives = props.entry.directives.map(d => ({ ...d }))
  }
}, { immediate: true })

function getDirective(key: string): string {
  return form.directives.find(d => d.key.toLowerCase() === key.toLowerCase())?.value || ''
}

function setDirective(key: string, value: string) {
  const idx = form.directives.findIndex(d => d.key.toLowerCase() === key.toLowerCase())
  if (idx >= 0) {
    if (value) {
      form.directives[idx] = { key, value }
    } else {
      form.directives.splice(idx, 1)
    }
  } else if (value) {
    form.directives.push({ key, value })
  }
}

function removeDirective(key: string) {
  const idx = form.directives.findIndex(d => d.key.toLowerCase() === key.toLowerCase())
  if (idx >= 0) form.directives.splice(idx, 1)
}

const commonKeys = ['HostName', 'User', 'Port', 'IdentityFile']
const extraDirectives = () => form.directives.filter(d => !commonKeys.includes(d.key) && d.key !== '#')

function addDirective() {
  form.directives.push({ key: '', value: '' })
}

function handleSave() {
  emit('save', props.index, { type: form.type, pattern: form.pattern, directives: [...form.directives] })
  emit('update:expanded', false)
}

function handleCancel() {
  emit('update:expanded', false)
}

function handleDelete() {
  emit('delete', props.index)
}

function displayMeta(entry: SSHConfigEntry): string {
  const parts: string[] = []
  const h = entry.directives.find(d => d.key === 'HostName')?.value
  const u = entry.directives.find(d => d.key === 'User')?.value
  const p = entry.directives.find(d => d.key === 'Port')?.value
  if (u) parts.push(u)
  if (h) parts.push('@' + h)
  if (p && p !== '22') parts.push(':' + p)
  return parts.length ? parts.join('') : ''
}
</script>

<template>
  <div :class="{ '': expanded }">
    <!-- Collapsed view -->
    <div v-if="!expanded" class="group flex items-center justify-between px-3 py-2 cursor-pointer transition-colors duration-150 hover:bg-[var(--hover-overlay)]">
      <div class="flex-1 min-w-0">
        <div class="flex items-center gap-[6px]">
          <span class="text-[10px] py-[1px] px-[5px] rounded-[3px] bg-[var(--color-primary)] text-white font-semibold leading-[1.4] shrink-0">Host</span>
          <span class="text-[var(--font-size-base)] text-[var(--text-primary)] font-medium whitespace-nowrap overflow-hidden text-ellipsis">{{ entry.pattern }}</span>
        </div>
        <div v-if="displayMeta(entry)" class="text-[11px] text-[var(--text-secondary)] mt-[2px] ml-[40px] whitespace-nowrap overflow-hidden text-ellipsis">{{ displayMeta(entry) }}</div>
      </div>
      <div class="flex gap-[2px] shrink-0 opacity-0 group-hover:opacity-100 transition-opacity duration-150">
        <button class="entry-action-btn" @click.stop="emit('update:expanded', true)" title="Edit">
          <svg width="12" height="12" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5">
            <path d="M11.5 1.5l3 3L5 14H2v-3L11.5 1.5z" />
          </svg>
        </button>
        <NPopconfirm @positive-click="handleDelete">
          <template #trigger>
            <button class="entry-action-btn entry-action-danger" @click.stop title="Delete">
              <svg width="12" height="12" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5">
                <path d="M3 4h10M5 4V3a1 1 0 011-1h4a1 1 0 011 1v1M6 7v4M10 7v4M4 4l.7 9.4a1 1 0 001 .6h4.6a1 1 0 001-.6L12 4" stroke-linecap="round" stroke-linejoin="round" />
              </svg>
            </button>
          </template>
          {{ t('sshConfig.deleteConfirm', { name: entry.pattern }) }}
        </NPopconfirm>
      </div>
    </div>

    <!-- Expanded edit form -->
    <div v-else class="px-3 py-[10px] flex flex-col gap-[6px] bg-[var(--hover-overlay)]">
      <div class="flex items-center gap-2">
        <label class="text-[var(--font-size-sm)] text-[var(--text-secondary)] w-[70px] shrink-0 text-right">{{ t('sshConfig.hostPattern') }}</label>
        <NInput v-model:value="form.pattern" size="small" :placeholder="t('sshConfig.hostPatternPlaceholder')" />
      </div>
      <div class="flex items-center gap-2">
        <label class="text-[var(--font-size-sm)] text-[var(--text-secondary)] w-[70px] shrink-0 text-right">{{ t('sshConfig.hostName') }}</label>
        <NInput :value="getDirective('HostName')" size="small" :placeholder="t('sshConfig.hostNamePlaceholder')" @update:value="(v: string) => setDirective('HostName', v)" />
      </div>
      <div class="flex items-center gap-2">
        <label class="text-[var(--font-size-sm)] text-[var(--text-secondary)] w-[70px] shrink-0 text-right">{{ t('sshConfig.user') }}</label>
        <NInput :value="getDirective('User')" size="small" :placeholder="t('sshConfig.userPlaceholder')" @update:value="(v: string) => setDirective('User', v)" />
      </div>
      <div class="flex items-center gap-2">
        <label class="text-[var(--font-size-sm)] text-[var(--text-secondary)] w-[70px] shrink-0 text-right">{{ t('sshConfig.port') }}</label>
        <NInputNumber :value="parseInt(getDirective('Port')) || null" size="small" :min="1" :max="65535" placeholder="22" style="width: 100%" @update:value="(v: number | null) => setDirective('Port', v ? String(v) : '')" />
      </div>
      <div class="flex items-center gap-2">
        <label class="text-[var(--font-size-sm)] text-[var(--text-secondary)] w-[70px] shrink-0 text-right">{{ t('sshConfig.identityFile') }}</label>
        <NInput :value="getDirective('IdentityFile')" size="small" :placeholder="t('sshConfig.identityFilePlaceholder')" @update:value="(v: string) => setDirective('IdentityFile', v)" />
      </div>

      <!-- Extra directives -->
      <div v-for="(dir, i) in extraDirectives()" :key="i" class="flex items-center gap-2 ml-[78px]">
        <NInput v-model:value="dir.key" size="small" :placeholder="t('sshConfig.directiveKey')" class="w-[100px] shrink-0" />
        <NInput v-model:value="dir.value" size="small" :placeholder="t('sshConfig.directiveValue')" class="flex-1" />
        <button class="directive-remove" @click="removeDirective(dir.key)">
          <svg width="10" height="10" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="2"><path d="M4 4l8 8M12 4l-8 8" /></svg>
        </button>
      </div>

      <button class="add-directive-btn ml-[78px]" @click="addDirective">+ {{ t('sshConfig.addDirective') }}</button>

      <div class="flex justify-end mt-1">
        <NSpace>
          <NButton size="small" @click="handleCancel">{{ t('common.cancel') }}</NButton>
          <NButton size="small" type="primary" @click="handleSave" :disabled="!form.pattern.trim()">{{ t('common.save') }}</NButton>
        </NSpace>
      </div>
    </div>
  </div>
</template>

<style scoped>
.entry-action-btn {
  background: none;
  border: none;
  color: var(--text-secondary);
  cursor: pointer;
  padding: 4px 6px;
  border-radius: 3px;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: color 0.15s, background 0.15s;
}
.entry-action-btn:hover {
  color: var(--text-primary);
  background: var(--hover-overlay);
}
.entry-action-danger:hover {
  color: var(--color-error);
}

.directive-remove {
  background: none;
  border: none;
  color: var(--text-secondary);
  cursor: pointer;
  padding: 2px;
  border-radius: 3px;
  display: flex;
  align-items: center;
  transition: color 0.15s;
}
.directive-remove:hover {
  color: var(--color-error);
}

.add-directive-btn {
  background: none;
  border: 1px dashed var(--border-color);
  color: var(--text-secondary);
  font-size: var(--font-size-sm);
  cursor: pointer;
  padding: 4px 8px;
  border-radius: 4px;
  transition: color 0.15s, border-color 0.15s;
}
.add-directive-btn:hover {
  color: var(--color-primary);
  border-color: var(--color-primary);
}
</style>
