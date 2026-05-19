<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { NSpin, NButton, useDialog, useMessage } from 'naive-ui'
import { GetHomeDir, ListLocalDir, DeleteLocalFile } from '../../../bindings/vshell/internal/app/appservice'
import { useDragSource, useDropTarget } from '../../composables/useDragTransfer'

const emit = defineEmits<{
  (e: 'upload', paths: string[], targetDir: string): void
  (e: 'pathChange', path: string): void
  (e: 'drop-files', paths: string[]): void
}>()

interface LocalEntry {
  name: string
  path: string
  size: number
  is_dir: boolean
  mod_time: number
}

const { t } = useI18n()
const dialog = useDialog()
const message = useMessage()

const showHidden = ref(false)
const currentPath = ref('')
const allFiles = ref<LocalEntry[]>([])
const loading = ref(true)
const loadingDir = ref(false)
const editing = ref(false)
const editPath = ref('')
const selected = ref(new Set<string>())
const sortKey = ref<'name' | 'size' | 'time'>('name')
const sortAsc = ref(true)

const files = computed(() => {
  const raw = showHidden.value ? allFiles.value : allFiles.value.filter(f => !f.name.startsWith('.'))
  const key = sortKey.value
  const asc = sortAsc.value
  return [...raw].sort((a, b) => {
    if (a.is_dir !== b.is_dir) return a.is_dir ? -1 : 1
    let cmp = 0
    if (key === 'name') cmp = a.name.localeCompare(b.name)
    else if (key === 'size') cmp = a.size - b.size
    else cmp = a.mod_time - b.mod_time
    return asc ? cmp : -cmp
  })
})

function toggleSort(key: 'name' | 'size' | 'time') {
  if (sortKey.value === key) sortAsc.value = !sortAsc.value
  else { sortKey.value = key; sortAsc.value = true }
}

// --- Drag source ---
const { onRowMouseDown: onLocalRowMouseDown, cleanup: cleanupLocalDrag } = useDragSource({
  source: 'local',
  getSelectedPaths: () => selected.value,
  getFilePath: (entry: LocalEntry) => entry.path,
  getFileLabel: (entry: LocalEntry) => entry.name,
})

// --- Drop target ---
const { targetRef: localBodyRef, isDragOver: localIsDragOver, register: registerLocalDrop, unregister: unregisterLocalDrop } = useDropTarget({
  acceptedSource: 'remote',
  onDrop: (paths: string[]) => emit('drop-files', paths),
})

const dirCache = ref<Record<string, LocalEntry[]>>({})

async function loadDir(dirPath: string): Promise<LocalEntry[]> {
  if (dirCache.value[dirPath]) return dirCache.value[dirPath]
  try {
    const result = await ListLocalDir(dirPath)
    const entries: LocalEntry[] = (result || []).map((e: any) => ({
      name: e.name || '',
      path: e.path || '',
      size: e.size || 0,
      is_dir: e.is_dir || false,
      mod_time: e.mod_time || 0,
    }))
    dirCache.value[dirPath] = entries
    return entries
  } catch {
    dirCache.value[dirPath] = []
    return []
  }
}

async function navigateTo(dirPath: string) {
  loadingDir.value = true
  selected.value = new Set()
  try {
    allFiles.value = await loadDir(dirPath)
    currentPath.value = dirPath
    emit('pathChange', dirPath)
  } finally {
    loadingDir.value = false
  }
}

function handleNameClick(entry: LocalEntry, e: MouseEvent) {
  if (entry.is_dir && !e.ctrlKey && !e.metaKey) {
    navigateTo(entry.path)
    return
  }
  toggleSelect(entry, e)
}

function handleRowClick(entry: LocalEntry, e: MouseEvent) {
  toggleSelect(entry, e)
}

function toggleSelect(entry: LocalEntry, e: MouseEvent) {
  if (e.ctrlKey || e.metaKey) {
    if (selected.value.has(entry.path)) selected.value.delete(entry.path)
    else selected.value.add(entry.path)
  } else {
    selected.value = new Set([entry.path])
  }
}

