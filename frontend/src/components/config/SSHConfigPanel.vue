<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { NEmpty, NModal, NCheckbox, NButton, NSpace, useMessage } from 'naive-ui'
import IconRefreshCw from '~icons/lucide/refresh-cw'
import IconPencil from '~icons/lucide/pencil'
import IconPlus from '~icons/lucide/plus'
import IconDownload from '~icons/lucide/download'
import IconCheckCircle from '~icons/lucide/check-circle'
import IconXCircle from '~icons/lucide/x-circle'
import { useSSHConfigStore } from '../../stores/sshconfig'
import { useTerminalStore } from '../../stores/terminal'
import { useConnectionStore } from '../../stores/connection'
import ConfigEntryForm from './ConfigEntryForm.vue'
import type { SSHConfigEntry, SSHConfigImportCandidate } from '../../types'

const { t } = useI18n()
const store = useSSHConfigStore()
const terminalStore = useTerminalStore()
const connectionStore = useConnectionStore()
const message = useMessage()

const expandedIndex = ref<number | null>(null)

// Import modal state
const showImportModal = ref(false)
const importCandidates = ref<SSHConfigImportCandidate[]>([])
const selectedPatterns = ref<Set<string>>(new Set())
const importing = ref(false)
const loadingCandidates = ref(false)

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

async function openImportModal() {
  selectedPatterns.value = new Set()
  loadingCandidates.value = true
  showImportModal.value = true
  try {
    importCandidates.value = await store.getImportCandidates()
  } catch (e: any) {
    message.error(e.message || String(e))
  } finally {
    loadingCandidates.value = false
  }
}

function toggleSelectAll() {
  if (selectedPatterns.value.size === importCandidates.value.length) {
    selectedPatterns.value = new Set()
  } else {
    selectedPatterns.value = new Set(importCandidates.value.map((c) => c.pattern))
  }
}

function togglePattern(pattern: string) {
  const next = new Set(selectedPatterns.value)
  if (next.has(pattern)) {
    next.delete(pattern)
  } else {
    next.add(pattern)
  }
  selectedPatterns.value = next
}

async function handleImport() {
  if (selectedPatterns.value.size === 0) {
    message.warning(t('sshConfig.importNothingSelected'))
    return
  }
  importing.value = true
  try {
    await store.importHosts([...selectedPatterns.value])
    await connectionStore.loadConnections()
    message.success(t('sshConfig.imported', { count: selectedPatterns.value.size }))
    showImportModal.value = false
  } catch (e: any) {
    message.error(e.message || String(e))
  } finally {
    importing.value = false
  }
}

function isAllSelected() {
  return importCandidates.value.length > 0 && selectedPatterns.value.size === importCandidates.value.length
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
        <button class="panel-action-btn" @click="openImportModal" :title="t('sshConfig.importHosts')">
          <IconDownload :width="14" :height="14" />
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

    <!-- Import Modal -->
    <NModal v-model:show="showImportModal" preset="card" :title="t('sshConfig.importHosts')" style="width: 640px" :mask-closable="false">
      <p class="text-[var(--text-secondary)] mb-3 text-sm">{{ t('sshConfig.importHostsDesc') }}</p>

      <div v-if="loadingCandidates" class="py-8 flex-center">
        <span class="text-[var(--text-secondary)]">{{ t('common.refresh') }}...</span>
      </div>

      <div v-else-if="importCandidates.length === 0" class="py-8 flex-center">
        <NEmpty :description="t('sshConfig.noEntries')" />
      </div>

      <template v-else>
        <div class="mb-2 flex items-center gap-2">
          <NButton size="small" text @click="toggleSelectAll">
            {{ isAllSelected() ? t('sshConfig.importDeselectAll') : t('sshConfig.importSelectAll') }}
          </NButton>
          <span class="text-xs text-[var(--text-secondary)]">{{ selectedPatterns.size }} / {{ importCandidates.length }}</span>
        </div>

        <div class="max-h-80 overflow-y-auto">
          <div
            v-for="candidate in importCandidates"
            :key="candidate.pattern"
            class="flex items-center gap-3 py-2 px-1 border-b border-[var(--border-color)] last:border-b-0 cursor-pointer hover:bg-[var(--hover-overlay)]"
            @click="togglePattern(candidate.pattern)"
          >
            <NCheckbox :checked="selectedPatterns.has(candidate.pattern)" @click.stop="togglePattern(candidate.pattern)" />
            <div class="flex-1 min-w-0">
              <div class="text-sm font-medium text-[var(--text-primary)] truncate">{{ candidate.pattern }}</div>
              <div class="text-xs text-[var(--text-secondary)] flex flex-wrap gap-x-3 gap-y-0.5 mt-0.5">
                <span v-if="candidate.hostname">{{ candidate.hostname }}:{{ candidate.port }}</span>
                <span v-else>{{ candidate.pattern }}:{{ candidate.port }}</span>
                <span v-if="candidate.user">{{ candidate.user }}</span>
                <span v-if="candidate.identity_file" class="flex items-center gap-1">
                  <IconCheckCircle v-if="candidate.has_key" :width="12" :height="12" class="text-green-500" />
                  <IconXCircle v-else :width="12" :height="12" class="text-red-400" />
                  {{ candidate.identity_file }}
                </span>
              </div>
            </div>
          </div>
        </div>
      </template>

      <template #footer>
        <NSpace justify="end">
          <NButton @click="showImportModal = false">{{ t('common.cancel') }}</NButton>
          <NButton type="primary" :loading="importing" :disabled="selectedPatterns.size === 0" @click="handleImport">
            {{ t('common.save') }} ({{ selectedPatterns.size }})
          </NButton>
        </NSpace>
      </template>
    </NModal>
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
