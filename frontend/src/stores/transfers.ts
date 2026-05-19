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
  }

  function removeTransfer(id: string) {
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
