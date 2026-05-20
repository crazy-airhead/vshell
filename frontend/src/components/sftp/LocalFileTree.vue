<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { NButton, useDialog, useMessage } from 'naive-ui'
import IconRefreshCw from '~icons/lucide/refresh-cw'
import IconUpload from '~icons/lucide/upload'
import IconTrash2 from '~icons/lucide/trash-2'
import IconFolder from '~icons/lucide/folder'
import IconFile from '~icons/lucide/file'
import { GetHomeDir, ListLocalDir, DeleteLocalFile, ReadLocalFileContent } from '../../../bindings/vshell/internal/app/appservice'
import { useDragSource, useDropTarget } from '../../composables/useDragTransfer'
import { isEditableFile } from '../../utils/fileType'
import { useTerminalStore } from '../../stores/terminal'

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
const terminalStore = useTerminalStore()

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

// Manual dblclick detection because drag overlay suppresses native dblclick
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

function handleNameClick(entry: LocalEntry, e: MouseEvent) {
  if (entry.is_dir && !e.ctrlKey && !e.metaKey) {
    navigateTo(entry.path)
    return
  }
  if (!entry.is_dir && checkLocalDblClick(entry)) {
    handleRowDblClick(entry)
    return
  }
  toggleSelect(entry, e)
}

function handleRowClick(entry: LocalEntry, e: MouseEvent) {
  if (!entry.is_dir && checkLocalDblClick(entry)) {
    handleRowDblClick(entry)
    return
  }
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

async function handleRowDblClick(entry: LocalEntry) {
  if (entry.is_dir) return
  if (!isEditableFile(entry.name, entry.size)) return

  const tabId = `editor-local:${entry.path}`

  if (terminalStore.tabs.find(t => t.id === tabId)) {
    terminalStore.activeTabID = tabId
    return
  }

  try {
    const content = await ReadLocalFileContent(entry.path)

    terminalStore.addEditorTab(tabId, entry.name, content, entry.path, {
      isRemote: false,
      editorMode: 'local-file',
      tooltip: entry.path,
    })
  } catch (e: any) {
    console.error('Failed to open local file:', e)
  }
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
  <div class="flex flex-col h-full overflow-hidden">
    <div class="flex items-center px-2 py-1 gap-1 shrink-0 toolbar-wrapper">
      <template v-if="!editing">
        <span class="flex-1 overflow-hidden whitespace-nowrap text-[var(--font-size-sm)] select-none cursor-default" @dblclick="startEdit">
          <span class="text-[var(--text-secondary)] cursor-pointer hover:text-[var(--text-primary)] hover:underline" @click="navigateTo('/')">/</span>
          <template v-for="p in pathParts" :key="p.path">
            <span class="text-[var(--text-secondary)] cursor-pointer hover:text-[var(--text-primary)] hover:underline" @click="navigateTo(p.path)">{{ p.name }}</span>
            <span class="text-[var(--text-secondary)]">/</span>
          </template>
        </span>
      </template>
      <input v-else v-model="editPath" class="flex-1 bg-[var(--bg-tertiary)] border border-solid border-[var(--border-color)] rounded-[3px] text-[var(--text-primary)] text-[var(--font-size-sm)] font-mono px-[6px] py-[2px] outline-none"
        @keyup.enter="commitEdit" @keyup.escape="editing = false" @blur="commitEdit" />
      <NButton size="tiny" quaternary @click="handleRefresh" title="Refresh"><IconRefreshCw :width="14" :height="14" /></NButton>
      <NButton size="tiny" quaternary :type="showHidden ? 'primary' : 'default'" @click="showHidden = !showHidden">.*</NButton>
      <NButton size="tiny" quaternary class="upload-btn" :class="{ active: selected.size > 0 }" @click="handleUpload" title="Upload selected">
        <IconUpload :width="14" :height="14" />
      </NButton>
      <NButton size="tiny" quaternary class="delete-btn" :class="{ active: selected.size > 0 }" @click="handleDeleteLocal" title="Delete selected">
        <IconTrash2 :width="14" :height="14" />
      </NButton>
      <div v-if="loading || loadingDir" class="loading-bar"></div>
    </div>

    <div class="flex-1 overflow-y-auto min-h-0" ref="localBodyRef" :class="{ 'drag-over': localIsDragOver }">
      <table v-if="!loading && !loadingDir" class="w-full border-collapse local-table">
        <thead>
          <tr>
            <th class="text-left py-1 px-2 text-[var(--text-secondary)] font-medium text-[var(--font-size-sm)] select-none cursor-pointer hover:text-[var(--text-primary)] max-w-[160px]" @click="toggleSort('name')">Name <span class="ml-[2px] text-[10px]" v-if="sortKey === 'name'">{{ sortAsc ? '↑' : '↓' }}</span></th>
            <th class="text-right py-1 px-2 text-[var(--text-secondary)] font-medium text-[var(--font-size-sm)] select-none cursor-pointer hover:text-[var(--text-primary)] w-[60px]" @click="toggleSort('size')">Size <span class="ml-[2px] text-[10px]" v-if="sortKey === 'size'">{{ sortAsc ? '↑' : '↓' }}</span></th>
            <th class="text-left py-1 px-2 text-[var(--text-secondary)] font-medium text-[var(--font-size-sm)] select-none cursor-pointer hover:text-[var(--text-primary)] w-[85px] text-[11px]" @click="toggleSort('time')">Modified <span class="ml-[2px] text-[10px]" v-if="sortKey === 'time'">{{ sortAsc ? '↑' : '↓' }}</span></th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="f in files" :key="f.name"
            class="local-row"
            :class="{ 'dir-row': f.is_dir, selected: selected.has(f.path) }"
            @mousedown="onLocalRowMouseDown($event, f)"
            @click="handleRowClick(f, $event)"
          >
            <td class="py-[3px] px-2 whitespace-nowrap overflow-hidden text-ellipsis max-w-[160px]">
              <span class="dir-name" @click.stop="handleNameClick(f, $event)">
                <component :is="f.is_dir ? IconFolder : IconFile" :width="14" :height="14" class="mr-1 inline-block align-[-2px]" />
                {{ f.name }}
              </span>
            </td>
            <td class="py-[3px] px-2 whitespace-nowrap text-right w-[60px]">{{ f.is_dir ? '-' : formatSize(f.size) }}</td>
            <td class="py-[3px] px-2 whitespace-nowrap w-[85px] text-[11px]">{{ formatTime(f.mod_time) }}</td>
          </tr>
          <tr v-if="files.length === 0">
            <td colspan="3" class="text-center text-[var(--text-secondary)] p-4">Empty directory</td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>

<style scoped>
.toolbar-wrapper {
  position: relative;
  border-bottom: 1px solid rgba(128, 128, 128, 0.12);
}

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

.local-row { cursor: default; }
.local-row:nth-child(even) { background: var(--hover-overlay-strong); }
.local-row.dir-row .dir-name { cursor: pointer; }
.local-row:hover { background: var(--hover-overlay); }
.dir-row:hover .dir-name:hover { color: var(--color-info); }
.local-row.selected { background: var(--action-hover-bg); }

.upload-btn { color: var(--text-secondary); }
.upload-btn.active { color: var(--color-primary); }
.delete-btn { color: var(--text-secondary); }
.delete-btn.active { color: var(--delete-hover-color); }

.drag-over {
  outline: 2px dashed rgba(100, 108, 255, 0.5);
  outline-offset: -2px;
  background: rgba(100, 108, 255, 0.06) !important;
  transition: outline 0.15s, background 0.15s;
}
</style>
