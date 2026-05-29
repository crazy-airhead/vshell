import { watch, onMounted, onUnmounted } from 'vue'
import { useSettingsStore } from '../stores/settings'
import type { ShortcutMap } from '../stores/settings'

type ShortcutAction = keyof ShortcutMap

function parseCombo(combo: string) {
  const parts = combo.split('+').map(p => p.trim())
  const ctrl = parts.includes('CommandOrControl')
  const shift = parts.includes('Shift')
  const alt = parts.includes('Alt')
  const key = parts.filter(p => !['CommandOrControl', 'Shift', 'Alt'].includes(p))[0] || ''
  return { ctrl, shift, alt, key: key.toUpperCase() }
}

export function useShortcuts(handlers: Partial<Record<ShortcutAction, () => void>>) {
  const settings = useSettingsStore()

  function handleKeydown(e: KeyboardEvent) {
    // Don't trigger shortcuts when typing in inputs
    const tag = (e.target as HTMLElement)?.tagName
    if (tag === 'INPUT' || tag === 'TEXTAREA' || tag === 'SELECT') return

    for (const [action, handler] of Object.entries(handlers)) {
      if (!handler) continue
      const combo = parseCombo(settings.shortcuts[action as ShortcutAction])
      const ctrlMatch = combo.ctrl ? (e.metaKey || e.ctrlKey) : (!e.metaKey && !e.ctrlKey)
      const shiftMatch = combo.shift ? e.shiftKey : !e.shiftKey
      const altMatch = combo.alt ? e.altKey : !e.altKey
      const keyMatch = e.key.toUpperCase() === combo.key

      if (ctrlMatch && shiftMatch && altMatch && keyMatch) {
        e.preventDefault()
        handler()
        return
      }
    }
  }

  onMounted(() => document.addEventListener('keydown', handleKeydown))
  onUnmounted(() => document.removeEventListener('keydown', handleKeydown))
}
