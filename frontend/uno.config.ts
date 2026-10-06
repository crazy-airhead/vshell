import { defineConfig, presetUno } from 'unocss'

export default defineConfig({
  presets: [
    presetUno({ dark: ['class', '[data-theme="dark"]'] }),
  ],
  theme: {
    colors: {
      primary: 'var(--color-primary)',
      info: 'var(--color-info)',
      success: 'var(--color-success)',
      warning: 'var(--color-warning)',
      error: 'var(--color-error)',
      'bg-canvas': 'var(--bg-canvas)',
      'bg-island': 'var(--bg-island)',
    'bg-chrome': 'var(--bg-chrome)',
      'bg-component': 'var(--bg-component)',
      'bg-inverted': 'var(--bg-inverted)',
      selection: 'var(--selection)',
      'text-primary': 'var(--text-primary)',
      'text-secondary': 'var(--text-secondary)',
      'text-muted': 'var(--text-muted)',
      border: 'var(--border-color)',
    },
  },
  shortcuts: {
    'flex-center': 'flex items-center justify-center',
    'flex-col-center': 'flex flex-col items-center justify-center',
    'island': 'bg-[var(--bg-island)] rounded-[var(--island-arc)]',
    // 铬条（工具栏/左条/状态栏）：融画布色，与画布构成一体外框（同 GMark --bg-chrome）
    'island-bar': 'bg-[var(--bg-chrome)] rounded-[var(--bar-arc)]',
    'icon-btn': 'flex-center bg-transparent border-none cursor-pointer rounded-[var(--radius-m)] text-[var(--text-secondary)] transition-colors duration-150 hover:text-[var(--text-primary)] hover:bg-[var(--hover-overlay)] active:bg-[var(--pressed-overlay)]',
    'stripe-btn': 'w-10 h-10 flex-center bg-transparent border-none cursor-pointer rounded-[var(--radius-m)] text-[var(--text-secondary)] transition-colors duration-150 hover:text-[var(--text-primary)] hover:bg-[var(--hover-overlay)]',
    'hover-overlay': 'transition-colors duration-150 hover:bg-[var(--hover-overlay)] hover:text-[var(--text-primary)]',
    'text-muted': 'text-[var(--text-secondary)]',
    'text-active': 'text-[var(--text-primary)]',
  },
})
