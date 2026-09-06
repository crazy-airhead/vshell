<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  NEmpty,
  NButton,
  NTag,
  NTooltip,
  NPopconfirm,
  NCollapse,
  NCollapseItem,
  useMessage,
} from 'naive-ui'
import IconPlus from '~icons/lucide/plus'
import IconPencil from '~icons/lucide/pencil'
import IconTrash2 from '~icons/lucide/trash-2'
import IconRefreshCw from '~icons/lucide/refresh-cw'
import IconFileClock from '~icons/lucide/file-clock'
import IconRotateCcw from '~icons/lucide/rotate-ccw'
import IconShieldOff from '~icons/lucide/shield-off'
import { useCertStore } from '../../stores/cert'
import type { CertTask } from '../../types'
import CertWizard from './CertWizard.vue'
import CertEditModal from './CertEditModal.vue'
import CertLogModal from './CertLogModal.vue'

const { t } = useI18n()
const message = useMessage()
const certStore = useCertStore()

const showWizard = ref(false)
const editingTask = ref<CertTask | null>(null)
const logTask = ref<CertTask | null>(null)

const groupedTasks = computed(() => {
  const groups: { connectionID: string; connectionName: string; connectionHost: string; items: CertTask[] }[] = []
  for (const task of certStore.tasks) {
    const name = task.connection_name || task.connection_id
    const host = task.connection_host || ''
    let group = groups.find(g => g.connectionID === task.connection_id)
    if (!group) {
      group = { connectionID: task.connection_id, connectionName: name, connectionHost: host, items: [] }
      groups.push(group)
    }
    group.items.push(task)
  }
  return groups
})

function statusTagType(task: CertTask): 'default' | 'info' | 'success' | 'error' {
  switch (task.last_status) {
    case 'issued': return 'success'
    case 'running': return 'info'
    case 'failed': return 'error'
    default: return 'default'
  }
}

function statusLabel(task: CertTask): string {
  switch (task.last_status) {
    case 'issued': return t('certs.statusIssued')
    case 'running': return t('certs.statusRunning')
    case 'failed': return t('certs.statusFailed')
    default: return t('certs.statusIdle')
  }
}

function expiry(task: CertTask): { text: string; type: 'default' | 'success' | 'warning' | 'error' } {
  const remote = certStore.remoteForTask(task)
  if (!remote) {
    return { text: t('certs.remoteMissing'), type: 'default' }
  }
  if (remote.days_left == null) {
    return { text: t('certs.unknownExpiry'), type: 'default' }
  }
  if (remote.days_left < 0) {
    return { text: t('certs.expired'), type: 'error' }
  }
  if (remote.days_left < 30) {
    return { text: t('certs.expiringSoon') + ` (${t('certs.daysLeft', { n: remote.days_left })})`, type: 'warning' }
  }
  return { text: t('certs.daysLeft', { n: remote.days_left }), type: 'success' }
}

function domainsText(task: CertTask): string {
  return [task.primary_domain, ...(task.san_domains ?? [])].join(', ')
}

async function refreshGroup(connectionID: string) {
  await Promise.all([certStore.detectEnv(connectionID), certStore.refreshRemote(connectionID)])
}

async function handleRenew(task: CertTask) {
  try {
    await certStore.startRenew(task.id)
    message.info(t('certs.stageRenew'))
  } catch (e: any) {
    message.error(t('certs.failed', { error: String(e) }))
  }
}

async function handleRemove(task: CertTask) {
  try {
    await certStore.startRemove(task.id, true)
  } catch (e: any) {
    message.error(t('certs.failed', { error: String(e) }))
  }
}

async function handleDelete(task: CertTask) {
  try {
    await certStore.deleteTask(task.id)
  } catch (e: any) {
    message.error(t('certs.failed', { error: String(e) }))
  }
}

function handleWizardClosed(connectionID?: string) {
  if (connectionID) refreshGroup(connectionID)
}

onMounted(() => {
  certStore.registerListeners()
  certStore.loadTasks()
  certStore.loadProviders()
})
</script>

