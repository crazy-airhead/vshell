<script setup lang="ts">
import { ref, computed, onMounted, h } from 'vue'
import { useI18n } from 'vue-i18n'
import { NTree, NButton, NSpin, NInputGroup, NInput, useMessage, useDialog } from 'naive-ui'
import type { TreeOption } from 'naive-ui'
import { useConnectionStore } from '../../stores/connection'
import { useTerminalStore } from '../../stores/terminal'
import ConnectionFormModal from './ConnectionFormModal.vue'
import type { Connection } from '../../types'

const { t } = useI18n()
const connectionStore = useConnectionStore()
const terminalStore = useTerminalStore()
const message = useMessage()
const dialog = useDialog()
const loading = ref(true)
const showModal = ref(false)
const editConn = ref<Connection | null>(null)
const expandedKeys = ref<string[]>([])
const showGroupInput = ref(false)
const newGroupName = ref('')
const newGroupParent = ref<string | null>(null)
const contextMenuKey = ref<string | null>(null)

onMounted(async () => {
  try {
    await Promise.all([
      connectionStore.loadConnections(),
      connectionStore.loadGroups(),
    ])
    expandedKeys.value = connectionStore.groups.map(g => g.id)
  } catch {
    message.error(t('connection.loadFailed'))
  } finally {
    loading.value = false
  }
})

function isGroupKey(key: string): boolean {
  return connectionStore.groups.some(g => g.id === key)
}

const treeData = computed<TreeOption[]>(() => {
  const groups = connectionStore.groups
  const connections = connectionStore.connections

  const groupNodes: TreeOption[] = []
  const groupNodeMap = new Map<string, TreeOption>()

  for (const group of groups) {
    const node: TreeOption = {
      key: group.id,
      label: group.name,
      prefix: () => h('span', { style: 'margin-right: 4px; font-size: 13px; opacity: 0.7' }, '\u{1F4C1}'),
      children: [],
    }
    groupNodeMap.set(group.id, node)
  }

  for (const group of groups) {
    const node = groupNodeMap.get(group.id)!
    if (group.parent_id && groupNodeMap.has(group.parent_id)) {
      groupNodeMap.get(group.parent_id)!.children!.push(node)
    } else {
      groupNodes.push(node)
    }
  }

  const ungrouped: TreeOption[] = []
  for (const conn of connections) {
    const connected = connectionStore.connectedIDs.has(conn.id)
    const node: TreeOption = {
      key: conn.id,
      label: conn.name,
      prefix: () => h('span', {
        style: `display:inline-block;width:8px;height:8px;border-radius:50%;margin-right:6px;background:${connected ? '#4caf50' : '#666'};flex-shrink:0`
      }),
      suffix: () => h('span', { style: 'color:var(--text-secondary);font-size:11px' }, conn.host),
    }
    if (conn.group_id && groupNodeMap.has(conn.group_id)) {
      groupNodeMap.get(conn.group_id)!.children!.push(node)
    } else {
      ungrouped.push(node)
    }
  }

  const result = [...groupNodes]
  if (ungrouped.length > 0) {
    result.push(...ungrouped)
  }
  return result
})

async function handleSelect(keys: string[]) {
  if (keys.length === 0) return
  const key = keys[0]
  if (isGroupKey(key)) return

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
    message.error(t('connection.connectFailed', { error: e }))
  }
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
    title: t('connection.deleteTitle'),
    content: t('connection.deleteContent', { name: conn.name }),
    positiveText: t('common.delete'),
    negativeText: t('common.cancel'),
    onPositiveClick: async () => {
      try {
        await connectionStore.removeConnection(connID)
        terminalStore.removeTab(connID)
        message.success(t('connection.deleted', { name: conn.name }))
      } catch (e: any) {
        message.error(t('connection.deleteFailed', { error: e }))
      }
    },
  })
}

function handleDeleteGroup(groupID: string) {
  const group = connectionStore.groups.find(g => g.id === groupID)
  if (!group) return
  dialog.warning({
    title: t('group.deleteGroup'),
    content: t('group.deleteContent', { name: group.name }),
    positiveText: t('common.delete'),
    negativeText: t('common.cancel'),
    onPositiveClick: async () => {
      try {
        await connectionStore.removeGroup(groupID)
      } catch (e: any) {
        message.error(t('connection.failed', { error: e }))
      }
    },
  })
}

