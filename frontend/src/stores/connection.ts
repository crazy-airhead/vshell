import { defineStore } from 'pinia'
import { ref } from 'vue'
import {
  ListConnections,
  ListGroups,
  ConnectSSH,
  DisconnectSSH,
  DisconnectSession,
  CreateConnection,
  UpdateConnection,
  DeleteConnection,
  ExportConnectionConfigs,
  ImportConnectionConfigs,
  IsConnectionConfigEncrypted,
  MoveConnection,
  CreateGroup,
  UpdateGroup,
  DeleteGroup,
} from '../../bindings/vshell/internal/app/appservice'
import { AuthType, ConnectionForm } from '../../bindings/vshell/internal/models/models'
import type { Connection, Group } from '../types'
import { useMonitorStore } from './monitor'

export { AuthType }

export interface ConnectionFormData {
  id: string
  name: string
  host: string
  port: number
  username: string
  authType: string
  password: string
  privateKey: string
  keyPassphrase: string
  groupID: string | null
}

export function newFormData(): ConnectionFormData {
  return {
    id: crypto.randomUUID(),
    name: '',
    host: '',
    port: 22,
    username: 'root',
    authType: AuthType.AuthPassword,
    password: '',
    privateKey: '',
    keyPassphrase: '',
    groupID: null,
  }
}

export const useConnectionStore = defineStore('connection', () => {
  const connections = ref<Connection[]>([])
  const groups = ref<Group[]>([])
  const activeConnectionID = ref<string | null>(null)
  const connectedIDs = ref<Set<string>>(new Set())
  const connecting = ref(false)

  async function loadConnections() {
    try {
      const result = await ListConnections()
      console.log('[DEBUG] ListConnections raw result:', result, 'type:', typeof result, Array.isArray(result))
      // Convert Wails class instances to plain objects for Vue reactivity
      const items = (result || []).map((c: any) => ({
        id: c.id || '',
        group_id: c.group_id || null,
        name: c.name || '',
        host: c.host || '',
        port: c.port || 22,
        username: c.username || '',
        auth_type: c.auth_type || '',
        upload_path: c.upload_path || '/',
        default_cmd: c.default_cmd || null,
        sort_order: c.sort_order || 0,
        color: c.color || null,
        last_used_at: c.last_used_at || null,
      }))
      connections.value = items as Connection[]
      console.log('[DEBUG] connections.value set:', connections.value.length, 'items', connections.value.map(c => c.name))
    } catch (e) {
      console.error('Failed to load connections:', e)
    }
  }

  async function loadGroups() {
    try {
      const result = await ListGroups()
      const items = (result || []).map((g: any) => ({
        id: g.id || '',
        name: g.name || '',
        parent_id: g.parent_id || null,
        sort_order: g.sort_order || 0,
      }))
      groups.value = items as Group[]
    } catch (e) {
      console.error('Failed to load groups:', e)
    }
  }

  async function connect(id: string): Promise<string> {
    connecting.value = true
    try {
      const sessionID = await ConnectSSH(id) as unknown as string
      connectedIDs.value.add(id)
      activeConnectionID.value = id
      const monitorStore = useMonitorStore()
      await monitorStore.startMonitoring(id)
      return sessionID
    } catch (e) {
      console.error('Failed to connect:', e)
      throw e
    } finally {
      connecting.value = false
    }
  }

  async function disconnect(id: string) {
    try {
      const monitorStore = useMonitorStore()
      await monitorStore.stopMonitoring(id)
      await DisconnectSSH(id)
      connectedIDs.value.delete(id)
      if (activeConnectionID.value === id) {
        activeConnectionID.value = null
      }
    } catch (e) {
      console.error('Failed to disconnect:', e)
    }
  }

  async function disconnectSession(sessionID: string, connectionID: string) {
    try {
      await DisconnectSession(sessionID, connectionID)
    } catch (e) {
      console.error('Failed to disconnect session:', e)
    }
  }

  async function createConnection(data: ConnectionFormData) {
    const form = new ConnectionForm({
      id: data.id,
      name: data.name,
      host: data.host,
      port: data.port,
      username: data.username,
      auth_type: data.authType as AuthType,
      password: data.password,
      private_key: data.privateKey,
      key_passphrase: data.keyPassphrase,
      group_id: data.groupID || null,
      upload_path: '/',
      sort_order: 0,
    })
    console.log('[DEBUG] Creating connection with form:', JSON.stringify(form))
    await CreateConnection(form)
    console.log('[DEBUG] CreateConnection succeeded, loading connections...')
    await loadConnections()
  }

  async function removeConnection(id: string) {
    await DeleteConnection(id)
    await loadConnections()
  }

  async function moveConnection(id: string, groupID: string | null) {
    await MoveConnection(id, groupID)
    await loadConnections()
  }

  async function updateConnection(data: ConnectionFormData) {
    const form = new ConnectionForm({
      id: data.id,
      name: data.name,
      host: data.host,
      port: data.port,
      username: data.username,
      auth_type: data.authType as AuthType,
      password: data.password,
      private_key: data.privateKey,
      key_passphrase: data.keyPassphrase,
      group_id: data.groupID || null,
      upload_path: '/',
      sort_order: 0,
    })
    await UpdateConnection(form)
    await loadConnections()
  }

  async function exportConfigs(filePath: string, password = '') {
    await ExportConnectionConfigs(filePath, password)
  }

  async function isConfigEncrypted(filePath: string) {
    return await IsConnectionConfigEncrypted(filePath)
  }

  async function importConfigs(filePath: string, password = '') {
    const result = await ImportConnectionConfigs(filePath, password)
    await Promise.all([loadConnections(), loadGroups()])
    return result
  }

  function getConnectionsByGroup(groupID: string | null): Connection[] {
    return connections.value.filter((c) => {
      const gid = c.group_id || null
      return gid === groupID
    })
  }

  async function createGroup(name: string, parentID: string | null) {
    await CreateGroup(crypto.randomUUID(), name, parentID, 0)
    await loadGroups()
  }

  async function updateGroup(id: string, name: string) {
    await UpdateGroup(id, name)
    await loadGroups()
  }

  async function removeGroup(id: string) {
    await DeleteGroup(id)
    await loadGroups()
    await loadConnections()
  }

  return {
    connections,
    groups,
    activeConnectionID,
    connectedIDs,
    connecting,
    loadConnections,
    loadGroups,
    connect,
    disconnect,
    disconnectSession,
    createConnection,
    removeConnection,
    moveConnection,
    updateConnection,
    exportConfigs,
    isConfigEncrypted,
    importConfigs,
    getConnectionsByGroup,
    createGroup,
    updateGroup,
    removeGroup,
  }
})
