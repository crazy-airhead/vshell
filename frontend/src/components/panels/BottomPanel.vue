<script setup lang="ts">
import { useLayoutStore } from '../../stores/layout'
import MonitorPanel from '../monitor/MonitorPanel.vue'
import SFTPArea from '../sftp/SFTPArea.vue'
import IconX from '~icons/lucide/x'

const layout = useLayoutStore()
</script>

<template>
  <div class="flex flex-col h-full overflow-hidden bg-[var(--bg-secondary)]">
    <div class="flex items-center justify-between px-[10px] py-1 shrink-0 bg-[var(--bg-tertiary)]">
      <span class="text-[var(--font-size-sm)] font-semibold text-[var(--text-secondary)] uppercase tracking-[0.5px]">{{ layout.activeBottomTool === 'monitor' ? 'Monitor' : 'SFTP' }}</span>
      <button
        class="bg-transparent border-none text-[var(--text-secondary)] cursor-pointer px-1 py-[2px] rounded-[3px] flex-center transition-colors duration-150 hover:text-[var(--text-primary)] hover:bg-[var(--hover-overlay)]"
        @click="layout.activeBottomTool = null"
      >
        <IconX :width="14" :height="14" />
      </button>
    </div>
    <div class="flex-1 min-h-0 overflow-auto">
      <MonitorPanel v-if="layout.activeBottomTool === 'monitor'" />
      <SFTPArea v-else-if="layout.activeBottomTool === 'sftp'" />
    </div>
  </div>
</template>
