<script setup lang="ts">
import { ref, onMounted, onUnmounted, nextTick } from 'vue'
import { Terminal } from '@xterm/xterm'
import { FitAddon } from '@xterm/addon-fit'
import { WebglAddon } from '@xterm/addon-webgl'
import { Events } from '@wailsio/runtime'
import { useTerminalManager } from '../../composables/useTerminalManager'

import '@xterm/xterm/css/xterm.css'

const props = defineProps<{
  sessionID: string
}>()

const terminalRef = ref<HTMLElement | null>(null)
const { registerTerminal, unregisterTerminal } = useTerminalManager()

let term: Terminal | null = null
let fitAddon: FitAddon | null = null
let resizeObserver: ResizeObserver | null = null

onMounted(() => {
  if (!terminalRef.value) return

  term = new Terminal({
    cursorBlink: true,
    fontSize: 14,
    fontFamily: 'Menlo, Monaco, "Courier New", monospace',
    theme: {
      background: '#1e1e1e',
      foreground: '#cccccc',
      cursor: '#ffffff',
      selectionBackground: '#264f78',
    },
    allowProposedApi: true,
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
  fitAddon.fit()

  registerTerminal(props.sessionID, term)

  // Input: xterm -> backend
  term.onData((data) => {
    Events.Emit('terminal:stdin', { sessionID: props.sessionID, data })
  })

  // Resize: xterm -> backend
  term.onResize(({ rows, cols }) => {
    Events.Emit('terminal:resize', { sessionID: props.sessionID, rows, cols })
  })

  // Auto-fit on container resize
  resizeObserver = new ResizeObserver(() => {
    fitAddon?.fit()
  })
  resizeObserver.observe(terminalRef.value)
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

defineExpose({ fit })
</script>

<template>
  <div ref="terminalRef" class="xterminal-container"></div>
</template>

<style scoped>
.xterminal-container {
  width: 100%;
  height: 100%;
  padding: 4px;
}
</style>
