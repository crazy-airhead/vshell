<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { NEmpty, useMessage } from 'naive-ui'
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
  <div class="config-panel">
    <div class="panel-header">
      <span class="panel-title">{{ t('sshConfig.title') }}</span>
      <div class="panel-header-actions">
        <button class="panel-action-btn" @click="store.loadEntries()" :title="t('common.refresh')">
          <svg width="14" height="14" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5">
            <path d="M13.65 2.35A7.5 7.5 0 1 0 15.5 8.5" stroke-linecap="round" />
            <path d="M13.65 0.5v2.5h2.5" stroke-linecap="round" stroke-linejoin="round" />
          </svg>
        </button>
        <button class="panel-action-btn" @click="handleEditRaw" :title="t('sshConfig.editRaw')">
          <svg width="14" height="14" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5">
            <path d="M11.5 1.5l3 3L5 14H2v-3L11.5 1.5z" />
          </svg>
        </button>
        <button class="panel-action-btn" @click="handleAdd" :title="t('sshConfig.addHost')">
          <svg width="14" height="14" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5">
            <path d="M2 8h12M8 2v12" stroke-linecap="round" />
          </svg>
        </button>
      </div>
    </div>

    <div class="entry-list">
      <div v-if="store.entries.length === 0 && !store.loading" class="panel-body-empty">
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
.config-panel {
  display: flex;
  flex-direction: column;
  height: 100%;
  overflow: hidden;
  background: var(--bg-secondary);
}

.panel-header {
  padding: 10px 12px;
  background: var(--bg-tertiary);
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-shrink: 0;
}

.panel-header-actions {
  display: flex;
  align-items: center;
  gap: 2px;
}

.panel-title {
  font-size: var(--font-size-base);
  font-weight: 600;
  color: var(--text-primary);
}

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
}

.panel-action-btn:hover {
  color: var(--text-primary);
  background: var(--hover-overlay);
}

.entry-list {
  flex: 1;
  overflow-y: auto;
  padding: 4px 0;
}

.panel-body-empty {
  padding: 40px 12px;
  display: flex;
  align-items: center;
  justify-content: center;
}
</style>
