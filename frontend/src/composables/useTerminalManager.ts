import { Terminal } from '@xterm/xterm'
import { Events } from '@wailsio/runtime'

const terminals = new Map<string, Terminal>()
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
}

export function useTerminalManager() {
  ensureListener()

  function registerTerminal(sessionID: string, term: Terminal) {
    terminals.set(sessionID, term)
  }

  function unregisterTerminal(sessionID: string) {
    terminals.delete(sessionID)
  }

  return { registerTerminal, unregisterTerminal }
}
