import { defineConfig, presetUno, presetIcons } from 'unocss'

export default defineConfig({
  presets: [
    presetUno({ dark: 'class' }),
    presetIcons(),
  ],
  theme: {
    colors: {
      primary: 'var(--color-primary)',
      info: 'var(--color-info)',
      success: 'var(--color-success)',
      warning: 'var(--color-warning)',
      error: 'var(--color-error)',
      'bg-primary': 'var(--bg-primary)',
      'bg-secondary': 'var(--bg-secondary)',
      'bg-tertiary': 'var(--bg-tertiary)',
      'bg-hover': 'var(--bg-hover)',
      'text-primary': 'var(--text-primary)',
      'text-secondary': 'var(--text-secondary)',
      'border': 'var(--border-color)',
    },
  },
  shortcuts: {
    'flex-center': 'flex items-center justify-center',
    'flex-col-center': 'flex flex-col items-center justify-center',
    'panel-bg': 'bg-[var(--bg-secondary)] rounded-[var(--border-radius)]',
    'hover-overlay': 'transition-colors duration-150 hover:bg-[var(--hover-overlay)] hover:text-[var(--text-primary)]',
    'text-muted': 'text-[var(--text-secondary)]',
    'text-active': 'text-[var(--text-primary)]',
  },
  rules: [
    [/^text-size-(\d+)$/, ([, d]) => ({ 'font-size': `${d}px` })],
  ],
})
