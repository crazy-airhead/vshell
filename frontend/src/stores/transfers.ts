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

  function addOrUpdateTransfer(t: TransferProgress) {
    const idx = transfers.value.findIndex(x => x.id === t.id)
    if (idx >= 0) {
      transfers.value[idx] = t
    } else {
      transfers.value.push(t)
    }
    // Auto-remove completed transfers after 5 seconds
    if (t.done) {
      setTimeout(() => {
        transfers.value = transfers.value.filter(x => x.id !== t.id)
      }, 5000)
    }
  }

  function removeTransfer(id: string) {
    transfers.value = transfers.value.filter(x => x.id !== id)
  }

  return {
    transfers,
    addOrUpdateTransfer,
    removeTransfer,
  }
})
