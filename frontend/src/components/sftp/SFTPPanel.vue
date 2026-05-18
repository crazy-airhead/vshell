<script setup lang="ts">
import { ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { NTree, NSpin, NButton } from 'naive-ui'
import type { TreeOption } from 'naive-ui'
import { h } from 'vue'
import { useSFTPStore } from '../../stores/sftp'
import type { SFTPFile } from '../../stores/sftp'

const props = defineProps<{ connectionID: string }>()
const { t } = useI18n()
const sftpStore = useSFTPStore()

const treeData = ref<TreeOption[]>([])
const expandedKeys = ref<string[]>([])

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
      return {
        key: fullPath,
        label: f.name,
        prefix: () => h('span', { style: 'margin-right: 4px; font-size: 13px' }, expandedKeys.value.includes(fullPath) ? '\u{1F4C2}' : '\u{1F4C1}'),
        children: (fullPath in cache) ? buildTreeNodes(fullPath, cache) : undefined,
        isLeaf: false,
      }
    })
}

function rebuildTree() {
  const p = sftpStore.getPanel(props.connectionID)
  treeData.value = buildTreeNodes('/', p.treeCache)
}

async function handleTreeExpand(keys: string[]) {
  const p = sftpStore.getPanel(props.connectionID)
  const oldKeys = new Set(expandedKeys.value)
  const newKeys = new Set(keys)

  const toLoad: string[] = []
  for (const k of keys) {
    if (!oldKeys.has(k)) {
      toLoad.push(k)
    }
  }
  await Promise.all(toLoad.map(k => sftpStore.loadTreeDir(props.connectionID, k)))

  rebuildTree()
  expandedKeys.value = keys
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
  sftpStore.navigateToDir(props.connectionID, newPath)
}

function handleRefresh() {
  const p = sftpStore.getPanel(props.connectionID)
  sftpStore.navigateToDir(props.connectionID, p.currentPath)
  rebuildTree()
}

function handleParentDir() {
  const p = sftpStore.getPanel(props.connectionID)
  const current = p.currentPath
  if (current === '/') return
  const parts = current.split('/').filter(Boolean)
  parts.pop()
  const parent = parts.length === 0 ? '/' : '/' + parts.join('/')
  sftpStore.navigateToDir(props.connectionID, parent)
  rebuildTree()
}

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

watch(() => sftpStore.treeVersion, () => {
  rebuildTree()
})
</script>

<template>
  <div class="sftp-panel">
    <div class="sftp-toolbar">
      <span class="sftp-path">{{ sftpStore.getPanel(props.connectionID).currentPath }}</span>
      <NButton size="tiny" quaternary @click="handleParentDir" :disabled="sftpStore.getPanel(props.connectionID).currentPath === '/'">..</NButton>
      <NButton size="tiny" quaternary @click="handleRefresh">{{ t('common.refresh') }}</NButton>
    </div>
    <div class="sftp-content">
      <div class="sftp-tree">
        <NTree
          :data="treeData"
          :expanded-keys="expandedKeys"
          selectable
          block-line
          @update:expanded-keys="handleTreeExpand"
          @update:selected-keys="handleTreeSelect"
        />
      </div>
      <div class="sftp-list">
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
              @click="handleFileClick(f)"
            >
              <td class="col-name">
                <span class="file-icon">{{ f.is_dir ? '\u{1F4C1}' : '\u{1F4C4}' }}</span>
                {{ f.name }}
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

.sftp-toolbar {
  display: flex;
  align-items: center;
  padding: 4px 8px;
  border-bottom: 1px solid var(--border-color);
  gap: 4px;
  flex-shrink: 0;
}

.sftp-path {
  flex: 1;
  font-family: monospace;
  color: var(--text-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.sftp-content {
  flex: 1;
  display: flex;
  overflow: hidden;
  min-height: 0;
}

.sftp-tree {
  width: 200px;
  min-width: 150px;
  border-right: 1px solid var(--border-color);
  overflow-y: auto;
  padding: 4px;
  flex-shrink: 0;
}

.sftp-list {
  flex: 1;
  overflow-y: auto;
  padding: 0;
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

.col-name {
  max-width: 300px;
}

.col-size {
  width: 70px;
  text-align: right;
}

.col-time {
  width: 100px;
}

.file-icon {
  margin-right: 4px;
}

.sftp-empty-msg {
  text-align: center;
  color: var(--text-secondary);
  padding: 16px;
}
</style>
