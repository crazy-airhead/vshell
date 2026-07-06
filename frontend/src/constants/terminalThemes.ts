export interface TerminalTheme {
  key: string
  name: string
  background: string
  foreground: string
  cursor: string
  selectionBackground: string
  palette: string[]
  black: string
  red: string
  green: string
  yellow: string
  blue: string
  magenta: string
  cyan: string
  white: string
  brightBlack: string
  brightRed: string
  brightGreen: string
  brightYellow: string
  brightBlue: string
  brightMagenta: string
  brightCyan: string
  brightWhite: string
}

type ThemeInput = Omit<TerminalTheme, 'palette'>

function theme(input: ThemeInput): TerminalTheme {
  return {
    ...input,
    palette: [input.green, input.cyan, input.blue, input.yellow, input.magenta],
  }
}

export const terminalThemes: TerminalTheme[] = [
  theme({
    key: 'default',
    name: 'Default',
    background: '#1e1e1e',
    foreground: '#cccccc',
    cursor: '#ffffff',
    selectionBackground: '#264f78',
    black: '#000000',
    red: '#cd3131',
    green: '#0dbc79',
    yellow: '#e5e510',
    blue: '#2472c8',
    magenta: '#bc3fbc',
    cyan: '#11a8cd',
    white: '#e5e5e5',
    brightBlack: '#666666',
    brightRed: '#f14c4c',
    brightGreen: '#23d18b',
    brightYellow: '#f5f543',
    brightBlue: '#3b8eea',
    brightMagenta: '#d670d6',
    brightCyan: '#29b8db',
    brightWhite: '#ffffff',
  }),
  theme({
    key: 'termius-dark',
    name: 'Termius Dark',
    background: '#111827',
    foreground: '#e5eef7',
    cursor: '#22d3a6',
    selectionBackground: '#26364c',
    black: '#0d1117',
    red: '#ff5f57',
    green: '#00d084',
    yellow: '#f4d35e',
    blue: '#4ea1ff',
    magenta: '#a78bfa',
    cyan: '#22d3ee',
    white: '#dce7f3',
    brightBlack: '#6b7280',
    brightRed: '#ff7b72',
    brightGreen: '#34f5a5',
    brightYellow: '#ffe07a',
    brightBlue: '#77b7ff',
    brightMagenta: '#c4b5fd',
    brightCyan: '#67e8f9',
    brightWhite: '#ffffff',
  }),
  theme({
    key: 'termius-light',
    name: 'Termius Light',
    background: '#f6f8fb',
    foreground: '#1f2937',
    cursor: '#2563eb',
    selectionBackground: '#dbeafe',
    black: '#111827',
    red: '#dc2626',
    green: '#059669',
    yellow: '#b7791f',
    blue: '#2563eb',
    magenta: '#7c3aed',
    cyan: '#0891b2',
    white: '#e5e7eb',
    brightBlack: '#6b7280',
    brightRed: '#ef4444',
    brightGreen: '#10b981',
    brightYellow: '#d97706',
    brightBlue: '#3b82f6',
    brightMagenta: '#8b5cf6',
    brightCyan: '#06b6d4',
    brightWhite: '#ffffff',
  }),
  theme({
    key: 'nord-dark',
    name: 'Nord Dark',
    background: '#2e3440',
    foreground: '#d8dee9',
    cursor: '#88c0d0',
    selectionBackground: '#434c5e',
    black: '#3b4252',
    red: '#bf616a',
    green: '#a3be8c',
    yellow: '#ebcb8b',
    blue: '#81a1c1',
    magenta: '#b48ead',
    cyan: '#88c0d0',
    white: '#e5e9f0',
    brightBlack: '#4c566a',
    brightRed: '#bf616a',
    brightGreen: '#a3be8c',
    brightYellow: '#ebcb8b',
    brightBlue: '#81a1c1',
    brightMagenta: '#b48ead',
    brightCyan: '#8fbcbb',
    brightWhite: '#eceff4',
  }),
  theme({
    key: 'flexoki-dark',
    name: 'Flexoki Dark',
    background: '#100f0f',
    foreground: '#cecdc3',
    cursor: '#cecdc3',
    selectionBackground: '#343331',
    black: '#100f0f',
    red: '#af3029',
    green: '#66800b',
    yellow: '#ad8301',
    blue: '#205ea6',
    magenta: '#a02f6f',
    cyan: '#24837b',
    white: '#cecdc3',
    brightBlack: '#575653',
    brightRed: '#d14d41',
    brightGreen: '#879a39',
    brightYellow: '#d0a215',
    brightBlue: '#4385be',
    brightMagenta: '#ce5d97',
    brightCyan: '#3aa99f',
    brightWhite: '#fffcf0',
  }),
  theme({
    key: 'flexoki-light',
    name: 'Flexoki Light',
    background: '#fffcf0',
    foreground: '#100f0f',
    cursor: '#100f0f',
    selectionBackground: '#e6e4d9',
    black: '#100f0f',
    red: '#af3029',
    green: '#66800b',
    yellow: '#ad8301',
    blue: '#205ea6',
    magenta: '#a02f6f',
    cyan: '#24837b',
    white: '#cecdc3',
    brightBlack: '#878580',
    brightRed: '#d14d41',
    brightGreen: '#879a39',
    brightYellow: '#d0a215',
    brightBlue: '#4385be',
    brightMagenta: '#ce5d97',
    brightCyan: '#3aa99f',
    brightWhite: '#fffcf0',
  }),
  theme({
    key: 'kanagawa-wave',
    name: 'Kanagawa Wave',
    background: '#1f1f28',
    foreground: '#dcd7ba',
    cursor: '#c8c093',
    selectionBackground: '#2d4f67',
    black: '#090618',
    red: '#c34043',
    green: '#76946a',
    yellow: '#c0a36e',
    blue: '#7e9cd8',
    magenta: '#957fb8',
    cyan: '#6a9589',
    white: '#c8c093',
    brightBlack: '#727169',
    brightRed: '#e82424',
    brightGreen: '#98bb6c',
    brightYellow: '#e6c384',
    brightBlue: '#7fb4ca',
    brightMagenta: '#938aa9',
    brightCyan: '#7aa89f',
    brightWhite: '#dcd7ba',
  }),
  theme({
    key: 'kanagawa-dragon',
    name: 'Kanagawa Dragon',
    background: '#181616',
    foreground: '#c5c9c5',
    cursor: '#c5c9c5',
    selectionBackground: '#2d2a2e',
    black: '#0d0c0c',
    red: '#c4746e',
    green: '#8a9a7b',
    yellow: '#c4b28a',
    blue: '#8ba4b0',
    magenta: '#a292a3',
    cyan: '#8ea4a2',
    white: '#c8c093',
    brightBlack: '#625e5a',
    brightRed: '#e46876',
    brightGreen: '#87a987',
    brightYellow: '#e6c384',
    brightBlue: '#7fb4ca',
    brightMagenta: '#938aa9',
    brightCyan: '#7aa89f',
    brightWhite: '#dcd7ba',
  }),
  theme({
    key: 'kanagawa-lotus',
    name: 'Kanagawa Lotus',
    background: '#f2ecbc',
    foreground: '#545464',
    cursor: '#545464',
    selectionBackground: '#d5cea3',
    black: '#1f1f28',
    red: '#c84053',
    green: '#6f894e',
    yellow: '#77713f',
    blue: '#4d699b',
    magenta: '#b35b79',
    cyan: '#597b75',
    white: '#545464',
    brightBlack: '#8a8980',
    brightRed: '#d7474b',
    brightGreen: '#6e915f',
    brightYellow: '#836f4a',
    brightBlue: '#6693bf',
    brightMagenta: '#624c83',
    brightCyan: '#5e857a',
    brightWhite: '#43436c',
  }),
  theme({
    key: 'hacker-blue',
    name: 'Hacker Blue',
    background: '#07111f',
    foreground: '#d7f4ff',
    cursor: '#00c8ff',
    selectionBackground: '#123452',
    black: '#03101c',
    red: '#ff4d6d',
    green: '#00d8ff',
    yellow: '#5eead4',
    blue: '#38bdf8',
    magenta: '#818cf8',
    cyan: '#22d3ee',
    white: '#d7f4ff',
    brightBlack: '#35617d',
    brightRed: '#fb7185',
    brightGreen: '#67e8f9',
    brightYellow: '#99f6e4',
    brightBlue: '#7dd3fc',
    brightMagenta: '#a5b4fc',
    brightCyan: '#a5f3fc',
    brightWhite: '#ffffff',
  }),
  theme({
    key: 'hacker-green',
    name: 'Hacker Green',
    background: '#031307',
    foreground: '#c9ffd8',
    cursor: '#22ff55',
    selectionBackground: '#0f3d1f',
    black: '#020803',
    red: '#ff4d4d',
    green: '#00ff41',
    yellow: '#aaff00',
    blue: '#00c2ff',
    magenta: '#00ff95',
    cyan: '#4dffb8',
    white: '#d8ffe2',
    brightBlack: '#28613a',
    brightRed: '#ff7070',
    brightGreen: '#5cff78',
    brightYellow: '#c4ff4d',
    brightBlue: '#4dd2ff',
    brightMagenta: '#5cffb3',
    brightCyan: '#8affd2',
    brightWhite: '#ffffff',
  }),
  theme({
    key: 'hacker-red',
    name: 'Hacker Red',
    background: '#190509',
    foreground: '#ffd6dd',
    cursor: '#ff1744',
    selectionBackground: '#4a1019',
    black: '#0b0204',
    red: '#ff1744',
    green: '#ff6b81',
    yellow: '#ffb3c1',
    blue: '#ef4444',
    magenta: '#f43f5e',
    cyan: '#fb7185',
    white: '#ffe4e9',
    brightBlack: '#71313c',
    brightRed: '#ff4d6d',
    brightGreen: '#ff8fa3',
    brightYellow: '#ffc2cd',
    brightBlue: '#f87171',
    brightMagenta: '#fb7185',
    brightCyan: '#fda4af',
    brightWhite: '#ffffff',
  }),
  theme({
    key: 'everforest-dark',
    name: 'Everforest Dark',
    background: '#2d353b',
    foreground: '#d3c6aa',
    cursor: '#d3c6aa',
    selectionBackground: '#475258',
    black: '#475258',
    red: '#e67e80',
    green: '#a7c080',
    yellow: '#dbbc7f',
    blue: '#7fbbb3',
    magenta: '#d699b6',
    cyan: '#83c092',
    white: '#d3c6aa',
    brightBlack: '#859289',
    brightRed: '#e67e80',
    brightGreen: '#a7c080',
    brightYellow: '#dbbc7f',
    brightBlue: '#7fbbb3',
    brightMagenta: '#d699b6',
    brightCyan: '#83c092',
    brightWhite: '#fff9e8',
  }),
  theme({
    key: 'everforest-light',
    name: 'Everforest Light',
    background: '#fdf6e3',
    foreground: '#5c6a72',
    cursor: '#5c6a72',
    selectionBackground: '#e6e2cc',
    black: '#5c6a72',
    red: '#f85552',
    green: '#8da101',
    yellow: '#dfa000',
    blue: '#3a94c5',
    magenta: '#df69ba',
    cyan: '#35a77c',
    white: '#e6e2cc',
    brightBlack: '#939f91',
    brightRed: '#f85552',
    brightGreen: '#8da101',
    brightYellow: '#dfa000',
    brightBlue: '#3a94c5',
    brightMagenta: '#df69ba',
    brightCyan: '#35a77c',
    brightWhite: '#fff9e8',
  }),
  theme({
    key: 'night-owl',
    name: 'Night Owl',
    background: '#011627',
    foreground: '#d6deeb',
    cursor: '#80a4c2',
    selectionBackground: '#1d3b53',
    black: '#011627',
    red: '#ef5350',
    green: '#22da6e',
    yellow: '#c5e478',
    blue: '#82aaff',
    magenta: '#c792ea',
    cyan: '#21c7a8',
    white: '#ffffff',
    brightBlack: '#575656',
    brightRed: '#ef5350',
    brightGreen: '#22da6e',
    brightYellow: '#ffeb95',
    brightBlue: '#82aaff',
    brightMagenta: '#c792ea',
    brightCyan: '#7fdbca',
    brightWhite: '#ffffff',
  }),
  theme({
    key: 'light-owl',
    name: 'Light Owl',
    background: '#fbfbfb',
    foreground: '#403f53',
    cursor: '#403f53',
    selectionBackground: '#d6deeb',
    black: '#011627',
    red: '#d3423e',
    green: '#2aa298',
    yellow: '#daaa01',
    blue: '#4876d6',
    magenta: '#994cc3',
    cyan: '#08916a',
    white: '#f0f0f0',
    brightBlack: '#989fb1',
    brightRed: '#de3d3b',
    brightGreen: '#49d0c5',
    brightYellow: '#e0af02',
    brightBlue: '#5ca7e4',
    brightMagenta: '#c967e6',
    brightCyan: '#00c990',
    brightWhite: '#ffffff',
  }),
  theme({
    key: 'aura',
    name: 'Aura',
    background: '#15141b',
    foreground: '#edecee',
    cursor: '#a277ff',
    selectionBackground: '#3d375e',
    black: '#110f18',
    red: '#ff6767',
    green: '#61ffca',
    yellow: '#ffca85',
    blue: '#a277ff',
    magenta: '#a277ff',
    cyan: '#61ffca',
    white: '#edecee',
    brightBlack: '#6d6d6d',
    brightRed: '#ff8f8f',
    brightGreen: '#8bffd9',
    brightYellow: '#ffd8a6',
    brightBlue: '#bda2ff',
    brightMagenta: '#bda2ff',
    brightCyan: '#8bffd9',
    brightWhite: '#ffffff',
  }),
  theme({
    key: 'rose-pine',
    name: 'Rose Pine',
    background: '#191724',
    foreground: '#e0def4',
    cursor: '#c4a7e7',
    selectionBackground: '#403d52',
    black: '#26233a',
    red: '#eb6f92',
    green: '#31748f',
    yellow: '#f6c177',
    blue: '#9ccfd8',
    magenta: '#c4a7e7',
    cyan: '#ebbcba',
    white: '#e0def4',
    brightBlack: '#6e6a86',
    brightRed: '#eb6f92',
    brightGreen: '#31748f',
    brightYellow: '#f6c177',
    brightBlue: '#9ccfd8',
    brightMagenta: '#c4a7e7',
    brightCyan: '#ebbcba',
    brightWhite: '#e0def4',
  }),
]

