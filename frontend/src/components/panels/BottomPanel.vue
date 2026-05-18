<script setup lang="ts">
import { useLayoutStore } from '../../stores/layout'
import MonitorPanel from '../monitor/MonitorPanel.vue'
import SFTPArea from '../sftp/SFTPArea.vue'

const layout = useLayoutStore()
</script>

<template>
  <div class="bottom-panel">
    <div class="pane-header">
      <span class="pane-label">{{ layout.activeBottomTool === 'monitor' ? 'Monitor' : 'SFTP' }}</span>
      <button class="pane-close-btn" @click="layout.activeBottomTool = null">
        <svg viewBox="0 0 16 16" fill="currentColor" width="12" height="12">
          <path d="M4.646 4.646a.5.5 0 01.708 0L8 7.293l2.646-2.647a.5.5 0 01.708.708L8.707 8l2.647 2.646a.5.5 0 01-.708.708L8 8.707l-2.646 2.647a.5.5 0 01-.708-.708L7.293 8 4.646 5.354a.5.5 0 010-.708z" />
        </svg>
      </button>
    </div>
    <div class="pane-body">
      <MonitorPanel v-if="layout.activeBottomTool === 'monitor'" />
      <SFTPArea v-else-if="layout.activeBottomTool === 'sftp'" />
    </div>
  </div>
</template>

<style scoped>
.bottom-panel {
  display: flex;
  flex-direction: column;
  height: 100%;
  overflow: hidden;
  background: var(--bg-secondary);
  border-radius: 6px;
}

.pane-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 4px 10px;
  flex-shrink: 0;
  border-bottom: 1px solid var(--border-color);
}

.pane-label {
  font-size: var(--font-size-sm);
  font-weight: 600;
  color: var(--text-secondary);
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.pane-close-btn {
  background: none;
  border: none;
  color: var(--text-secondary);
  cursor: pointer;
  padding: 2px 4px;
  border-radius: 3px;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: color 0.15s, background 0.15s;
}

.pane-close-btn:hover {
  color: var(--text-primary);
  background: var(--hover-overlay);
}

.pane-body {
  flex: 1;
  min-height: 0;
  overflow: auto;
}
</style>