function handleNew() {
  editConn.value = null
  showModal.value = true
}

function startNewGroup(parentID: string | null) {
  newGroupParent.value = parentID
  newGroupName.value = ''
  showGroupInput.value = true
}

async function confirmNewGroup() {
  const name = newGroupName.value.trim()
  if (!name) {
    message.warning(t('group.nameRequired'))
    return
  }
  try {
    await connectionStore.createGroup(name, newGroupParent.value)
    showGroupInput.value = false
    newGroupName.value = ''
  } catch (e: any) {
    message.error(t('connection.failed', { error: e }))
  }
}

function onContextMenu(e: MouseEvent, option: TreeOption) {
  e.preventDefault()
  contextMenuKey.value = option.key as string
}

function clearContextMenu() {
  contextMenuKey.value = null
}

const contextIsGroup = computed(() => contextMenuKey.value ? isGroupKey(contextMenuKey.value) : false)
const contextIsConnection = computed(() => contextMenuKey.value ? !isGroupKey(contextMenuKey.value) : false)
</script>

<template>
  <div class="connection-tree" @click="clearContextMenu">
    <div class="tree-header">
      <span class="tree-title">{{ t('connection.title') }}</span>
      <div class="tree-header-actions">
        <NButton size="tiny" quaternary @click="startNewGroup(null)" :title="t('group.newGroup')">
          <span style="font-size:13px">&#x2295;</span>
        </NButton>
        <NButton size="tiny" quaternary @click="handleNew" :title="t('connection.newConnection')">
          +
        </NButton>
      </div>
    </div>

    <!-- New group inline input -->
    <div v-if="showGroupInput" class="group-input-row">
      <NInputGroup>
        <NInput
          v-model:value="newGroupName"
          size="tiny"
          :placeholder="t('group.namePlaceholder')"
          @keyup.enter="confirmNewGroup"
          @keyup.escape="showGroupInput = false"
        />
        <NButton size="tiny" type="primary" @click="confirmNewGroup">&#10003;</NButton>
        <NButton size="tiny" @click="showGroupInput = false">&#10005;</NButton>
      </NInputGroup>
    </div>

    <div class="tree-content">
      <NSpin v-if="loading" />
      <NTree
        v-else
        :data="treeData"
        :expanded-keys="expandedKeys"
        selectable
        block-line
        @update:expanded-keys="(keys: string[]) => expandedKeys = keys"
        @update:selected-keys="handleSelect"
        @contextmenu="onContextMenu"
      />
    </div>

    <!-- Context actions -->
    <div v-if="contextMenuKey" class="tree-actions">
      <template v-if="contextIsConnection">
        <button class="action-btn" @click="handleEdit(contextMenuKey!)">{{ t('common.edit') }}</button>
        <button class="action-btn delete" @click="handleDelete(contextMenuKey!)">{{ t('common.delete') }}</button>
      </template>
      <template v-if="contextIsGroup">
        <button class="action-btn" @click="startNewGroup(contextMenuKey!)">{{ t('group.newSubGroup') }}</button>
        <button class="action-btn delete" @click="handleDeleteGroup(contextMenuKey!)">{{ t('common.delete') }}</button>
      </template>
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
  padding: 10px 12px;
  border-bottom: 1px solid var(--border-color);
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-shrink: 0;
}

.tree-title {
  font-size: var(--font-size-base);
  font-weight: 600;
  color: var(--text-primary);
}

.tree-header-actions {
  display: flex;
  gap: 2px;
}

.group-input-row {
  padding: 6px 12px;
  border-bottom: 1px solid var(--border-color);
  flex-shrink: 0;
}

.tree-content {
  flex: 1;
  overflow-y: auto;
  padding: 8px;
}

.tree-content :deep(.n-tree-node-content) {
  font-size: var(--font-size-base);
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
  color: var(--text-secondary);
  font-size: var(--font-size-sm);
  cursor: pointer;
  padding: 2px 8px;
  border-radius: 3px;
}
.action-btn:hover {
  color: var(--action-hover-color);
  background: var(--action-hover-bg);
}
.action-btn.delete:hover {
  color: var(--delete-hover-color);
  background: var(--delete-hover-bg);
}
</style>
