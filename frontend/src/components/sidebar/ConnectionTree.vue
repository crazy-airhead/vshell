<script setup lang="ts">
import { ref, computed, onMounted, h } from 'vue'
import { useI18n } from 'vue-i18n'
import { NTree, NButton, NInputGroup, NInput, useMessage, useDialog } from 'naive-ui'
import type { TreeOption, TreeDropInfo } from 'naive-ui'
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
const defaultGroupID = ref<string | null>(null)
const expandedKeys = ref<string[]>([])
const showGroupInput = ref(false)
const newGroupName = ref('')
const newGroupParent = ref<string | null>(null)
const showRenameInput = ref(false)
const renameGroupID = ref<string | null>(null)
const renameGroupName = ref('')

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
      // 分组图标 14px / 间距 4px；状态点同占位（14px 槽 + 4px），文字起点对齐
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
      // 状态点（设计语言 §8）：外层 14px 占位槽，内层 7px 圆点显色——
      // 有绿点 = 在线；未连接不显点但保留占位，与分组行文字起点对齐
      prefix: () => h('span', { class: 'conn-status-dot' }, [
        h('span', {
          class: 'conn-status-dot-dot',
          style: `background:${connectionStore.connectedIDs.has(conn.id) ? 'var(--status-online)' : 'transparent'}`,
        }),
      ]),
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
    return h('div', { class: 'conn-label group-label' }, [
      h('span', { class: 'conn-name' }, option.label as string),
      h('span', { class: 'conn-actions flex gap-[2px]' }, [
        h('button', {
          class: 'conn-hover-btn',
          title: t('connection.newConnection'),
          onClick: (e: MouseEvent) => { e.stopPropagation(); handleNewInGroup(key) },
        }, h(IconPlus, { width: 12, height: 12 })),
        h('button', {
          class: 'conn-hover-btn',
          title: t('group.renameGroup'),
          onClick: (e: MouseEvent) => { e.stopPropagation(); startRenameGroup(key) },
        }, h(IconPencil, { width: 12, height: 12 })),
        h('button', {
          class: 'conn-hover-btn conn-hover-btn-danger',
          title: t('group.deleteGroup'),
          onClick: (e: MouseEvent) => { e.stopPropagation(); handleDeleteGroup(key) },
        }, h(IconTrash2, { width: 12, height: 12 })),
      ]),
    ])
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
  const key = option.key as string
  if (isGroupKey(key)) {
    return {
      // 分组双击展开/收起（§8）
      onDblclick: () => {
        const i = expandedKeys.value.indexOf(key)
        expandedKeys.value = i >= 0
          ? expandedKeys.value.filter(k => k !== key)
          : [...expandedKeys.value, key]
      },
    }
  }
  return {
    // 连接行双行（名称 + IP），行高自适应；分组行保持单行 24
    class: 'conn-node',
    onDblclick: () => {
      handleConnect(key)
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

  // Check if group has connections
  const connsInGroup = connectionStore.getConnectionsByGroup(groupID)
  if (connsInGroup.length > 0) {
    message.warning(t('group.deleteDisabled', { name: group.name }))
    return
  }

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

function startRenameGroup(groupID: string) {
  const group = connectionStore.groups.find(g => g.id === groupID)
  if (!group) return
  renameGroupID.value = groupID
  renameGroupName.value = group.name
  showRenameInput.value = true
}

async function confirmRenameGroup() {
  const name = renameGroupName.value.trim()
  if (!name || !renameGroupID.value) return
  try {
    await connectionStore.updateGroup(renameGroupID.value, name)
    showRenameInput.value = false
    renameGroupID.value = null
    renameGroupName.value = ''
  } catch (e: any) {
    message.error(t('connection.failed', { error: e }))
  }
}

function handleNew() {
  editConn.value = null
  defaultGroupID.value = null
  showModal.value = true
}

function handleNewInGroup(groupID: string) {
  editConn.value = null
  defaultGroupID.value = groupID
  showModal.value = true
}

function startNewGroup(parentID: string | null) {
  newGroupParent.value = parentID
  newGroupName.value = ''
  showGroupInput.value = true
}

/* 壳层头部动作（AppShell #actions，设计语言 §4.3） */
defineExpose({ handleNew, startNewGroup })

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

</script>

<template>
  <div class="flex flex-col h-full overflow-hidden bg-[var(--bg-island)]">
    <!-- 头部标题与动作由壳层 ToolWindow 提供（设计语言 §4.3）；加载条保留在顶部 -->
    <div v-if="loading" class="relative shrink-0" style="height: 0">
      <div class="loading-bar"></div>
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

    <div v-if="showRenameInput" class="px-3 py-[6px] thin-border-b shrink-0">
      <NInputGroup>
        <NInput
          v-model:value="renameGroupName"
          size="tiny"
          :placeholder="t('group.renamePlaceholder')"
          @keyup.enter="confirmRenameGroup"
          @keyup.escape="showRenameInput = false; renameGroupID = null"
        />
        <NButton size="tiny" type="primary" @click="confirmRenameGroup">&#10003;</NButton>
        <NButton size="tiny" @click="showRenameInput = false; renameGroupID = null">&#10005;</NButton>
      </NInputGroup>
    </div>

    <div class="flex-1 overflow-y-auto p-2 tree-content">
      <NTree
        v-if="!loading"
        :data="treeData"
        :expanded-keys="expandedKeys"
        :render-label="renderLabel"
        :node-props="nodeProps"
        :indent="8"
        selectable
        block-line
        draggable
        :allow-drop="allowDrop"
        @update:expanded-keys="(keys: string[]) => expandedKeys = keys"
        @update:selected-keys="handleSelect"
        @drop="handleDrop"
      />
    </div>

    <ConnectionFormModal v-model:show="showModal" :edit-connection="editConn" :defaultGroupID="defaultGroupID" />
  </div>
</template>

<style scoped>
.thin-border-b { border-bottom: 1px solid var(--border-color); }
.thin-border-t { border-top: 1px solid var(--border-color); }

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

/* 行高：分组 24 单行；连接行双行自适应（内容 31px + 余量，防挤） */
.tree-content :deep(.n-tree-node) {
  height: var(--row-h);
  border-radius: var(--radius-m);
  transition: background-color 150ms ease;
}
.tree-content :deep(.n-tree-node.conn-node) {
  height: auto;
  min-height: 34px;
}

/* 单层全行高亮（§8）：统一上在 node 级（覆盖缩进+折叠箭头区，压掉
   Naive 自带的小圆角底色），content 级不再单独上色——避免双层观感 */
.tree-content :deep(.n-tree-node:hover) {
  background: var(--hover) !important;
}
.tree-content :deep(.n-tree-node--selected),
.tree-content :deep(.n-tree-node--selected:hover) {
  background: var(--selection) !important;
}
.tree-content :deep(.n-tree-node-content) {
  font-size: var(--font-size-base);
  user-select: none;
  -webkit-user-select: none;
}
/* 状态点：外层 14px 占位槽（间距 4px，与分组图标一致，文字起点对齐）；
   背景色只上内层 7px 圆点，未连接透明但保留占位 */
.tree-content :deep(.conn-status-dot) {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 14px;
  height: 14px;
  margin-right: 4px;
  flex-shrink: 0;
}
.tree-content :deep(.conn-status-dot-dot) {
  width: 7px;
  height: 7px;
  border-radius: 50%;
}

/* 双行标签：名称（主色）/ host:port（次要色）上下排布，行距 1.3 */
.tree-content :deep(.conn-label) {
  display: flex;
  flex-direction: column;
  justify-content: center;
  gap: 1px;
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
  font-size: var(--font-size-xs);
  color: var(--text-muted);
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

.tree-content :deep(.n-tree-node:hover .conn-actions),
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
</style>
