<script setup lang="ts">
import { ref, computed, watch, onMounted, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { NTree, NButton, useDialog, useMessage } from 'naive-ui'
import type { TreeOption } from 'naive-ui'
import { Events } from '@wailsio/runtime'
import IconRefreshCw from '~icons/lucide/refresh-cw'
import IconDownload from '~icons/lucide/download'
import IconTrash2 from '~icons/lucide/trash-2'
import IconFolder from '~icons/lucide/folder'
import IconFile from '~icons/lucide/file'
import { useSFTPStore } from '../../stores/sftp'
import { useTransferStore } from '../../stores/transfers'
import { useTerminalStore } from '../../stores/terminal'
import { useConnectionStore } from '../../stores/connection'
import type { SFTPFile } from '../../stores/sftp'
import type { TransferProgress } from '../../stores/transfers'
import LocalFileTree from './LocalFileTree.vue'
import { SFTPUpload, SFTPDownload, SFTPDelete, SFTPReadFileContent } from '../../../bindings/vshell/internal/app/appservice'
import { useDragSource, useDropTarget } from '../../composables/useDragTransfer'
import { isEditableFile } from '../../utils/fileType'

const localTreeRef = ref<InstanceType<typeof LocalFileTree> | null>(null)

const props = defineProps<{ connectionID: string }>()
const { t } = useI18n()
const sftpStore = useSFTPStore()
const transferStore = useTransferStore()
const terminalStore = useTerminalStore()
const connectionStore = useConnectionStore()
const dialog = useDialog()
const message = useMessage()

const treeData = ref<TreeOption[]>([])
const expandedKeys = ref<string[]>([])
const treeWidth = ref(170)
const editingRemotePath = ref(false)
const editRemotePath = ref('')
const localDir = ref('')
const selectedRemote = ref(new Set<string>())
const sortKey = ref<'name' | 'size' | 'time'>('name')
const sortAsc = ref(true)

const sortedRemoteFiles = computed(() => {
  const p = sftpStore.getPanel(props.connectionID)
  const files = [...p.files]
  const key = sortKey.value
  const asc = sortAsc.value
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

function toggleSort(key: 'name' | 'size' | 'time') {
  if (sortKey.value === key) sortAsc.value = !sortAsc.value
  else { sortKey.value = key; sortAsc.value = true }
}

// --- Drag & Drop ---
const { onRowMouseDown: onRemoteRowMouseDown, cleanup: cleanupRemoteDrag } = useDragSource({
  source: 'remote',
  getSelectedPaths: () => selectedRemote.value,
  getFilePath: (file: SFTPFile) => remoteFilePath(file.name),
  getFileLabel: (file: SFTPFile) => file.name,
})

const { targetRef: remoteDropRef, isDragOver: remoteIsDragOver, register: registerRemoteDrop, unregister: unregisterRemoteDrop } = useDropTarget({
  acceptedSource: 'local',
  onDrop: (paths: string[]) => handleUpload(paths),
})

function handleLocalDrop(paths: string[]) {
  for (const remotePath of paths) {
    const fileName = remotePath.split('/').pop() || remotePath
    const localPath = localDir.value ? `${localDir.value}/${fileName}` : fileName
    SFTPDownload(props.connectionID, remotePath, localPath).catch(console.error)
  }
}

// --- Remote path ---
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

// --- Tree ---
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

// --- Remote file list ---
function remoteFilePath(name: string): string {
  const p = sftpStore.getPanel(props.connectionID)
  return p.currentPath === '/' ? `/${name}` : `${p.currentPath}/${name}`
}

function handleRemoteNameClick(file: SFTPFile, e: MouseEvent) {
  if (file.is_dir && !e.ctrlKey && !e.metaKey) {
    const newPath = remoteFilePath(file.name)
    if (!expandedKeys.value.includes(sftpStore.getPanel(props.connectionID).currentPath)) {
      expandedKeys.value = [...expandedKeys.value, sftpStore.getPanel(props.connectionID).currentPath]
    }
    sftpStore.navigateToDir(props.connectionID, newPath)
    return
  }
  if (!file.is_dir && checkRemoteDblClick(file)) {
    handleRemoteRowDblClick(file)
    return
  }
  toggleRemoteSelect(file, e)
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

// Manual dblclick detection because drag overlay suppresses native dblclick
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

function handleRemoteRowClick(file: SFTPFile, e: MouseEvent) {
  if (!file.is_dir && checkRemoteDblClick(file)) {
    handleRemoteRowDblClick(file)
    return
  }
  toggleRemoteSelect(file, e)
}

async function handleRemoteRowDblClick(file: SFTPFile) {
  if (file.is_dir) return
  if (!isEditableFile(file.name, file.size)) return

  const fullPath = remoteFilePath(file.name)
  const tabId = `editor-remote:${props.connectionID}:${fullPath}`

  if (terminalStore.tabs.find(t => t.id === tabId)) {
    terminalStore.activeTabID = tabId
    return
  }

  try {
    const content = await SFTPReadFileContent(props.connectionID, fullPath)

    const conn = connectionStore.connections.find(c => c.id === props.connectionID)
    const username = conn?.username || 'user'
    const host = conn?.host || 'unknown'
    const tooltip = `${username}@${host}:${fullPath}`

    terminalStore.addEditorTab(tabId, file.name, content, fullPath, {
      isRemote: true,
      editorMode: 'remote-sftp',
      tooltip,
      connectionID: props.connectionID,
    })
  } catch (e: any) {
    console.error('Failed to open remote file:', e)
  }
}

// --- Transfer ---
function handleDownload() {
  if (selectedRemote.value.size === 0) return
  for (const remotePath of selectedRemote.value) {
    const fileName = remotePath.split('/').pop() || remotePath
    const localPath = localDir.value ? `${localDir.value}/${fileName}` : fileName
    SFTPDownload(props.connectionID, remotePath, localPath).catch(console.error)
  }
}

function handleUpload(localPaths: string[]) {
  const p = sftpStore.getPanel(props.connectionID)
  for (const localPath of localPaths) {
    const fileName = localPath.split('/').pop() || localPath
    const remotePath = p.currentPath === '/' ? `/${fileName}` : `${p.currentPath}/${fileName}`
    SFTPUpload(props.connectionID, localPath, remotePath).catch(console.error)
  }
}

function handleLocalPathChange(path: string) {
  localDir.value = path
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
        try { await SFTPDelete(props.connectionID, remotePath) }
        catch { /* handled via event */ }
      }
      setTimeout(refreshRemote, 300)
    },
  })
}

