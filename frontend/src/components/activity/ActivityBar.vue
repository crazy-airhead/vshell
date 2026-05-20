<script setup lang="ts">
import { useLayoutStore } from '../../stores/layout'
import IconServer from '~icons/lucide/server'
import IconKeyRound from '~icons/lucide/key-round'
import IconFileText from '~icons/lucide/file-text'
import IconActivity from '~icons/lucide/activity'
import IconFolder from '~icons/lucide/folder'
import IconSettings from '~icons/lucide/settings'

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
        <IconServer :width="20" :height="20" />
        <span v-if="layout.activeSidebar === 'connections'" class="absolute left-0 top-2 bottom-2 w-[2px] bg-[var(--color-primary)] rounded-r-[2px]" />
      </button>
      <button
        class="w-10 h-10 flex-center bg-transparent border-none rounded-[var(--border-radius)] cursor-pointer text-[var(--text-secondary)] relative transition-colors duration-150 hover:text-[var(--text-primary)] hover:bg-[var(--hover-overlay)]"
        :class="{ '!text-[var(--color-primary)]': layout.activeSidebar === 'keys' }"
        @click="layout.setSidebar('keys')"
        title="SSH Keys"
      >
        <IconKeyRound :width="20" :height="20" />
        <span v-if="layout.activeSidebar === 'keys'" class="absolute left-0 top-2 bottom-2 w-[2px] bg-[var(--color-primary)] rounded-r-[2px]" />
      </button>
      <button
        class="w-10 h-10 flex-center bg-transparent border-none rounded-[var(--border-radius)] cursor-pointer text-[var(--text-secondary)] relative transition-colors duration-150 hover:text-[var(--text-primary)] hover:bg-[var(--hover-overlay)]"
        :class="{ '!text-[var(--color-primary)]': layout.activeSidebar === 'ssh-config' }"
        @click="layout.setSidebar('ssh-config')"
        title="SSH Config"
      >
        <IconFileText :width="20" :height="20" />
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
        <IconActivity :width="20" :height="20" />
      </button>
      <button
        class="w-10 h-10 flex-center bg-transparent border-none rounded-[var(--border-radius)] cursor-pointer text-[var(--text-secondary)] relative transition-colors duration-150 hover:text-[var(--text-primary)] hover:bg-[var(--hover-overlay)]"
        :class="{ '!text-[var(--color-primary)]': layout.activeBottomTool === 'sftp' }"
        @click="layout.toggleBottomTool('sftp')"
        title="SFTP"
      >
        <IconFolder :width="20" :height="20" />
      </button>
      <button
        class="w-10 h-10 flex-center bg-transparent border-none rounded-[var(--border-radius)] cursor-pointer text-[var(--text-secondary)] transition-colors duration-150 hover:text-[var(--text-primary)] hover:bg-[var(--hover-overlay)]"
        @click="emit('openSettings')"
        title="Settings"
      >
        <IconSettings :width="20" :height="20" />
      </button>
    </div>
  </div>
</template>
