<script setup lang="ts">
import { computed, ref, watch, nextTick, onBeforeUnmount, onMounted, h } from 'vue'
import { useI18n } from 'vue-i18n'
import { NEmpty, NDataTable } from 'naive-ui'
import * as echarts from 'echarts'
import IconCopy from '~icons/lucide/copy'
import IconNetwork from '~icons/lucide/network'
import IconListTree from '~icons/lucide/list-tree'
import { useMonitorStore } from '../../stores/monitor'
import { useConnectionStore } from '../../stores/connection'
import { useTerminalStore } from '../../stores/terminal'
import { useLayoutStore } from '../../stores/layout'
import type { ProcessInfo, NetConnProcess, NetHistoryPoint, NetIO } from '../../types'
import type { DataTableColumns } from 'naive-ui'

interface NetIfaceRow {
  name: string
  receive_kbps: number
  transmit_kbps: number
  receive_bytes: number
  transmit_bytes: number
}

const { t } = useI18n()
const monitorStore = useMonitorStore()
const connectionStore = useConnectionStore()
const terminalStore = useTerminalStore()
const layoutStore = useLayoutStore()

const lastConnID = ref<string | null>(null)
type DetailType = 'cpu' | 'memory' | 'disk' | 'network' | 'processes' | null
const activeDetail = ref<DetailType>(null)

// Network tab state
const netTab = ref<'interface' | 'process'>('interface')
const selectedIface = ref<string | null>(null)
const prevSelectedIface = ref<string | null>(null)
const chartRef = ref<HTMLElement | null>(null)
let chartInstance: echarts.ECharts | null = null

// Dynamic table height via ResizeObserver
const netContentRef = ref<HTMLElement | null>(null)
const rightPanelRef = ref<HTMLElement | null>(null)
const netTableHeight = ref(300)
const processTableHeight = ref(400)
let resizeObserver: ResizeObserver | null = null

function recalcTableHeight() {
  if (netContentRef.value) {
    const h = netContentRef.value.clientHeight
    const chartH = netTab.value === 'interface' ? 142 : 0
    const gapH = 8
    netTableHeight.value = Math.max(h - chartH - gapH, 100)
  }
  if (rightPanelRef.value) {
    // Right panel padding py-2 = 8px * 2 = 16px
    processTableHeight.value = Math.max(rightPanelRef.value.clientHeight - 16, 100)
  }
}

onMounted(() => {
  resizeObserver = new ResizeObserver(() => recalcTableHeight())
})

watch(netContentRef, (el) => {
  if (resizeObserver) resizeObserver.disconnect()
  if (el) resizeObserver?.observe(el)
})

watch(rightPanelRef, (el) => {
  if (el) resizeObserver?.observe(el)
})

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

const netConns = computed(() => {
  if (!activeConnectionID.value) return []
  return monitorStore.getNetConns(activeConnectionID.value)
})