// --- Progress & Refresh ---
async function refreshRemote() {
  const p = sftpStore.getPanel(props.connectionID)
  p.treeCache = {}
  selectedRemote.value = new Set()
  await sftpStore.navigateToDir(props.connectionID, p.currentPath)
  // Reload root so the tree has data
  await sftpStore.loadTreeDir(props.connectionID, '/')
  rebuildTree()
}

function refreshLocal() {
  localTreeRef.value?.refresh()
}

onMounted(() => {
  registerRemoteDrop()
  Events.On('sftp:progress', (ev: any) => {
    const d = ev?.data
    if (d) transferStore.addOrUpdateTransfer(d as TransferProgress)
  })
  Events.On('sftp:transfer-done', (ev: any) => {
    const d = ev?.data
    if (d?.direction === 'upload' || d?.direction === 'delete') {
      refreshRemote()
    } else {
      refreshLocal()
    }
    setTimeout(() => transferStore.clearDone(), 2000)
  })
})

onUnmounted(() => {
  Events.Off('sftp:progress')
  Events.Off('sftp:transfer-done')
  cleanupRemoteDrag()
  unregisterRemoteDrop()
})

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

const downloadTransfers = computed(() => transferStore.transfers.filter(t => t.direction === 'download'))
const uploadTransfers = computed(() => transferStore.transfers.filter(t => t.direction === 'upload'))

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
          <NButton size="tiny" quaternary class="download-btn" :class="{ active: selectedRemote.size > 0 }" @click="handleDownload" title="Download selected">
            <IconDownload :width="14" :height="14" />
          </NButton>
          <NButton size="tiny" quaternary class="delete-btn" :class="{ active: selectedRemote.size > 0 }" @click="handleDeleteRemote" title="Delete selected">
            <IconTrash2 :width="14" :height="14" />
          </NButton>
          <div v-if="sftpStore.getPanel(props.connectionID).loading" class="loading-bar"></div>
        </div>

        <div class="flex-1 flex overflow-hidden min-h-0">
          <div class="overflow-y-auto thin-border-r" :style="{ width: treeWidth + 'px', flexShrink: 0 }">
            <NTree v-if="treeData.length > 0" :data="treeData" :expanded-keys="expandedKeys" :on-load="handleLoad"
              selectable block-line
              @update:expanded-keys="(keys: string[]) => expandedKeys = keys"
              @update:selected-keys="handleTreeSelect" />
            <div v-else-if="!sftpStore.getPanel(props.connectionID).loading" class="h-full flex-center text-[var(--text-secondary)] text-[var(--font-size-sm)] px-2 text-center">{{ t('sftp.treeEmpty') }}</div>
          </div>

          <div class="flex-1 overflow-y-auto" ref="remoteDropRef" :class="{ 'drag-over': remoteIsDragOver }">
            <div v-if="sftpStore.getPanel(props.connectionID).error" class="h-full flex-center text-[var(--color-error)] px-4">{{ sftpStore.getPanel(props.connectionID).error }}</div>
            <table v-else-if="!sftpStore.getPanel(props.connectionID).loading" class="w-full border-collapse sftp-table">
              <thead>
                <tr>
                  <th class="text-left py-1 px-2 text-[var(--text-secondary)] font-medium text-[var(--font-size-sm)] select-none cursor-pointer hover:text-[var(--text-primary)] max-w-[300px]" @click="toggleSort('name')">{{ t('sftp.name') }} <span class="ml-[2px] text-[10px]" v-if="sortKey === 'name'">{{ sortAsc ? '↑' : '↓' }}</span></th>
                  <th class="text-right py-1 px-2 text-[var(--text-secondary)] font-medium text-[var(--font-size-sm)] select-none cursor-pointer hover:text-[var(--text-primary)] w-[70px]" @click="toggleSort('size')">{{ t('sftp.size') }} <span class="ml-[2px] text-[10px]" v-if="sortKey === 'size'">{{ sortAsc ? '↑' : '↓' }}</span></th>
                  <th class="text-left py-1 px-2 text-[var(--text-secondary)] font-medium text-[var(--font-size-sm)] select-none cursor-pointer hover:text-[var(--text-primary)] w-[100px]" @click="toggleSort('time')">{{ t('sftp.modified') }} <span class="ml-[2px] text-[10px]" v-if="sortKey === 'time'">{{ sortAsc ? '↑' : '↓' }}</span></th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="f in sortedRemoteFiles" :key="f.name"
                  class="sftp-row"
                  :class="{ 'sftp-dir': f.is_dir, selected: selectedRemote.has(remoteFilePath(f.name)) }"
                  @mousedown="onRemoteRowMouseDown($event, f)"
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
        <!-- Download status bar -->
        <div class="status-bar">
          <div class="status-bg"><div class="status-fill" :style="{ width: transferSummary(downloadTransfers).percent + '%' }"></div></div>
          <span class="status-path">{{ transferSummary(downloadTransfers).path }}</span>
          <span class="status-speed">{{ transferSummary(downloadTransfers).speed > 0 ? formatSpeed(transferSummary(downloadTransfers).speed) : '' }}</span>
        </div>
      </div>

      <!-- Local side -->
      <div class="min-w-0 flex flex-col" :style="{ flex: 4 }">
        <LocalFileTree ref="localTreeRef" @upload="handleUpload" @path-change="handleLocalPathChange" @drop-files="handleLocalDrop" />
        <!-- Upload status bar -->
        <div class="status-bar">
          <div class="status-bg"><div class="status-fill" :style="{ width: transferSummary(uploadTransfers).percent + '%' }"></div></div>
          <span class="status-path">{{ transferSummary(uploadTransfers).path }}</span>
          <span class="status-speed">{{ transferSummary(uploadTransfers).speed > 0 ? formatSpeed(transferSummary(uploadTransfers).speed) : '' }}</span>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.toolbar-wrapper {
  position: relative;
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

.sftp-row { cursor: default; }
.sftp-row:nth-child(even) { background: var(--hover-overlay-strong); }
.sftp-row.sftp-dir .dir-name { cursor: pointer; }
.sftp-row:hover { background: var(--hover-overlay); }
.sftp-dir:hover .dir-name:hover { color: var(--color-info); }
.sftp-row.selected { background: var(--action-hover-bg); }

.download-btn { color: var(--text-secondary); }
.download-btn.active { color: var(--color-primary); }
.delete-btn { color: var(--text-secondary); }
.delete-btn.active { color: var(--delete-hover-color); }

.drag-over {
  outline: 2px dashed rgba(100, 108, 255, 0.5);
  outline-offset: -2px;
  background: rgba(100, 108, 255, 0.06) !important;
  transition: outline 0.15s, background 0.15s;
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
</style>
