<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { NButton, useMessage, useDialog } from 'naive-ui'
import IconPlus from '~icons/lucide/plus'
import IconPencil from '~icons/lucide/pencil'
import IconTrash2 from '~icons/lucide/trash-2'
import { useConnectionStore } from '../../stores/connection'
import { useTerminalStore } from '../../stores/terminal'
import ConnectionFormModal from './ConnectionFormModal.vue'
import type { Connection } from '../../types'

const connectionStore = useConnectionStore()
const terminalStore = useTerminalStore()
const message = useMessage()
const dialog = useDialog()
const loading = ref(true)
const showModal = ref(false)
const editConn = ref<Connection | null>(null)

onMounted(async () => {
  try {
    await Promise.all([
      connectionStore.loadConnections(),
      connectionStore.loadGroups(),
    ])
  } catch {
    message.error('Failed to load data')
  } finally {
    loading.value = false
  }
})

async function handleConnClick(connectionID: string) {
  if (connectionStore.connectedIDs.has(connectionID)) {
    terminalStore.activeTabID = connectionID
    return
  }
  try {
    await connectionStore.connect(connectionID)
    const conn = connectionStore.connections.find((c) => c.id === connectionID)
    terminalStore.addTab({
      id: connectionID,
      connectionID,
      title: conn?.name || conn?.host || connectionID,
    })
  } catch (e: any) {
    message.error(`Connection failed: ${e}`)
  }
}

function handleDelete(connID: string, connName: string) {
  dialog.warning({
    title: 'Delete Connection',
    content: `Are you sure you want to delete "${connName}"?`,
    positiveText: 'Delete',
    negativeText: 'Cancel',
    onPositiveClick: async () => {
      try {
        await connectionStore.removeConnection(connID)
        terminalStore.removeTab(connID)
        message.success(`Deleted "${connName}"`)
      } catch (e: any) {
        message.error(`Delete failed: ${e}`)
      }
    },
  })
}

function handleEdit(conn: Connection) {
  editConn.value = conn
  showModal.value = true
}

function handleNew() {
  editConn.value = null
  showModal.value = true
}
</script>

<template>
  <div class="w-[280px] min-w-[200px] h-full bg-[var(--bg-secondary)] border-r border-solid border-[var(--border-color)] flex flex-col overflow-hidden">
    <div class="px-4 py-3 flex items-center justify-between relative" style="border-bottom: 1px solid rgba(128, 128, 128, 0.12)">
      <span class="text-[13px] font-semibold text-[var(--text-primary)]">Connections ({{ connectionStore.connections.length }})</span>
      <NButton size="tiny" quaternary @click="handleNew" title="New Connection">
        <IconPlus :width="14" :height="14" />
      </NButton>
      <div v-if="loading" class="loading-bar"></div>
    </div>
    <div class="flex-1 overflow-y-auto p-2">
      <div
        v-if="!loading"
        v-for="conn in connectionStore.connections"
        :key="conn.id"
        class="flex items-center py-[6px] px-2 cursor-pointer rounded-[4px] text-[var(--text-primary)] text-[13px] transition-colors duration-150 hover:bg-[var(--hover-overlay)]"
        @click="handleConnClick(conn.id)"
      >
        <span class="inline-block w-2 h-2 rounded-full mr-2 shrink-0" :style="{ backgroundColor: connectionStore.connectedIDs.has(conn.id) ? 'var(--color-success)' : 'var(--text-secondary)' }"></span>
        <span class="flex-1 overflow-hidden text-ellipsis whitespace-nowrap">{{ conn.name }}</span>
        <span class="text-[var(--text-secondary)] text-[11px] ml-2 shrink-0">{{ conn.host }}</span>
        <button class="conn-action" title="Edit" @click.stop="handleEdit(conn)"><IconPencil :width="12" :height="12" /></button>
        <button class="conn-action conn-action-danger" title="Delete" @click.stop="handleDelete(conn.id, conn.name)"><IconTrash2 :width="12" :height="12" /></button>
      </div>
    </div>
    <ConnectionFormModal v-model:show="showModal" :edit-connection="editConn" />
  </div>
</template>

<style scoped>
.loading-bar {
  position: absolute;
  bottom: -1px;
  left: 0;
  height: 2px;
  width: 100%;
  background: linear-gradient(90deg, transparent, var(--color-primary), transparent);
  animation: loading-slide 0.8s ease-in-out infinite;
}
.loading-bar::after {
  content: '';
  position: absolute;
  top: 0;
  left: -100%;
  width: 100%;
  height: 100%;
  background: linear-gradient(90deg, transparent, var(--color-primary), transparent);
  animation: loading-slide 1.6s ease-in-out 0.4s infinite;
}

@keyframes loading-slide {
  0% { transform: translateX(-100%); }
  100% { transform: translateX(100%); }
}

.conn-action {
  background: none;
  border: none;
  color: var(--text-secondary);
  font-size: 13px;
  cursor: pointer;
  padding: 2px 6px;
  border-radius: 3px;
  margin-left: 4px;
  opacity: 0;
  transition: opacity 0.15s, color 0.15s;
  flex-shrink: 0;
}
.conn-action:hover {
  color: var(--color-primary);
  background: var(--action-hover-bg);
}
.conn-action-danger:hover {
  color: var(--delete-hover-color);
  background: var(--delete-hover-bg);
}
</style>
