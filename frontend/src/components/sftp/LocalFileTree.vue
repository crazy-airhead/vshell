<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { NSpin, NButton } from 'naive-ui'
import { GetHomeDir, ListLocalDir } from '../../../bindings/vshell/internal/app/appservice'

const emit = defineEmits<{
  (e: 'dropFiles', paths: string[], targetDir: string): void
}>()

interface LocalEntry {
  name: string
  path: string
  size: number
  is_dir: boolean
  mod_time: number
}

const showHidden = ref(false)
const currentPath = ref('')
const allFiles = ref<LocalEntry[]>([])
const loading = ref(true)
const loadingDir = ref(false)
const dragOver = ref(false)
const editing = ref(false)
const editPath = ref('')

const files = computed(() =>
  showHidden.value ? allFiles.value : allFiles.value.filter(f => !f.name.startsWith('.'))
)

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
  try {
    allFiles.value = await loadDir(dirPath)
    currentPath.value = dirPath
  } finally {
    loadingDir.value = false
  }
}

function handleFileClick(entry: LocalEntry) {
  if (entry.is_dir) navigateTo(entry.path)
}

function handleRefresh() {
  dirCache.value = {}
  navigateTo(currentPath.value)
}

// Breadcrumb
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

// Drag & Drop
function onFileDragStart(e: DragEvent, entry: LocalEntry) {
  if (entry.is_dir) {
    e.preventDefault()
    return
  }
  if (e.dataTransfer) {
    e.dataTransfer.effectAllowed = 'copy'
    e.dataTransfer.setData('text/plain', entry.path)
  }
}

function onDragOver(e: DragEvent) {
  e.preventDefault()
  dragOver.value = true
  if (e.dataTransfer) {
    e.dataTransfer.dropEffect = 'copy'
  }
}

function onDragLeave() {
  dragOver.value = false
}

function onDrop(e: DragEvent) {
  e.preventDefault()
  dragOver.value = false
  if (!e.dataTransfer) return
  const data = e.dataTransfer.getData('text/plain')
  if (!data) return
  emit('dropFiles', data.split('\n').filter(Boolean), currentPath.value)
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
})
</script>

<template>
  <div
    class="local-panel"
    :class="{ 'drop-active': dragOver }"
    @dragover="onDragOver"
    @dragleave="onDragLeave"
    @drop="onDrop"
  >
    <!-- Breadcrumb toolbar -->
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
      <input
        v-else
        v-model="editPath"
        class="local-path-input"
        @keyup.enter="commitEdit"
        @keyup.escape="editing = false"
        @blur="commitEdit"
      />
      <NButton size="tiny" quaternary @click="handleRefresh">&#x21bb;</NButton>
      <NButton size="tiny" quaternary :type="showHidden ? 'primary' : 'default'" @click="showHidden = !showHidden" title="Toggle hidden files">.*</NButton>
    </div>

    <!-- File list -->
    <div class="local-body">
      <NSpin v-if="loading || loadingDir" size="small" />
      <table v-else class="local-table">
        <thead>
          <tr>
            <th class="col-name">Name</th>
            <th class="col-size">Size</th>
            <th class="col-time">Modified</th>
          </tr>
        </thead>
        <tbody>
          <tr
            v-for="f in files"
            :key="f.name"
            class="local-row"
            :class="{ 'dir-row': f.is_dir }"
            :draggable="!f.is_dir"
            @click="handleFileClick(f)"
            @dragstart="onFileDragStart($event, f)"
          >
            <td class="col-name">
              <span class="file-icon">{{ f.is_dir ? '\u{1F4C1}' : '\u{1F4C4}' }}</span>
              {{ f.name }}
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

.local-panel.drop-active {
  background: var(--action-hover-bg);
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

.crumb-part {
  color: var(--text-secondary);
  cursor: pointer;
}

.crumb-part:hover {
  color: var(--text-primary);
  text-decoration: underline;
}

.crumb-sep {
  color: var(--text-secondary);
}

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

.local-body {
  flex: 1;
  overflow-y: auto;
  min-height: 0;
}

.local-table {
  width: 100%;
  border-collapse: collapse;
}

.local-table th {
  text-align: left;
  padding: 4px 8px;
  border-bottom: 1px solid var(--border-color);
  color: var(--text-secondary);
  font-weight: 500;
  font-size: var(--font-size-sm);
}

.local-table td {
  padding: 3px 8px;
  border-bottom: 1px solid var(--border-color);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.local-row { cursor: default; }
.local-row.dir-row { cursor: pointer; }
.local-row:hover { background: var(--hover-overlay-strong); }
.dir-row:hover { background: var(--action-hover-bg); }

.col-name { max-width: 160px; }
.col-size { width: 60px; text-align: right; }
.col-time { width: 85px; font-size: 11px; }
.file-icon { margin-right: 4px; }

.local-empty {
  text-align: center;
  color: var(--text-secondary);
  padding: 16px;
}
</style>
