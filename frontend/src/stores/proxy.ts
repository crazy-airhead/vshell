import { defineStore } from 'pinia'
import { ref } from 'vue'
import {
  CreateProxy,
  DeleteProxy,
  ListProxies,
  UpdateProxy,
} from '../../bindings/vshell/internal/app/appservice'
import { ProxyConfig as ProxyConfigModel } from '../../bindings/vshell/internal/models/models'
import type { ProxyConfig } from '../types'

export interface ProxyFormData {
  id: string
  name: string
  type: 'http' | 'socks5'
  host: string
  port: number
  username: string
  password: string
}

export function newProxyFormData(): ProxyFormData {
  return {
    id: crypto.randomUUID(),
    name: '',
    type: 'http',
    host: '',
    port: 3128,
    username: '',
    password: '',
  }
}

export const useProxyStore = defineStore('proxy', () => {
  const proxies = ref<ProxyConfig[]>([])
  const loading = ref(false)

  async function loadProxies() {
    loading.value = true
    try {
      const result = await ListProxies()
      proxies.value = (result || []).map((item: any) => ({
        id: item.id || '',
        name: item.name || '',
        type: item.type || 'http',
        host: item.host || '',
        port: item.port || 3128,
        username: item.username || '',
      }))
    } finally {
      loading.value = false
    }
  }

  async function createProxy(data: ProxyFormData) {
    await CreateProxy(new ProxyConfigModel({
      id: data.id,
      name: data.name,
      type: data.type,
      host: data.host,
      port: data.port,
      username: data.username,
      password: data.password,
    }))
    await loadProxies()
  }

  async function updateProxy(data: ProxyFormData) {
    await UpdateProxy(new ProxyConfigModel({
      id: data.id,
      name: data.name,
      type: data.type,
      host: data.host,
      port: data.port,
      username: data.username,
      password: data.password,
    }))
    await loadProxies()
  }

  async function deleteProxy(id: string) {
    await DeleteProxy(id)
    await loadProxies()
  }

  return {
    proxies,
    loading,
    loadProxies,
    createProxy,
    updateProxy,
    deleteProxy,
  }
})
