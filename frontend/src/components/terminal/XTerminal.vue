<script setup lang="ts">
import { ref, onMounted, onUnmounted, watch } from 'vue'
import { Terminal } from '@xterm/xterm'
import { FitAddon } from '@xterm/addon-fit'
import { WebglAddon } from '@xterm/addon-webgl'
import { Events } from '@wailsio/runtime'
import { useTerminalManager } from '../../composables/useTerminalManager'
import { useSettingsStore } from '../../stores/settings'

import '@xterm/xterm/css/xterm.css'

const props = defineProps<{
  sessionID: string
}>()

const terminalRef = ref<HTMLElement | null>(null)
const { registerTerminal, unregisterTerminal } = useTerminalManager()
const settings = useSettingsStore()

let term: Terminal | null = null
let fitAddon: FitAddon | null = null
let resizeObserver: ResizeObserver | null = null

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

  term.onData((data) => {
    Events.Emit('terminal:stdin', { sessionID: props.sessionID, data })
  })

  term.onResize(({ rows, cols }) => {
    Events.Emit('terminal:resize', { sessionID: props.sessionID, rows, cols })
  })

  resizeObserver = new ResizeObserver(() => {
    fitAddon?.fit()
  })
  resizeObserver.observe(terminalRef.value)
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
