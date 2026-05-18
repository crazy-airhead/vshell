<script setup lang="ts">
import { useLayoutStore } from '../../stores/layout'

const emit = defineEmits<{ (e: 'openSettings'): void }>()
const layout = useLayoutStore()
</script>

<template>
  <div class="activity-bar">
    <div class="activity-top">
      <button
        class="activity-btn"
        :class="{ active: layout.activeSidebar === 'connections' }"
        @click="layout.setSidebar('connections')"
        title="Connections"
      >
        <svg viewBox="0 0 20 20" fill="none" stroke="currentColor" stroke-width="1.5">
          <rect x="3" y="3" width="14" height="14" rx="2" />
          <path d="M7 17V7l6 5-6 5z" />
        </svg>
      </button>
      <button
        class="activity-btn"
        :class="{ active: layout.activeSidebar === 'keys' }"
        @click="layout.setSidebar('keys')"
        title="SSH Keys"
      >
        <svg viewBox="0 0 20 20" fill="none" stroke="currentColor" stroke-width="1.5">
          <circle cx="7" cy="7" r="5" />
          <path d="M11 11l7 7" stroke-linecap="round" />
          <path d="M10.5 3.5h6M10.5 6.5h5M10.5 9.5h3" stroke-linecap="round" />
        </svg>
      </button>
      <button
        class="activity-btn"
        :class="{ active: layout.activeSidebar === 'ssh-config' }"
        @click="layout.setSidebar('ssh-config')"
        title="SSH Config"
      >
        <svg viewBox="0 0 20 20" fill="none" stroke="currentColor" stroke-width="1.5">
          <rect x="3" y="2" width="14" height="16" rx="2" />
          <path d="M7 6h6M7 9h6M7 12h4M7 15h3" stroke-linecap="round" />
        </svg>
      </button>
    </div>

    <div class="activity-spacer" />

    <div class="activity-bottom">
      <button
        class="activity-btn"
        :class="{ active: layout.activeBottomTool === 'monitor' }"
        @click="layout.toggleBottomTool('monitor')"
        title="Monitor"
      >
        <svg viewBox="0 0 20 20" fill="none" stroke="currentColor" stroke-width="1.5">
          <rect x="2" y="2" width="7" height="8" rx="1" />
          <rect x="11" y="5" width="7" height="13" rx="1" />
          <rect x="2" y="13" width="7" height="5" rx="1" />
        </svg>
      </button>
      <button
        class="activity-btn"
        :class="{ active: layout.activeBottomTool === 'sftp' }"
        @click="layout.toggleBottomTool('sftp')"
        title="SFTP"
      >
        <svg viewBox="0 0 20 20" fill="none" stroke="currentColor" stroke-width="1.5">
          <path d="M3 5a1 1 0 011-1h4l2 2h6a1 1 0 011 1v7a1 1 0 01-1 1H4a1 1 0 01-1-1V5z" />
        </svg>
      </button>
      <button
        class="activity-btn"
        @click="emit('openSettings')"
        title="Settings"
      >
        <svg viewBox="0 0 20 20" fill="none" stroke="currentColor" stroke-width="1.5">
          <circle cx="10" cy="10" r="3" />
          <path d="M10 2v2M10 16v2M2 10h2M16 10h2M4.93 4.93l1.41 1.41M13.66 13.66l1.41 1.41M4.93 15.07l1.41-1.41M13.66 6.34l1.41-1.41" stroke-linecap="round" />
        </svg>
      </button>
    </div>
  </div>
</template>

<style scoped>
.activity-bar {
  width: 48px;
  flex-shrink: 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  background: var(--bg-secondary);
  border-right: 1px solid var(--border-color);
  padding: 4px 0;
}

.activity-top,
.activity-bottom {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 2px;
}

.activity-spacer {
  flex: 1;
}

.activity-btn {
  width: 40px;
  height: 40px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: none;
  border: none;
  border-radius: 6px;
  cursor: pointer;
  color: var(--text-secondary);
  position: relative;
  transition: color 0.15s, background 0.15s;
}

.activity-btn svg {
  width: 20px;
  height: 20px;
}

.activity-btn:hover {
  color: var(--text-primary);
  background: var(--hover-overlay);
}

.activity-btn.active {
  color: var(--accent-color);
}

.activity-btn.active::before {
  content: '';
  position: absolute;
  left: 0;
  top: 8px;
  bottom: 8px;
  width: 2px;
  background: var(--accent-color);
  border-radius: 0 2px 2px 0;
}
</style>
