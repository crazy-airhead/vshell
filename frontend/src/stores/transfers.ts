import { defineStore } from 'pinia'
import { ref } from 'vue'

export interface TransferProgress {
  id: string
  file_name: string
  total_bytes: number
  transferred: number
  percent: number
  speed_kbps: number
  done: boolean
  error?: string
  direction: 'upload' | 'download'
  cancelled?: boolean
}

export const useTransferStore = defineStore('transfers', () => {
  const transfers = ref<TransferProgress[]>([])
  const doneTimers = new Map<string, ReturnType<typeof setTimeout>>()

  function addOrUpdateTransfer(t: TransferProgress) {
    if (t.done && t.error && (t.error === 'context canceled' || t.error === 'cancelled')) {
      t.cancelled = true
    }
    const idx = transfers.value.findIndex(x => x.id === t.id)
    if (idx >= 0) {
      transfers.value[idx] = t
    } else {
      transfers.value.push(t)
    }
    // Keep done transfers visible while there are still active ones,
    // so the details view can show all files in the batch.
    // When all transfers are done, auto-clear after 2s.
    if (t.done) {
      scheduleClear()
    }
  }

  function scheduleClear() {
    const hasActive = transfers.value.some(x => !x.done)
    if (hasActive) return
    // All done — clear after a short delay
    setTimeout(() => {
      // Re-check in case new transfers started
      if (!transfers.value.some(x => !x.done)) {
        clearDone()
      }
    }, 2000)
  }

  function removeTransfer(id: string) {
    const timer = doneTimers.get(id)
    if (timer) {
      clearTimeout(timer)
      doneTimers.delete(id)
    }
    transfers.value = transfers.value.filter(x => x.id !== id)
  }

  function markTransfersDone(direction: TransferProgress['direction']) {
    let changed = false
    transfers.value = transfers.value.map((t) => {
      if (t.direction !== direction || t.done) return t
      changed = true
      return {
        ...t,
        transferred: t.total_bytes > 0 ? t.total_bytes : t.transferred,
        percent: 100,
        speed_kbps: 0,
        done: true,
      }
    })
    if (changed) scheduleClear()
  }

  function clearDone() {
    transfers.value = transfers.value.filter(x => !x.done)
  }

  function clearAll() {
    doneTimers.forEach(timer => clearTimeout(timer))
    doneTimers.clear()
    transfers.value = []
  }

  return {
    transfers,
    addOrUpdateTransfer,
    removeTransfer,
    markTransfersDone,
    clearDone,
    clearAll,
  }
})