function handleDeleteLocal() {
  if (selected.value.size === 0) return
  const names = Array.from(selected.value).map(p => p.split('/').pop() || p)
  dialog.warning({
    title: t('sftp.deleteTitle'),
    content: t('sftp.deleteContent', { name: names.length === 1 ? names[0] : `${names.length} items` }),
    positiveText: t('common.delete'),
    negativeText: t('common.cancel'),
    onPositiveClick: async () => {
      for (const p of selected.value) {
        try { await DeleteLocalFile(p) } catch { /* ignore */ }
      }
      selected.value = new Set()
      handleRefresh()
    },
  })
}

function handleRefresh() {
  dirCache.value = {}
  navigateTo(currentPath.value)
}

defineExpose({ refresh: handleRefresh })

function handleUpload() {
  if (selected.value.size === 0) return
  emit('upload', Array.from(selected.value), currentPath.value)
}

const pathParts = computed(() => {
  const parts = currentPath.value.split('/').filter(Boolean)
  return parts.map((name, i) => ({
    name,
    path: '/' + parts.slice(0, i + 1).join('/'),
  }))
})

function startEdit() {
  editPath.value = currentPath.value
  editing.value = true
}

function commitEdit() {
  editing.value = false
  navigateTo(editPath.value)
}

function formatSize(bytes: number): string {
  if (bytes === 0) return '-'
  if (bytes < 1024) return bytes + ' B'
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' K'
  if (bytes < 1024 * 1024 * 1024) return (bytes / (1024 * 1024)).toFixed(1) + ' M'
  return (bytes / (1024 * 1024 * 1024)).toFixed(1) + ' G'
}

function formatTime(ts: number): string {
  if (!ts) return '-'
  return new Date(ts * 1000).toLocaleDateString()
}

onMounted(async () => {
  try {
    const home = await GetHomeDir()
    await navigateTo(home)
  } catch {
    try { await navigateTo('/') } catch { /* empty */ }
  } finally {
    loading.value = false
  }
  registerLocalDrop()
})

onUnmounted(() => {
  cleanupLocalDrag()
  unregisterLocalDrop()
})
</script>

<template>
  <div class="local-panel">
    <div class="local-toolbar">
      <template v-if="!editing">
        <span class="local-breadcrumb" @dblclick="startEdit">
          <span class="crumb-part" @click="navigateTo('/')">/</span>
          <template v-for="p in pathParts" :key="p.path">
            <span class="crumb-part" @click="navigateTo(p.path)">{{ p.name }}</span>
            <span class="crumb-sep">/</span>
          </template>
        </span>
      </template>
      <input v-else v-model="editPath" class="local-path-input"
        @keyup.enter="commitEdit" @keyup.escape="editing = false" @blur="commitEdit" />
      <NButton size="tiny" quaternary @click="handleRefresh" title="Refresh">&#x21bb;</NButton>
      <NButton size="tiny" quaternary :type="showHidden ? 'primary' : 'default'" @click="showHidden = !showHidden">.*</NButton>
      <NButton size="tiny" quaternary class="upload-btn" :class="{ active: selected.size > 0 }" @click="handleUpload" title="Upload selected">
        <svg width="14" height="14" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.8"><path d="M8 12V3M4 6l4-4 4 4"/><path d="M2 12v2h12v-2"/></svg>
      </NButton>
      <NButton size="tiny" quaternary class="delete-btn" :class="{ active: selected.size > 0 }" @click="handleDeleteLocal" title="Delete selected">
        <svg width="14" height="14" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.8"><path d="M2 4h12M5 4V3a1 1 0 011-1h4a1 1 0 011 1v1M6 7v5M10 7v5M3 4l1 9a1 1 0 001 1h6a1 1 0 001-1l1-9"/></svg>
      </NButton>
    </div>

    <div class="local-body" ref="localBodyRef" :class="{ 'drag-over': localIsDragOver }">
      <NSpin v-if="loading || loadingDir" size="small" />
      <table v-else class="local-table">
        <thead>
          <tr>
            <th class="col-name sortable" @click="toggleSort('name')">Name <span class="sort-arrow" v-if="sortKey === 'name'">{{ sortAsc ? '↑' : '↓' }}</span></th>
            <th class="col-size sortable" @click="toggleSort('size')">Size <span class="sort-arrow" v-if="sortKey === 'size'">{{ sortAsc ? '↑' : '↓' }}</span></th>
            <th class="col-time sortable" @click="toggleSort('time')">Modified <span class="sort-arrow" v-if="sortKey === 'time'">{{ sortAsc ? '↑' : '↓' }}</span></th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="f in files" :key="f.name"
            class="local-row"
            :class="{ 'dir-row': f.is_dir, selected: selected.has(f.path) }"
            @mousedown="onLocalRowMouseDown($event, f)"
            @click="handleRowClick(f, $event)"
          >
            <td class="col-name">
              <span class="dir-name" @click.stop="handleNameClick(f, $event)">
                <span class="file-icon">{{ f.is_dir ? '\u{1F4C1}' : '\u{1F4C4}' }}</span>
                {{ f.name }}
              </span>
            </td>
            <td class="col-size">{{ f.is_dir ? '-' : formatSize(f.size) }}</td>
            <td class="col-time">{{ formatTime(f.mod_time) }}</td>
          </tr>
          <tr v-if="files.length === 0">
            <td colspan="3" class="local-empty">Empty directory</td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>

