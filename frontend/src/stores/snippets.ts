import { defineStore } from 'pinia'
import { ref } from 'vue'
import {
  CreateQuickCommand,
  DeleteQuickCommand,
  ListQuickCommands,
  UpdateQuickCommand,
} from '../../bindings/vshell/internal/app/appservice'
import type { QuickCommand } from '../types'

export interface SnippetFormData {
  id: string
  name: string
  command: string
  sort_order: number
}

export const useSnippetsStore = defineStore('snippets', () => {
  const snippets = ref<QuickCommand[]>([])
  const loading = ref(false)

  async function loadSnippets() {
    loading.value = true
    try {
      const result = await ListQuickCommands(null)
      snippets.value = (result || []).map((item: any) => ({
        id: item.id || '',
        name: item.name || '',
        command: item.command || '',
        connection_id: item.connection_id || null,
        sort_order: item.sort_order || 0,
      }))
    } finally {
      loading.value = false
    }
  }

  async function createSnippet(data: SnippetFormData) {
    await CreateQuickCommand({
      id: data.id,
      name: data.name,
      command: data.command,
      connection_id: null,
      sort_order: data.sort_order,
    } as any)
    await loadSnippets()
  }

  async function updateSnippet(data: SnippetFormData) {
    await UpdateQuickCommand({
      id: data.id,
      name: data.name,
      command: data.command,
      connection_id: null,
      sort_order: data.sort_order,
    } as any)
    await loadSnippets()
  }

  async function deleteSnippet(id: string) {
    await DeleteQuickCommand(id)
    await loadSnippets()
  }

  return {
    snippets,
    loading,
    loadSnippets,
    createSnippet,
    updateSnippet,
    deleteSnippet,
  }
})
