<script setup lang="ts">
import { ref, onMounted, onUnmounted, watch } from 'vue'
import { Terminal } from '@xterm/xterm'
import { FitAddon } from '@xterm/addon-fit'
import { WebglAddon } from '@xterm/addon-webgl'
import { NDropdown } from 'naive-ui'
import type { DropdownOption } from 'naive-ui'
import { Events } from '@wailsio/runtime'
import { useTerminalManager } from '../../composables/useTerminalManager'
import { matchesShortcut } from '../../composables/useShortcuts'
import { useTerminalStore } from '../../stores/terminal'
import { useConnectionStore } from '../../stores/connection'
import { useSettingsStore } from '../../stores/settings'
import { writeClipboard, readClipboard } from '../../utils/clipboard'
import { useI18n } from 'vue-i18n'

import '@xterm/xterm/css/xterm.css'

const props = defineProps<{
  sessionID: string
}>()

const terminalRef = ref<HTMLElement | null>(null)
const { registerTerminal, unregisterTerminal, isDisconnected, clearDisconnected } = useTerminalManager()
const terminalStore = useTerminalStore()
const connectionStore = useConnectionStore()
const settings = useSettingsStore()
const { t } = useI18n()

let reconnecting = false
const isMac = navigator.platform.includes('Mac')

let term: Terminal | null = null
let fitAddon: FitAddon | null = null
let resizeObserver: ResizeObserver | null = null
let dprQuery: MediaQueryList | null = null

const ctxShow = ref(false)
const ctxX = ref(0)
const ctxY = ref(0)
const ctxCanCopy = ref(false)

const colorSchemes: Record<string, Record<string, string>> = {
  'default-dark': {
    background: '#1E1F22',
    foreground: '#DFE1E5',
    cursor: '#FFFFFF',
    selectionBackground: '#2E436E',
  },
  'default-light': {
    background: '#FFFFFF',
    foreground: '#1F2329',
    cursor: '#1F2329',
    selectionBackground: '#D0DFFE',
  },
  'solarized-dark': {
    background: '#002b36',
    foreground: '#839496',
    cursor: '#93a1a1',
    selectionBackground: '#073642',
  },
  'solarized-light': {
    background: '#fdf6e3',
    foreground: '#657b83',
    cursor: '#586e75',
    selectionBackground: '#eee8d5',
  },
  'dracula': {
    background: '#282a36',
    foreground: '#f8f8f2',
    cursor: '#f8f8f2',
    selectionBackground: '#44475a',
  },
  'monokai': {
    background: '#272822',
    foreground: '#f8f8f2',
    cursor: '#f8f8f0',
    selectionBackground: '#49483e',
  },
  'one-dark': {
    background: '#282c34',
    foreground: '#abb2bf',
    cursor: '#528bff',
    selectionBackground: '#3e4451',
  },
}

function getTermTheme() {
  const scheme = settings.terminalColorScheme
  const key = scheme === 'default'
    ? (settings.isDark ? 'default-dark' : 'default-light')
    : scheme
  return colorSchemes[key] || colorSchemes['default-dark']
}

