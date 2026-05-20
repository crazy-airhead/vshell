import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { TreeNode } from '../types'

export type TabType = 'terminal' | 'editor'
export type EditorMode = 'ssh-config' | 'remote-sftp' | 'local-file'

export interface TerminalTab {
  id: string
  connectionID: string
  title: string
  type?: TabType
  editorContent?: string
  filePath?: string
  dirty?: boolean
  isRemote?: boolean
  editorMode?: EditorMode
  tooltip?: string
  connected?: boolean
}

export const useTerminalStore = defineStore('terminal', () => {
  const tabs = ref<TerminalTab[]>([])
  const activeTabID = ref<string | null>(null)
  const splitTree = ref<TreeNode | null>(null)

  function addTab(tab: TerminalTab) {
    tabs.value.push(tab)
    activeTabID.value = tab.id
    if (!splitTree.value) {
      splitTree.value = { type: 'leaf', sessionID: tab.id }
    }
  }

  function addEditorTab(
    id: string,
    title: string,
    content: string,
    filePath: string,
    opts?: { isRemote?: boolean; editorMode?: EditorMode; tooltip?: string; connectionID?: string },
  ) {
    if (tabs.value.find((t) => t.id === id)) {
      activeTabID.value = id
      return
    }
    addTab({
      id,
      connectionID: opts?.connectionID || '',
      title,
      type: 'editor',
      editorContent: content,
      filePath,
      dirty: false,
      isRemote: opts?.isRemote,
      editorMode: opts?.editorMode,
      tooltip: opts?.tooltip,
    })
  }

  function updateTabContent(id: string, content: string) {
    const tab = tabs.value.find((t) => t.id === id)
    if (tab) {
      tab.editorContent = content
    }
  }

  function markTabDirty(id: string, dirty: boolean) {
    const tab = tabs.value.find((t) => t.id === id)
    if (tab) {
      tab.dirty = dirty
    }
  }

  function removeTab(id: string) {
    const idx = tabs.value.findIndex((t) => t.id === id)
    if (idx >= 0) {
      tabs.value.splice(idx, 1)
    }
    if (activeTabID.value === id) {
      activeTabID.value = tabs.value.length > 0 ? tabs.value[0].id : null
    }
    if (splitTree.value?.type === 'leaf' && splitTree.value.sessionID === id) {
      splitTree.value = tabs.value.length > 0
        ? { type: 'leaf', sessionID: tabs.value[0].id }
        : null
    }
  }

  function markTabDisconnected(sessionID: string) {
    const tab = tabs.value.find((t) => t.id === sessionID)
    if (tab) {
      tab.connected = false
    }
  }

  function closeOtherTabs(id: string) {
    const keep = tabs.value.find((t) => t.id === id)
    tabs.value = keep ? [keep] : []
    activeTabID.value = id
  }

  function closeAllTabs() {
    tabs.value = []
    activeTabID.value = null
    splitTree.value = null
  }

  return {
    tabs,
    activeTabID,
    splitTree,
    addTab,
    addEditorTab,
    updateTabContent,
    markTabDirty,
    removeTab,
    markTabDisconnected,
    closeOtherTabs,
    closeAllTabs,
  }
})
