<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { NEmpty } from 'naive-ui'
import { useMonitorStore } from '../../stores/monitor'
import { useConnectionStore } from '../../stores/connection'
import { useTerminalStore } from '../../stores/terminal'

const { t } = useI18n()
const monitorStore = useMonitorStore()
const connectionStore = useConnectionStore()
const terminalStore = useTerminalStore()

const activeConnectionID = computed(() => {
  const tab = terminalStore.tabs.find(t => t.id === terminalStore.activeTabID)
  return tab?.connectionID ?? null
})

const connName = computed(() => {
  if (!activeConnectionID.value) return ''
  const conn = connectionStore.connections.find(c => c.id === activeConnectionID.value)
  return conn?.name || conn?.host || ''
})

const stats = computed(() => {
  if (!activeConnectionID.value) return null
  return monitorStore.getStats(activeConnectionID.value)
})

function formatUptime(seconds: number): string {
  if (seconds <= 0) return '-'
  const days = Math.floor(seconds / 86400)
  const hours = Math.floor((seconds % 86400) / 3600)
  const minutes = Math.floor((seconds % 3600) / 60)
  if (days > 0) return `${days}d ${hours}h`
  if (hours > 0) return `${hours}h ${minutes}m`
  return `${minutes}m`
}

const diskTotal = computed(() => {
  if (!stats.value || stats.value.disk_stats.length === 0) return null
  let total = 0, used = 0
  for (const d of stats.value.disk_stats) {
    total += d.total
    used += d.used
  }
  const pct = total > 0 ? (used / total) * 100 : 0
  return { total, used, pct }
})

const netTotal = computed(() => {
  if (!stats.value) return null
  const ifaces = stats.value.net_interfaces
  if (!ifaces || Object.keys(ifaces).length === 0) return null
  let rx = 0, tx = 0
  for (const nio of Object.values(ifaces)) {
    rx += nio.receive_kbps
    tx += nio.transmit_kbps
  }
  return { rx, tx }
})

function barColor(pct: number): string {
  if (pct < 60) return '#4caf50'
  if (pct < 85) return '#ff9800'
  return '#f44336'
}

function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(1024))
  return (bytes / Math.pow(1024, i)).toFixed(1) + ' ' + units[i]
}

function formatKbps(kbps: number): string {
  if (kbps < 1024) return kbps.toFixed(0) + ' KB/s'
  return (kbps / 1024).toFixed(1) + ' MB/s'
}
</script>

<template>
  <div class="monitor-panel">
    <div v-if="!activeConnectionID" class="monitor-empty">
      <NEmpty :description="t('monitor.noConnection')" size="small" />
    </div>
    <template v-else-if="stats">
      <div class="monitor-header">
        <span class="monitor-name">{{ connName }}</span>
      </div>
      <div class="monitor-stats">
        <!-- Uptime -->
        <div class="stat-row">
          <div class="stat-label">
            <span>{{ t('monitor.uptime') }}</span>
            <span class="stat-value">{{ formatUptime(stats.uptime_seconds) }}</span>
          </div>
        </div>

        <!-- CPU -->
        <div class="stat-row">
          <div class="stat-label">
            <span>{{ t('monitor.cpu') }}</span>
            <span class="stat-value">{{ stats.cpu_percent.toFixed(1) }}%</span>
          </div>
          <div class="stat-bar">
            <div class="stat-bar-fill" :style="{ width: stats.cpu_percent + '%', background: barColor(stats.cpu_percent) }"></div>
          </div>
        </div>

        <!-- Memory -->
        <div class="stat-row">
          <div class="stat-label">
            <span>{{ t('monitor.memory') }}</span>
            <span class="stat-value">{{ formatBytes(stats.mem_used * 1024) }} / {{ formatBytes(stats.mem_total * 1024) }}</span>
          </div>
          <div class="stat-bar">
            <div class="stat-bar-fill" :style="{ width: stats.mem_percent + '%', background: barColor(stats.mem_percent) }"></div>
          </div>
        </div>

        <!-- Disk (total) -->
        <div v-if="diskTotal" class="stat-row">
          <div class="stat-label">
            <span>{{ t('monitor.disk') }}</span>
            <span class="stat-value">{{ formatBytes(diskTotal.used) }} / {{ formatBytes(diskTotal.total) }}</span>
          </div>
          <div class="stat-bar">
            <div class="stat-bar-fill" :style="{ width: diskTotal.pct + '%', background: barColor(diskTotal.pct) }"></div>
          </div>
        </div>

        <!-- Network -->
        <div v-if="netTotal" class="stat-row">
          <div class="stat-label">
            <span>{{ t('monitor.network') }}</span>
            <span class="stat-value">&#8595;{{ formatKbps(netTotal.rx) }} &#8593;{{ formatKbps(netTotal.tx) }}</span>
          </div>
        </div>
      </div>
    </template>
    <div v-else class="monitor-empty">
      <NEmpty :description="t('monitor.waitingStats')" size="small" />
    </div>
  </div>
</template>

<style scoped>
.monitor-panel {
  display: flex;
  flex-direction: column;
  height: 100%;
  overflow-y: auto;
  background: var(--bg-secondary);
  color: var(--text-primary);
  font-size: var(--font-size-sm);
}

.monitor-empty {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
}

.monitor-header {
  padding: 8px 12px;
  border-bottom: 1px solid var(--border-color);
  flex-shrink: 0;
}

.monitor-name {
  font-weight: 600;
  font-size: var(--font-size-base);
}

.monitor-stats {
  padding: 8px 12px;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.stat-row {
  display: flex;
  flex-direction: column;
  gap: 3px;
}

.stat-label {
  display: flex;
  justify-content: space-between;
  align-items: center;
  color: var(--text-secondary);
  font-size: var(--font-size-sm);
}

.stat-value {
  color: var(--text-primary);
  font-family: monospace;
  font-size: var(--font-size-sm);
}

.stat-bar {
  height: 5px;
  background: var(--stat-bar-bg);
  border-radius: 3px;
  overflow: hidden;
}

.stat-bar-fill {
  height: 100%;
  border-radius: 3px;
  transition: width 0.5s ease;
}

</style>
