<script setup lang="ts">
import { ref, onMounted, onUnmounted, watch } from 'vue'
import { Terminal } from '@xterm/xterm'
import { FitAddon } from '@xterm/addon-fit'
import { WebglAddon } from '@xterm/addon-webgl'
import { Events } from '@wailsio/runtime'
import { useTerminalManager } from '../../composables/useTerminalManager'
import { useTerminalStore } from '../../stores/terminal'
import { useConnectionStore } from '../../stores/connection'
import { useSettingsStore } from '../../stores/settings'
import { resolveTerminalTheme } from '../../constants/terminalThemes'
import { shortcutActionFromKeyboardEvent, shortcutDigitIndex } from '../../composables/useShortcuts'
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

let term: Terminal | null = null
let fitAddon: FitAddon | null = null
let resizeObserver: ResizeObserver | null = null
let offNativeFileDrop: (() => void) | null = null

const dropTargetID = `terminal-drop-${props.sessionID}`

function getTermTheme() {
  return resolveTerminalTheme(settings.terminalColorScheme, settings.isDark)
}

function shellQuotePath(path: string): string {
  if (/^[A-Za-z0-9_./:@%+=,-]+$/.test(path)) return path
  return `'${path.replace(/'/g, `'\\''`)}'`
}

function formatDroppedPaths(paths: string[]): string {
  return paths.map(shellQuotePath).join(' ') + ' '
}

function handleNativeFileDrop(ev: any) {
  const data = ev?.data
  if (!data?.files || data.targetId !== dropTargetID) return

  const paths = (data.files as string[]).filter(Boolean)
  if (paths.length === 0) return

  const text = formatDroppedPaths(paths)
  Events.Emit('terminal:stdin', { sessionID: props.sessionID, data: text })
}

function handleCustomKeyEvent(e: KeyboardEvent): boolean {
  if (e.type !== 'keydown') return true
  if (document.documentElement.hasAttribute('data-shortcut-capturing')) return true

  if ((e.metaKey || e.ctrlKey) && !e.altKey && !e.shiftKey) {
    const index = shortcutDigitIndex(e)
    if (index !== null) {
      e.preventDefault()
      e.stopPropagation()
      window.dispatchEvent(new CustomEvent('vshell:activate-tab-index', { detail: { index } }))
      return false
    }
  }

  const action = shortcutActionFromKeyboardEvent(settings.shortcuts, e)
  if (action) {
    e.preventDefault()
    e.stopPropagation()
    window.dispatchEvent(new CustomEvent('vshell:app-shortcut', { detail: { action } }))
    return false
  }

  return true
}

onMounted(() => {
  if (!terminalRef.value) return

  term = new Terminal({
    cursorBlink: true,
    fontSize: settings.terminalFontSize,
    fontFamily: settings.terminalFontFamily,
    theme: getTermTheme(),
    allowProposedApi: true,
  })

  term.attachCustomKeyEventHandler(handleCustomKeyEvent)

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
  offNativeFileDrop = Events.On('native:file-drop', handleNativeFileDrop)

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
  offNativeFileDrop?.()
  offNativeFileDrop = null
  resizeObserver?.disconnect()
  term?.dispose()
  term = null
  fitAddon = null
})

function fit() {
  fitAddon?.fit()
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
  <div
    :id="dropTargetID"
    ref="terminalRef"
    class="xterminal-container"
    data-file-drop-target
  ></div>
</template>

<style scoped>
.xterminal-container {
  width: 100%;
  height: 100%;
  padding: 4px;
}
</style>
