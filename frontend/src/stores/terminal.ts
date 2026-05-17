import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { TreeNode } from '../types'

export interface TerminalTab {
  id: string
  connectionID: string
  title: string
}

export const useTerminalStore = defineStore('terminal', () => {
  const tabs = ref<TerminalTab[]>([])
  const activeTabID = ref<string | null>(null)
  const splitTree = ref<TreeNode | null>(null)

  function addTab(tab: TerminalTab) {
    if (!tabs.value.find((t) => t.id === tab.id)) {
      tabs.value.push(tab)
    }
    activeTabID.value = tab.id
    if (!splitTree.value) {
      splitTree.value = { type: 'leaf', sessionID: tab.id }
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

  return {
    tabs,
    activeTabID,
    splitTree,
    addTab,
    removeTab,
  }
})
