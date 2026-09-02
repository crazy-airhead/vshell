import { watch, onMounted, onUnmounted } from 'vue'
import { useSettingsStore } from '../stores/settings'
import type { ShortcutMap } from '../stores/settings'

type ShortcutAction = keyof ShortcutMap

export function parseCombo(combo: string) {
  const parts = combo.split('+').map(p => p.trim())
  const ctrl = parts.includes('CommandOrControl')
  const shift = parts.includes('Shift')
  const alt = parts.includes('Alt')
  const key = parts.filter(p => !['CommandOrControl', 'Shift', 'Alt'].includes(p))[0] || ''
  return { ctrl, shift, alt, key: key.toUpperCase() }
}

// Whether a keyboard event matches a shortcut combo like 'CommandOrControl+Shift+C'.
// Used by the global handler here and by xterm's attachCustomKeyEventHandler
// (the global handler skips events targeted at xterm's hidden textarea).
export function matchesShortcut(e: KeyboardEvent, combo: string | undefined): boolean {
  if (!combo) return false
  const parsed = parseCombo(combo)
  if (!parsed.key) return false
  const ctrlMatch = parsed.ctrl ? (e.metaKey || e.ctrlKey) : (!e.metaKey && !e.ctrlKey)
  const shiftMatch = parsed.shift ? e.shiftKey : !e.shiftKey
  const altMatch = parsed.alt ? e.altKey : !e.altKey
  const keyMatch = e.key.toUpperCase() === parsed.key
  return ctrlMatch && shiftMatch && altMatch && keyMatch
}

export function useShortcuts(handlers: Partial<Record<ShortcutAction, () => void>>) {
  const settings = useSettingsStore()

  function handleKeydown(e: KeyboardEvent) {
    // Don't trigger shortcuts when typing in inputs
    const tag = (e.target as HTMLElement)?.tagName
    if (tag === 'INPUT' || tag === 'TEXTAREA' || tag === 'SELECT') return

    for (const [action, handler] of Object.entries(handlers)) {
      if (!handler) continue
      if (matchesShortcut(e, settings.shortcuts[action as ShortcutAction])) {
        e.preventDefault()
        handler()
        return
      }
    }
  }

  onMounted(() => document.addEventListener('keydown', handleKeydown))
  onUnmounted(() => document.removeEventListener('keydown', handleKeydown))
}
