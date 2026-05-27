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
}

export const useTransferStore = defineStore('transfers', () => {
  const transfers = ref<TransferProgress[]>([])
  const doneTimers = new Map<string, ReturnType<typeof setTimeout>>()

  function addOrUpdateTransfer(t: TransferProgress) {
    const idx = transfers.value.findIndex(x => x.id === t.id)
    if (idx >= 0) {
      transfers.value[idx] = t
    } else {
      transfers.value.push(t)
    }
    // Auto-clear done transfers after 2s as safety net against
    // race between final progress event and transfer-done event
    if (t.done) {
      const existing = doneTimers.get(t.id)
      if (existing) clearTimeout(existing)
      doneTimers.set(t.id, setTimeout(() => {
        removeTransfer(t.id)
        doneTimers.delete(t.id)
      }, 2000))
    }
  }

  function removeTransfer(id: string) {
    const timer = doneTimers.get(id)
    if (timer) {
      clearTimeout(timer)
      doneTimers.delete(id)
    }
    transfers.value = transfers.value.filter(x => x.id !== id)
  }

  function clearDone() {
    transfers.value = transfers.value.filter(x => !x.done)
  }

  return {
    transfers,
    addOrUpdateTransfer,
    removeTransfer,
    clearDone,
  }
})
