import { defineStore } from 'pinia'
import { ref, reactive } from 'vue'
import { Events } from '@wailsio/runtime'
import {
  ListCertTasks,
  CreateCertTask,
  UpdateCertTask,
  DeleteCertTask,
  GetCertTaskCredentials,
  GetCertTaskLog,
  ListDNSProviders,
  DetectCertEnvironment,
  ListRemoteCerts,
  GetCertServerLog,
  CancelCertOp,
  StartAcmeShInstall,
  StartCertIssue,
  StartCertRenew,
  StartCertRemove,
} from '../../bindings/vshell/internal/app/appservice'
import type { CertTask, CertTaskForm, DNSProvider, RemoteCert, CertEnvironment, CertStage } from '../types'

const MAX_LOG_LINES = 2000

interface RunningOp {
  stage: CertStage | ''
  stageStatus: string
  startedAt: number
}

export const useCertStore = defineStore('cert', () => {
  const tasks = ref<CertTask[]>([])
  const providers = ref<DNSProvider[]>([])
  // key: connectionID
  const remoteCerts = reactive<Map<string, RemoteCert[]>>(new Map())
  const envs = reactive<Map<string, CertEnvironment>>(new Map())
  const remoteLoading = reactive<Map<string, boolean>>(new Map())
  // key: taskID or opID
  const logs = reactive<Map<string, string[]>>(new Map())
  const running = reactive<Map<string, RunningOp>>(new Map())
  let listenersRegistered = false

  function pushLog(key: string, stream: string, line: string) {
    let arr = logs.get(key)
    if (!arr) {
      arr = []
      logs.set(key, arr)
    }
    arr.push(stream === 'stderr' ? `[stderr] ${line}` : line)
    if (arr.length > MAX_LOG_LINES) {
      arr.splice(0, arr.length - MAX_LOG_LINES)
    }
  }

  function getLog(key: string): string[] {
    return logs.get(key) ?? []
  }

  function clearLog(key: string) {
    logs.delete(key)
  }

  function isRunning(key: string): boolean {
    return running.has(key)
  }

  function mergeTask(task: CertTask) {
    const idx = tasks.value.findIndex(t => t.id === task.id)
    if (idx >= 0) {
      tasks.value[idx] = task
    } else {
      tasks.value.push(task)
    }
  }

  function registerListeners() {
    if (listenersRegistered) return
    listenersRegistered = true

    Events.On('cert:log', (ev: any) => {
      const d = ev?.data
      if (!d) return
      const key = d.taskID || d.opID
      if (!key) return
      pushLog(key, d.stream ?? 'stdout', d.line ?? '')
    })

    Events.On('cert:stage', (ev: any) => {
      const d = ev?.data
      if (!d) return
      const key = d.taskID || d.opID
      if (!key) return
      if (d.status === 'start') {
        running.set(key, { stage: d.stage ?? '', stageStatus: 'start', startedAt: Date.now() })
      } else if (d.status === 'fail') {
        running.set(key, { stage: d.stage ?? '', stageStatus: 'fail', startedAt: running.get(key)?.startedAt ?? Date.now() })
      } else if (d.stage === 'done') {
        running.delete(key)
      }
    })

    Events.On('cert:task-updated', (ev: any) => {
      const d = ev?.data
      if (d?.id) mergeTask(d as CertTask)
    })

    Events.On('cert:task-deleted', (ev: any) => {
      const d = ev?.data
      if (!d?.taskID) return
      tasks.value = tasks.value.filter(t => t.id !== d.taskID)
      running.delete(d.taskID)
    })

    Events.On('cert:op-done', (ev: any) => {
      const d = ev?.data
      if (!d?.opID) return
      running.delete(d.opID)
    })
  }

  async function loadTasks() {
    registerListeners()
    try {
      tasks.value = (await ListCertTasks()) ?? []
    } catch (e) {
      console.error('Failed to load cert tasks:', e)
    }
  }

  async function loadProviders() {
    try {
      providers.value = (await ListDNSProviders()) ?? []
    } catch (e) {
      console.error('Failed to load DNS providers:', e)
    }
  }

  async function detectEnv(connectionID: string): Promise<CertEnvironment | null> {
    try {
      const env = await DetectCertEnvironment(connectionID)
      envs.set(connectionID, env)
      return env
    } catch (e) {
      console.error('Failed to detect cert environment:', e)
      envs.delete(connectionID)
      return null
    }
  }

  async function installAcmeSh(connectionID: string, email: string): Promise<string | null> {
    try {
      return await StartAcmeShInstall(connectionID, email)
    } catch (e) {
      console.error('Failed to start acme.sh install:', e)
      return null
    }
  }

  async function refreshRemote(connectionID: string): Promise<RemoteCert[]> {
    remoteLoading.set(connectionID, true)
    try {
      const certs = (await ListRemoteCerts(connectionID)) ?? []
      remoteCerts.set(connectionID, certs)
      return certs
    } catch (e) {
      console.error('Failed to list remote certs:', e)
      remoteCerts.delete(connectionID)
      return []
    } finally {
      remoteLoading.delete(connectionID)
    }
  }

  async function createTask(form: CertTaskForm): Promise<CertTask | null> {
    const task = await CreateCertTask(form)
    mergeTask(task)
    return task
  }

  async function updateTask(form: CertTaskForm): Promise<CertTask | null> {
    const task = await UpdateCertTask(form)
    mergeTask(task)
    return task
  }

  async function deleteTask(taskID: string) {
    await DeleteCertTask(taskID)
    tasks.value = tasks.value.filter(t => t.id !== taskID)
  }

  async function revealCredentials(taskID: string): Promise<Record<string, string>> {
    return (await GetCertTaskCredentials(taskID)) ?? {}
  }

  // Logs are in-memory; after an app restart refill the view from the
  // persisted last-operation log so the log window is never silently empty.
  async function ensureOpLog(taskID: string) {
    if ((logs.get(taskID) ?? []).length > 0) return
    try {
      const persisted = await GetCertTaskLog(taskID)
      if (persisted) {
        logs.set(taskID, persisted.split('\n').filter(l => l !== '').slice(-MAX_LOG_LINES))
      }
    } catch (e) {
      console.error('Failed to load persisted cert log:', e)
    }
  }

  async function startIssue(taskID: string, email: string): Promise<boolean> {
    clearLog(taskID)
    await StartCertIssue(taskID, email)
    running.set(taskID, { stage: 'detect', stageStatus: 'start', startedAt: Date.now() })
    return true
  }

  async function startRenew(taskID: string): Promise<boolean> {
    clearLog(taskID)
    await StartCertRenew(taskID)
    running.set(taskID, { stage: 'detect', stageStatus: 'start', startedAt: Date.now() })
    return true
  }

  async function startRemove(taskID: string, deleteTask: boolean): Promise<boolean> {
    clearLog(taskID)
    await StartCertRemove(taskID, deleteTask)
    running.set(taskID, { stage: 'remove', stageStatus: 'start', startedAt: Date.now() })
    return true
  }

  async function readServerLog(connectionID: string): Promise<string> {
    return (await GetCertServerLog(connectionID)) ?? ''
  }

  async function cancelOp(id: string) {
    try {
      await CancelCertOp(id)
    } catch (e) {
      console.error('Failed to cancel cert op:', e)
    }
  }

  // Find the remote cert row matching a task (by main domain).
  function remoteForTask(task: CertTask): RemoteCert | undefined {
    const certs = remoteCerts.get(task.connection_id)
    if (!certs) return undefined
    return certs.find(c => c.main_domain === task.primary_domain)
  }

  return {
    tasks,
    providers,
    remoteCerts,
    envs,
    remoteLoading,
    logs,
    running,
    registerListeners,
    loadTasks,
    loadProviders,
    detectEnv,
    installAcmeSh,
    refreshRemote,
    createTask,
    updateTask,
    deleteTask,
    revealCredentials,
    ensureOpLog,
    startIssue,
    startRenew,
    startRemove,
    readServerLog,
    cancelOp,
    getLog,
    clearLog,
    isRunning,
    remoteForTask,
  }
})
