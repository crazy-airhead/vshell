<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { NButton, NSpin, useMessage, useDialog } from 'naive-ui'
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
  <div class="sidebar">
    <div class="sidebar-header">
      <span class="sidebar-title">Connections ({{ connectionStore.connections.length }})</span>
      <NButton size="tiny" quaternary @click="handleNew" title="New Connection">
        +
      </NButton>
    </div>
    <div class="sidebar-content">
      <NSpin v-if="loading" />
      <div
        v-else
        v-for="conn in connectionStore.connections"
        :key="conn.id"
        class="conn-item"
        @click="handleConnClick(conn.id)"
      >
        <span class="conn-dot" :style="{ backgroundColor: connectionStore.connectedIDs.has(conn.id) ? '#4caf50' : '#666' }"></span>
        <span class="conn-name">{{ conn.name }}</span>
        <span class="conn-host">{{ conn.host }}</span>
        <button class="conn-action" title="Edit" @click.stop="handleEdit(conn)">&#9998;</button>
        <button class="conn-delete" title="Delete" @click.stop="handleDelete(conn.id, conn.name)">x</button>
      </div>
    </div>
    <ConnectionFormModal v-model:show="showModal" :edit-connection="editConn" />
  </div>
</template>

<style scoped>
.sidebar {
  width: 280px;
  min-width: 200px;
  height: 100%;
  background: var(--bg-secondary);
  border-right: 1px solid var(--border-color);
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.sidebar-header {
  padding: 12px 16px;
  border-bottom: 1px solid var(--border-color);
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.sidebar-title {
  font-size: 13px;
  font-weight: 600;
  color: var(--text-primary);
}

.sidebar-content {
  flex: 1;
  overflow-y: auto;
  padding: 8px;
}

.conn-item {
  display: flex;
  align-items: center;
  padding: 6px 8px;
  cursor: pointer;
  border-radius: 4px;
  color: var(--text-primary, #e0e0e0);
  font-size: 13px;
}
.conn-item:hover {
  background: rgba(255, 255, 255, 0.06);
}
.conn-dot {
  display: inline-block;
  width: 8px;
  height: 8px;
  border-radius: 50%;
  margin-right: 8px;
  flex-shrink: 0;
}
.conn-name {
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.conn-host {
  color: #858585;
  font-size: 11px;
  margin-left: 8px;
  flex-shrink: 0;
}
.conn-action {
  background: none;
  border: none;
  color: #666;
  font-size: 13px;
  cursor: pointer;
  padding: 2px 6px;
  border-radius: 3px;
  margin-left: 4px;
  opacity: 0;
  transition: opacity 0.15s, color 0.15s;
  flex-shrink: 0;
}
.conn-item:hover .conn-action {
  opacity: 1;
}
.conn-action:hover {
  color: #59a8f5;
  background: rgba(80, 160, 255, 0.1);
}
.conn-delete {
  background: none;
  border: none;
  color: #666;
  font-size: 12px;
  cursor: pointer;
  padding: 2px 6px;
  border-radius: 3px;
  margin-left: 4px;
  opacity: 0;
  transition: opacity 0.15s, color 0.15s;
  flex-shrink: 0;
}
.conn-item:hover .conn-delete {
  opacity: 1;
}
.conn-delete:hover {
  color: #e55;
  background: rgba(255, 80, 80, 0.1);
}
</style>
