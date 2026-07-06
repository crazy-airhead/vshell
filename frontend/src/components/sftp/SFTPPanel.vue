<script setup lang="ts">
import { ref, computed, watch, onMounted, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { NTree, NButton, useDialog, useMessage } from 'naive-ui'
import type { TreeOption } from 'naive-ui'
import { Events } from '@wailsio/runtime'
import IconRefreshCw from '~icons/lucide/refresh-cw'
import IconDownload from '~icons/lucide/download'
import IconUpload from '~icons/lucide/upload'
import IconTrash2 from '~icons/lucide/trash-2'
import IconFolder from '~icons/lucide/folder'
import IconFile from '~icons/lucide/file'
import IconFolderOpen from '~icons/lucide/folder-open'
import IconX from '~icons/lucide/x'
import { useSFTPStore } from '../../stores/sftp'
import { useTransferStore } from '../../stores/transfers'
import { useTerminalStore } from '../../stores/terminal'
import { useConnectionStore } from '../../stores/connection'
import type { SFTPFile } from '../../stores/sftp'
import type { TransferProgress } from '../../stores/transfers'
import { SFTPUpload, SFTPDownload, SFTPDelete, SFTPReadFileContent, SFTPCancelTransfers, GetHomeDir, ListLocalDir, DeleteLocalFile, ReadLocalFileContent, OpenInFileManager } from '../../../bindings/vshell/internal/app/appservice'
import { useDragSource, useDropTarget } from '../../composables/useDragTransfer'
import { isEditableFile } from '../../utils/fileType'

const props = defineProps<{ connectionID: string }>()
const { t } = useI18n()
const sftpStore = useSFTPStore()
const transferStore = useTransferStore()
const terminalStore = useTerminalStore()
const connectionStore = useConnectionStore()
const dialog = useDialog()
const message = useMessage()

// ===========================
// Remote side state & methods
// ===========================
const treeData = ref<TreeOption[]>([])
const expandedKeys = ref<string[]>([])
const treeWidth = ref(170)
const editingRemotePath = ref(false)
const editRemotePath = ref('')
const loadingRemoteFile = ref(false)
const selectedRemote = ref(new Set<string>())
const remoteSortKey = ref<'name' | 'size' | 'time'>('name')
const remoteSortAsc = ref(true)
const remoteDir = ref('')

const sortedRemoteFiles = computed(() => {
  const p = sftpStore.getPanel(props.connectionID)
  const files = [...p.files]
  const key = remoteSortKey.value
  const asc = remoteSortAsc.value
  files.sort((a: SFTPFile, b: SFTPFile) => {
    if (a.is_dir !== b.is_dir) return a.is_dir ? -1 : 1
    let cmp = 0
    if (key === 'name') cmp = a.name.localeCompare(b.name)
    else if (key === 'size') cmp = a.size - b.size
    else cmp = a.mod_time - b.mod_time
    return asc ? cmp : -cmp
  })
  return files
})

function toggleRemoteSort(key: 'name' | 'size' | 'time') {
  if (remoteSortKey.value === key) remoteSortAsc.value = !remoteSortAsc.value
  else { remoteSortKey.value = key; remoteSortAsc.value = true }
}

// Remote drag source
const { onRowPointerDown: onRemoteRowPointerDown, cleanup: cleanupRemoteDrag } = useDragSource({
  source: 'remote',
  getSelectedPaths: () => selectedRemote.value,
  getFilePath: (file: SFTPFile) => remoteFilePath(file.name),
  getFileLabel: (file: SFTPFile) => file.name,
})

// Remote drop target (accepts local→remote uploads)
const { targetRef: remoteDropRef, isDragOver: remoteIsDragOver, register: registerRemoteDrop, unregister: unregisterRemoteDrop } = useDropTarget({
  acceptedSource: 'local',
  onDrop: (paths: string[]) => handleUploadDrop(paths),
})

// Remote path
const remotePathParts = computed(() => {
  const p = sftpStore.getPanel(props.connectionID)
  const parts = p.currentPath.split('/').filter(Boolean)
  return parts.map((name, i) => ({
    name,
    path: '/' + parts.slice(0, i + 1).join('/'),
  }))
})

function navigateRemoteTo(dirPath: string) {
  selectedRemote.value = new Set()
  sftpStore.navigateToDir(props.connectionID, dirPath)
  rebuildTree()
}

function startRemoteEdit() {
  editRemotePath.value = sftpStore.getPanel(props.connectionID).currentPath
  editingRemotePath.value = true
}

function commitRemoteEdit() {
  editingRemotePath.value = false
  navigateRemoteTo(editRemotePath.value)
}

// Remote tree
function buildTreeNodes(parentPath: string, cache: Record<string, SFTPFile[]>): TreeOption[] {
  const items = cache[parentPath]
  if (!items) return []
  return items.filter((f) => f.is_dir).map((f) => {
    const fullPath = parentPath === '/' ? `/${f.name}` : `${parentPath}/${f.name}`
    return {
      key: fullPath,
      label: f.name,
      children: fullPath in cache ? buildTreeNodes(fullPath, cache) : undefined,
      isLeaf: false,
    }
  })
}

function rebuildTree() {
  const p = sftpStore.getPanel(props.connectionID)
  treeData.value = buildTreeNodes('/', p.treeCache)
}

async function handleLoad(node: TreeOption) {
  const path = node.key as string
  await sftpStore.loadTreeDir(props.connectionID, path)
  const p = sftpStore.getPanel(props.connectionID)
  node.children = buildTreeNodes(path, p.treeCache)
}

function handleTreeSelect(keys: string[]) {
  if (keys.length === 0) return
  navigateRemoteTo(keys[0])
}

// Remote file list
function remoteFilePath(name: string): string {
  const p = sftpStore.getPanel(props.connectionID)
  return p.currentPath === '/' ? `/${name}` : `${p.currentPath}/${name}`
}

function toggleRemoteSelect(file: SFTPFile, e: MouseEvent) {
  const fp = remoteFilePath(file.name)
  if (e.ctrlKey || e.metaKey) {
    if (selectedRemote.value.has(fp)) selectedRemote.value.delete(fp)
    else selectedRemote.value.add(fp)
  } else {
    selectedRemote.value = new Set([fp])
  }
}

let lastRemoteClickTime = 0
let lastRemoteClickName = ''

function checkRemoteDblClick(file: SFTPFile): boolean {
  const now = Date.now()
  if (now - lastRemoteClickTime < 400 && lastRemoteClickName === file.name) {
    lastRemoteClickTime = 0
    lastRemoteClickName = ''
    return true
  }
  lastRemoteClickTime = now
  lastRemoteClickName = file.name
  return false
}

function handleRemoteNameClick(file: SFTPFile, e: MouseEvent) {
  if (checkRemoteDblClick(file)) {
    if (file.is_dir) navigateToRemoteDir(file)
    else handleRemoteRowDblClick(file)
    return
  }
  toggleRemoteSelect(file, e)
}

function handleRemoteRowClick(file: SFTPFile, e: MouseEvent) {
  if (checkRemoteDblClick(file)) {
    if (file.is_dir) navigateToRemoteDir(file)
    else handleRemoteRowDblClick(file)
    return
  }
  toggleRemoteSelect(file, e)
}

function navigateToRemoteDir(file: SFTPFile) {
  const newPath = remoteFilePath(file.name)
  if (!expandedKeys.value.includes(sftpStore.getPanel(props.connectionID).currentPath)) {
    expandedKeys.value = [...expandedKeys.value, sftpStore.getPanel(props.connectionID).currentPath]
  }
  sftpStore.navigateToDir(props.connectionID, newPath)
}

async function handleRemoteRowDblClick(file: SFTPFile) {
  if (file.is_dir) return
  if (!isEditableFile(file.name, file.size)) {
    message.warning(t('sftp.fileTooLarge', { name: file.name, size: (file.size / 1024 / 1024).toFixed(1) }))
    return
  }

  const fullPath = remoteFilePath(file.name)
  const tabId = `editor-remote:${props.connectionID}:${fullPath}`

  if (terminalStore.tabs.find(t => t.id === tabId)) {
    terminalStore.activeTabID = tabId
    return
  }

  loadingRemoteFile.value = true
  try {
    const content = await SFTPReadFileContent(props.connectionID, fullPath)
    const conn = connectionStore.connections.find(c => c.id === props.connectionID)
    const username = conn?.username || 'user'
    const host = conn?.host || 'unknown'
    terminalStore.addEditorTab(tabId, file.name, content, fullPath, {
      isRemote: true,
      editorMode: 'remote-sftp',
      tooltip: `${username}@${host}:${fullPath}`,
      connectionID: props.connectionID,
    })
  } catch (e: any) {
    message.error(t('sftp.openFileFailed', { name: file.name, error: e instanceof Error ? e.message : String(e) }))
  } finally {
    loadingRemoteFile.value = false
  }
}

function handleDeleteRemote() {
  if (selectedRemote.value.size === 0) return
  const names = Array.from(selectedRemote.value).map(p => p.split('/').pop() || p)
  dialog.warning({
    title: t('sftp.deleteTitle'),
    content: t('sftp.deleteContent', { name: names.length === 1 ? names[0] : `${names.length} items` }),
    positiveText: t('common.delete'),
    negativeText: t('common.cancel'),
    onPositiveClick: async () => {
      for (const remotePath of selectedRemote.value) {
        try { await SFTPDelete(props.connectionID, remotePath) } catch { /* handled via event */ }
      }
      setTimeout(refreshRemote, 300)
    },
  })
}

async function refreshRemote() {
  const p = sftpStore.getPanel(props.connectionID)
  p.treeCache = {}
  selectedRemote.value = new Set()
  await sftpStore.navigateToDir(props.connectionID, p.currentPath)
  await sftpStore.loadTreeDir(props.connectionID, '/')
  rebuildTree()
}

// ===========================
// Local side state & methods
// ===========================
const showHidden = ref(false)
const localCurrentPath = ref('')
const localAllFiles = ref<LocalEntry[]>([])
const localLoading = ref(true)
const localLoadingDir = ref(false)
const localLoadingFile = ref(false)
const localEditing = ref(false)
const localEditPath = ref('')
const localSelected = ref(new Set<string>())
const localSortKey = ref<'name' | 'size' | 'time'>('name')
const localSortAsc = ref(true)

interface LocalEntry {
  name: string
  path: string
  size: number
  is_dir: boolean
  mod_time: number
}

const localFiles = computed(() => {
  const raw = showHidden.value ? localAllFiles.value : localAllFiles.value.filter(f => !f.name.startsWith('.'))
  const key = localSortKey.value
  const asc = localSortAsc.value
  return [...raw].sort((a, b) => {
    if (a.is_dir !== b.is_dir) return a.is_dir ? -1 : 1
    let cmp = 0
    if (key === 'name') cmp = a.name.localeCompare(b.name)
    else if (key === 'size') cmp = a.size - b.size
    else cmp = a.mod_time - b.mod_time
    return asc ? cmp : -cmp
  })
})

function toggleLocalSort(key: 'name' | 'size' | 'time') {
  if (localSortKey.value === key) localSortAsc.value = !localSortAsc.value
  else { localSortKey.value = key; localSortAsc.value = true }
}

// Local drag source
const { onRowPointerDown: onLocalRowPointerDown, cleanup: cleanupLocalDrag } = useDragSource({
  source: 'local',
  getSelectedPaths: () => localSelected.value,
  getFilePath: (entry: LocalEntry) => entry.path,
  getFileLabel: (entry: LocalEntry) => entry.name,
})

// Local drop target (accepts remote→local downloads)
const { targetRef: localDropRef, isDragOver: localIsDragOver, register: registerLocalDrop, unregister: unregisterLocalDrop } = useDropTarget({
  acceptedSource: 'remote',
  onDrop: (paths: string[]) => handleLocalDrop(paths),
})

const localDirCache = ref<Record<string, LocalEntry[]>>({})

async function loadLocalDir(dirPath: string): Promise<LocalEntry[]> {
  if (localDirCache.value[dirPath]) return localDirCache.value[dirPath]
  try {
    const result = await ListLocalDir(dirPath)
    const entries: LocalEntry[] = (result || []).map((e: any) => ({
      name: e.name || '',
      path: e.path || '',
      size: e.size || 0,
      is_dir: e.is_dir || false,
      mod_time: e.mod_time || 0,
    }))
    localDirCache.value[dirPath] = entries
    return entries
  } catch {
    localDirCache.value[dirPath] = []
    return []
  }
}

async function navigateLocalTo(dirPath: string) {
  localLoadingDir.value = true
  localSelected.value = new Set()
  try {
    localAllFiles.value = await loadLocalDir(dirPath)
    localCurrentPath.value = dirPath
    remoteDir.value = dirPath
  } finally {
    localLoadingDir.value = false
  }
}

const localPathParts = computed(() => {
  const parts = localCurrentPath.value.split('/').filter(Boolean)
  return parts.map((name, i) => ({
    name,
    path: '/' + parts.slice(0, i + 1).join('/'),
  }))
})

function startLocalEdit() {
  localEditPath.value = localCurrentPath.value
  localEditing.value = true
}

function commitLocalEdit() {
  localEditing.value = false
  navigateLocalTo(localEditPath.value)
}

let lastLocalClickTime = 0
let lastLocalClickName = ''

function checkLocalDblClick(entry: LocalEntry): boolean {
  const now = Date.now()
  if (now - lastLocalClickTime < 400 && lastLocalClickName === entry.name) {
    lastLocalClickTime = 0
    lastLocalClickName = ''
    return true
  }
  lastLocalClickTime = now
  lastLocalClickName = entry.name
  return false
}

function handleLocalNameClick(entry: LocalEntry, e: MouseEvent) {
  if (checkLocalDblClick(entry)) {
    if (entry.is_dir) navigateLocalTo(entry.path)
    else handleLocalRowDblClick(entry)
    return
  }
  toggleLocalSelect(entry, e)
}

function handleLocalRowClick(entry: LocalEntry, e: MouseEvent) {
  if (checkLocalDblClick(entry)) {
    if (entry.is_dir) navigateLocalTo(entry.path)
    else handleLocalRowDblClick(entry)
    return
  }
  toggleLocalSelect(entry, e)
}

function toggleLocalSelect(entry: LocalEntry, e: MouseEvent) {
  if (e.ctrlKey || e.metaKey) {
    if (localSelected.value.has(entry.path)) localSelected.value.delete(entry.path)
    else localSelected.value.add(entry.path)
  } else {
    localSelected.value = new Set([entry.path])
  }
}

function handleLocalUploadClick() {
  if (localSelected.value.size === 0) return
  handleUploadDrop(Array.from(localSelected.value))
}

function handleOpenInFileManager() {
  if (!localCurrentPath.value) return
  OpenInFileManager(localCurrentPath.value).catch(() => {})
}

function handleDeleteLocal() {
  if (localSelected.value.size === 0) return
  const names = Array.from(localSelected.value).map(p => p.split('/').pop() || p)
  dialog.warning({
    title: t('sftp.deleteTitle'),
    content: t('sftp.deleteContent', { name: names.length === 1 ? names[0] : `${names.length} items` }),
    positiveText: t('common.delete'),
    negativeText: t('common.cancel'),
    onPositiveClick: async () => {
      for (const p of localSelected.value) {
        try { await DeleteLocalFile(p) } catch { /* ignore */ }
      }
      localSelected.value = new Set()
      refreshLocal()
    },
  })
}

async function handleLocalRowDblClick(entry: LocalEntry) {
  if (entry.is_dir) return
  if (!isEditableFile(entry.name, entry.size)) return

  const tabId = `editor-local:${entry.path}`

  if (terminalStore.tabs.find(t => t.id === tabId)) {
    terminalStore.activeTabID = tabId
    return
  }

  localLoadingFile.value = true
  try {
    const content = await ReadLocalFileContent(entry.path)
    terminalStore.addEditorTab(tabId, entry.name, content, entry.path, {
      isRemote: false,
      editorMode: 'local-file',
      tooltip: entry.path,
    })
  } catch (e: any) {
    console.error('Failed to open local file:', e)
  } finally {
    localLoadingFile.value = false
  }
}

function refreshLocal() {
  localDirCache.value = {}
  navigateLocalTo(localCurrentPath.value)
}

// ===========================
// Transfer (shared)
// ===========================
function handleDownload() {
  if (selectedRemote.value.size === 0) return
  for (const remotePath of selectedRemote.value) {
    const fileName = remotePath.split('/').pop() || remotePath
    const localPath = remoteDir.value ? `${remoteDir.value}/${fileName}` : fileName
    SFTPDownload(props.connectionID, remotePath, localPath).catch(console.error)
  }
}

function handleUploadDrop(localPaths: string[]) {
  const p = sftpStore.getPanel(props.connectionID)
  for (const localPath of localPaths) {
    const fileName = localPath.split('/').pop() || localPath
    const remotePath = p.currentPath === '/' ? `/${fileName}` : `${p.currentPath}/${fileName}`
    SFTPUpload(props.connectionID, localPath, remotePath).catch(console.error)
  }
}

function handleLocalDrop(paths: string[]) {
  for (const remotePath of paths) {
    const fileName = remotePath.split('/').pop() || remotePath
    const localPath = remoteDir.value ? `${remoteDir.value}/${fileName}` : fileName
    SFTPDownload(props.connectionID, remotePath, localPath).catch(console.error)
  }
}

// Wails native file drop from OS file browser
function handleNativeFileDrop(ev: any) {
  const data = ev?.data
  if (!data?.files || !data?.targetId) return
  if (data.targetId !== 'remote-drop-zone-' + props.connectionID) return
  const paths: string[] = data.files as string[]
  if (paths.length > 0) handleUploadDrop(paths)
}

// ===========================
// Formatting utilities
// ===========================
function formatSize(bytes: number): string {
  if (bytes === 0) return '-'
  if (bytes < 1024) return bytes + 'B'
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + 'K'
  if (bytes < 1024 * 1024 * 1024) return (bytes / (1024 * 1024)).toFixed(1) + 'M'
  return (bytes / (1024 * 1024 * 1024)).toFixed(1) + 'G'
}

function formatTime(ts: number): string {
  if (!ts) return '-'
  return new Date(ts * 1000).toLocaleDateString()
}

function formatSpeed(kbps: number): string {
  if (kbps < 1024) return kbps.toFixed(0) + ' KB/s'
  return (kbps / 1024).toFixed(1) + ' MB/s'
}

const hasActiveTransfers = computed(() => allTransfers.value.some(tr => !tr.done))

async function handleCancelAll() {
  try {
    await SFTPCancelTransfers()
    setTimeout(() => {
      if (hasActiveTransfers.value) transferStore.clearAll()
    }, 300)
  } catch (e) {
    console.error('Failed to cancel transfers:', e)
    transferStore.clearAll()
  }
}

const allTransfers = computed(() => transferStore.transfers)

function transferSummary(transfers: TransferProgress[]) {
  if (transfers.length === 0) return { path: '', percent: 0, speed: 0 }
  let totalBytes = 0
  let transferred = 0
  let speed = 0
  let currentPath = ''
  for (const t of transfers) {
    totalBytes += t.total_bytes
    transferred += t.transferred
    if (!t.done) {
      speed += t.speed_kbps
      currentPath = t.file_name
    }
  }
  if (!currentPath) currentPath = transfers[transfers.length - 1].file_name
  const percent = totalBytes > 0 ? (transferred / totalBytes) * 100 : 0
  return { path: currentPath, percent: Math.min(100, percent), speed }
}

// ===========================
// Lifecycle
// ===========================
onMounted(async () => {
  registerRemoteDrop()
  registerLocalDrop()

  Events.On('sftp:progress', (ev: any) => {
    const d = ev?.data
    if (d) transferStore.addOrUpdateTransfer(d as TransferProgress)
  })
  Events.On('sftp:transfer-done', (ev: any) => {
    const d = ev?.data
    if (d?.direction === 'upload' || d?.direction === 'download') {
      transferStore.markTransfersDone(d.direction)
    }
    if (d?.direction === 'upload' || d?.direction === 'delete') {
      refreshRemote()
    } else {
      refreshLocal()
    }
  })
  Events.On('sftp:download:error', (ev: any) => {
    message.error(t('sftp.downloadFailed', { error: ev?.data || '' }))
  })
  Events.On('sftp:upload:error', (ev: any) => {
    message.error(t('sftp.uploadFailed', { error: ev?.data || '' }))
  })
  Events.On('native:file-drop', handleNativeFileDrop)

  // Load local home dir
  try {
    const home = await GetHomeDir()
    await navigateLocalTo(home)
  } catch {
    try { await navigateLocalTo('/') } catch { /* empty */ }
  } finally {
    localLoading.value = false
  }
})

onUnmounted(() => {
  Events.Off('sftp:progress')
  Events.Off('sftp:transfer-done')
  Events.Off('sftp:download:error')
  Events.Off('sftp:upload:error')
  Events.Off('native:file-drop')
  cleanupRemoteDrag()
  cleanupLocalDrag()
  unregisterRemoteDrop()
  unregisterLocalDrop()
})

watch(() => sftpStore.treeVersion, rebuildTree, { immediate: true })
</script>

<template>
  <div class="flex flex-col h-full bg-[var(--bg-secondary)] text-[var(--text-primary)] text-[var(--font-size-sm)]">
    <div class="flex-1 flex overflow-hidden min-h-0">
      <!-- Remote side -->
      <div class="flex flex-col min-w-0 thin-border-r" :style="{ flex: 6 }">
        <div class="flex items-center px-2 py-1 gap-1 shrink-0 toolbar-wrapper">
          <template v-if="!editingRemotePath">
            <span class="flex-1 overflow-hidden whitespace-nowrap text-[var(--font-size-sm)] select-none cursor-default" @dblclick="startRemoteEdit">
              <span class="text-[var(--text-secondary)] cursor-pointer hover:text-[var(--text-primary)] hover:underline" @click="navigateRemoteTo('/')">/</span>
              <template v-for="p in remotePathParts" :key="p.path">
                <span class="text-[var(--text-secondary)] cursor-pointer hover:text-[var(--text-primary)] hover:underline" @click="navigateRemoteTo(p.path)">{{ p.name }}</span>
                <span class="text-[var(--text-secondary)]">/</span>
              </template>
            </span>
          </template>
          <input v-else v-model="editRemotePath" class="flex-1 bg-[var(--bg-tertiary)] border border-solid border-[var(--border-color)] rounded-[3px] text-[var(--text-primary)] text-[var(--font-size-sm)] font-mono px-[6px] py-[2px] outline-none"
            @keyup.enter="commitRemoteEdit" @keyup.escape="editingRemotePath = false" @blur="commitRemoteEdit" />
          <NButton size="tiny" quaternary @click="refreshRemote" title="Refresh"><IconRefreshCw :width="14" :height="14" /></NButton>
          <NButton size="tiny" quaternary :type="selectedRemote.size > 0 ? 'primary' : 'default'" @click="handleDownload" title="Download selected">
            <IconDownload :width="14" :height="14" />
          </NButton>
          <NButton size="tiny" quaternary :type="selectedRemote.size > 0 ? 'primary' : 'default'" @click="handleDeleteRemote" title="Delete selected">
            <IconTrash2 :width="14" :height="14" />
          </NButton>
          <div v-if="sftpStore.getPanel(props.connectionID).loading || loadingRemoteFile" class="loading-bar"></div>
        </div>

        <div class="flex-1 flex overflow-hidden min-h-0">
          <div class="overflow-y-auto thin-border-r" :style="{ width: treeWidth + 'px', flexShrink: 0 }">
            <NTree v-if="treeData.length > 0" :data="treeData" :expanded-keys="expandedKeys" :on-load="handleLoad"
              selectable block-line
              @update:expanded-keys="(keys: string[]) => expandedKeys = keys"
              @update:selected-keys="handleTreeSelect" />
            <div v-else-if="!sftpStore.getPanel(props.connectionID).loading" class="h-full flex-center text-[var(--text-secondary)] text-[var(--font-size-sm)] px-2 text-center">{{ t('sftp.treeEmpty') }}</div>
          </div>

          <div class="flex-1 overflow-y-auto" ref="remoteDropRef" :id="'remote-drop-zone-' + props.connectionID" data-file-drop-target :class="{ 'drag-over': remoteIsDragOver }">
            <div v-if="sftpStore.getPanel(props.connectionID).error" class="h-full flex-center text-[var(--color-error)] px-4">{{ sftpStore.getPanel(props.connectionID).error }}</div>
            <table v-else-if="!sftpStore.getPanel(props.connectionID).loading" class="w-full border-collapse file-table">
              <thead>
                <tr>
                  <th class="text-left py-1 px-2 text-[var(--text-secondary)] font-medium text-[var(--font-size-sm)] select-none cursor-pointer hover:text-[var(--text-primary)] max-w-[300px]" @click="toggleRemoteSort('name')">{{ t('sftp.name') }} <span class="ml-[2px] text-[10px]" v-if="remoteSortKey === 'name'">{{ remoteSortAsc ? '↑' : '↓' }}</span></th>
                  <th class="text-right py-1 px-2 text-[var(--text-secondary)] font-medium text-[var(--font-size-sm)] select-none cursor-pointer hover:text-[var(--text-primary)] w-[70px]" @click="toggleRemoteSort('size')">{{ t('sftp.size') }} <span class="ml-[2px] text-[10px]" v-if="remoteSortKey === 'size'">{{ remoteSortAsc ? '↑' : '↓' }}</span></th>
                  <th class="text-left py-1 px-2 text-[var(--text-secondary)] font-medium text-[var(--font-size-sm)] select-none cursor-pointer hover:text-[var(--text-primary)] w-[100px]" @click="toggleRemoteSort('time')">{{ t('sftp.modified') }} <span class="ml-[2px] text-[10px]" v-if="remoteSortKey === 'time'">{{ remoteSortAsc ? '↑' : '↓' }}</span></th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="f in sortedRemoteFiles" :key="f.name"
                  class="file-row"
                  :class="{ 'dir-row': f.is_dir, selected: selectedRemote.has(remoteFilePath(f.name)) }"
                  @pointerdown="onRemoteRowPointerDown($event, f)"
                  @click="handleRemoteRowClick(f, $event)"
                >
                  <td class="py-[3px] px-2 whitespace-nowrap overflow-hidden text-ellipsis max-w-[300px]">
                    <span class="dir-name" @click.stop="handleRemoteNameClick(f, $event)">
                      <component :is="f.is_dir ? IconFolder : IconFile" :width="14" :height="14" class="mr-1 inline-block align-[-2px]" />{{ f.name }}
                    </span>
                  </td>
                  <td class="py-[3px] px-2 whitespace-nowrap text-right w-[70px]">{{ f.is_dir ? '-' : formatSize(f.size) }}</td>
                  <td class="py-[3px] px-2 whitespace-nowrap w-[100px]">{{ formatTime(f.mod_time) }}</td>
                </tr>
                <tr v-if="sftpStore.getPanel(props.connectionID).files.length === 0">
                  <td colspan="3" class="text-center text-[var(--text-secondary)] p-4">Empty directory</td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
      </div>

      <!-- Local side -->
      <div class="min-w-0 flex flex-col" :style="{ flex: 4 }">
        <div class="flex items-center px-2 py-1 gap-1 shrink-0 toolbar-wrapper">
          <template v-if="!localEditing">
            <span class="flex-1 overflow-hidden whitespace-nowrap text-[var(--font-size-sm)] select-none cursor-default" @dblclick="startLocalEdit">
              <span class="text-[var(--text-secondary)] cursor-pointer hover:text-[var(--text-primary)] hover:underline" @click="navigateLocalTo('/')">/</span>
              <template v-for="p in localPathParts" :key="p.path">
                <span class="text-[var(--text-secondary)] cursor-pointer hover:text-[var(--text-primary)] hover:underline" @click="navigateLocalTo(p.path)">{{ p.name }}</span>
                <span class="text-[var(--text-secondary)]">/</span>
              </template>
            </span>
          </template>
          <input v-else v-model="localEditPath" class="flex-1 bg-[var(--bg-tertiary)] border border-solid border-[var(--border-color)] rounded-[3px] text-[var(--text-primary)] text-[var(--font-size-sm)] font-mono px-[6px] py-[2px] outline-none"
            @keyup.enter="commitLocalEdit" @keyup.escape="localEditing = false" @blur="commitLocalEdit" />
          <NButton size="tiny" quaternary @click="refreshLocal" title="Refresh"><IconRefreshCw :width="14" :height="14" /></NButton>
          <NButton size="tiny" quaternary @click="handleOpenInFileManager" :title="t('sftp.openInFileManager')"><IconFolderOpen :width="14" :height="14" /></NButton>
          <NButton size="tiny" quaternary :type="showHidden ? 'primary' : 'default'" @click="showHidden = !showHidden">.*</NButton>
          <NButton size="tiny" quaternary :type="localSelected.size > 0 ? 'primary' : 'default'" @click="handleLocalUploadClick" title="Upload selected">
            <IconUpload :width="14" :height="14" />
          </NButton>
          <NButton size="tiny" quaternary :type="localSelected.size > 0 ? 'primary' : 'default'" @click="handleDeleteLocal" title="Delete selected">
            <IconTrash2 :width="14" :height="14" />
          </NButton>
          <div v-if="localLoading || localLoadingDir || localLoadingFile" class="loading-bar"></div>
        </div>

        <div class="flex-1 flex overflow-hidden min-h-0">
          <div class="flex-1 overflow-y-auto" ref="localDropRef" :class="{ 'drag-over': localIsDragOver }">
            <table class="w-full border-collapse file-table">
            <thead>
              <tr>
                <th class="text-left py-1 px-2 text-[var(--text-secondary)] font-medium text-[var(--font-size-sm)] select-none cursor-pointer hover:text-[var(--text-primary)] max-w-[160px]" @click="toggleLocalSort('name')">Name <span class="ml-[2px] text-[10px]" v-if="localSortKey === 'name'">{{ localSortAsc ? '↑' : '↓' }}</span></th>
                <th class="text-right py-1 px-2 text-[var(--text-secondary)] font-medium text-[var(--font-size-sm)] select-none cursor-pointer hover:text-[var(--text-primary)] w-[60px]" @click="toggleLocalSort('size')">Size <span class="ml-[2px] text-[10px]" v-if="localSortKey === 'size'">{{ localSortAsc ? '↑' : '↓' }}</span></th>
                <th class="text-left py-1 px-2 text-[var(--text-secondary)] font-medium text-[var(--font-size-sm)] select-none cursor-pointer hover:text-[var(--text-primary)] w-[85px] text-[11px]" @click="toggleLocalSort('time')">Modified <span class="ml-[2px] text-[10px]" v-if="localSortKey === 'time'">{{ localSortAsc ? '↑' : '↓' }}</span></th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="f in localFiles" :key="f.name"
                class="file-row"
                :class="{ 'dir-row': f.is_dir, selected: localSelected.has(f.path) }"
                @pointerdown="onLocalRowPointerDown($event, f)"
                @click="handleLocalRowClick(f, $event)"
              >
                <td class="py-[3px] px-2 whitespace-nowrap overflow-hidden text-ellipsis max-w-[160px]">
                  <span class="dir-name" @click.stop="handleLocalNameClick(f, $event)">
                    <component :is="f.is_dir ? IconFolder : IconFile" :width="14" :height="14" class="mr-1 inline-block align-[-2px]" />
                    {{ f.name }}
                  </span>
                </td>
                <td class="py-[3px] px-2 whitespace-nowrap text-right w-[60px]">{{ f.is_dir ? '-' : formatSize(f.size) }}</td>
                <td class="py-[3px] px-2 whitespace-nowrap w-[85px] text-[11px]">{{ formatTime(f.mod_time) }}</td>
              </tr>
              <tr v-if="localFiles.length === 0">
                <td colspan="3" class="text-center text-[var(--text-secondary)] p-4">Empty directory</td>
              </tr>
            </tbody>
          </table>
        </div>
        </div>
      </div>
    </div>

    <!-- Unified transfer status bar (always visible) -->
    <div class="status-bar">
      <template v-if="hasActiveTransfers">
        <div class="status-bg"><div class="status-fill" :style="{ width: transferSummary(allTransfers).percent + '%' }"></div></div>
        <span class="status-path">{{ transferSummary(allTransfers).path }}</span>
        <span class="status-speed">{{ transferSummary(allTransfers).speed > 0 ? formatSpeed(transferSummary(allTransfers).speed) : '' }}</span>
      </template>
      <div class="status-actions" v-if="hasActiveTransfers">
        <NButton size="tiny" quaternary @click="handleCancelAll" :title="t('sftp.cancelAll')">
          <IconX :width="14" :height="14" />
        </NButton>
      </div>
    </div>
  </div>
</template>

<style scoped>
.toolbar-wrapper {
  position: relative;
  overflow: hidden;
  border-bottom: 1px solid rgba(128, 128, 128, 0.12);
}
.thin-border-r { border-right: 1px solid rgba(128, 128, 128, 0.12); }

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

.file-row { cursor: default; }
.file-row:nth-child(even) { background: var(--hover-overlay-strong); }
.file-row.dir-row .dir-name { cursor: default; }
.file-row:hover { background: var(--hover-overlay); }
.file-row.selected { background: var(--action-hover-bg); }

.drag-over, :global(.file-drop-target-active) {
  outline: 2px dashed rgba(100, 108, 255, 0.5);
  outline-offset: -2px;
  background: rgba(100, 108, 255, 0.06) !important;
}

.status-bar {
  position: relative;
  height: 22px;
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: space-between;
  border-top: 1px solid var(--border-color);
  padding: 0 8px;
  font-size: 11px;
}
.status-bg {
  position: absolute;
  inset: 0;
  background: var(--stat-bar-bg);
  overflow: hidden;
}
.status-fill {
  height: 100%;
  background: rgba(100, 108, 255, 0.35);
  transition: width 0.15s ease;
}
.status-path {
  position: relative;
  z-index: 1;
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: var(--text-primary);
}
.status-speed {
  position: relative;
  z-index: 1;
  flex-shrink: 0;
  margin-left: 8px;
  color: var(--text-primary);
  font-variant-numeric: tabular-nums;
}
.status-actions {
  position: relative;
  z-index: 1;
  display: flex;
  align-items: center;
  gap: 2px;
  flex-shrink: 0;
  margin-left: 8px;
}
</style>
