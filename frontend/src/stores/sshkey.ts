import { defineStore } from 'pinia'
import { ref } from 'vue'
import { ListSSHKeys, SaveSSHKey, RenameSSHKey, DeleteSSHKey, ReadSSHKeyContent, GenerateSSHKey, GetSSHKeyUsage } from '../../bindings/vshell/internal/app/appservice'
import type { SSHKeyInfo } from '../types'

export const useSSHKeyStore = defineStore('sshkey', () => {
  const keys = ref<SSHKeyInfo[]>([])

  async function loadKeys() {
    try {
      const result = await ListSSHKeys()
      keys.value = (result || []).map((k: any) => ({
        name: k.name || '',
        type: k.type || '',
        fingerprint: k.fingerprint || '',
        public_key: k.public_key || '',
        comment: k.comment || '',
        has_passphrase: !!k.has_passphrase,
      })) as SSHKeyInfo[]
    } catch (e) {
      console.error('Failed to load SSH keys:', e)
    }
  }

  async function saveKey(name: string, privateKey: string, publicKey: string) {
    await SaveSSHKey(name, privateKey, publicKey)
    await loadKeys()
  }

  async function renameKey(oldName: string, newName: string) {
    await RenameSSHKey(oldName, newName)
    await loadKeys()
  }

  async function deleteKey(name: string) {
    await DeleteSSHKey(name)
    await loadKeys()
  }

  async function getKeyUsage(name: string): Promise<string[]> {
    return (await GetSSHKeyUsage(name)) || []
  }

  async function readContent(name: string, kind: string): Promise<string> {
    return await ReadSSHKeyContent(name, kind)
  }

  async function generateKey(name: string, keyType: string, bits: number, comment: string, passphrase: string) {
    await GenerateSSHKey(name, keyType, bits, comment, passphrase)
    await loadKeys()
  }

  return {
    keys,
    loadKeys,
    saveKey,
    renameKey,
    deleteKey,
    getKeyUsage,
    readContent,
    generateKey,
  }
})
