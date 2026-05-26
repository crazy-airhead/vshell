import { defineStore } from 'pinia'
import { reactive } from 'vue'
import { Events } from '@wailsio/runtime'
import { StartMonitor, StopMonitor } from '../../bindings/vshell/internal/app/appservice'
import type { SystemStats, NetConnProcess, NetHistoryPoint } from '../types'

function newEmptyStats(): SystemStats {
  return {
    cpu_percent: 0,
    mem_percent: 0,
    mem_total: 0,
    mem_used: 0,
    swap_total: 0,
    swap_used: 0,
    net_interfaces: {},
    load_avg: [0, 0, 0],
    disk_stats: [],
    uptime_seconds: 0,
    os: '',
    cpu_cores: [],
    top_processes: [],
    hostname: '',
    ip_addresses: [],
  }
}

const MAX_NET_HISTORY = 60

export const useMonitorStore = defineStore('monitor', () => {
  const stats = reactive<Map<string, SystemStats>>(new Map())
  const connectedSince = reactive<Map<string, number>>(new Map())
  const netConnProcesses = reactive<Map<string, NetConnProcess[]>>(new Map())
  const netHistory = reactive<Map<string, Map<string, NetHistoryPoint[]>>>(new Map())
  let listenersRegistered = false

  function appendNetHistory(connectionID: string, ifaces: Record<string, { receive_bytes: number; transmit_bytes: number }>) {
    let connHistory = netHistory.get(connectionID)
    if (!connHistory) {
      connHistory = new Map()
      netHistory.set(connectionID, connHistory)
    }

    const now = Date.now()
    for (const [name, io] of Object.entries(ifaces)) {
      let points = connHistory.get(name)
      if (!points) {
        points = []
        connHistory.set(name, points)
      }
      points.push({ ts: now, rx: io.receive_bytes, tx: io.transmit_bytes })
      if (points.length > MAX_NET_HISTORY) {
        points.splice(0, points.length - MAX_NET_HISTORY)
      }
    }
  }

  function registerListeners() {
    if (listenersRegistered) return
    listenersRegistered = true

    Events.On('monitor:stats', (ev: any) => {
      const d = ev?.data
      if (!d?.connectionID) return
      const s = stats.get(d.connectionID)
      if (!s) return
      const incoming = d.stats
      if (incoming) {
        s.cpu_percent = incoming.cpu_percent ?? s.cpu_percent
        s.mem_percent = incoming.mem_percent ?? s.mem_percent
        s.mem_total = incoming.mem_total ?? s.mem_total
        s.mem_used = incoming.mem_used ?? s.mem_used
        s.swap_total = incoming.swap_total ?? s.swap_total
        s.swap_used = incoming.swap_used ?? s.swap_used
        s.net_interfaces = incoming.net_interfaces ?? s.net_interfaces
        s.uptime_seconds = incoming.uptime_seconds ?? s.uptime_seconds
        s.os = incoming.os ?? s.os
        s.cpu_cores = incoming.cpu_cores ?? s.cpu_cores
        s.hostname = incoming.hostname ?? s.hostname
        s.ip_addresses = incoming.ip_addresses ?? s.ip_addresses
      }
      // Append network history
      if (incoming?.net_interfaces) {
        appendNetHistory(d.connectionID, incoming.net_interfaces)
      }
    })

    Events.On('monitor:disk', (ev: any) => {
      const d = ev?.data
      if (!d?.connectionID) return
      const s = stats.get(d.connectionID)
      if (!s) return
      if (d.disks) s.disk_stats = d.disks
    })

    Events.On('monitor:load', (ev: any) => {
      const d = ev?.data
      if (!d?.connectionID) return
      const s = stats.get(d.connectionID)
      if (!s) return
      if (d.load) s.load_avg = d.load
    })

    Events.On('monitor:processes', (ev: any) => {
      const d = ev?.data
      if (!d?.connectionID) return
      const s = stats.get(d.connectionID)
      if (!s) return
      if (d.processes) s.top_processes = d.processes
    })

    Events.On('monitor:netconns', (ev: any) => {
      const d = ev?.data
      if (!d?.connectionID) return
      if (d.netconns) netConnProcesses.set(d.connectionID, d.netconns)
    })
  }

  async function startMonitoring(connectionID: string) {
    registerListeners()
    if (!stats.has(connectionID)) {
      stats.set(connectionID, newEmptyStats())
      connectedSince.set(connectionID, Date.now())
    }
    await StartMonitor(connectionID)
  }

  async function stopMonitoring(connectionID: string) {
    stats.delete(connectionID)
    connectedSince.delete(connectionID)
    netConnProcesses.delete(connectionID)
    netHistory.delete(connectionID)

    try {
      await StopMonitor(connectionID)
    } catch {
      // connection may already be closed
    }
  }

  function getStats(connectionID: string): SystemStats | undefined {
    return stats.get(connectionID)
  }

  function getNetConns(connectionID: string): NetConnProcess[] {
    return netConnProcesses.get(connectionID) ?? []
  }

  function getNetHistory(connectionID: string): Map<string, NetHistoryPoint[]> | undefined {
    return netHistory.get(connectionID)
  }

  function getUptime(connectionID: string): number | undefined {
    return connectedSince.get(connectionID)
  }

  return {
    stats,
    connectedSince,
    netConnProcesses,
    netHistory,
    startMonitoring,
    stopMonitoring,
    getStats,
    getNetConns,
    getNetHistory,
    getUptime,
  }
})