const legacyThemes = new Map<string, TerminalTheme>([
  ['default-dark', terminalThemes.find(t => t.key === 'default')!],
  ['default-light', terminalThemes.find(t => t.key === 'termius-light')!],
  ['solarized-dark', theme({
    key: 'solarized-dark',
    name: 'Solarized Dark',
    background: '#002b36',
    foreground: '#839496',
    cursor: '#93a1a1',
    selectionBackground: '#073642',
    black: '#073642',
    red: '#dc322f',
    green: '#859900',
    yellow: '#b58900',
    blue: '#268bd2',
    magenta: '#d33682',
    cyan: '#2aa198',
    white: '#eee8d5',
    brightBlack: '#002b36',
    brightRed: '#cb4b16',
    brightGreen: '#586e75',
    brightYellow: '#657b83',
    brightBlue: '#839496',
    brightMagenta: '#6c71c4',
    brightCyan: '#93a1a1',
    brightWhite: '#fdf6e3',
  })],
  ['solarized-light', theme({
    key: 'solarized-light',
    name: 'Solarized Light',
    background: '#fdf6e3',
    foreground: '#657b83',
    cursor: '#586e75',
    selectionBackground: '#eee8d5',
    black: '#073642',
    red: '#dc322f',
    green: '#859900',
    yellow: '#b58900',
    blue: '#268bd2',
    magenta: '#d33682',
    cyan: '#2aa198',
    white: '#eee8d5',
    brightBlack: '#002b36',
    brightRed: '#cb4b16',
    brightGreen: '#586e75',
    brightYellow: '#657b83',
    brightBlue: '#839496',
    brightMagenta: '#6c71c4',
    brightCyan: '#93a1a1',
    brightWhite: '#fdf6e3',
  })],
  ['dracula', theme({
    key: 'dracula',
    name: 'Dracula',
    background: '#282a36',
    foreground: '#f8f8f2',
    cursor: '#f8f8f2',
    selectionBackground: '#44475a',
    black: '#21222c',
    red: '#ff5555',
    green: '#50fa7b',
    yellow: '#f1fa8c',
    blue: '#bd93f9',
    magenta: '#ff79c6',
    cyan: '#8be9fd',
    white: '#f8f8f2',
    brightBlack: '#6272a4',
    brightRed: '#ff6e6e',
    brightGreen: '#69ff94',
    brightYellow: '#ffffa5',
    brightBlue: '#d6acff',
    brightMagenta: '#ff92df',
    brightCyan: '#a4ffff',
    brightWhite: '#ffffff',
  })],
  ['monokai', theme({
    key: 'monokai',
    name: 'Monokai',
    background: '#272822',
    foreground: '#f8f8f2',
    cursor: '#f8f8f0',
    selectionBackground: '#49483e',
    black: '#272822',
    red: '#f92672',
    green: '#a6e22e',
    yellow: '#f4bf75',
    blue: '#66d9ef',
    magenta: '#ae81ff',
    cyan: '#a1efe4',
    white: '#f8f8f2',
    brightBlack: '#75715e',
    brightRed: '#f92672',
    brightGreen: '#a6e22e',
    brightYellow: '#f4bf75',
    brightBlue: '#66d9ef',
    brightMagenta: '#ae81ff',
    brightCyan: '#a1efe4',
    brightWhite: '#f9f8f5',
  })],
  ['one-dark', theme({
    key: 'one-dark',
    name: 'One Dark',
    background: '#282c34',
    foreground: '#abb2bf',
    cursor: '#528bff',
    selectionBackground: '#3e4451',
    black: '#282c34',
    red: '#e06c75',
    green: '#98c379',
    yellow: '#e5c07b',
    blue: '#61afef',
    magenta: '#c678dd',
    cyan: '#56b6c2',
    white: '#abb2bf',
    brightBlack: '#5c6370',
    brightRed: '#e06c75',
    brightGreen: '#98c379',
    brightYellow: '#e5c07b',
    brightBlue: '#61afef',
    brightMagenta: '#c678dd',
    brightCyan: '#56b6c2',
    brightWhite: '#ffffff',
  })],
])

