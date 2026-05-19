<script setup lang="ts">
import { ref, computed, watch, onMounted, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { NTree, NSpin, NButton } from 'naive-ui'
import type { TreeOption } from 'naive-ui'
import { Events } from '@wailsio/runtime'
import { useSFTPStore } from '../../stores/sftp'
import { useTransferStore } from '../../stores/transfers'
import type { SFTPFile } from '../../stores/sftp'
import type { TransferProgress } from '../../stores/transfers'
import LocalFileTree from './LocalFileTree.vue'
import { SFTPUpload, SFTPDownload } from '../../../bindings/vshell/internal/app/appservice'

const localTreeRef = ref<InstanceType<typeof LocalFileTree> | null>(null)

const props = defineProps<{ connectionID: string }>()
const { t } = useI18n()
const sftpStore = useSFTPStore()
const transferStore = useTransferStore()

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
  toggleRemoteSelect(file, e)
}

function handleRemoteRowClick(file: SFTPFile, e: MouseEvent) {
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

function handleRefresh() {
  const p = sftpStore.getPanel(props.connectionID)
  navigateRemoteTo(p.currentPath)
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

// --- Progress & Refresh ---
function refreshRemote() {
  const p = sftpStore.getPanel(props.connectionID)
  selectedRemote.value = new Set()
  sftpStore.navigateToDir(props.connectionID, p.currentPath)
  rebuildTree()
}

function refreshLocal() {
  localTreeRef.value?.refresh()
}

onMounted(() => {
  Events.On('sftp:progress', (ev: any) => {
    const d = ev?.data
    if (d) transferStore.addOrUpdateTransfer(d as TransferProgress)
  })
  Events.On('sftp:transfer-done', (ev: any) => {
    const d = ev?.data
    if (d?.direction === 'upload') {
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

watch(() => sftpStore.treeVersion, rebuildTree)
</script>

<template>
  <div class="sftp-panel">
    <div class="sftp-body">
      <!-- Remote side -->
      <div class="remote-side" :style="{ flex: 6 }">
        <div class="remote-toolbar">
          <template v-if="!editingRemotePath">
            <span class="remote-breadcrumb" @dblclick="startRemoteEdit">
              <span class="breadcrumb-part" @click="navigateRemoteTo('/')">/</span>
              <template v-for="p in remotePathParts" :key="p.path">
                <span class="breadcrumb-part" @click="navigateRemoteTo(p.path)">{{ p.name }}</span>
                <span class="breadcrumb-sep">/</span>
              </template>
            </span>
          </template>
          <input v-else v-model="editRemotePath" class="remote-path-input"
            @keyup.enter="commitRemoteEdit" @keyup.escape="editingRemotePath = false" @blur="commitRemoteEdit" />
          <NButton size="tiny" quaternary @click="handleRefresh">{{ t('common.refresh') }}</NButton>
          <NButton size="tiny" quaternary class="download-btn" :class="{ active: selectedRemote.size > 0 }" @click="handleDownload" title="Download selected">
            <svg width="14" height="14" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.8"><path d="M8 3v9M4 9l4 4 4-4"/><path d="M2 12v2h12v-2"/></svg>
          </NButton>
        </div>

        <div class="remote-content">
          <div class="remote-tree" :style="{ width: treeWidth + 'px', flexShrink: 0 }">
            <NTree :data="treeData" :expanded-keys="expandedKeys" :on-load="handleLoad"
              selectable block-line
              @update:expanded-keys="(keys: string[]) => expandedKeys = keys"
              @update:selected-keys="handleTreeSelect" />
          </div>

          <div class="remote-list">
            <NSpin v-if="sftpStore.getPanel(props.connectionID).loading" size="small" />
            <div v-else-if="sftpStore.getPanel(props.connectionID).error" class="sftp-error">{{ sftpStore.getPanel(props.connectionID).error }}</div>
            <table v-else class="sftp-table">
              <thead>
                <tr>
                  <th class="col-name sortable" @click="toggleSort('name')">{{ t('sftp.name') }} <span class="sort-arrow" v-if="sortKey === 'name'">{{ sortAsc ? '↑' : '↓' }}</span></th>
                  <th class="col-size sortable" @click="toggleSort('size')">{{ t('sftp.size') }} <span class="sort-arrow" v-if="sortKey === 'size'">{{ sortAsc ? '↑' : '↓' }}</span></th>
                  <th class="col-time sortable" @click="toggleSort('time')">{{ t('sftp.modified') }} <span class="sort-arrow" v-if="sortKey === 'time'">{{ sortAsc ? '↑' : '↓' }}</span></th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="f in sortedRemoteFiles" :key="f.name"
                  class="sftp-row"
                  :class="{ 'sftp-dir': f.is_dir, selected: selectedRemote.has(remoteFilePath(f.name)) }"
                  @click="handleRemoteRowClick(f, $event)"
                >
                  <td class="col-name">
                    <span class="dir-name" @click.stop="handleRemoteNameClick(f, $event)">
                      <span class="file-icon">{{ f.is_dir ? '\u{1F4C1}' : '\u{1F4C4}' }}</span>{{ f.name }}
                    </span>
                  </td>
                  <td class="col-size">{{ f.is_dir ? '-' : formatSize(f.size) }}</td>
                  <td class="col-time">{{ formatTime(f.mod_time) }}</td>
                </tr>
                <tr v-if="sftpStore.getPanel(props.connectionID).files.length === 0">
                  <td colspan="3" class="sftp-empty-msg">{{ t('sftp.emptyDir') }}</td>
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
      <div class="local-side" :style="{ flex: 4 }">
        <LocalFileTree ref="localTreeRef" @upload="handleUpload" @path-change="handleLocalPathChange" />
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
.sftp-panel {
  display: flex;
  flex-direction: column;
  height: 100%;
  background: var(--bg-secondary);
  color: var(--text-primary);
  font-size: var(--font-size-sm);
}

.sftp-body {
  flex: 1;
  display: flex;
  overflow: hidden;
  min-height: 0;
}

/* ---- Remote side ---- */
.remote-side {
  display: flex;
  flex-direction: column;
  min-width: 0;
  border-right: 1px solid var(--border-color);
}

.remote-toolbar {
  display: flex;
  align-items: center;
  padding: 4px 8px;
  border-bottom: 1px solid var(--border-color);
  gap: 4px;
  flex-shrink: 0;
}

.remote-breadcrumb {
  flex: 1;
  overflow: hidden;
  white-space: nowrap;
  font-size: var(--font-size-sm);
  cursor: default;
  user-select: none;
}

.breadcrumb-part { color: var(--text-secondary); cursor: pointer; }
.breadcrumb-part:hover { color: var(--text-primary); text-decoration: underline; }
.breadcrumb-sep { color: var(--text-secondary); }

.remote-path-input {
  flex: 1;
  background: var(--bg-tertiary);
  border: 1px solid var(--border-color);
  border-radius: 3px;
  color: var(--text-primary);
  font-size: var(--font-size-sm);
  font-family: monospace;
  padding: 2px 6px;
  outline: none;
}

.download-btn { color: var(--text-secondary); }
.download-btn.active { color: var(--accent-color, #0078d4); }

.remote-content { flex: 1; display: flex; overflow: hidden; min-height: 0; }
.remote-tree { overflow-y: auto; border-right: 1px solid var(--border-color); }
.remote-list { flex: 1; overflow-y: auto; }

.sftp-error { padding: 8px; color: var(--error-color, #e55); }

.sftp-table { width: 100%; border-collapse: collapse; }
.sftp-table th {
  text-align: left; padding: 4px 8px;
  border-bottom: 1px solid var(--border-color);
  color: var(--text-secondary); font-weight: 500; font-size: var(--font-size-sm);
  user-select: none;
}
th.sortable { cursor: pointer; }
th.sortable:hover { color: var(--text-primary); }
.sort-arrow { margin-left: 2px; font-size: 10px; }
.sftp-table td {
  padding: 3px 8px;
  border-bottom: 1px solid var(--border-color);
  white-space: nowrap; overflow: hidden; text-overflow: ellipsis;
}

.sftp-row { cursor: default; }
.sftp-row.sftp-dir .dir-name { cursor: pointer; }
.sftp-row:hover { background: var(--hover-overlay-strong); }
.sftp-dir:hover .dir-name:hover { color: #5dade2; }
.sftp-row.selected { background: var(--action-hover-bg); }

.col-name { max-width: 300px; }
.col-size { width: 70px; text-align: right; }
.col-time { width: 100px; }
.file-icon { margin-right: 4px; }
.sftp-empty-msg { text-align: center; color: var(--text-secondary); padding: 16px; }

/* ---- Local side ---- */
.local-side { min-width: 0; display: flex; flex-direction: column; }

/* ---- Status bar ---- */
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
  background: rgba(56, 132, 244, 0.35);
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
