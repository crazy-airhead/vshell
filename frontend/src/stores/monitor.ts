import { defineStore } from 'pinia'
import { reactive } from 'vue'
import { Events } from '@wailsio/runtime'
import { StartMonitor, StopMonitor } from '../../bindings/vshell/internal/app/appservice'
import type { SystemStats } from '../types'

function newEmptyStats(): SystemStats {
  return {
    cpu_percent: 0,
    mem_percent: 0,
    mem_total: 0,
    mem_used: 0,
    net_interfaces: {},
    load_avg: [0, 0, 0],
    disk_stats: [],
    uptime_seconds: 0,
    os: '',
  }
}

export const useMonitorStore = defineStore('monitor', () => {
  const stats = reactive<Map<string, SystemStats>>(new Map())
  const connectedSince = reactive<Map<string, number>>(new Map())
  let listenersRegistered = false

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
        s.net_interfaces = incoming.net_interfaces ?? s.net_interfaces
        s.uptime_seconds = incoming.uptime_seconds ?? s.uptime_seconds
        s.os = incoming.os ?? s.os
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
  }

  async function startMonitoring(connectionID: string) {
    if (stats.has(connectionID)) return

    registerListeners()
    stats.set(connectionID, newEmptyStats())
    connectedSince.set(connectionID, Date.now())

    await StartMonitor(connectionID)
  }

  async function stopMonitoring(connectionID: string) {
    stats.delete(connectionID)
    connectedSince.delete(connectionID)

    try {
      await StopMonitor(connectionID)
    } catch {
      // connection may already be closed
    }
  }

  function getStats(connectionID: string): SystemStats | undefined {
    return stats.get(connectionID)
  }

  function getUptime(connectionID: string): number | undefined {
    return connectedSince.get(connectionID)
  }

  return {
    stats,
    connectedSince,
    startMonitoring,
    stopMonitoring,
    getStats,
    getUptime,
  }
})