export const terminalThemeMap = new Map(terminalThemes.map(item => [item.key, item]))

export function resolveTerminalTheme(key: string, isDark: boolean): TerminalTheme {
  if (key === 'default') {
    return terminalThemeMap.get(isDark ? 'termius-dark' : 'termius-light')!
  }
  return terminalThemeMap.get(key) || legacyThemes.get(key) || terminalThemeMap.get('termius-dark')!
}

interface RGB {
  r: number
  g: number
  b: number
}

export interface AppThemeVars {
  isDark: boolean
  colorPrimary: string
  colorInfo: string
  colorSuccess: string
  colorWarning: string
  colorError: string
  bgPrimary: string
  bgSecondary: string
  bgTertiary: string
  bgHover: string
  textPrimary: string
  textSecondary: string
  borderColor: string
  shadowHeader: string
  shadowSider: string
  shadowTab: string
  shadowElevated: string
  hoverOverlay: string
  hoverOverlayStrong: string
  actionHoverBg: string
  deleteHoverColor: string
  deleteHoverBg: string
  statBarBg: string
}

function hexToRgb(hex: string): RGB {
  const value = hex.replace('#', '').trim()
  const normalized = value.length === 3
    ? value.split('').map(char => char + char).join('')
    : value.padEnd(6, '0').slice(0, 6)
  const parsed = Number.parseInt(normalized, 16)
  return {
    r: (parsed >> 16) & 255,
    g: (parsed >> 8) & 255,
    b: parsed & 255,
  }
}