onMounted(() => {
  if (!terminalRef.value) return

  term = new Terminal({
    cursorBlink: true,
    fontSize: settings.terminalFontSize,
    fontFamily: settings.terminalFontFamily,
    theme: getTermTheme(),
    allowProposedApi: true,
    rightClickSelectsWord: true,
  })

  fitAddon = new FitAddon()
  term.loadAddon(fitAddon)

  try {
    const webgl = new WebglAddon()
    webgl.onContextLoss(() => webgl.dispose())
    term.loadAddon(webgl)
  } catch {
    // Fallback to canvas renderer
  }

  term.open(terminalRef.value)

  // Set up resize observer before initial fit so it can pick up
  // the correct dimensions once layout settles.
  resizeObserver = new ResizeObserver(() => {
    fitAddon?.fit()
  })
  resizeObserver.observe(terminalRef.value)

  // dpr 变化时容器 css 尺寸可能不变（如两块同分辨率不同缩放的屏），
  // ResizeObserver 不触发，需单独监听 refit（ISSUE-0011）
  dprQuery = window.matchMedia(`(resolution: ${window.devicePixelRatio}dppx)`)
  dprQuery.addEventListener('change', onDprChange)

  // Defer initial fit past the first paint to ensure the container
  // has its final flex layout dimensions. Without this, the first
  // tab's terminal may get 0 height and the WebGL renderer won't
  // repaint buffered content until a manual resize.
  requestAnimationFrame(() => {
    fitAddon?.fit()
    if (term && term.rows > 0) {
      term.refresh(0, term.rows - 1)
    }
  })

  registerTerminal(props.sessionID, term)

  // The global shortcut handler skips events targeting xterm's hidden
  // textarea, so copy/paste shortcuts are intercepted here instead.
  // Returning false stops xterm from sending the key to the PTY.
  term.attachCustomKeyEventHandler((event) => {
    if (event.type !== 'keydown') return true
    // On macOS "CommandOrControl" must mean Cmd only for copy/paste:
    // plain Ctrl+C is SIGINT and Ctrl+V is quoted-insert — never swallow them.
    const ctrlOnly = event.ctrlKey && !event.metaKey
    if (!(isMac && ctrlOnly) && matchesShortcut(event, settings.shortcuts.copy)) {
      event.preventDefault()
      copySelection()
      return false
    }
    if (!(isMac && ctrlOnly) && matchesShortcut(event, settings.shortcuts.paste)) {
      // preventDefault keeps the native paste action (menu accelerator path)
      // from ALSO firing — otherwise the clipboard would land twice.
      event.preventDefault()
      pasteFromClipboard()
      return false
    }
    // macOS convention: plain Cmd+C copies when a selection exists,
    // otherwise let it through (covers users who rebound the copy shortcut).
    if (
      isMac && event.metaKey && !event.ctrlKey && !event.altKey && !event.shiftKey &&
      event.key.toLowerCase() === 'c' && term?.hasSelection()
    ) {
      event.preventDefault()
      copySelection()
      return false
    }
    return true
  })

  term.onData((data) => {
    if (isDisconnected(props.sessionID)) {
      if (data === '\r' && !reconnecting) {
        reconnecting = true
        handleReconnect()
      }
      return
    }
    Events.Emit('terminal:stdin', { sessionID: props.sessionID, data })
  })

  term.onResize(({ rows, cols }) => {
    Events.Emit('terminal:resize', { sessionID: props.sessionID, rows, cols })
  })
})

/**
 * 行高变化（字号 / 字体 / dpr）后主动 refit：此类变化不改变容器尺寸，
 * ResizeObserver 不会触发；xterm 只重测行高、不重算行数，行数仍按旧行高
 * 计算时 rows × 新行高会超出容器，最后一行溢出下缘被裁（ISSUE-0011）。
 * option 变更时 xterm 同步重测，双 rAF 等布局稳定后按新行高取整行数。
 */
function scheduleFit() {
  requestAnimationFrame(() => requestAnimationFrame(() => fitAddon?.fit()))
}

/** dpr 变化（窗口在 Retina / 外接屏间移动）时跟随新档位继续监听并 refit */
function onDprChange() {
  scheduleFit()
  dprQuery?.removeEventListener('change', onDprChange)
  dprQuery = window.matchMedia(`(resolution: ${window.devicePixelRatio}dppx)`)
  dprQuery.addEventListener('change', onDprChange)
}

watch(() => settings.isDark, () => {
  if (term) term.options.theme = getTermTheme()
})

watch(() => settings.terminalColorScheme, () => {
  if (term) term.options.theme = getTermTheme()
})

watch(() => settings.terminalFontSize, (size) => {
  if (term) term.options.fontSize = size
  scheduleFit()
})

watch(() => settings.terminalFontFamily, (family) => {
  if (term) term.options.fontFamily = family
  scheduleFit()
})