<style scoped>
.local-panel {
  display: flex;
  flex-direction: column;
  height: 100%;
  overflow: hidden;
}

.local-toolbar {
  display: flex;
  align-items: center;
  padding: 4px 8px;
  border-bottom: 1px solid var(--border-color);
  gap: 4px;
  flex-shrink: 0;
}

.local-breadcrumb {
  flex: 1;
  overflow: hidden;
  white-space: nowrap;
  font-size: var(--font-size-sm);
  user-select: none;
}

.crumb-part { color: var(--text-secondary); cursor: pointer; }
.crumb-part:hover { color: var(--text-primary); text-decoration: underline; }
.crumb-sep { color: var(--text-secondary); }

.local-path-input {
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

.upload-btn { color: var(--text-secondary); }
.upload-btn.active { color: var(--accent-color, #0078d4); }
.delete-btn { color: var(--text-secondary); }
.delete-btn.active { color: var(--delete-hover-color, #e55); }

.local-body { flex: 1; overflow-y: auto; min-height: 0; }

.drag-over {
  outline: 2px dashed rgba(56, 132, 244, 0.5);
  outline-offset: -2px;
  background: rgba(56, 132, 244, 0.06) !important;
  transition: outline 0.15s, background 0.15s;
}

.local-table { width: 100%; border-collapse: collapse; }
.local-table th {
  text-align: left; padding: 4px 8px;
  border-bottom: 1px solid var(--border-color);
  color: var(--text-secondary); font-weight: 500; font-size: var(--font-size-sm);
  user-select: none;
}
th.sortable { cursor: pointer; }
th.sortable:hover { color: var(--text-primary); }
.sort-arrow { margin-left: 2px; font-size: 10px; }
.local-table td {
  padding: 3px 8px;
  border-bottom: 1px solid var(--border-color);
  white-space: nowrap; overflow: hidden; text-overflow: ellipsis;
}

.local-row { cursor: default; }
.local-row.dir-row .dir-name { cursor: pointer; }
.local-row:hover { background: var(--hover-overlay-strong); }
.dir-row:hover .dir-name:hover { color: #5dade2; }
.local-row.selected { background: var(--action-hover-bg); }

.col-name { max-width: 160px; }
.col-size { width: 60px; text-align: right; }
.col-time { width: 85px; font-size: 11px; }
.file-icon { margin-right: 4px; }
.local-empty { text-align: center; color: var(--text-secondary); padding: 16px; }
</style>
