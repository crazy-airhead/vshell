import { onMounted, onUnmounted } from 'vue'
import { useSettingsStore } from '../stores/settings'
import type { ShortcutMap } from '../stores/settings'

type ShortcutAction = keyof ShortcutMap

const modifierKeys = new Set(['Control', 'Shift', 'Alt', 'Meta'])

export function normalizeShortcutEventKey(e: KeyboardEvent): string {
  if (e.code === 'Space') return 'Space'
  if (e.code === 'Backquote') return '`'
  if (e.code === 'Escape') return 'Escape'
  if (e.code.startsWith('Key')) return e.code.slice(3)
  if (e.code.startsWith('Digit')) return e.code.slice(5)
  if (e.code.startsWith('Numpad')) return e.code.slice(6)
  if (e.key.length === 1) return e.key.toUpperCase()
  return e.key
}

export function comboFromKeyboardEvent(e: KeyboardEvent): string | null {
  const key = normalizeShortcutEventKey(e)
  if (modifierKeys.has(key)) return null

  const parts: string[] = []
  if (e.metaKey || e.ctrlKey) parts.push('CommandOrControl')
  if (e.shiftKey) parts.push('Shift')
  if (e.altKey) parts.push('Alt')
  parts.push(key)
  return parts.join('+')
}

export function formatShortcutCombo(combo: string): string {
  return combo.replace(/CommandOrControl/g, navigator.platform.includes('Mac') ? '⌘' : 'Ctrl')
    .replace(/Alt/g, navigator.platform.includes('Mac') ? '⌥' : 'Alt')
    .replace(/Shift/g, navigator.platform.includes('Mac') ? '⇧' : 'Shift')
    .replace(/Space/g, 'Space')
    .replace(/\+/g, ' + ')
}

function parseCombo(combo: string) {
  const parts = combo.split('+').map(p => p.trim())
  const ctrl = parts.includes('CommandOrControl')
  const shift = parts.includes('Shift')
  const alt = parts.includes('Alt')
  const key = parts.filter(p => !['CommandOrControl', 'Shift', 'Alt'].includes(p))[0] || ''
  return { ctrl, shift, alt, key }
}

export function shortcutDigitIndex(e: KeyboardEvent): number | null {
  const key = normalizeShortcutEventKey(e)
  if (/^[1-9]$/.test(key)) return Number(key) - 1
  return null
}

export function shortcutActionFromKeyboardEvent(shortcuts: ShortcutMap, e: KeyboardEvent): ShortcutAction | null {
  for (const action of Object.keys(shortcuts) as ShortcutAction[]) {
    const combo = parseCombo(shortcuts[action])
    const ctrlMatch = combo.ctrl ? (e.metaKey || e.ctrlKey) : (!e.metaKey && !e.ctrlKey)
    const shiftMatch = combo.shift ? e.shiftKey : !e.shiftKey
    const altMatch = combo.alt ? e.altKey : !e.altKey
    const keyMatch = normalizeShortcutEventKey(e) === combo.key
    if (ctrlMatch && shiftMatch && altMatch && keyMatch) return action
  }
  return null
}

export function useShortcuts(handlers: Partial<Record<ShortcutAction, () => void>>) {
  const settings = useSettingsStore()

  function handleKeydown(e: KeyboardEvent) {
    if (document.documentElement.hasAttribute('data-shortcut-capturing')) return

    // Don't trigger shortcuts when typing in inputs
    const target = e.target as HTMLElement | null
    const tag = target?.tagName
    if (
      tag === 'INPUT' ||
      tag === 'TEXTAREA' ||
      tag === 'SELECT' ||
      target?.isContentEditable ||
      target?.closest('[data-shortcut-scope="settings"]')
    ) return

    const action = shortcutActionFromKeyboardEvent(settings.shortcuts, e)
    const handler = action ? handlers[action] : undefined
    if (!handler) return

    e.preventDefault()
    e.stopPropagation()
    handler()
  }

  onMounted(() => window.addEventListener('keydown', handleKeydown, true))
  onUnmounted(() => window.removeEventListener('keydown', handleKeydown, true))
}
