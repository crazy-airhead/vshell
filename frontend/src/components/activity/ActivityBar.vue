<script setup lang="ts">
import { useLayoutStore } from '../../stores/layout'

const emit = defineEmits<{ (e: 'openSettings'): void }>()
const layout = useLayoutStore()
</script>

<template>
  <div class="w-[48px] shrink-0 flex flex-col items-center bg-[var(--bg-tertiary)] py-1">
    <div class="flex flex-col items-center gap-[2px]">
      <button
        class="w-10 h-10 flex-center bg-transparent border-none rounded-[var(--border-radius)] cursor-pointer text-[var(--text-secondary)] relative transition-colors duration-150 hover:text-[var(--text-primary)] hover:bg-[var(--hover-overlay)]"
        :class="{ '!text-[var(--color-primary)]': layout.activeSidebar === 'connections' }"
        @click="layout.setSidebar('connections')"
        title="Connections"
      >
        <svg class="w-5 h-5" viewBox="0 0 20 20" fill="none" stroke="currentColor" stroke-width="1.5">
          <rect x="3" y="3" width="14" height="14" rx="2" />
          <path d="M7 17V7l6 5-6 5z" />
        </svg>
        <span v-if="layout.activeSidebar === 'connections'" class="absolute left-0 top-2 bottom-2 w-[2px] bg-[var(--color-primary)] rounded-r-[2px]" />
      </button>
      <button
        class="w-10 h-10 flex-center bg-transparent border-none rounded-[var(--border-radius)] cursor-pointer text-[var(--text-secondary)] relative transition-colors duration-150 hover:text-[var(--text-primary)] hover:bg-[var(--hover-overlay)]"
        :class="{ '!text-[var(--color-primary)]': layout.activeSidebar === 'keys' }"
        @click="layout.setSidebar('keys')"
        title="SSH Keys"
      >
        <svg class="w-5 h-5" viewBox="0 0 20 20" fill="none" stroke="currentColor" stroke-width="1.5">
          <circle cx="7" cy="7" r="5" />
          <path d="M11 11l7 7" stroke-linecap="round" />
          <path d="M10.5 3.5h6M10.5 6.5h5M10.5 9.5h3" stroke-linecap="round" />
        </svg>
        <span v-if="layout.activeSidebar === 'keys'" class="absolute left-0 top-2 bottom-2 w-[2px] bg-[var(--color-primary)] rounded-r-[2px]" />
      </button>
      <button
        class="w-10 h-10 flex-center bg-transparent border-none rounded-[var(--border-radius)] cursor-pointer text-[var(--text-secondary)] relative transition-colors duration-150 hover:text-[var(--text-primary)] hover:bg-[var(--hover-overlay)]"
        :class="{ '!text-[var(--color-primary)]': layout.activeSidebar === 'ssh-config' }"
        @click="layout.setSidebar('ssh-config')"
        title="SSH Config"
      >
        <svg class="w-5 h-5" viewBox="0 0 20 20" fill="none" stroke="currentColor" stroke-width="1.5">
          <rect x="3" y="2" width="14" height="16" rx="2" />
          <path d="M7 6h6M7 9h6M7 12h4M7 15h3" stroke-linecap="round" />
        </svg>
        <span v-if="layout.activeSidebar === 'ssh-config'" class="absolute left-0 top-2 bottom-2 w-[2px] bg-[var(--color-primary)] rounded-r-[2px]" />
      </button>
    </div>

    <div class="flex-1" />

    <div class="flex flex-col items-center gap-[2px]">
      <button
        class="w-10 h-10 flex-center bg-transparent border-none rounded-[var(--border-radius)] cursor-pointer text-[var(--text-secondary)] relative transition-colors duration-150 hover:text-[var(--text-primary)] hover:bg-[var(--hover-overlay)]"
        :class="{ '!text-[var(--color-primary)]': layout.activeBottomTool === 'monitor' }"
        @click="layout.toggleBottomTool('monitor')"
        title="Monitor"
      >
        <svg class="w-5 h-5" viewBox="0 0 20 20" fill="none" stroke="currentColor" stroke-width="1.5">
          <rect x="2" y="2" width="7" height="8" rx="1" />
          <rect x="11" y="5" width="7" height="13" rx="1" />
          <rect x="2" y="13" width="7" height="5" rx="1" />
        </svg>
      </button>
      <button
        class="w-10 h-10 flex-center bg-transparent border-none rounded-[var(--border-radius)] cursor-pointer text-[var(--text-secondary)] relative transition-colors duration-150 hover:text-[var(--text-primary)] hover:bg-[var(--hover-overlay)]"
        :class="{ '!text-[var(--color-primary)]': layout.activeBottomTool === 'sftp' }"
        @click="layout.toggleBottomTool('sftp')"
        title="SFTP"
      >
        <svg class="w-5 h-5" viewBox="0 0 20 20" fill="none" stroke="currentColor" stroke-width="1.5">
          <path d="M3 5a1 1 0 011-1h4l2 2h6a1 1 0 011 1v7a1 1 0 01-1 1H4a1 1 0 01-1-1V5z" />
        </svg>
      </button>
      <button
        class="w-10 h-10 flex-center bg-transparent border-none rounded-[var(--border-radius)] cursor-pointer text-[var(--text-secondary)] transition-colors duration-150 hover:text-[var(--text-primary)] hover:bg-[var(--hover-overlay)]"
        @click="emit('openSettings')"
        title="Settings"
      >
        <svg class="w-5 h-5" viewBox="0 0 20 20" fill="none" stroke="currentColor" stroke-width="1.5">
          <path d="M10 12.5a2.5 2.5 0 100-5 2.5 2.5 0 000 5z" />
          <path d="M17.3 10c0-.35-.04-.69-.1-1.02l1.77-1.38-1.77-3.07-2.1.55a5.47 5.47 0 00-.88-.51l-.44-2.12h-3.56l-.44 2.12c-.32.13-.61.3-.88.51l-2.1-.55-1.77 3.07L6.8 8.98c-.06.33-.1.67-.1 1.02s.04.69.1 1.02l-1.77 1.38 1.77 3.07 2.1-.55c.27.21.56.38.88.51l.44 2.12h3.56l.44-2.12c.32-.13.61-.3.88-.51l2.1.55 1.77-3.07-1.77-1.38z" />
        </svg>
      </button>
    </div>
  </div>
</template>