const netHistoryMap = computed(() => {
  if (!activeConnectionID.value) return null
  return monitorStore.getNetHistory(activeConnectionID.value)
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

const top5Processes = computed(() => {
  if (!stats.value) return []
  return stats.value.top_processes.slice(0, 5)
})

// Process table for NDataTable
const processColumns = computed<DataTableColumns<ProcessInfo>>(() => [
  {
    title: 'PID',
    key: 'pid',
    width: 72,
    sorter: (a, b) => a.pid - b.pid,
    render: (row) => h('span', { class: 'font-mono' }, String(row.pid)),
  },
  {
    title: 'CPU',
    key: 'cpu_percent',
    width: 72,
    sorter: (a, b) => a.cpu_percent - b.cpu_percent,
    render: (row) => {
      const color = barColor(row.cpu_percent)
      return h('span', { class: 'font-mono', style: { color } }, row.cpu_percent.toFixed(1))
    },
  },
  {
    title: t('monitor.memory'),
    key: 'mem_bytes',
    width: 94,
    sorter: (a, b) => a.mem_bytes - b.mem_bytes,
    render: (row) => h('span', { class: 'font-mono' }, formatBytes(row.mem_bytes)),
  },
  {
    title: t('monitor.user'),
    key: 'user',
    width: 76,
    sorter: (a, b) => a.user.localeCompare(b.user),
    render: (row) => h('span', {}, row.user),
  },
  {
    title: t('monitor.command'),
    key: 'command',
    sorter: (a, b) => (a.command || '').localeCompare(b.command || ''),
    ellipsis: { tooltip: true },
    render: (row) => {
      const text = row.command || row.name
      return h('div', { class: 'flex items-center gap-1' }, [
        h(IconCopy, {
          width: 12,
          height: 12,
          class: 'shrink-0 cursor-pointer text-[var(--text-secondary)] hover:text-[var(--text-primary)]',
          onClick: (e: Event) => { e.stopPropagation(); navigator.clipboard.writeText(text) },
        }),
        h('span', { class: 'truncate' }, text),
      ])
    },
  },
  {
    title: t('monitor.location'),
    key: 'exe_path',
    width: 200,
    sorter: (a, b) => (a.exe_path || '').localeCompare(b.exe_path || ''),
    ellipsis: { tooltip: true },
    render: (row) => {
      const text = row.exe_path || '-'
      return h('div', { class: 'flex items-center gap-1' }, [
        h(IconCopy, {
          width: 12,
          height: 12,
          class: 'shrink-0 cursor-pointer text-[var(--text-secondary)] hover:text-[var(--text-primary)]',
          onClick: (e: Event) => { e.stopPropagation(); navigator.clipboard.writeText(text) },
        }),
        h('span', { class: 'truncate' }, text),
      ])
    },
  },
])

const processData = computed(() => {
  if (!stats.value) return []
  return stats.value.top_processes
})

// Network interface table columns
const netIfaceColumns = computed<DataTableColumns<NetIfaceRow>>(() => [
  {
    title: t('monitor.interface'),
    key: 'name',
    sorter: (a, b) => a.name.localeCompare(b.name),
    defaultSortOrder: 'ascend',
    render: (row) => h('span', { class: 'font-mono cursor-pointer' }, row.name),
  },
  {
    title: t('monitor.receive'),
    key: 'receive_kbps',
    sorter: (a, b) => a.receive_kbps - b.receive_kbps,
    render: (row) => h('span', { class: 'font-mono' }, formatKbps(row.receive_kbps)),
  },
  {
    title: t('monitor.transmit'),
    key: 'transmit_kbps',
    sorter: (a, b) => a.transmit_kbps - b.transmit_kbps,
    render: (row) => h('span', { class: 'font-mono' }, formatKbps(row.transmit_kbps)),
  },
  {
    title: t('monitor.totalReceived'),
    key: 'receive_bytes',
    sorter: (a, b) => a.receive_bytes - b.receive_bytes,
    render: (row) => h('span', { class: 'font-mono text-[var(--text-secondary)]' }, formatBytes(row.receive_bytes)),
  },
  {
    title: t('monitor.totalSent'),
    key: 'transmit_bytes',
    sorter: (a, b) => a.transmit_bytes - b.transmit_bytes,
    render: (row) => h('span', { class: 'font-mono text-[var(--text-secondary)]' }, formatBytes(row.transmit_bytes)),
  },
])

const netIfaceData = computed<NetIfaceRow[]>(() => {
  if (!stats.value) return []
  return Object.entries(stats.value.net_interfaces).map(([name, io]) => ({
    name,
    receive_kbps: io.receive_kbps,
    transmit_kbps: io.transmit_kbps,
    receive_bytes: io.receive_bytes,
    transmit_bytes: io.transmit_bytes,
  }))
})

// Default selected interface: first physical nic
const firstPhysicalIface = computed(() => {
  if (!stats.value) return null
  const names = Object.keys(stats.value.net_interfaces).filter(n => !n.startsWith('lo'))
  return names.length > 0 ? names[0] : null
})

// Network process table columns
const netProcColumns = computed<DataTableColumns<NetConnProcess>>(() => [
  {
    title: 'PID',
    key: 'pid',
    sorter: (a, b) => a.pid - b.pid,
    render: (row) => h('span', { class: 'font-mono' }, String(row.pid)),
  },
  {
    title: t('monitor.processName'),
    key: 'name',
    sorter: (a, b) => a.name.localeCompare(b.name),
    ellipsis: { tooltip: true },
  },
  {
    title: t('monitor.listenAddr'),
    key: 'listen_addrs',
    sorter: (a, b) => (a.listen_addrs || []).join(',').localeCompare((b.listen_addrs || []).join(',')),
    render: (row) => h('span', { class: 'font-mono text-[var(--text-secondary)]' }, (row.listen_addrs || []).join(', ') || '-'),
  },
  {
    title: t('monitor.port'),
    key: 'ports',
    sorter: (a, b) => (a.ports || []).join(',').localeCompare((b.ports || []).join(',')),
    render: (row) => h('span', { class: 'font-mono text-[var(--text-secondary)]' }, (row.ports || []).join(', ') || '-'),
  },
  {
    title: t('monitor.connCount'),
    key: 'conn_count',
    sorter: (a, b) => a.conn_count - b.conn_count,
    render: (row) => h('span', { class: 'font-mono' }, String(row.conn_count)),
  },
])

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

function toggleDetail(type: DetailType) {
  activeDetail.value = type
  if (type === 'network') {
    selectedIface.value = firstPhysicalIface.value ?? null
  }
}

// Ensure selectedIface stays in sync with available interfaces
watch(firstPhysicalIface, (val) => {
  if (activeDetail.value === 'network' && netTab.value === 'interface' && val) {
    if (!selectedIface.value || !stats.value?.net_interfaces[selectedIface.value]) {
      selectedIface.value = val
    }
  }
})

const hasChartData = computed(() => {
  if (!selectedIface.value || !netHistoryMap.value) return false
  const points = netHistoryMap.value.get(selectedIface.value)
  return !!points && points.length >= 2
})

// ECharts trend chart
function renderChart() {
  if (!chartRef.value || !selectedIface.value || !netHistoryMap.value) return
  if (!hasChartData.value) {
    if (chartInstance) {
      chartInstance.clear()
    }
    return
  }

  const points = netHistoryMap.value.get(selectedIface.value)!

  if (!chartInstance) {
    chartInstance = echarts.init(chartRef.value)
  }

  const times = points.map(p => {
    const d = new Date(p.ts)
    return `${d.getMinutes().toString().padStart(2, '0')}:${d.getSeconds().toString().padStart(2, '0')}`
  })

  // Calculate rates between consecutive points
  const rxRates: number[] = []
  const txRates: number[] = []
  for (let i = 1; i < points.length; i++) {
    const dt = (points[i].ts - points[i - 1].ts) / 1000
    if (dt > 0) {
      rxRates.push(+((points[i].rx - points[i - 1].rx) / dt / 1024).toFixed(1))
      txRates.push(+((points[i].tx - points[i - 1].tx) / dt / 1024).toFixed(1))
    } else {
      rxRates.push(0)
      txRates.push(0)
    }
  }
  const chartTimes = times.slice(1)

  chartInstance.setOption({
    grid: { top: 20, right: 12, bottom: 24, left: 50 },
    textStyle: { fontSize: 10, color: 'var(--text-secondary)' },
    xAxis: { type: 'category', data: chartTimes, boundaryGap: false },
    yAxis: { type: 'value', axisLabel: { formatter: '{value} KB/s' } },
    tooltip: { trigger: 'axis', textStyle: { fontSize: 11 } },
    series: [
      { name: '↓ RX', type: 'line', data: rxRates, smooth: true, symbol: 'none', lineStyle: { width: 1.5 }, itemStyle: { color: '#4fc3f7' } },
      { name: '↑ TX', type: 'line', data: txRates, smooth: true, symbol: 'none', lineStyle: { width: 1.5 }, itemStyle: { color: '#ff8a65' } },
    ],
  })
}

watch([selectedIface, netHistoryMap], () => {
  nextTick(() => renderChart())
})

// Save/restore selectedIface when switching net tabs
watch(netTab, (tab) => {
  if (tab === 'process') {
    prevSelectedIface.value = selectedIface.value
    selectedIface.value = null
    if (chartInstance) {
      chartInstance.dispose()
      chartInstance = null
    }
  } else {
    selectedIface.value = prevSelectedIface.value ?? firstPhysicalIface.value ?? null
  }
  // Recalculate table height after tab switch (chart area changes)
  nextTick(() => recalcTableHeight())
})

watch(activeDetail, (val) => {
  nextTick(() => recalcTableHeight())
  if (val !== 'network') {
    if (chartInstance) {
      chartInstance.dispose()
      chartInstance = null
    }
  } else {
    // Re-enter network: ensure chart renders after DOM is ready
    nextTick(() => renderChart())
  }
})

onBeforeUnmount(() => {
  if (chartInstance) {
    chartInstance.dispose()
    chartInstance = null
  }
  if (resizeObserver) {
    resizeObserver.disconnect()
    resizeObserver = null
  }
})

// Swap percent
const swapPercent = computed(() => {
  if (!stats.value || stats.value.swap_total === 0) return 0
  return (stats.value.swap_used / stats.value.swap_total) * 100
})
</script>

<template>
  <div class="flex flex-col h-full bg-[var(--bg-secondary)] text-[var(--text-primary)] text-[var(--font-size-sm)]">
    <div v-if="!activeConnectionID" class="flex-1 flex-center">
      <NEmpty :description="t('monitor.noConnection')" size="small" />
    </div>
    <template v-else-if="stats">
      <!-- Header: connection name -->
      <div class="px-3 py-2 shrink-0">
        <span class="font-semibold text-[var(--font-size-base)]">{{ connName }}</span>
      </div>
      <div class="flex-1 min-h-0 flex">
        <!-- ===== Left Panel ===== -->
        <div class="w-[280px] shrink-0 min-w-0 px-3 py-2 flex flex-col gap-2 overflow-y-auto border-r border-[var(--border-color)]">
          <!-- Host info -->
          <div v-if="stats.hostname || stats.os || stats.ip_addresses.length" class="flex flex-wrap gap-x-3 gap-y-[2px] text-[var(--text-secondary)]">
            <span v-if="stats.hostname" class="font-mono">{{ stats.hostname }}</span>
            <span v-if="stats.os" class="font-mono">{{ stats.os }}</span>
            <span v-for="ip in stats.ip_addresses" :key="ip" class="font-mono">{{ ip }}</span>
          </div>

          <!-- Uptime -->
          <div class="flex flex-col gap-[3px]">
            <div class="flex justify-between items-center text-[var(--text-secondary)]">
              <span>{{ t('monitor.uptime') }}</span>
              <span class="font-mono">{{ formatUptime(stats.uptime_seconds) }}</span>
            </div>
          </div>

          <!-- CPU -->
          <div
            class="flex flex-col gap-[3px] cursor-pointer rounded px-2 py-1 -mx-2 transition-colors duration-150"
            :class="activeDetail === 'cpu' ? 'bg-[var(--hover-overlay)]' : 'hover:bg-[var(--hover-overlay)]'"
            @click="toggleDetail('cpu')"
          >
            <div class="flex justify-between items-center text-[var(--text-secondary)]">
              <span>{{ t('monitor.cpu') }}</span>
              <span class="font-mono">{{ stats.cpu_percent.toFixed(1) }}%</span>
            </div>
            <div class="h-[5px] bg-[var(--stat-bar-bg)] rounded-[3px] overflow-hidden">
              <div class="h-full rounded-[3px] transition-[width] duration-500 ease-out" :style="{ width: stats.cpu_percent + '%', background: barColor(stats.cpu_percent) }"></div>
            </div>
          </div>

          <!-- Memory -->
          <div
            class="flex flex-col gap-[3px] cursor-pointer rounded px-2 py-1 -mx-2 transition-colors duration-150"
            :class="activeDetail === 'memory' ? 'bg-[var(--hover-overlay)]' : 'hover:bg-[var(--hover-overlay)]'"
            @click="toggleDetail('memory')"
          >
            <div class="flex justify-between items-center text-[var(--text-secondary)]">
              <span>{{ t('monitor.memory') }}</span>
              <span class="font-mono">{{ formatBytes(stats.mem_used * 1024) }} / {{ formatBytes(stats.mem_total * 1024) }}</span>
            </div>
            <div class="h-[5px] bg-[var(--stat-bar-bg)] rounded-[3px] overflow-hidden">
              <div class="h-full rounded-[3px] transition-[width] duration-500 ease-out" :style="{ width: stats.mem_percent + '%', background: barColor(stats.mem_percent) }"></div>
            </div>
          </div>

          <!-- Processes -->
          <div
            class="mt-1 border-t border-[var(--border-color)] pt-2 cursor-pointer rounded px-2 -mx-2 transition-colors duration-150"
            :class="activeDetail === 'processes' ? 'bg-[var(--hover-overlay)]' : 'hover:bg-[var(--hover-overlay)]'"
            @click="toggleDetail('processes')"
          >
            <div class="text-[var(--text-secondary)] font-semibold">{{ t('monitor.processes') }}</div>
            <div v-if="top5Processes.length > 0" class="flex flex-col gap-[2px] mt-1">
              <div v-for="p in top5Processes" :key="p.pid" class="flex justify-between items-center">
                <span class="truncate text-[var(--text-primary)]">{{ p.name }}</span>
                <span class="flex gap-2 shrink-0 ml-2 font-mono">
                  <span :style="{ color: barColor(p.cpu_percent) }">{{ p.cpu_percent.toFixed(1) }}%</span>
                  <span class="text-[var(--text-secondary)]">{{ p.mem_percent.toFixed(1) }}%</span>
                </span>
              </div>
            </div>
          </div>

          <!-- Disk -->
          <div
            v-if="diskTotal"
            class="flex flex-col gap-[3px] cursor-pointer rounded px-2 py-1 -mx-2 transition-colors duration-150"
            :class="activeDetail === 'disk' ? 'bg-[var(--hover-overlay)]' : 'hover:bg-[var(--hover-overlay)]'"
            @click="toggleDetail('disk')"
          >
            <div class="flex justify-between items-center text-[var(--text-secondary)]">
              <span>{{ t('monitor.disk') }}</span>
              <span class="font-mono">{{ formatBytes(diskTotal.used) }} / {{ formatBytes(diskTotal.total) }}</span>
            </div>
            <div class="h-[5px] bg-[var(--stat-bar-bg)] rounded-[3px] overflow-hidden">
              <div class="h-full rounded-[3px] transition-[width] duration-500 ease-out" :style="{ width: diskTotal.pct + '%', background: barColor(diskTotal.pct) }"></div>
            </div>
          </div>

          <!-- Network -->
          <div
            v-if="netTotal"
            class="flex flex-col gap-[3px] cursor-pointer rounded px-2 py-1 -mx-2 transition-colors duration-150"
            :class="activeDetail === 'network' ? 'bg-[var(--hover-overlay)]' : 'hover:bg-[var(--hover-overlay)]'"
            @click="toggleDetail('network')"
          >
            <div class="flex justify-between items-center text-[var(--text-secondary)]">
              <span>{{ t('monitor.network') }}</span>
              <span class="font-mono">&#8595;{{ formatKbps(netTotal.rx) }} &#8593;{{ formatKbps(netTotal.tx) }}</span>
            </div>
          </div>
        </div>

        <!-- ===== Right Panel ===== -->
        <div ref="rightPanelRef" class="flex-1 min-w-0 px-3 py-2" :class="activeDetail === 'processes' ? 'overflow-hidden' : 'overflow-y-auto'">
          <!-- No selection -->
          <template v-if="!activeDetail">
            <div class="h-full flex-center">
              <NEmpty :description="t('monitor.selectHint')" size="small" />
            </div>
          </template>

          <!-- CPU Detail -->
          <template v-else-if="activeDetail === 'cpu'">
            <div class="flex flex-col gap-2">
              <div class="text-[var(--text-secondary)] font-semibold">{{ t('monitor.cpuCores') }}</div>
              <div v-if="stats.cpu_cores.length > 0" class="flex flex-col gap-2">
                <div v-for="c in stats.cpu_cores" :key="c.core" class="flex flex-col gap-[3px]">
                  <div class="flex justify-between items-center text-[var(--text-secondary)]">
                    <span>CPU {{ c.core }}</span>
                    <span class="font-mono">{{ c.percent.toFixed(1) }}%</span>
                  </div>
                  <div class="h-[5px] bg-[var(--stat-bar-bg)] rounded-[3px] overflow-hidden">
                    <div class="h-full rounded-[3px] transition-[width] duration-500 ease-out" :style="{ width: c.percent + '%', background: barColor(c.percent) }"></div>
                  </div>
                </div>
              </div>
              <NEmpty v-else :description="t('monitor.waitingStats')" size="small" />
            </div>
          </template>

          <!-- Memory Detail -->
          <template v-else-if="activeDetail === 'memory'">
            <div class="flex flex-col gap-3">
              <!-- Physical Memory -->
              <div class="flex flex-col gap-1">
                <div class="text-[var(--text-secondary)] font-semibold">{{ t('monitor.physicalMemory') }}</div>
                <div class="flex justify-between items-center">
                  <span class="text-[var(--text-secondary)]">{{ t('monitor.used') }}</span>
                  <span class="font-mono">{{ formatBytes(stats.mem_used * 1024) }} / {{ formatBytes(stats.mem_total * 1024) }}</span>
                </div>
                <div class="h-[6px] bg-[var(--stat-bar-bg)] rounded-[3px] overflow-hidden">
                  <div class="h-full rounded-[3px] transition-[width] duration-500 ease-out" :style="{ width: stats.mem_percent + '%', background: barColor(stats.mem_percent) }"></div>
                </div>
              </div>
              <!-- Swap -->
              <div class="flex flex-col gap-1">
                <div class="text-[var(--text-secondary)] font-semibold">{{ t('monitor.swapMemory') }}</div>
                <template v-if="stats.swap_total > 0">
                  <div class="flex justify-between items-center">
                    <span class="text-[var(--text-secondary)]">{{ t('monitor.used') }}</span>
                    <span class="font-mono">{{ formatBytes(stats.swap_used * 1024) }} / {{ formatBytes(stats.swap_total * 1024) }}</span>
                  </div>
                  <div class="h-[6px] bg-[var(--stat-bar-bg)] rounded-[3px] overflow-hidden">
                    <div class="h-full rounded-[3px] transition-[width] duration-500 ease-out" :style="{ width: swapPercent + '%', background: barColor(swapPercent) }"></div>
                  </div>
                </template>
                <div v-else class="text-[var(--text-secondary)]">{{ t('monitor.noSwap') }}</div>
              </div>
            </div>
          </template>

          <!-- Disk Detail -->
          <template v-else-if="activeDetail === 'disk'">
            <div class="flex flex-col gap-2">
              <div v-for="d in stats.disk_stats" :key="d.mount_point" class="flex flex-col gap-1 py-2 border-b border-[var(--border-color)] last:border-b-0">
                <div class="flex justify-between items-center">
                  <span class="font-mono text-[var(--text-primary)]">{{ d.mount_point }}</span>
                  <span class="font-mono text-[var(--text-secondary)]">{{ d.percent.toFixed(1) }}%</span>
                </div>
                <div class="h-[5px] bg-[var(--stat-bar-bg)] rounded-[3px] overflow-hidden">
                  <div class="h-full rounded-[3px] transition-[width] duration-500 ease-out" :style="{ width: d.percent + '%', background: barColor(d.percent) }"></div>
                </div>
                <div class="flex justify-between text-[var(--text-secondary)]">
                  <span>{{ d.device }}</span>
                  <span>{{ formatBytes(d.used) }} / {{ formatBytes(d.total) }}</span>
                </div>
              </div>
              <NEmpty v-if="stats.disk_stats.length === 0" :description="t('monitor.waitingStats')" size="small" />
            </div>
          </template>

          <!-- Network Detail -->
          <template v-else-if="activeDetail === 'network'">
            <div class="flex h-full overflow-hidden">
              <!-- Left: content -->
              <div ref="netContentRef" class="flex-1 min-w-0 flex flex-col gap-2">
                <!-- Trend chart (top) — always show when on interface tab -->
                <div v-if="netTab === 'interface'" class="shrink-0">
                  <div class="text-[var(--text-secondary)] mb-1">{{ selectedIface || '-' }} - {{ t('monitor.trend') }}</div>
                  <div class="w-full h-[120px] relative">
                    <div ref="chartRef" class="w-full h-full"></div>
                    <div v-if="!hasChartData" class="absolute inset-0 flex-center text-[var(--text-secondary)] text-xs">{{ t('monitor.noConnection') }}</div>
                  </div>
                </div>
                <!-- Table -->
                <NDataTable
                  v-if="netTab === 'interface'"
                  :columns="netIfaceColumns"
                  :data="netIfaceData"
                  :row-props="(row: NetIfaceRow) => ({ style: 'font-size: 11px; cursor: pointer', onClick: () => { selectedIface = row.name } })"
                  :pagination="false"
                  :bordered="false"
                  :single-line="false"
                  size="small"
                  :row-class-name="(row: NetIfaceRow) => selectedIface === row.name ? 'bg-[var(--hover-overlay)]' : ''"
                  :max-height="netTableHeight"
                />
                <NDataTable
                  v-else
                  :columns="netProcColumns"
                  :data="netConns"
                  :row-props="() => ({ style: 'font-size: 11px' })"
                  :pagination="false"
                  :bordered="false"
                  :single-line="false"
                  size="small"
                  striped
                  :max-height="netTableHeight"
                />
              </div>
              <!-- Right: icon tabs (ActivityBar style) -->
              <div class="shrink-0 flex flex-col items-center border-l border-[var(--border-color)] ml-2">
                <button
                  class="w-9 h-9 flex-center bg-transparent border-none rounded-[var(--border-radius)] cursor-pointer text-[var(--text-secondary)] relative transition-colors duration-150 hover:text-[var(--text-primary)] hover:bg-[var(--hover-overlay)]"
                  :class="{ '!text-[var(--color-primary)]': netTab === 'interface' }"
                  @click="netTab = 'interface'"
                  :title="t('monitor.byInterface')"
                >
                  <IconNetwork :width="18" :height="18" />
                  <span v-if="netTab === 'interface'" class="absolute right-0 top-2 bottom-2 w-[2px] bg-[var(--color-primary)] rounded-l-[2px]" />
                </button>
                <button
                  class="w-9 h-9 flex-center bg-transparent border-none rounded-[var(--border-radius)] cursor-pointer text-[var(--text-secondary)] relative transition-colors duration-150 hover:text-[var(--text-primary)] hover:bg-[var(--hover-overlay)]"
                  :class="{ '!text-[var(--color-primary)]': netTab === 'process' }"
                  @click="netTab = 'process'"
                  :title="t('monitor.byProcess')"
                >
                  <IconListTree :width="18" :height="18" />
                  <span v-if="netTab === 'process'" class="absolute right-0 top-2 bottom-2 w-[2px] bg-[var(--color-primary)] rounded-l-[2px]" />
                </button>
              </div>
            </div>
          </template>

          <!-- Process Detail -->
          <template v-else-if="activeDetail === 'processes'">
            <NDataTable
              :columns="processColumns"
              :data="processData"
              :row-props="() => ({ style: 'font-size: 11px' })"
              :pagination="false"
              :bordered="false"
              :single-line="false"
              size="small"
              striped
              :max-height="processTableHeight"
            />
          </template>
        </div>
      </div>
    </template>
    <div v-else class="flex-1 flex-center">
      <NEmpty :description="t('monitor.waitingStats')" size="small" />
    </div>
  </div>
</template>
