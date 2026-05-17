import { defineStore } from 'pinia'
import { ref, reactive } from 'vue'
import { SFTPReadDir } from '../../bindings/vshell/internal/app/appservice'

export interface SFTPFile {
  name: string
  size: number
  mode: number
  mod_time: number
  is_dir: boolean
}

export interface SFTPPanelState {
  connectionID: string
  currentPath: string
  files: SFTPFile[]
  loading: boolean
  error: string | null
  expandedPaths: string[]
  treeCache: Record<string, SFTPFile[]>
}

function createPanel(connectionID: string): SFTPPanelState {
  return {
    connectionID,
    currentPath: '/',
    files: [],
    loading: false,
    error: null,
    expandedPaths: [],
    treeCache: {},
  }
}

export const useSFTPStore = defineStore('sftp', () => {
  const panels = ref(new Map<string, SFTPPanelState>())
  const treeVersion = ref(0)

  function getPanel(connectionID: string): SFTPPanelState {
    if (!panels.value.has(connectionID)) {
      panels.value.set(connectionID, createPanel(connectionID))
    }
    return panels.value.get(connectionID)!
  }

  async function readDir(connectionID: string, path: string): Promise<SFTPFile[]> {
    const result = await SFTPReadDir(connectionID, path)
    return (result || []).map((f: any) => ({
      name: f.name || '',
      size: f.size || 0,
      mode: f.mode || 0,
      mod_time: f.mod_time || 0,
      is_dir: f.is_dir || false,
    }))
  }

  async function navigateToDir(connectionID: string, path: string) {
    const panel = getPanel(connectionID)
    panel.loading = true
    panel.error = null
    try {
      const items = await readDir(connectionID, path)
      panel.files = items
      panel.currentPath = path
      panel.treeCache = { ...panel.treeCache, [path]: items }
      treeVersion.value++
    } catch (e: any) {
      panel.error = String(e)
    } finally {
      panel.loading = false
    }
  }

  async function loadTreeDir(connectionID: string, path: string) {
    const panel = getPanel(connectionID)
    if (path in panel.treeCache) return
    try {
      const items = await readDir(connectionID, path)
      panel.treeCache = { ...panel.treeCache, [path]: items }
      treeVersion.value++
    } catch {
      panel.treeCache = { ...panel.treeCache, [path]: [] }
      treeVersion.value++
    }
  }

  function toggleExpandPath(connectionID: string, path: string, expand: boolean) {
    const panel = getPanel(connectionID)
    if (expand) {
      if (!panel.expandedPaths.includes(path)) {
        panel.expandedPaths = [...panel.expandedPaths, path]
      }
    } else {
      panel.expandedPaths = panel.expandedPaths.filter(p => p !== path)
    }
    treeVersion.value++
  }

  function closePanel(connectionID: string) {
    panels.value.delete(connectionID)
  }

  return {
    panels,
    treeVersion,
    getPanel,
    readDir,
    navigateToDir,
    loadTreeDir,
    toggleExpandPath,
    closePanel,
  }
})
