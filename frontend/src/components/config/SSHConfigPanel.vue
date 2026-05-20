<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { NEmpty, useMessage } from 'naive-ui'
import IconRefreshCw from '~icons/lucide/refresh-cw'
import IconPencil from '~icons/lucide/pencil'
import IconPlus from '~icons/lucide/plus'
import { useSSHConfigStore } from '../../stores/sshconfig'
import { useTerminalStore } from '../../stores/terminal'
import ConfigEntryForm from './ConfigEntryForm.vue'
import type { SSHConfigEntry } from '../../types'

const { t } = useI18n()
const store = useSSHConfigStore()
const terminalStore = useTerminalStore()
const message = useMessage()

const expandedIndex = ref<number | null>(null)

onMounted(() => {
  store.loadEntries()
})

function handleSave(index: number, entry: SSHConfigEntry) {
  store.updateEntry(index, entry)
  expandedIndex.value = null
}

function handleDelete(index: number) {
  store.deleteEntry(index)
  expandedIndex.value = null
}

function handleAdd() {
  const newEntry: SSHConfigEntry = { type: 'Host', pattern: '', directives: [] }
  store.entries.push(newEntry)
  expandedIndex.value = store.entries.length - 1
}

async function handleEditRaw() {
  const EDITOR_TAB_ID = 'ssh-config-raw'
  try {
    const content = await store.readRaw()
    terminalStore.addEditorTab(EDITOR_TAB_ID, '~/.ssh/config', content, '~/.ssh/config')
  } catch (e: any) {
    message.error(e.message || String(e))
  }
}
</script>

<template>
  <div class="flex flex-col h-full overflow-hidden bg-[var(--bg-secondary)]">
    <div class="px-3 py-[10px] bg-[var(--bg-tertiary)] flex items-center justify-between shrink-0">
      <span class="text-[var(--font-size-base)] font-semibold text-[var(--text-primary)]">{{ t('sshConfig.title') }}</span>
      <div class="flex items-center gap-[2px]">
        <button class="panel-action-btn" @click="store.loadEntries()" :title="t('common.refresh')">
          <IconRefreshCw :width="14" :height="14" />
        </button>
        <button class="panel-action-btn" @click="handleEditRaw" :title="t('sshConfig.editRaw')">
          <IconPencil :width="14" :height="14" />
        </button>
        <button class="panel-action-btn" @click="handleAdd" :title="t('sshConfig.addHost')">
          <IconPlus :width="14" :height="14" />
        </button>
      </div>
    </div>

    <div class="flex-1 overflow-y-auto py-1">
      <div v-if="store.entries.length === 0 && !store.loading" class="px-3 py-10 flex-center">
        <NEmpty :description="t('sshConfig.noEntries')" />
      </div>

      <ConfigEntryForm
        v-for="(entry, index) in store.entries"
        :key="index"
        :entry="entry"
        :index="index"
        :expanded="expandedIndex === index"
        @update:expanded="(v: boolean) => expandedIndex = v ? index : null"
        @save="handleSave"
        @delete="handleDelete"
      />
    </div>
  </div>
</template>

<style scoped>
.panel-action-btn {
  background: none;
  border: none;
  color: var(--text-secondary);
  font-size: 16px;
  cursor: pointer;
  padding: 2px 8px;
  border-radius: 3px;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: color 0.15s, background 0.15s;
}
.panel-action-btn:hover {
  color: var(--text-primary);
  background: var(--hover-overlay);
}
</style>
