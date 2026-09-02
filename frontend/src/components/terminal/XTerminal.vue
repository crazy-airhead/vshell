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

const ctxShow = ref(false)
const ctxX = ref(0)
const ctxY = ref(0)
const ctxCanCopy = ref(false)

const colorSchemes: Record<string, Record<string, string>> = {
  'default-dark': {
    background: '#1e1e1e',
    foreground: '#cccccc',
    cursor: '#ffffff',
    selectionBackground: '#264f78',
  },
  'default-light': {
    background: '#ffffff',
    foreground: '#1e1e1e',
    cursor: '#1e1e1e',
    selectionBackground: '#add6ff',
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

watch(() => settings.isDark, () => {
  if (term) term.options.theme = getTermTheme()
})

watch(() => settings.terminalColorScheme, () => {
  if (term) term.options.theme = getTermTheme()
})

watch(() => settings.terminalFontSize, (size) => {
  if (term) term.options.fontSize = size
})

watch(() => settings.terminalFontFamily, (family) => {
  if (term) term.options.fontFamily = family
})

onUnmounted(() => {
  unregisterTerminal(props.sessionID)
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
  <div ref="terminalRef" class="xterminal-container" @contextmenu="onContextMenu"></div>
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
  padding: 4px;
}
</style>
