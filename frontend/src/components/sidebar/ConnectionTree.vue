<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, watch, h } from 'vue'
import { useI18n } from 'vue-i18n'
import { NTree, NButton, NInputGroup, NInput, NModal, NCheckbox, useMessage, useDialog } from 'naive-ui'
import type { TreeOption, TreeDropInfo } from 'naive-ui'
import { Dialogs } from '@wailsio/runtime'
import IconFolderPlus from '~icons/lucide/folder-plus'
import IconPlus from '~icons/lucide/plus'
import IconFolder from '~icons/lucide/folder'
import IconPencil from '~icons/lucide/pencil'
import IconTrash2 from '~icons/lucide/trash-2'
import IconZap from '~icons/lucide/zap'
import IconDownload from '~icons/lucide/download'
import IconUpload from '~icons/lucide/upload'
import IconPanelLeftClose from '~icons/lucide/panel-left-close'
import { useConnectionStore } from '../../stores/connection'
import { useTerminalStore } from '../../stores/terminal'
import { LookupIPCountry } from '../../../bindings/vshell/internal/app/appservice'
import { countryForIPv4, flagForCountry } from '../../utils/ipCountry'
import ConnectionFormModal from './ConnectionFormModal.vue'
import type { Connection } from '../../types'

const { t } = useI18n()
const emit = defineEmits<{ (e: 'collapseSidebar'): void }>()
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
const transferringConfig = ref(false)
const showExportModal = ref(false)
const exportPath = ref('')
const encryptExport = ref(false)
const exportPassword = ref('')
const exportPasswordConfirm = ref('')
const showImportPasswordModal = ref(false)
const importPath = ref('')
const importPassword = ref('')
const countryByHost = ref<Record<string, string | null>>({})

function handleGlobalNewConnection() {
  handleNew()
}

