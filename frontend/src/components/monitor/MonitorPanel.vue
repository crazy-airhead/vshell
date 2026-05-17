<script setup lang="ts">
import { computed } from 'vue'
import { NEmpty } from 'naive-ui'
import { useMonitorStore } from '../../stores/monitor'
import { useConnectionStore } from '../../stores/connection'
import { useTerminalStore } from '../../stores/terminal'

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

const uptime = computed(() => {
  if (!activeConnectionID.value) return ''
  const since = monitorStore.getUptime(activeConnectionID.value)
  if (!since) return ''
  const diff = Date.now() - since
  const seconds = Math.floor(diff / 1000)
  if (seconds < 60) return `${seconds}s`
  const minutes = Math.floor(seconds / 60)
  if (minutes < 60) return `${minutes}m ${seconds % 60}s`
  const hours = Math.floor(minutes / 60)
  return `${hours}h ${minutes % 60}m`
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
      <NEmpty description="No active connection" size="small" />
    </div>
    <template v-else-if="stats">
      <div class="monitor-header">
        <span class="monitor-name">{{ connName }}</span>
        <span class="monitor-uptime">{{ uptime }}</span>
      </div>
      <div class="monitor-stats">
        <!-- CPU -->
        <div class="stat-row">
          <div class="stat-label">
            <span>CPU</span>
            <span class="stat-value">{{ stats.cpu_percent.toFixed(1) }}%</span>
          </div>
          <div class="stat-bar">
            <div class="stat-bar-fill" :style="{ width: stats.cpu_percent + '%', background: barColor(stats.cpu_percent) }"></div>
          </div>
        </div>

        <!-- Memory -->
        <div class="stat-row">
          <div class="stat-label">
            <span>Memory</span>
            <span class="stat-value">{{ formatBytes(stats.mem_used * 1024) }} / {{ formatBytes(stats.mem_total * 1024) }}</span>
          </div>
          <div class="stat-bar">
            <div class="stat-bar-fill" :style="{ width: stats.mem_percent + '%', background: barColor(stats.mem_percent) }"></div>
          </div>
        </div>

        <!-- Load Average -->
        <div class="stat-row">
          <div class="stat-label">
            <span>Load Avg</span>
            <span class="stat-value">{{ stats.load_avg[0].toFixed(2) }} {{ stats.load_avg[1].toFixed(2) }} {{ stats.load_avg[2].toFixed(2) }}</span>
          </div>
        </div>

        <!-- Network -->
        <div v-if="Object.keys(stats.net_interfaces).length > 0" class="stat-row">
          <div class="stat-label"><span>Network</span></div>
          <div v-for="(nio, iface) in stats.net_interfaces" :key="iface" class="net-row">
            <span class="net-iface">{{ iface }}</span>
            <span class="net-io">&#8595;{{ formatKbps(nio.receive_kbps) }} &#8593;{{ formatKbps(nio.transmit_kbps) }}</span>
          </div>
        </div>

        <!-- Disk -->
        <div v-if="stats.disk_stats.length > 0" class="stat-row">
          <div class="stat-label"><span>Disk</span></div>
          <div v-for="disk in stats.disk_stats" :key="disk.mount_point" class="disk-row">
            <div class="disk-info">
              <span class="disk-mount">{{ disk.mount_point }}</span>
              <span class="stat-value">{{ formatBytes(disk.used) }} / {{ formatBytes(disk.total) }}</span>
            </div>
            <div class="stat-bar">
              <div class="stat-bar-fill" :style="{ width: disk.percent + '%', background: barColor(disk.percent) }"></div>
            </div>
          </div>
        </div>
      </div>
    </template>
    <div v-else class="monitor-empty">
      <NEmpty description="Waiting for stats..." size="small" />
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
  font-size: 12px;
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
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-shrink: 0;
}

.monitor-name {
  font-weight: 600;
  font-size: 13px;
}

.monitor-uptime {
  color: var(--text-secondary);
  font-size: 11px;
}

.monitor-stats {
  padding: 8px 12px;
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.stat-row {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.stat-label {
  display: flex;
  justify-content: space-between;
  align-items: center;
  color: var(--text-secondary);
  font-size: 11px;
}

.stat-value {
  color: var(--text-primary);
  font-family: monospace;
  font-size: 11px;
}

.stat-bar {
  height: 6px;
  background: #3a3a3a;
  border-radius: 3px;
  overflow: hidden;
}

.stat-bar-fill {
  height: 100%;
  border-radius: 3px;
  transition: width 0.5s ease;
}

.net-row {
  display: flex;
  justify-content: space-between;
  padding: 1px 0;
  font-size: 11px;
}

.net-iface {
  color: var(--text-secondary);
}

.net-io {
  color: var(--text-primary);
  font-family: monospace;
  font-size: 10px;
}

.disk-row {
  display: flex;
  flex-direction: column;
  gap: 3px;
  padding: 2px 0;
}

.disk-info {
  display: flex;
  justify-content: space-between;
  font-size: 11px;
}

.disk-mount {
  color: var(--text-secondary);
}
</style>
