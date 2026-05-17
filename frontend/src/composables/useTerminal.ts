import { ref, onUnmounted } from 'vue'
import { Terminal } from '@xterm/xterm'
import { FitAddon } from '@xterm/addon-fit'
import { WebglAddon } from '@xterm/addon-webgl'
import { SearchAddon } from '@xterm/addon-search'
import { SerializeAddon } from '@xterm/addon-serialize'
import { Events } from '@wailsio/runtime'

export function useTerminal(sessionID: string) {
  const terminal = ref<Terminal | null>(null)
  const fitAddon = ref<FitAddon | null>(null)

  function init(el: HTMLElement) {
    const term = new Terminal({
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

    const fit = new FitAddon()
    const search = new SearchAddon()
    const serialize = new SerializeAddon()

    term.loadAddon(fit)
    term.loadAddon(search)
    term.loadAddon(serialize)

    try {
      const webgl = new WebglAddon()
      webgl.onContextLoss(() => webgl.dispose())
      term.loadAddon(webgl)
    } catch {
      // Fallback to canvas renderer
    }

    term.open(el)
    fit.fit()

    terminal.value = term
    fitAddon.value = fit

    term.onData((data) => {
      Events.Emit('terminal:stdin', { sessionID, data })
    })

    Events.On('terminal:stdout', (ev: any) => {
      const d = ev?.data
      if (d?.sessionID === sessionID && d?.data) {
        term.write(d.data)
      }
    })

    term.onResize(({ rows, cols }) => {
      Events.Emit('terminal:resize', { sessionID, rows, cols })
    })

    const observer = new ResizeObserver(() => {
      fit.fit()
    })
    observer.observe(el)

    onUnmounted(() => {
      observer.disconnect()
      Events.Off('terminal:stdout')
      term.dispose()
    })
  }

  function fit() {
    fitAddon.value?.fit()
  }

  function write(data: string) {
    terminal.value?.write(data)
  }

  return { terminal, init, fit, write }
}
