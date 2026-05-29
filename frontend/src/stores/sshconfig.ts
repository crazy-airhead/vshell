import { defineStore } from 'pinia'
import { ref } from 'vue'
import { ReadSSHConfig, WriteSSHConfig, ReadSSHConfigRaw, WriteSSHConfigRaw, GetSSHConfigImportCandidates, ImportSSHConfigHosts } from '../../bindings/vshell/internal/app/appservice'
import type { SSHConfigEntry, SSHConfigDirective, SSHConfigImportCandidate } from '../types'

export const useSSHConfigStore = defineStore('sshconfig', () => {
  const entries = ref<SSHConfigEntry[]>([])
  const loading = ref(false)

  async function loadEntries() {
    loading.value = true
    try {
      const result = await ReadSSHConfig()
      entries.value = (result || []).map((e: any) => ({
        type: e.type || 'Host',
        pattern: e.pattern || '',
        directives: (e.directives || []).map((d: any) => ({
          key: d.key || '',
          value: d.value || '',
        })),
      })) as SSHConfigEntry[]
    } catch (e) {
      console.error('Failed to load SSH config:', e)
    } finally {
      loading.value = false
    }
  }

  async function saveEntries(newEntries: SSHConfigEntry[]) {
    await WriteSSHConfig(newEntries as any)
    await loadEntries()
  }

  async function addEntry(entry: SSHConfigEntry) {
    const updated = [...entries.value, entry]
    await saveEntries(updated)
  }

  async function updateEntry(index: number, entry: SSHConfigEntry) {
    const updated = [...entries.value]
    updated[index] = entry
    await saveEntries(updated)
  }

  async function deleteEntry(index: number) {
    const updated = entries.value.filter((_, i) => i !== index)
    await saveEntries(updated)
  }

  async function readRaw(): Promise<string> {
    return await ReadSSHConfigRaw()
  }

  async function writeRaw(content: string) {
    await WriteSSHConfigRaw(content)
    await loadEntries()
  }

  async function getImportCandidates(): Promise<SSHConfigImportCandidate[]> {
    const result = await GetSSHConfigImportCandidates()
    return (result || []).map((c: any) => ({
      pattern: c.pattern || '',
      hostname: c.hostname || '',
      port: c.port || 22,
      user: c.user || '',
      identity_file: c.identity_file || '',
      has_key: !!c.has_key,
    })) as SSHConfigImportCandidate[]
  }

  async function importHosts(patterns: string[]): Promise<void> {
    await ImportSSHConfigHosts(patterns)
  }

  return {
    entries,
    loading,
    loadEntries,
    saveEntries,
    addEntry,
    updateEntry,
    deleteEntry,
    readRaw,
    writeRaw,
    getImportCandidates,
    importHosts,
  }
})
