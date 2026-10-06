import { defineStore } from 'pinia'
import { ref, computed, watch } from 'vue'

export type SidebarView = 'connections' | 'keys' | 'ssh-config' | 'port-forward' | 'certs'
export type BottomTool = 'monitor' | 'sftp'

const STORAGE_KEY = 'vshell:layout'

interface PersistedLayout {
  activeSidebar: SidebarView
  leftPanelVisible: boolean
  sidebarWidth: number
  bottomHeight: number
}

function clamp(v: number, min: number, max: number): number {
  return Math.min(max, Math.max(min, v))
}

function loadPersisted(): Partial<PersistedLayout> {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (!raw) return {}
    const parsed = JSON.parse(raw) as Partial<PersistedLayout>
    if (typeof parsed !== 'object' || parsed === null) return {}
    return parsed
  } catch {
    return {}
  }
}

const SIDEBAR_VIEWS: SidebarView[] = ['connections', 'keys', 'ssh-config', 'port-forward', 'certs']

function sanitizeSidebar(v: unknown): SidebarView | undefined {
  return SIDEBAR_VIEWS.includes(v as SidebarView) ? (v as SidebarView) : undefined
}

export const useLayoutStore = defineStore('layout', () => {
  const saved = loadPersisted()

  /** 记住「最后视图」——隐藏期间不丢失，重开恢复（岛式 §3.4） */
  const activeSidebar = ref<SidebarView>(sanitizeSidebar(saved.activeSidebar) ?? 'connections')
  const leftPanelVisible = ref(typeof saved.leftPanelVisible === 'boolean' ? saved.leftPanelVisible : true)
  const sidebarWidth = ref(clamp(typeof saved.sidebarWidth === 'number' ? saved.sidebarWidth : 280, 150, 600))

  const bottomTool = ref<BottomTool | null>(null)
  const bottomHeight = ref(clamp(typeof saved.bottomHeight === 'number' ? saved.bottomHeight : 300, 80, 600))

  const bottomVisible = computed(() => bottomTool.value !== null)

  function setSidebar(view: SidebarView) {
    activeSidebar.value = view
  }

  /** 无参 = ⌘B 翻转；带参 = 条按钮语义：同视图再点收起，异视图切换并展开 */
  function toggleSidebar(view?: SidebarView) {
    if (!view) {
      leftPanelVisible.value = !leftPanelVisible.value
      return
    }
    if (!leftPanelVisible.value) {
      activeSidebar.value = view
      leftPanelVisible.value = true
    } else if (activeSidebar.value === view) {
      leftPanelVisible.value = false
    } else {
      activeSidebar.value = view
    }
  }

  function setSidebarWidth(w: number) {
    sidebarWidth.value = clamp(w, 150, 600)
  }

  function toggleBottomTool(tool: BottomTool) {
    bottomTool.value = bottomTool.value === tool ? null : tool
  }

  function setBottomHeight(h: number) {
    bottomHeight.value = clamp(h, 80, 600)
  }

  // 尺寸与显隐状态持久化（bottomTool 不持久化：每次启动从收起开始）
  watch(
    () => [activeSidebar.value, leftPanelVisible.value, sidebarWidth.value, bottomHeight.value] as const,
    ([view, visible, width, height]) => {
      try {
        const payload: PersistedLayout = {
          activeSidebar: view,
          leftPanelVisible: visible,
          sidebarWidth: width,
          bottomHeight: height,
        }
        localStorage.setItem(STORAGE_KEY, JSON.stringify(payload))
      } catch {
        // localStorage 不可用时静默降级为会话内状态
      }
    },
  )

  return {
    activeSidebar,
    leftPanelVisible,
    sidebarWidth,
    bottomTool,
    bottomHeight,
    bottomVisible,
    setSidebar,
    toggleSidebar,
    setSidebarWidth,
    toggleBottomTool,
    setBottomHeight,
  }
})