onMounted(async () => {
  window.addEventListener('vshell:new-connection', handleGlobalNewConnection)
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

onUnmounted(() => {
  window.removeEventListener('vshell:new-connection', handleGlobalNewConnection)
})

watch(
  () => connectionStore.connections.map(conn => conn.host).join('|'),
  () => {
    void resolveConnectionCountries()
  },
)

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

  return h('div', { class: 'conn-label conn-label-with-flag' }, [
    h('span', { class: 'conn-flag-wrap' }, [
      h('img', {
        class: 'conn-flag',
        src: flagForCountry(countryByHost.value[conn.host] ?? countryForIPv4(conn.host)),
        alt: '',
        onError: (e: Event) => {
          (e.target as HTMLImageElement).src = '/flags/un.png'
        },
      }),
    ]),
    h('span', { class: 'conn-info' }, [
      h('span', { class: 'conn-name' }, conn.name),
      h('span', { class: 'conn-host-text' }, `${conn.host}:${conn.port}`),
    ]),
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

async function resolveConnectionCountries() {
  const hosts = Array.from(new Set(connectionStore.connections.map(conn => conn.host).filter(Boolean)))
  await Promise.all(hosts.map(async (host) => {
    if (Object.prototype.hasOwnProperty.call(countryByHost.value, host)) return
    const fallback = countryForIPv4(host)
    try {
      const country = await LookupIPCountry(host)
      countryByHost.value[host] = country || fallback
    } catch {
      countryByHost.value[host] = fallback
    }
  }))
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

async function handleExportConfigs() {
  const filePath = await Dialogs.SaveFile({
    Title: t('connection.exportConfigs'),
    Filename: 'vshell-connections.json',
    Filters: [{ DisplayName: 'JSON', Pattern: '*.json' }],
  })
  if (!filePath) return

  exportPath.value = filePath
  encryptExport.value = false
  exportPassword.value = ''
  exportPasswordConfirm.value = ''
  showExportModal.value = true
}

async function confirmExportConfigs() {
  if (encryptExport.value) {
    if (!exportPassword.value) {
      message.warning(t('connection.exportPasswordRequired'))
      return
    }
    if (exportPassword.value !== exportPasswordConfirm.value) {
      message.warning(t('connection.exportPasswordMismatch'))
      return
    }
  }

  transferringConfig.value = true
  try {
    await connectionStore.exportConfigs(exportPath.value, encryptExport.value ? exportPassword.value : '')
    showExportModal.value = false
    message.success(t('connection.exported'))
  } catch (e: any) {
    message.error(t('connection.exportFailed', { error: e }))
  } finally {
    transferringConfig.value = false
  }
}

async function importSelectedConfig(filePath: string, password = '') {
  transferringConfig.value = true
  try {
    const result = await connectionStore.importConfigs(filePath, password)
    expandedKeys.value = connectionStore.groups.map(g => g.id)
    showImportPasswordModal.value = false
    message.success(t('connection.imported', { connections: result.connections, groups: result.groups }))
  } catch (e: any) {
    message.error(t('connection.importFailed', { error: e }))
  } finally {
    transferringConfig.value = false
  }
}

async function handleImportConfigs() {
  const filePath = await Dialogs.OpenFile({
    Title: t('connection.importConfigs'),
    CanChooseFiles: true,
    CanChooseDirectories: false,
    AllowsMultipleSelection: false,
    Filters: [{ DisplayName: 'JSON', Pattern: '*.json' }],
  })
  if (!filePath || Array.isArray(filePath)) return

  importPath.value = filePath
  importPassword.value = ''

  let encrypted = false
  try {
    encrypted = await connectionStore.isConfigEncrypted(filePath)
  } catch (e: any) {
    message.error(t('connection.importFailed', { error: e }))
    return
  }

  dialog.warning({
    title: t('connection.importConfigs'),
    content: t('connection.importConfirm'),
    positiveText: t('connection.importConfigs'),
    negativeText: t('common.cancel'),
    onPositiveClick: async () => {
      if (encrypted) {
        showImportPasswordModal.value = true
        return
      }
      await importSelectedConfig(filePath)
    },
  })
}

async function confirmEncryptedImport() {
  if (!importPassword.value) {
    message.warning(t('connection.importPasswordRequired'))
    return
  }
  await importSelectedConfig(importPath.value, importPassword.value)
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

</script>

<template>
  <div class="flex flex-col h-full overflow-hidden bg-[var(--bg-secondary)]">
    <div class="px-3 py-[10px] bg-[var(--bg-tertiary)] flex items-center justify-between shrink-0 relative thin-border-b">
      <span class="text-[var(--font-size-base)] font-semibold text-[var(--text-primary)]">{{ t('connection.title') }}</span>
      <div class="flex gap-[2px]">
        <NButton size="tiny" quaternary @click="emit('collapseSidebar')" :title="t('common.collapse')">
          <IconPanelLeftClose :width="14" :height="14" />
        </NButton>
        <NButton size="tiny" quaternary :loading="transferringConfig" @click="handleImportConfigs" :title="t('connection.importConfigs')">
          <IconUpload :width="14" :height="14" />
        </NButton>
        <NButton size="tiny" quaternary :loading="transferringConfig" @click="handleExportConfigs" :title="t('connection.exportConfigs')">
          <IconDownload :width="14" :height="14" />
        </NButton>
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

    <div class="flex-1 overflow-y-auto px-1 py-2 tree-content">
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
      />
    </div>

    <ConnectionFormModal v-model:show="showModal" :edit-connection="editConn" :defaultGroupID="defaultGroupID" />

    <NModal v-model:show="showExportModal" preset="card" :title="t('connection.exportConfigs')" style="width: 420px" :mask-closable="false">
      <div class="flex flex-col gap-3">
        <NCheckbox v-model:checked="encryptExport">{{ t('connection.encryptExport') }}</NCheckbox>
        <template v-if="encryptExport">
          <NInput
            v-model:value="exportPassword"
            type="password"
            show-password-on="click"
            :placeholder="t('connection.exportPassword')"
            @keyup.enter="confirmExportConfigs"
          />
          <NInput
            v-model:value="exportPasswordConfirm"
            type="password"
            show-password-on="click"
            :placeholder="t('connection.exportPasswordConfirm')"
            @keyup.enter="confirmExportConfigs"
          />
        </template>
        <div class="text-[12px] text-[var(--text-secondary)] leading-relaxed">
          {{ encryptExport ? t('connection.encryptedExportHint') : t('connection.plainExportHint') }}
        </div>
      </div>
      <template #footer>
        <div class="flex justify-end gap-2">
          <NButton @click="showExportModal = false">{{ t('common.cancel') }}</NButton>
          <NButton type="primary" :loading="transferringConfig" @click="confirmExportConfigs">{{ t('connection.exportConfigs') }}</NButton>
        </div>
      </template>
    </NModal>

    <NModal v-model:show="showImportPasswordModal" preset="card" :title="t('connection.decryptImport')" style="width: 420px" :mask-closable="false">
      <div class="flex flex-col gap-3">
        <NInput
          v-model:value="importPassword"
          type="password"
          show-password-on="click"
          :placeholder="t('connection.importPassword')"
          @keyup.enter="confirmEncryptedImport"
        />
        <div class="text-[12px] text-[var(--text-secondary)] leading-relaxed">
          {{ t('connection.encryptedImportHint') }}
        </div>
      </div>
      <template #footer>
        <div class="flex justify-end gap-2">
          <NButton @click="showImportPasswordModal = false">{{ t('common.cancel') }}</NButton>
          <NButton type="primary" :loading="transferringConfig" @click="confirmEncryptedImport">{{ t('connection.importConfigs') }}</NButton>
        </div>
      </template>
    </NModal>
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
  user-select: none;
  -webkit-user-select: none;
}

.tree-content :deep(.n-tree-node-switcher--hide) {
  width: 0;
  margin-right: 0;
}

.tree-content :deep(.conn-label) {
  display: flex;
  flex-direction: column;
  width: 100%;
  min-width: 0;
  line-height: 1.3;
}

.tree-content :deep(.conn-label-with-flag) {
  flex-direction: row;
  align-items: center;
  gap: 9px;
}

.tree-content :deep(.conn-info) {
  display: flex;
  flex-direction: column;
  min-width: 0;
  flex: 1;
}

.tree-content :deep(.conn-name) {
  font-size: var(--font-size-base);
  color: var(--text-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.tree-content :deep(.conn-host) {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 11px;
  color: var(--text-secondary);
  overflow: hidden;
  min-width: 0;
}

.tree-content :deep(.conn-flag-wrap) {
  width: 30px;
  height: 22px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.tree-content :deep(.conn-flag) {
  width: 30px;
  height: 20px;
  object-fit: cover;
  border-radius: 3px;
  flex-shrink: 0;
}

.tree-content :deep(.conn-host-text) {
  font-size: 11px;
  color: var(--text-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  min-width: 0;
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
