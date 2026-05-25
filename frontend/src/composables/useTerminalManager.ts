import { Terminal } from '@xterm/xterm'
import { Events } from '@wailsio/runtime'
import { useTerminalStore } from '../stores/terminal'

const terminals = new Map<string, Terminal>()
const disconnectedSessions = new Set<string>()
let listenerRegistered = false

function ensureListener() {
  if (listenerRegistered) return
  listenerRegistered = true
  Events.On('terminal:stdout', (ev: any) => {
    const d = ev?.data
    if (d?.sessionID && d?.data) {
      const term = terminals.get(d.sessionID)
      if (term) term.write(d.data)
    }
  })
  Events.On('terminal:closed', (ev: any) => {
    const d = ev?.data
    if (d?.sessionID) {
      disconnectedSessions.add(d.sessionID)
      const terminalStore = useTerminalStore()
      terminalStore.markTabDisconnected(d.sessionID)
      const term = terminals.get(d.sessionID)
      if (term) {
        term.write('\r\n\x1b[33m--- 连接已断开，按回车键重连 ---\x1b[0m\r\n')
      }
    }
  })
}

export function useTerminalManager() {
  ensureListener()

  function registerTerminal(sessionID: string, term: Terminal) {
    terminals.set(sessionID, term)
  }

  function unregisterTerminal(sessionID: string) {
    terminals.delete(sessionID)
    disconnectedSessions.delete(sessionID)
  }

  function isDisconnected(sessionID: string): boolean {
    return disconnectedSessions.has(sessionID)
  }

  function clearDisconnected(sessionID: string) {
    disconnectedSessions.delete(sessionID)
  }

  return { registerTerminal, unregisterTerminal, isDisconnected, clearDisconnected }
}