function rgbToHex({ r, g, b }: RGB): string {
  const toHex = (value: number) => Math.round(Math.max(0, Math.min(255, value)))
    .toString(16)
    .padStart(2, '0')
  return `#${toHex(r)}${toHex(g)}${toHex(b)}`
}

function mix(color: string, target: string, amount: number): string {
  const a = hexToRgb(color)
  const b = hexToRgb(target)
  return rgbToHex({
    r: a.r + (b.r - a.r) * amount,
    g: a.g + (b.g - a.g) * amount,
    b: a.b + (b.b - a.b) * amount,
  })
}

function relativeLuminance(color: string): number {
  const { r, g, b } = hexToRgb(color)
  const channel = (value: number) => {
    const c = value / 255
    return c <= 0.03928 ? c / 12.92 : ((c + 0.055) / 1.055) ** 2.4
  }
  return 0.2126 * channel(r) + 0.7152 * channel(g) + 0.0722 * channel(b)
}

export function isTerminalThemeDark(theme: TerminalTheme): boolean {
  return relativeLuminance(theme.background) < 0.48
}

export function createAppThemeVars(theme: TerminalTheme): AppThemeVars {
  const isDark = isTerminalThemeDark(theme)
  const primary = theme.blue || theme.cyan || theme.cursor

  if (isDark) {
    return {
      isDark,
      colorPrimary: primary,
      colorInfo: theme.cyan,
      colorSuccess: theme.green,
      colorWarning: theme.yellow,
      colorError: theme.red,
      bgPrimary: mix(theme.background, '#000000', 0.18),
      bgSecondary: theme.background,
      bgTertiary: mix(theme.background, theme.foreground, 0.08),
      bgHover: mix(theme.background, theme.foreground, 0.14),
      textPrimary: theme.foreground,
      textSecondary: mix(theme.foreground, theme.background, 0.36),
      borderColor: mix(theme.background, theme.foreground, 0.16),
      shadowHeader: '0 1px 2px rgba(0, 0, 0, 0.3)',
      shadowSider: '2px 0 8px 0 rgba(0, 0, 0, 0.25)',
      shadowTab: '0 1px 2px rgba(0, 0, 0, 0.3)',
      shadowElevated: '0 4px 12px rgba(0, 0, 0, 0.4)',
      hoverOverlay: 'rgba(255, 255, 255, 0.06)',
      hoverOverlayStrong: 'rgba(255, 255, 255, 0.04)',
      actionHoverBg: `${primary}26`,
      deleteHoverColor: theme.red,
      deleteHoverBg: `${theme.red}26`,
      statBarBg: mix(theme.background, theme.foreground, 0.13),
    }
  }

  return {
    isDark,
    colorPrimary: primary,
    colorInfo: theme.cyan,
    colorSuccess: theme.green,
    colorWarning: theme.yellow,
    colorError: theme.red,
    bgPrimary: mix(theme.background, '#000000', 0.04),
    bgSecondary: theme.background,
    bgTertiary: mix(theme.background, '#000000', 0.09),
    bgHover: mix(theme.background, '#000000', 0.14),
    textPrimary: theme.foreground,
    textSecondary: mix(theme.foreground, theme.background, 0.38),
    borderColor: mix(theme.background, '#000000', 0.18),
    shadowHeader: '0 1px 2px rgb(0, 21, 41, 0.08)',
    shadowSider: '2px 0 8px 0 rgb(29, 35, 41, 0.05)',
    shadowTab: '0 1px 2px rgb(0, 21, 41, 0.08)',
    shadowElevated: '0 4px 12px rgba(0, 0, 0, 0.08)',
    hoverOverlay: 'rgba(0, 0, 0, 0.04)',
    hoverOverlayStrong: 'rgba(0, 0, 0, 0.03)',
    actionHoverBg: `${primary}1a`,
    deleteHoverColor: theme.red,
    deleteHoverBg: `${theme.red}1a`,
    statBarBg: mix(theme.background, '#000000', 0.12),
  }
}