// 标签激活时兜底 fit：隐藏期间若发生未触发 ResizeObserver 的失同步，此时收敛
watch(() => terminalStore.activeTabID === props.sessionID, (active) => {
  if (active) scheduleFit()
})

onUnmounted(() => {
  unregisterTerminal(props.sessionID)
  dprQuery?.removeEventListener('change', onDprChange)
  dprQuery = null
  resizeObserver?.disconnect()
  term?.dispose()
  term = null
  fitAddon = null
})

function fit() {
  fitAddon?.fit()
}

function copySelection() {
  if (!term || !term.hasSelection()) return
  void writeClipboard(term.getSelection())
}

async function pasteFromClipboard() {
  if (!term) return
  const text = await readClipboard()
  if (text) term.paste(text)
}

function onContextMenu(e: MouseEvent) {
  e.preventDefault()
  // rightClickSelectsWord has already applied (xterm handles its own
  // contextmenu first, on the inner .xterm element), so the word under
  // the cursor — or a pre-existing selection — is available here.
  ctxCanCopy.value = term?.hasSelection() ?? false
  ctxX.value = e.clientX
  ctxY.value = e.clientY
  ctxShow.value = true
}

function getContextMenuOptions(): DropdownOption[] {
  return [
    { label: t('terminal.copy'), key: 'copy', disabled: !ctxCanCopy.value },
    { label: t('terminal.paste'), key: 'paste' },
    { type: 'divider', key: 'd1' },
    { label: t('terminal.selectAll'), key: 'selectAll' },
    { label: t('terminal.clear'), key: 'clear' },
  ]
}

function handleContextMenuSelect(action: string) {
  ctxShow.value = false
  switch (action) {
    case 'copy':
      copySelection()
      break
    case 'paste':
      void pasteFromClipboard()
      break
    case 'selectAll':
      term?.selectAll()
      break
    case 'clear':
      term?.clear()
      break
  }
}

async function handleReconnect() {
  const tab = terminalStore.tabs.find((t) => t.id === props.sessionID)
  if (!tab || !tab.connectionID) {
    reconnecting = false
    return
  }

  term?.write(`\r\n\x1b[33m${t('tab.reconnectingNotice')}\x1b[0m\r\n`)

  try {
    await connectionStore.disconnectSession(props.sessionID, tab.connectionID)
    const newSessionID = await connectionStore.connect(tab.connectionID)
    clearDisconnected(props.sessionID)
    terminalStore.removeTab(props.sessionID)
    terminalStore.addTab({
      id: newSessionID,
      connectionID: tab.connectionID,
      title: tab.title,
      connected: true,
    })
  } catch {
    term?.write(`\x1b[31m${t('tab.reconnectFailedNotice')}\x1b[0m\r\n`)
    reconnecting = false
  }
}

defineExpose({ fit })
</script>

<template>
  <div class="xterminal-container">
    <div ref="terminalRef" class="xterm-host" @contextmenu="onContextMenu"></div>
  </div>
  <NDropdown
    trigger="manual"
    :show="ctxShow"
    :x="ctxX"
    :y="ctxY"
    :options="getContextMenuOptions()"
    @select="handleContextMenuSelect"
    @clickoutside="ctxShow = false"
    placement="bottom-start"
  />
</template>

<style scoped>
.xterminal-container {
  width: 100%;
  height: 100%;
  /* 岛式 §5：内容与 20px 岛圆角之间留 ≥6px，防角部字形被裁。
     padding 必须留在本层、.xterm 挂在内层无 padding 的 .xterm-host 上：
     FitAddon 按 `.xterm` 父元素 computed height 减 `.xterm` 自身 padding
     算行数，而 box-sizing:border-box 下 computed height 含 padding —— 若
     .xterm 直接挂本层，上下 6px 永不被扣除，行数多算，最后一行溢出底缘
     被中心岛 overflow-hidden 裁切（ISSUE-0011）。 */
  padding: var(--space-3);
}
.xterm-host {
  width: 100%;
  height: 100%;
}
</style>
