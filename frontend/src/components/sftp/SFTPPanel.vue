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

const props = defineProps<{ connectionID: string }>()
const { t } = useI18n()
const sftpStore = useSFTPStore()
const transferStore = useTransferStore()

const treeData = ref<TreeOption[]>([])
const expandedKeys = ref<string[]>([])
const treeWidth = ref(170)
const dropOver = ref(false)
const editingRemotePath = ref(false)
const editRemotePath = ref('')

const remotePathParts = computed(() => {
  const p = sftpStore.getPanel(props.connectionID)
  const parts = p.currentPath.split('/').filter(Boolean)
  return parts.map((name, i) => ({
    name,
    path: '/' + parts.slice(0, i + 1).join('/'),
  }))
})

function navigateRemoteTo(dirPath: string) {
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

function buildTreeNodes(
  parentPath: string,
  cache: Record<string, SFTPFile[]>,
): TreeOption[] {
  const items = cache[parentPath]
  if (!items) return []
  return items
    .filter((f) => f.is_dir)
    .map((f) => {
      const fullPath = parentPath === '/' ? `/${f.name}` : `${parentPath}/${f.name}`
      const hasChildren = fullPath in cache
      return {
        key: fullPath,
        label: f.name,
        children: hasChildren ? buildTreeNodes(fullPath, cache) : undefined,
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
  const path = keys[0]
  sftpStore.navigateToDir(props.connectionID, path)
  rebuildTree()
}

function handleFileClick(file: SFTPFile) {
  if (!file.is_dir) return
  const p = sftpStore.getPanel(props.connectionID)
  const newPath = p.currentPath === '/' ? `/${file.name}` : `${p.currentPath}/${file.name}`
  if (!expandedKeys.value.includes(p.currentPath)) {
    expandedKeys.value = [...expandedKeys.value, p.currentPath]
  }
  sftpStore.navigateToDir(props.connectionID, newPath)
}

function handleRefresh() {
  const p = sftpStore.getPanel(props.connectionID)
  sftpStore.navigateToDir(props.connectionID, p.currentPath)
  rebuildTree()
}

// --- Drag & Drop: Remote → Local (Download) ---
function onFileDragStart(e: DragEvent, file: SFTPFile) {
  if (file.is_dir) {
    e.preventDefault()
    return
  }
  const p = sftpStore.getPanel(props.connectionID)
  const remotePath = p.currentPath === '/' ? `/${file.name}` : `${p.currentPath}/${file.name}`
  if (e.dataTransfer) {
    e.dataTransfer.effectAllowed = 'copy'
    e.dataTransfer.setData('text/plain', remotePath)
  }
}

// --- Drag & Drop: Local → Remote (Upload) ---
function onListDragOver(e: DragEvent) {
  e.preventDefault()
  dropOver.value = true
  if (e.dataTransfer) {
    e.dataTransfer.dropEffect = 'copy'
  }
}

function onListDragLeave() {
  dropOver.value = false
}

function onListDrop(e: DragEvent) {
  e.preventDefault()
  dropOver.value = false
  if (!e.dataTransfer) return
  const data = e.dataTransfer.getData('text/plain')
  if (!data) return
  const paths = data.split('\n').filter(Boolean)
  const p = sftpStore.getPanel(props.connectionID)
  for (const localPath of paths) {
    const fileName = localPath.split('/').pop() || localPath
    const remotePath = p.currentPath === '/' ? `/${fileName}` : `${p.currentPath}/${fileName}`
    SFTPUpload(props.connectionID, localPath, remotePath).catch(() => {})
  }
}

function onLocalDropFiles(paths: string[], targetDir: string) {
  for (const remotePath of paths) {
    const fileName = remotePath.split('/').pop() || remotePath
    const localPath = targetDir.endsWith('/') ? `${targetDir}${fileName}` : `${targetDir}/${fileName}`
    SFTPDownload(props.connectionID, remotePath, localPath).catch(() => {})
  }
}

// --- Progress Events ---
onMounted(() => {
  Events.On('sftp:progress', (e: any) => {
    transferStore.addOrUpdateTransfer(e as TransferProgress)
  })
})

onUnmounted(() => {
  Events.Off('sftp:progress')
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

watch(() => sftpStore.treeVersion, () => {
  rebuildTree()
})
</script>

<template>
  <div class="sftp-panel">
    <!-- Body: Remote (70%) | Local (30%) -->
    <div class="sftp-body">
      <!-- Remote side -->
      <div class="remote-side" :style="{ flex: 6 }">
        <!-- Remote path bar -->
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
          <input
            v-else
            v-model="editRemotePath"
            class="remote-path-input"
            @keyup.enter="commitRemoteEdit"
            @keyup.escape="editingRemotePath = false"
            @blur="commitRemoteEdit"
          />
          <NButton size="tiny" quaternary @click="handleRefresh">{{ t('common.refresh') }}</NButton>
        </div>

        <!-- Remote: tree + file list -->
        <div class="remote-content">
          <div class="remote-tree" :style="{ width: treeWidth + 'px', flexShrink: 0 }">
            <NTree
              :data="treeData"
              :expanded-keys="expandedKeys"
              :on-load="handleLoad"
              selectable
              block-line
              @update:expanded-keys="(keys: string[]) => expandedKeys = keys"
              @update:selected-keys="handleTreeSelect"
            />
          </div>

          <div
            class="remote-list"
            :class="{ 'drop-active': dropOver }"
            @dragover="onListDragOver"
            @dragleave="onListDragLeave"
            @drop="onListDrop"
          >
            <NSpin v-if="sftpStore.getPanel(props.connectionID).loading" size="small" />
            <div v-else-if="sftpStore.getPanel(props.connectionID).error" class="sftp-error">{{ sftpStore.getPanel(props.connectionID).error }}</div>
            <table v-else class="sftp-table">
              <thead>
                <tr>
                  <th class="col-name">{{ t('sftp.name') }}</th>
                  <th class="col-size">{{ t('sftp.size') }}</th>
                  <th class="col-time">{{ t('sftp.modified') }}</th>
                </tr>
              </thead>
              <tbody>
                <tr
                  v-for="f in sftpStore.getPanel(props.connectionID).files"
                  :key="f.name"
                  class="sftp-row"
                  :class="{ 'sftp-dir': f.is_dir }"
                  :draggable="!f.is_dir"
                  @click="handleFileClick(f)"
                  @dragstart="onFileDragStart($event, f)"
                >
                  <td class="col-name"><span class="file-icon">{{ f.is_dir ? '\u{1F4C1}' : '\u{1F4C4}' }}</span>{{ f.name }}</td>
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
      </div>

      <!-- Local side -->
      <div class="local-side" :style="{ flex: 4 }">
        <LocalFileTree @drop-files="onLocalDropFiles" />
      </div>
    </div>

    <!-- Progress area -->
    <div v-if="transferStore.transfers.length > 0" class="sftp-progress">
      <div v-for="t in transferStore.transfers" :key="t.id" class="transfer-item">
        <span class="transfer-dir">{{ t.direction === 'upload' ? '⬆' : '⬇' }}</span>
        <span class="transfer-name">{{ t.file_name }}</span>
        <div class="transfer-bar">
          <div class="transfer-bar-fill" :class="{ error: t.error }" :style="{ width: Math.min(100, t.percent) + '%' }"></div>
        </div>
        <span class="transfer-pct">{{ t.percent.toFixed(0) }}%</span>
        <span class="transfer-speed">{{ formatSpeed(t.speed_kbps) }}</span>
        <button v-if="t.done" class="transfer-dismiss" @click="transferStore.removeTransfer(t.id)">&#10005;</button>
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

.breadcrumb-part {
  color: var(--text-secondary);
  cursor: pointer;
}

.breadcrumb-part:hover {
  color: var(--text-primary);
  text-decoration: underline;
}

.breadcrumb-sep {
  color: var(--text-secondary);
}

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

.remote-content {
  flex: 1;
  display: flex;
  overflow: hidden;
  min-height: 0;
}

.remote-tree {
  overflow-y: auto;
  border-right: 1px solid var(--border-color);
}

.remote-list {
  flex: 1;
  overflow-y: auto;
}

.remote-list.drop-active {
  background: var(--action-hover-bg);
}

.sftp-error {
  padding: 8px;
  color: var(--error-color, #e55);
}

.sftp-table {
  width: 100%;
  border-collapse: collapse;
}

.sftp-table th {
  text-align: left;
  padding: 4px 8px;
  border-bottom: 1px solid var(--border-color);
  color: var(--text-secondary);
  font-weight: 500;
  font-size: var(--font-size-sm);
}

.sftp-table td {
  padding: 3px 8px;
  border-bottom: 1px solid var(--border-color);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.sftp-row {
  cursor: default;
}

.sftp-row.sftp-dir {
  cursor: pointer;
}

.sftp-row:hover {
  background: var(--hover-overlay-strong);
}

.sftp-dir:hover {
  background: var(--action-hover-bg);
}

.col-name { max-width: 300px; }
.col-size { width: 70px; text-align: right; }
.col-time { width: 100px; }
.file-icon { margin-right: 4px; }

.sftp-empty-msg {
  text-align: center;
  color: var(--text-secondary);
  padding: 16px;
}

/* ---- Local side ---- */
.local-side {
  min-width: 0;
  display: flex;
  flex-direction: column;
}

/* ---- Progress ---- */
.sftp-progress {
  border-top: 1px solid var(--border-color);
  padding: 4px 8px;
  flex-shrink: 0;
  display: flex;
  flex-direction: column;
  gap: 3px;
  max-height: 120px;
  overflow-y: auto;
}

.transfer-item {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: var(--font-size-sm);
}

.transfer-dir {
  font-size: 12px;
  width: 16px;
  text-align: center;
}

.transfer-name {
  width: 140px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: var(--text-primary);
  font-size: var(--font-size-sm);
}

.transfer-bar {
  flex: 1;
  height: 4px;
  background: var(--stat-bar-bg);
  border-radius: 2px;
  overflow: hidden;
}

.transfer-bar-fill {
  height: 100%;
  background: var(--accent-color);
  border-radius: 2px;
  transition: width 0.3s ease;
}

.transfer-bar-fill.error {
  background: var(--error-color, #e55);
}

.transfer-pct {
  width: 32px;
  text-align: right;
  color: var(--text-secondary);
  font-variant-numeric: tabular-nums;
}

.transfer-speed {
  width: 64px;
  text-align: right;
  color: var(--text-secondary);
  font-variant-numeric: tabular-nums;
}

.transfer-dismiss {
  background: none;
  border: none;
  color: var(--text-secondary);
  cursor: pointer;
  font-size: 10px;
  padding: 0 2px;
}

.transfer-dismiss:hover {
  color: var(--text-primary);
}
</style>
