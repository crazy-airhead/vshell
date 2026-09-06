import { defineStore } from 'pinia'
import { ref, computed } from 'vue'

export type SidebarView = 'connections' | 'keys' | 'ssh-config' | 'port-forward' | 'certs'
export type BottomTool = 'monitor' | 'sftp'

export const useLayoutStore = defineStore('layout', () => {
  const activeSidebar = ref<SidebarView>('connections')
  const sidebarWidth = ref(280)

  const activeBottomTool = ref<BottomTool | null>(null)
  const bottomPanelHeight = ref(300)

  const bottomAnyVisible = computed(() => activeBottomTool.value !== null)

  function setSidebar(view: SidebarView) {
    activeSidebar.value = view
  }

  function setSidebarWidth(w: number) {
    sidebarWidth.value = w
  }

  function toggleBottomTool(tool: BottomTool) {
    if (activeBottomTool.value === tool) {
      activeBottomTool.value = null
    } else {
      activeBottomTool.value = tool
    }
  }

  function setBottomPanelHeight(h: number) {
    bottomPanelHeight.value = h
  }

  return {
    activeSidebar,
    sidebarWidth,
    activeBottomTool,
    bottomPanelHeight,
    bottomAnyVisible,
    setSidebar,
    setSidebarWidth,
    toggleBottomTool,
    setBottomPanelHeight,
  }
})
