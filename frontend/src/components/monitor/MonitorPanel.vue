<script setup lang="ts">
import { computed, watch, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { NEmpty } from 'naive-ui'
import { useMonitorStore } from '../../stores/monitor'
import { useConnectionStore } from '../../stores/connection'
import { useTerminalStore } from '../../stores/terminal'
import { useLayoutStore } from '../../stores/layout'

const { t } = useI18n()
const monitorStore = useMonitorStore()
const connectionStore = useConnectionStore()
const terminalStore = useTerminalStore()
const layoutStore = useLayoutStore()

const lastConnID = ref<string | null>(null)

const activeConnectionID = computed(() => {
  const tab = terminalStore.tabs.find(t => t.id === terminalStore.activeTabID)
  const connID = tab?.connectionID ?? null
  if (connID) lastConnID.value = connID
  const resolved = connID || lastConnID.value
  if (resolved && !terminalStore.tabs.some(t => t.connectionID === resolved)) {
    lastConnID.value = null
    if (layoutStore.activeBottomTool === 'monitor') {
      layoutStore.activeBottomTool = null
    }
    return null
  }
  return resolved
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
  if (pct < 60) return 'var(--color-success)'
  if (pct < 85) return 'var(--color-warning)'
  return 'var(--color-error)'
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
  <div class="flex flex-col h-full overflow-y-auto bg-[var(--bg-secondary)] text-[var(--text-primary)] text-[var(--font-size-sm)]">
    <div v-if="!activeConnectionID" class="flex-1 flex-center">
      <NEmpty :description="t('monitor.noConnection')" size="small" />
    </div>
    <template v-else-if="stats">
      <div class="px-3 py-2 shrink-0">
        <span class="font-semibold text-[var(--font-size-base)]">{{ connName }}</span>
      </div>
      <div class="px-3 py-2 flex flex-col gap-2">
        <!-- Uptime -->
        <div class="flex flex-col gap-[3px]">
          <div class="flex justify-between items-center text-[var(--text-secondary)] text-[var(--font-size-sm)]">
            <span>{{ t('monitor.uptime') }}</span>
            <span class="text-[var(--text-primary)] font-mono text-[var(--font-size-sm)]">{{ formatUptime(stats.uptime_seconds) }}</span>
          </div>
        </div>

        <!-- CPU -->
        <div class="flex flex-col gap-[3px]">
          <div class="flex justify-between items-center text-[var(--text-secondary)] text-[var(--font-size-sm)]">
            <span>{{ t('monitor.cpu') }}</span>
            <span class="text-[var(--text-primary)] font-mono text-[var(--font-size-sm)]">{{ stats.cpu_percent.toFixed(1) }}%</span>
          </div>
          <div class="h-[5px] bg-[var(--stat-bar-bg)] rounded-[3px] overflow-hidden">
            <div class="h-full rounded-[3px] transition-[width] duration-500 ease-out" :style="{ width: stats.cpu_percent + '%', background: barColor(stats.cpu_percent) }"></div>
          </div>
        </div>

        <!-- Memory -->
        <div class="flex flex-col gap-[3px]">
          <div class="flex justify-between items-center text-[var(--text-secondary)] text-[var(--font-size-sm)]">
            <span>{{ t('monitor.memory') }}</span>
            <span class="text-[var(--text-primary)] font-mono text-[var(--font-size-sm)]">{{ formatBytes(stats.mem_used * 1024) }} / {{ formatBytes(stats.mem_total * 1024) }}</span>
          </div>
          <div class="h-[5px] bg-[var(--stat-bar-bg)] rounded-[3px] overflow-hidden">
            <div class="h-full rounded-[3px] transition-[width] duration-500 ease-out" :style="{ width: stats.mem_percent + '%', background: barColor(stats.mem_percent) }"></div>
          </div>
        </div>

        <!-- Disk (total) -->
        <div v-if="diskTotal" class="flex flex-col gap-[3px]">
          <div class="flex justify-between items-center text-[var(--text-secondary)] text-[var(--font-size-sm)]">
            <span>{{ t('monitor.disk') }}</span>
            <span class="text-[var(--text-primary)] font-mono text-[var(--font-size-sm)]">{{ formatBytes(diskTotal.used) }} / {{ formatBytes(diskTotal.total) }}</span>
          </div>
          <div class="h-[5px] bg-[var(--stat-bar-bg)] rounded-[3px] overflow-hidden">
            <div class="h-full rounded-[3px] transition-[width] duration-500 ease-out" :style="{ width: diskTotal.pct + '%', background: barColor(diskTotal.pct) }"></div>
          </div>
        </div>

        <!-- Network -->
        <div v-if="netTotal" class="flex flex-col gap-[3px]">
          <div class="flex justify-between items-center text-[var(--text-secondary)] text-[var(--font-size-sm)]">
            <span>{{ t('monitor.network') }}</span>
            <span class="text-[var(--text-primary)] font-mono text-[var(--font-size-sm)]">&#8595;{{ formatKbps(netTotal.rx) }} &#8593;{{ formatKbps(netTotal.tx) }}</span>
          </div>
        </div>
      </div>
    </template>
    <div v-else class="flex-1 flex-center">
      <NEmpty :description="t('monitor.waitingStats')" size="small" />
    </div>
  </div>
</template>
