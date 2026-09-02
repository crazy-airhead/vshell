import { defineStore } from 'pinia'
import { ref, computed } from 'vue'

export type ThemeMode = 'light' | 'dark'
export type LocaleCode = 'en' | 'zh-CN'

export interface ShortcutMap {
  newConnection: string
  closeTab: string
  toggleTheme: string
  toggleSidebar: string
  focusTerminal: string
  copy: string
  paste: string
}

function loadJSON<T>(key: string, fallback: T): T {
  try {
    const raw = localStorage.getItem(key)
    if (raw) return JSON.parse(raw) as T
  } catch {}
  return fallback
}

// Terminal convention: plain Ctrl+C (SIGINT) / Ctrl+V (quoted insert) must
// keep working, so on Windows/Linux copy/paste defaults follow the terminal
// standard Ctrl+Shift+C/V; macOS uses the native Cmd+C/Cmd+V instead.
const isMac = navigator.platform.includes('Mac')

const defaultShortcuts: ShortcutMap = {
  newConnection: 'CommandOrControl+Shift+N',
  closeTab: 'CommandOrControl+W',
  toggleTheme: 'CommandOrControl+Shift+T',
  toggleSidebar: 'CommandOrControl+B',
  focusTerminal: 'CommandOrControl+`',
  copy: isMac ? 'CommandOrControl+C' : 'CommandOrControl+Shift+C',
  paste: isMac ? 'CommandOrControl+V' : 'CommandOrControl+Shift+V',
}

export const useSettingsStore = defineStore('settings', () => {
  const themeMode = ref<ThemeMode>((localStorage.getItem('theme') as ThemeMode) || 'dark')
  const localeCode = ref<LocaleCode>((localStorage.getItem('locale') as LocaleCode) || 'zh-CN')
  const uiFontSize = ref<number>(loadJSON<number>('uiFontSize', 13))
  const uiFontFamily = ref<string>(localStorage.getItem('uiFontFamily') || 'system-ui')
  const accentColor = ref<string>(localStorage.getItem('accentColor') || '#0078d4')
  const terminalFontSize = ref<number>(loadJSON<number>('terminalFontSize', 14))
  const terminalFontFamily = ref<string>(localStorage.getItem('terminalFontFamily') || 'Menlo')
  const terminalColorScheme = ref<string>(localStorage.getItem('terminalColorScheme') || 'default')
  // Merge with defaults so shortcuts added in newer versions (e.g. copy/paste)
  // are present for users who saved an older shortcut map.
  const shortcuts = ref<ShortcutMap>({ ...defaultShortcuts, ...loadJSON<Partial<ShortcutMap>>('shortcuts', {}) })

  const isDark = computed(() => themeMode.value === 'dark')

  function setTheme(mode: ThemeMode) {
    themeMode.value = mode
    localStorage.setItem('theme', mode)
  }

  function toggleTheme() {
    setTheme(isDark.value ? 'light' : 'dark')
  }

  function setLocale(code: LocaleCode) {
    localeCode.value = code
    localStorage.setItem('locale', code)
  }

  function setUIFontSize(size: number) {
    uiFontSize.value = size
    localStorage.setItem('uiFontSize', JSON.stringify(size))
  }

  function setUIFontFamily(family: string) {
    uiFontFamily.value = family
    localStorage.setItem('uiFontFamily', family)
  }

  function setAccentColor(color: string) {
    accentColor.value = color
    localStorage.setItem('accentColor', color)
  }

  function setTerminalFontSize(size: number) {
    terminalFontSize.value = size
    localStorage.setItem('terminalFontSize', JSON.stringify(size))
  }

  function setTerminalFontFamily(family: string) {
    terminalFontFamily.value = family
    localStorage.setItem('terminalFontFamily', family)
  }

  function setTerminalColorScheme(scheme: string) {
    terminalColorScheme.value = scheme
    localStorage.setItem('terminalColorScheme', scheme)
  }

  function setShortcut(action: keyof ShortcutMap, combo: string) {
    shortcuts.value = { ...shortcuts.value, [action]: combo }
    localStorage.setItem('shortcuts', JSON.stringify(shortcuts.value))
  }

  function resetShortcuts() {
    shortcuts.value = { ...defaultShortcuts }
    localStorage.setItem('shortcuts', JSON.stringify(shortcuts.value))
  }

  return {
    themeMode, localeCode, uiFontSize, uiFontFamily, accentColor,
    terminalFontSize, terminalFontFamily, terminalColorScheme, shortcuts,
    isDark,
    setTheme, toggleTheme, setLocale,
    setUIFontSize, setUIFontFamily, setAccentColor,
    setTerminalFontSize, setTerminalFontFamily, setTerminalColorScheme,
    setShortcut, resetShortcuts,
  }
})