<template>
  <div class="flex flex-col h-full overflow-hidden">
    <!-- Toolbar -->
    <div class="flex items-center justify-between px-3 py-2 shrink-0">
      <span class="text-[var(--font-size-sm)] font-semibold text-[var(--text-secondary)]">{{ t('certs.title') }}</span>
      <NTooltip>
        <template #trigger>
          <NButton size="tiny" quaternary @click="showWizard = true">
            <template #icon><IconPlus :width="14" :height="14" /></template>
          </NButton>
        </template>
        {{ t('certs.add') }}
      </NTooltip>
    </div>

    <!-- Task list grouped by connection -->
    <div class="flex-1 min-h-0 overflow-auto px-3">
      <div v-if="certStore.tasks.length === 0" class="flex-center py-12">
        <NEmpty :description="t('certs.empty')" size="small" />
      </div>
      <NCollapse v-else>
        <NCollapseItem v-for="group in groupedTasks" :key="group.connectionID" :name="group.connectionID"
          :title="group.connectionHost ? `${group.connectionName}  ·  ${group.connectionHost}` : group.connectionName">
          <template #header-extra>
            <NTooltip>
              <template #trigger>
                <NButton size="tiny" quaternary :loading="certStore.remoteLoading.has(group.connectionID)"
                  @click.stop="refreshGroup(group.connectionID)">
                  <template #icon><IconRefreshCw :width="14" :height="14" /></template>
                </NButton>
              </template>
              {{ t('certs.refreshRemote') }}
            </NTooltip>
          </template>
          <div v-if="certStore.envs.has(group.connectionID) && !certStore.envs.get(group.connectionID)?.cron_present"
            class="text-[11px] text-[var(--color-warning)] px-1 pb-1">
            {{ t('certs.cronMissing') }}
          </div>
          <div v-for="task in group.items" :key="task.id"
            class="group flex items-center gap-2 py-1.5 px-1 border-b border-[var(--border-color)] last:border-b-0 rounded-[var(--border-radius)] transition-colors duration-150 hover:bg-[var(--hover-overlay)]">
            <div class="flex-1 min-w-0">
              <div class="flex items-center gap-1.5 flex-wrap">
                <NTag :bordered="false" size="tiny" :type="statusTagType(task)">{{ statusLabel(task) }}</NTag>
                <span class="text-[var(--font-size-sm)] font-medium truncate" :title="domainsText(task)">{{ domainsText(task) }}</span>
                <NTag :bordered="false" size="tiny" :type="expiry(task).type">{{ expiry(task).text }}</NTag>
                <NTag v-if="task.test_mode" :bordered="false" size="tiny" type="warning">staging</NTag>
              </div>
              <div class="text-[11px] text-[var(--text-secondary)] truncate">
                {{ task.auto_install ? `${task.cert_dir}/${task.key_file ? task.key_file : ''}` : t('certs.stepInstall') + ' · acme.sh' }}
                <span v-if="task.reload_cmd"> · {{ task.reload_cmd }}</span>
              </div>
              <div v-if="task.last_status === 'failed' && task.last_error" class="text-[11px] text-[var(--color-error)] truncate" :title="task.last_error">
                {{ task.last_error.split('\n')[0] }}
              </div>
            </div>
            <div class="flex gap-[2px] shrink-0 opacity-0 transition-opacity duration-150 group-hover:opacity-100">
              <NTooltip v-if="!certStore.isRunning(task.id)">
                <template #trigger>
                  <NButton size="tiny" quaternary @click="handleRenew(task)">
                    <template #icon><IconRotateCcw :width="14" :height="14" /></template>
                  </NButton>
                </template>
                {{ t('certs.renew') }}
              </NTooltip>
              <NTooltip>
                <template #trigger>
                  <NButton size="tiny" quaternary @click="logTask = task">
                    <template #icon><IconFileClock :width="14" :height="14" /></template>
                  </NButton>
                </template>
                {{ t('certs.logs') }}
              </NTooltip>
              <NTooltip>
                <template #trigger>
                  <NButton size="tiny" quaternary @click="editingTask = task">
                    <template #icon><IconPencil :width="14" :height="14" /></template>
                  </NButton>
                </template>
                {{ t('common.edit') }}
              </NTooltip>
              <NPopconfirm @positive-click="handleRemove(task)">
                <template #trigger>
                  <NTooltip>
                    <template #trigger>
                      <NButton size="tiny" quaternary>
                        <template #icon><IconShieldOff :width="14" :height="14" /></template>
                      </NButton>
                    </template>
                    {{ t('certs.removeFromServer') }}
                  </NTooltip>
                </template>
                {{ t('certs.confirmRemove') }}
              </NPopconfirm>
              <NPopconfirm @positive-click="handleDelete(task)">
                <template #trigger>
                  <NTooltip>
                    <template #trigger>
                      <NButton size="tiny" quaternary>
                        <template #icon><IconTrash2 :width="14" :height="14" /></template>
                      </NButton>
                    </template>
                    {{ t('common.delete') }}
                  </NTooltip>
                </template>
                {{ t('certs.confirmDelete', { name: task.name }) }}
              </NPopconfirm>
            </div>
          </div>
        </NCollapseItem>
      </NCollapse>

      <div v-if="certStore.tasks.length > 0" class="text-[11px] text-[var(--text-secondary)] px-1 py-2">
        {{ t('certs.runningHint') }}
      </div>
    </div>

    <!-- Modals -->
    <CertWizard v-model:show="showWizard" @closed="handleWizardClosed" />
    <CertEditModal :show="!!editingTask" :task="editingTask" @update:show="(v: boolean) => { if (!v) editingTask = null }" />
    <CertLogModal :show="!!logTask" :task="logTask" @update:show="(v: boolean) => { if (!v) logTask = null }" />
  </div>
</template>
