<script setup lang="ts">
import { ref, computed, onMounted, h } from 'vue'
import { NTree, NButton, NSpin, useMessage, useDialog } from 'naive-ui'
import type { TreeOption } from 'naive-ui'
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
const expandedKeys = ref<string[]>([])

onMounted(async () => {
  try {
    await Promise.all([
      connectionStore.loadConnections(),
      connectionStore.loadGroups(),
    ])
    // Expand all groups by default
    expandedKeys.value = connectionStore.groups.map(g => g.id)
  } catch {
    message.error('Failed to load data')
  } finally {
    loading.value = false
  }
})

const treeData = computed<TreeOption[]>(() => {
  const groups = connectionStore.groups
  const connections = connectionStore.connections
  const groupMap = new Map(groups.map(g => [g.id, g]))

  // Build group hierarchy
  const groupNodes: TreeOption[] = []
  const groupNodeMap = new Map<string, TreeOption>()

  // First pass: create all group nodes
  for (const group of groups) {
    const node: TreeOption = {
      key: group.id,
      label: group.name,
      prefix: () => h('span', { style: 'margin-right: 4px; font-size: 13px; opacity: 0.7' }, '\u{1F4C1}'),
      children: [],
    }
    groupNodeMap.set(group.id, node)
  }

  // Build tree structure for nested groups
  for (const group of groups) {
    const node = groupNodeMap.get(group.id)!
    if (group.parent_id && groupNodeMap.has(group.parent_id)) {
      groupNodeMap.get(group.parent_id)!.children!.push(node)
    } else {
      groupNodes.push(node)
    }
  }

  // Assign connections to their groups
  const ungrouped: TreeOption[] = []
  for (const conn of connections) {
    const connected = connectionStore.connectedIDs.has(conn.id)
    const node: TreeOption = {
      key: conn.id,
      label: conn.name,
      prefix: () => h('span', {
        style: `display:inline-block;width:8px;height:8px;border-radius:50%;margin-right:6px;background:${connected ? '#4caf50' : '#666'};flex-shrink:0`
      }),
      suffix: () => h('span', { style: 'color:#858585;font-size:11px' }, conn.host),
    }
    if (conn.group_id && groupNodeMap.has(conn.group_id)) {
      groupNodeMap.get(conn.group_id)!.children!.push(node)
    } else {
      ungrouped.push(node)
    }
  }

  // Remove empty group nodes
  function pruneEmpty(nodes: TreeOption[]): TreeOption[] {
    return nodes.filter(n => {
      if (n.children) {
        n.children = pruneEmpty(n.children)
        return n.children.length > 0
      }
      return true
    })
  }

  const result = pruneEmpty(groupNodes)
  if (ungrouped.length > 0) {
    result.push(...ungrouped)
  }
  return result
})

async function handleSelect(keys: string[]) {
  if (keys.length === 0) return
  const key = keys[0]

  // Check if it's a connection (not a group)
  const conn = connectionStore.connections.find(c => c.id === key)
  if (!conn) return

  if (connectionStore.connectedIDs.has(conn.id)) {
    terminalStore.activeTabID = conn.id
    return
  }
  try {
    await connectionStore.connect(conn.id)
    terminalStore.addTab({
      id: conn.id,
      connectionID: conn.id,
      title: conn.name || conn.host || conn.id,
    })
  } catch (e: any) {
    message.error(`Connection failed: ${e}`)
  }
}

function handleContextMenu(e: MouseEvent, option: TreeOption) {
  // Connection context actions
  const conn = connectionStore.connections.find(c => c.id === option.key)
  if (!conn) return
  // For now, use hover actions (edit/delete below)
}

function handleEdit(connID: string) {
  const conn = connectionStore.connections.find(c => c.id === connID)
  if (conn) {
    editConn.value = conn
    showModal.value = true
  }
}

function handleDelete(connID: string) {
  const conn = connectionStore.connections.find(c => c.id === connID)
  if (!conn) return
  dialog.warning({
    title: 'Delete Connection',
    content: `Are you sure you want to delete "${conn.name}"?`,
    positiveText: 'Delete',
    negativeText: 'Cancel',
    onPositiveClick: async () => {
      try {
        await connectionStore.removeConnection(connID)
        terminalStore.removeTab(connID)
        message.success(`Deleted "${conn.name}"`)
      } catch (e: any) {
        message.error(`Delete failed: ${e}`)
      }
    },
  })
}

function handleNew() {
  editConn.value = null
  showModal.value = true
}

// Track hovered node for action visibility
const hoveredKey = ref<string | null>(null)
</script>

<template>
  <div class="connection-tree">
    <div class="tree-header">
      <span class="tree-title">Connections</span>
      <NButton size="tiny" quaternary @click="handleNew" title="New Connection">
        +
      </NButton>
    </div>
    <div class="tree-content" @mouseleave="hoveredKey = null">
      <NSpin v-if="loading" />
      <NTree
        v-else
        :data="treeData"
        :expanded-keys="expandedKeys"
        selectable
        block-line
        @update:expanded-keys="(keys: string[]) => expandedKeys = keys"
        @update:selected-keys="handleSelect"
      />
    </div>
    <div v-if="hoveredKey && connectionStore.connections.some(c => c.id === hoveredKey)" class="tree-actions">
      <button class="action-btn" @click="handleEdit(hoveredKey!)">Edit</button>
      <button class="action-btn delete" @click="handleDelete(hoveredKey!)">Del</button>
    </div>
    <ConnectionFormModal v-model:show="showModal" :edit-connection="editConn" />
  </div>
</template>

<style scoped>
.connection-tree {
  display: flex;
  flex-direction: column;
  height: 100%;
  overflow: hidden;
  background: var(--bg-secondary);
}

.tree-header {
  padding: 12px 16px;
  border-bottom: 1px solid var(--border-color);
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-shrink: 0;
}

.tree-title {
  font-size: 13px;
  font-weight: 600;
  color: var(--text-primary);
}

.tree-content {
  flex: 1;
  overflow-y: auto;
  padding: 8px;
}

.tree-content :deep(.n-tree-node-content) {
  font-size: 13px;
}

.tree-content :deep(.n-tree-node-content__suffix) {
  margin-left: 8px;
}

.tree-actions {
  display: flex;
  gap: 4px;
  padding: 4px 8px;
  border-top: 1px solid var(--border-color);
  flex-shrink: 0;
}

.action-btn {
  background: none;
  border: none;
  color: #858585;
  font-size: 11px;
  cursor: pointer;
  padding: 2px 8px;
  border-radius: 3px;
}
.action-btn:hover {
  color: #59a8f5;
  background: rgba(80, 160, 255, 0.1);
}
.action-btn.delete:hover {
  color: #e55;
  background: rgba(255, 80, 80, 0.1);
}
</style>
