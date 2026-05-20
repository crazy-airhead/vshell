<script setup lang="ts">
import { ref, computed, onMounted, h } from 'vue'
import { useI18n } from 'vue-i18n'
import { NTree, NButton, NInputGroup, NInput, useMessage, useDialog } from 'naive-ui'
import type { TreeOption, TreeDropInfo } from 'naive-ui'
import IconFolderPlus from '~icons/lucide/folder-plus'
import IconPlus from '~icons/lucide/plus'
import IconFolder from '~icons/lucide/folder'
import IconPencil from '~icons/lucide/pencil'
import IconTrash2 from '~icons/lucide/trash-2'
import IconZap from '~icons/lucide/zap'
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
      prefix: () => h(IconFolder, { width: 14, height: 14, style: 'margin-right: 4px; opacity: 0.7' }),
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
    const node: TreeOption = {
      key: conn.id,
      label: conn.name,
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

function renderLabel({ option }: { option: TreeOption }) {
  const key = option.key as string
  if (isGroupKey(key)) {
    return option.label as string
  }
  const conn = connectionStore.connections.find(c => c.id === key)
  if (!conn) return option.label as string

  return h('div', { class: 'conn-label' }, [
    h('span', { class: 'conn-name' }, conn.name),
    h('span', { class: 'conn-host' }, `${conn.host}:${conn.port}`),
    h('span', { class: 'conn-actions flex gap-[2px]' }, [
      h('button', {
        class: 'conn-hover-btn',
        title: t('connection.newConnection'),
        onClick: (e: MouseEvent) => { e.stopPropagation(); handleConnect(conn.id) },
      }, h(IconZap, { width: 12, height: 12 })),
      h('button', {
        class: 'conn-hover-btn',
        title: t('common.edit'),
        onClick: (e: MouseEvent) => { e.stopPropagation(); handleEdit(conn.id) },
      }, h(IconPencil, { width: 12, height: 12 })),
      h('button', {
        class: 'conn-hover-btn conn-hover-btn-danger',
        title: t('common.delete'),
        onClick: (e: MouseEvent) => { e.stopPropagation(); handleDelete(conn.id) },
      }, h(IconTrash2, { width: 12, height: 12 })),
    ]),
  ])
}

function nodeProps({ option }: { option: TreeOption }) {
  if (isGroupKey(option.key as string)) return {}
  return {
    onDblclick: () => {
      handleConnect(option.key as string)
    },
  }
}

function handleSelect(keys: string[]) {
  if (keys.length === 0) return
  const key = keys[0]
  if (isGroupKey(key)) return

  // Switch to the first tab belonging to this connection
  const tab = terminalStore.tabs.find(t => t.connectionID === key && t.type !== 'editor')
  if (tab) {
    terminalStore.activeTabID = tab.id
  }
}

async function handleConnect(connID: string) {
  const conn = connectionStore.connections.find(c => c.id === connID)
  if (!conn) return

  try {
    const sessionID = await connectionStore.connect(connID)
    terminalStore.addTab({
      id: sessionID,
      connectionID: conn.id,
      title: conn.name || conn.host || conn.id,
      connected: true,
    })
  } catch (e: any) {
    dialog.error({
      title: t('connection.connectFailed'),
      content: () => h('div', { style: 'line-height:1.6' }, [
        h('div', { style: 'color:var(--text-secondary);font-size:13px;margin-bottom:8px' }, extractErrorMessage(e)),
        h('div', { style: 'font-size:13px' }, t('connection.connectFailedDetail', { host: conn.host })),
      ]),
      positiveText: t('common.close'),
    })
  }
}

function extractErrorMessage(e: any): string {
  if (!e) return 'Unknown error'

  // If it's an Error, unwrap it first
  let raw: any = e
  if (raw instanceof Error) raw = raw.message

  // If it's a string, try parsing as JSON to unwrap nested Wails error
  if (typeof raw === 'string') {
    try {
      const parsed = JSON.parse(raw)
      if (typeof parsed === 'object' && parsed !== null) {
        return extractErrorMessage(parsed)
      }
    } catch { /* not JSON */ }
    return raw
  }

  // Object: check known wrapper keys
  if (typeof raw === 'object') {
    // Prefer specific error fields over generic 'message'
    if (raw.err) return extractErrorMessage(raw.err)
    if (raw.error) return extractErrorMessage(raw.error)
    if (raw.msg) return extractErrorMessage(raw.msg)
    if (raw.message) return extractErrorMessage(raw.message)
  }

  // Last resort
  try { return JSON.stringify(raw) } catch { return String(raw) }
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

function allowDrop({ dropPosition, node }: { dropPosition: 'before' | 'inside' | 'after'; node: TreeOption }) {
  if (isGroupKey(node.key as string)) {
    return dropPosition === 'inside'
  }
  return dropPosition === 'before' || dropPosition === 'after'
}

async function handleDrop({ node, dragNode, dropPosition }: TreeDropInfo) {
  const connID = dragNode.key as string
  if (!connID || isGroupKey(connID)) return

  let groupID: string | null = null
  if (dropPosition === 'inside' && isGroupKey(node.key as string)) {
    groupID = node.key as string
  } else {
    const targetConn = connectionStore.connections.find(c => c.id === node.key)
    groupID = targetConn?.group_id ?? null
  }

  try {
    await connectionStore.moveConnection(connID, groupID)
  } catch (e: any) {
    message.error(t('connection.failed', { error: e }))
  }
}

function onContextMenu(e: MouseEvent, option: TreeOption) {
  e.preventDefault()
  if (isGroupKey(option.key as string)) {
    contextMenuKey.value = option.key as string
  } else {
    contextMenuKey.value = null
  }
}

function clearContextMenu() {
  contextMenuKey.value = null
}
</script>

<template>
  <div class="flex flex-col h-full overflow-hidden bg-[var(--bg-secondary)]" @click="clearContextMenu">
    <div class="px-3 py-[10px] bg-[var(--bg-tertiary)] flex items-center justify-between shrink-0 relative thin-border-b">
      <span class="text-[var(--font-size-base)] font-semibold text-[var(--text-primary)]">{{ t('connection.title') }}</span>
      <div class="flex gap-[2px]">
        <NButton size="tiny" quaternary @click="startNewGroup(null)" :title="t('group.newGroup')">
          <IconFolderPlus :width="14" :height="14" />
        </NButton>
        <NButton size="tiny" quaternary @click="handleNew" :title="t('connection.newConnection')">
          <IconPlus :width="14" :height="14" />
        </NButton>
      </div>
      <div v-if="loading" class="loading-bar"></div>
    </div>

    <div v-if="showGroupInput" class="px-3 py-[6px] thin-border-b shrink-0">
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

    <div class="flex-1 overflow-y-auto p-2 tree-content">
      <NTree
        v-if="!loading"
        :data="treeData"
        :expanded-keys="expandedKeys"
        :render-label="renderLabel"
        :node-props="nodeProps"
        selectable
        block-line
        draggable
        :allow-drop="allowDrop"
        @update:expanded-keys="(keys: string[]) => expandedKeys = keys"
        @update:selected-keys="handleSelect"
        @drop="handleDrop"
        @contextmenu="onContextMenu"
      />
    </div>

    <div v-if="contextMenuKey" class="flex gap-1 px-2 py-1 thin-border-t shrink-0">
      <button class="action-btn" @click="startNewGroup(contextMenuKey!)">{{ t('group.newSubGroup') }}</button>
      <button class="action-btn action-btn-danger" @click="handleDeleteGroup(contextMenuKey!)">{{ t('common.delete') }}</button>
    </div>

    <ConnectionFormModal v-model:show="showModal" :edit-connection="editConn" />
  </div>
</template>

<style scoped>
.thin-border-b { border-bottom: 1px solid rgba(128, 128, 128, 0.12); }
.thin-border-t { border-top: 1px solid rgba(128, 128, 128, 0.12); }

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

.tree-content :deep(.n-tree-node-content) {
  font-size: var(--font-size-base);
}

.tree-content :deep(.conn-label) {
  display: flex;
  flex-direction: column;
  width: 100%;
  min-width: 0;
  line-height: 1.3;
}

.tree-content :deep(.conn-name) {
  font-size: var(--font-size-base);
  color: var(--text-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.tree-content :deep(.conn-host) {
  font-size: 11px;
  color: var(--text-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.tree-content :deep(.conn-actions) {
  position: absolute;
  right: 4px;
  top: 50%;
  transform: translateY(-50%);
  opacity: 0;
  transition: opacity 0.15s;
}

.tree-content :deep(.n-tree-node-content:hover .conn-actions),
.tree-content :deep(.n-tree-node--selected .conn-actions) {
  opacity: 1;
}

.tree-content :deep(.conn-hover-btn) {
  background: none;
  border: none;
  color: var(--text-secondary);
  cursor: pointer;
  padding: 4px 6px;
  border-radius: 3px;
  display: inline-flex;
  align-items: center;
  transition: color 0.15s, background 0.15s;
}
.tree-content :deep(.conn-hover-btn:hover) {
  color: var(--text-primary);
  background: var(--hover-overlay);
}
.tree-content :deep(.conn-hover-btn-danger:hover) {
  color: var(--color-error);
}

.action-btn {
  background: none;
  border: none;
  color: var(--text-secondary);
  font-size: var(--font-size-sm);
  cursor: pointer;
  padding: 2px 8px;
  border-radius: 3px;
  transition: color 0.15s, background 0.15s;
}
.action-btn:hover {
  color: var(--color-primary);
  background: var(--action-hover-bg);
}
.action-btn-danger:hover {
  color: var(--delete-hover-color);
  background: var(--delete-hover-bg);
}
</style>
