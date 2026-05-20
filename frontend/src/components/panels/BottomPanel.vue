<script setup lang="ts">
import { useLayoutStore } from '../../stores/layout'
import MonitorPanel from '../monitor/MonitorPanel.vue'
import SFTPArea from '../sftp/SFTPArea.vue'

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
        <svg viewBox="0 0 16 16" fill="currentColor" width="12" height="12">
          <path d="M4.646 4.646a.5.5 0 01.708 0L8 7.293l2.646-2.647a.5.5 0 01.708.708L8.707 8l2.647 2.646a.5.5 0 01-.708.708L8 8.707l-2.646 2.647a.5.5 0 01-.708-.708L7.293 8 4.646 5.354a.5.5 0 010-.708z" />
        </svg>
      </button>
    </div>
    <div class="flex-1 min-h-0 overflow-auto">
      <MonitorPanel v-if="layout.activeBottomTool === 'monitor'" />
      <SFTPArea v-else-if="layout.activeBottomTool === 'sftp'" />
    </div>
  </div>
</template>
