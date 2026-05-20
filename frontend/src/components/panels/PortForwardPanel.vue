<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  NEmpty,
  NButton,
  NInput,
  NSelect,
  NCheckbox,
  NTag,
  NPopconfirm,
  NCollapse,
  NCollapseItem,
  NModal,
  NForm,
  NFormItem,
  NSpace,
  NTooltip,
  useMessage,
} from 'naive-ui'
import {
  ListAllPortForwards,
  ListRunningPortForwards,
  CreatePortForward,
  UpdatePortForward,
  DeletePortForward,
  StartPortForward,
  StopPortForward,
} from '../../../bindings/vshell/internal/app/appservice'
import { useConnectionStore } from '../../stores/connection'
import type { PortForward } from '../../types'

import IconPlay from '~icons/lucide/play'
import IconSquare from '~icons/lucide/square'
import IconPencil from '~icons/lucide/pencil'
import IconTrash2 from '~icons/lucide/trash-2'
import IconPlus from '~icons/lucide/plus'

interface ServicePreset {
  label: string
  type: string
  localHost: string
  localPort: string
  remoteHost: string
  remotePort: string
}

const servicePresets: ServicePreset[] = [
  { label: 'SSH', type: 'local', localHost: '127.0.0.1', localPort: '2222', remoteHost: '127.0.0.1', remotePort: '22' },
  { label: 'HTTP', type: 'local', localHost: '127.0.0.1', localPort: '8080', remoteHost: '127.0.0.1', remotePort: '80' },
  { label: 'HTTPS', type: 'local', localHost: '127.0.0.1', localPort: '8443', remoteHost: '127.0.0.1', remotePort: '443' },
  { label: 'MySQL', type: 'local', localHost: '127.0.0.1', localPort: '3306', remoteHost: '127.0.0.1', remotePort: '3306' },
  { label: 'PostgreSQL', type: 'local', localHost: '127.0.0.1', localPort: '5432', remoteHost: '127.0.0.1', remotePort: '5432' },
  { label: 'Redis', type: 'local', localHost: '127.0.0.1', localPort: '6379', remoteHost: '127.0.0.1', remotePort: '6379' },
  { label: 'MongoDB', type: 'local', localHost: '127.0.0.1', localPort: '27017', remoteHost: '127.0.0.1', remotePort: '27017' },
]

const { t } = useI18n()
const message = useMessage()
const connectionStore = useConnectionStore()

const forwards = ref<PortForward[]>([])
const running = ref<Set<string>>(new Set())
const saving = ref(false)

const connectionOptions = computed(() =>
  connectionStore.connections.map(c => ({
    label: `${c.name} (${c.host})`,
    value: c.id,
  }))
)

const presetOptions = servicePresets.map(p => ({ label: p.label, value: p.label }))
const typeOptions = computed(() => [
  { label: t('portForward.typeLocal'), value: 'local' },
  { label: t('portForward.typeRemote'), value: 'remote' },
  { label: t('portForward.typeDynamic'), value: 'dynamic' },
])

function typeLabel(type: string): string {
  switch (type) {
    case 'local': return t('portForward.typeLocal')
    case 'remote': return t('portForward.typeRemote')
    case 'dynamic': return t('portForward.typeDynamic')
    default: return type
  }
}

const groupedForwards = computed(() => {
  const groups: { connectionID: string; connectionName: string; connectionHost: string; items: PortForward[] }[] = []
  for (const fwd of forwards.value) {
    const name = fwd.connection_name || fwd.connection_id
    const host = fwd.connection_host || ''
    let group = groups.find(g => g.connectionID === fwd.connection_id)
    if (!group) {
      group = { connectionID: fwd.connection_id, connectionName: name, connectionHost: host, items: [] }
      groups.push(group)
    }
    group.items.push(fwd)
  }
  return groups
})

// Modal form
const showModal = ref(false)
const editingFwd = ref<PortForward | null>(null)
const form = ref({
  name: '',
  connectionID: '',
  preset: null as string | null,
  type: 'local' as string,
  localHost: '127.0.0.1',
  localPort: '',
  remoteHost: '127.0.0.1',
  remotePort: '',
  autoStart: false,
})

const isEdit = computed(() => !!editingFwd.value)

function applyPreset(label: string | null) {
  const preset = servicePresets.find(p => p.label === label)
  if (preset) {
    form.value.type = preset.type
    form.value.localHost = preset.localHost
    form.value.localPort = preset.localPort
    form.value.remoteHost = preset.remoteHost
    form.value.remotePort = preset.remotePort
  }
}

function resetForm() {
  form.value = {
    name: '',
    connectionID: '',
    preset: null,
    type: 'local',
    localHost: '127.0.0.1',
    localPort: '',
    remoteHost: '127.0.0.1',
    remotePort: '',
    autoStart: false,
  }
  editingFwd.value = null
}

function openCreate() {
  resetForm()
  showModal.value = true
}

function openEdit(fwd: PortForward) {
  editingFwd.value = fwd
  form.value = {
    name: fwd.name,
    connectionID: fwd.connection_id,
    preset: null,
    type: fwd.type,
    localHost: fwd.local_host,
    localPort: String(fwd.local_port),
    remoteHost: fwd.remote_host,
    remotePort: String(fwd.remote_port),
    autoStart: fwd.auto_start,
  }
  showModal.value = true
}

async function loadForwards() {
  try {
    forwards.value = await ListAllPortForwards()
  } catch (e) {
    console.error('Failed to load port forwards:', e)
  }
}

async function loadRunningState() {
  try {
    const ids = await ListRunningPortForwards()
    running.value = new Set(ids)
  } catch (e) {
    console.error('Failed to load running state:', e)
  }
}

async function handleSave() {
  const localPort = parseInt(form.value.localPort, 10)
  const remotePort = parseInt(form.value.remotePort, 10)
  if (!form.value.name || !form.value.connectionID || isNaN(localPort) || isNaN(remotePort)) return

  saving.value = true
  try {
    if (isEdit.value) {
      await UpdatePortForward(
        editingFwd.value!.id,
        form.value.name,
        form.value.connectionID,
        form.value.type,
        form.value.localHost,
        localPort,
        form.value.remoteHost,
        remotePort,
        form.value.autoStart,
      )
      message.success(t('portForward.updated'))
    } else {
      await CreatePortForward(
        form.value.name,
        form.value.connectionID,
        form.value.type,
        form.value.localHost,
        localPort,
        form.value.remoteHost,
        remotePort,
        form.value.autoStart,
      )
      message.success(t('portForward.created'))
    }
    showModal.value = false
    resetForm()
    await loadForwards()
  } catch (e: any) {
    message.error(t('portForward.failed', { error: e }))
  } finally {
    saving.value = false
  }
}

async function handleStart(fwd: PortForward) {
  try {
    console.log('[port-forward] Starting:', fwd.id, fwd.name)
    await StartPortForward(fwd.id)
    console.log('[port-forward] Started successfully:', fwd.id)
    running.value.add(fwd.id)
  } catch (e: any) {
    console.error('[port-forward] Start failed:', e)
    message.error(String(e))
  }
}

async function handleStop(fwd: PortForward) {
  try {
    console.log('[port-forward] Stopping:', fwd.id, fwd.name)
    await StopPortForward(fwd.id)
    console.log('[port-forward] Stopped successfully:', fwd.id)
    running.value.delete(fwd.id)
  } catch (e: any) {
    console.error('[port-forward] Stop failed:', e)
    message.error(String(e))
  }
}

async function handleDelete(fwd: PortForward) {
  try {
    await DeletePortForward(fwd.id)
    running.value.delete(fwd.id)
    message.success(t('portForward.deleted', { name: fwd.name }))
    await loadForwards()
  } catch (e: any) {
    message.error(t('portForward.failed', { error: e }))
  }
}

onMounted(() => {
  loadForwards()
  loadRunningState()
})
</script>

<template>
  <div class="flex flex-col h-full overflow-hidden">
    <!-- Toolbar -->
    <div class="flex items-center justify-between px-3 py-2 shrink-0">
      <span class="text-[var(--font-size-sm)] font-semibold text-[var(--text-secondary)]">{{ t('portForward.title') }}</span>
      <NTooltip>
        <template #trigger>
          <NButton size="tiny" quaternary @click="openCreate">
            <template #icon><IconPlus :width="14" :height="14" /></template>
          </NButton>
        </template>
        {{ t('portForward.add') }}
      </NTooltip>
    </div>

    <!-- Forwards List grouped by connection -->
    <div class="flex-1 min-h-0 overflow-auto px-3">
      <div v-if="forwards.length === 0" class="flex-center py-12">
        <NEmpty :description="t('portForward.noForwards')" size="small" />
      </div>
      <NCollapse v-else>
        <NCollapseItem v-for="group in groupedForwards" :key="group.connectionID" :name="group.connectionID"
          :title="group.connectionHost ? `${group.connectionName}  ·  ${group.connectionHost}` : group.connectionName">
          <template #header-extra>
            <span class="text-[11px] text-[var(--text-secondary)]">{{ group.items.length }} {{ t('portForward.rules') }}</span>
          </template>
          <div v-for="fwd in group.items" :key="fwd.id"
            class="group flex items-center gap-2 py-1.5 px-1 border-b border-[var(--border-color)] last:border-b-0 rounded-[var(--border-radius)] transition-colors duration-150 hover:bg-[var(--hover-overlay)]">
            <div class="flex-1 min-w-0">
              <div class="flex items-center gap-1.5">
                <NTag :bordered="false" size="tiny" type="info">{{ typeLabel(fwd.type) }}</NTag>
                <span class="text-[var(--font-size-sm)] font-medium truncate">{{ fwd.name }}</span>
              </div>
              <div class="text-[11px] text-[var(--text-secondary)] truncate">
                <span class="text-[var(--text-primary)] opacity-70">{{ t('portForward.local') }}</span>
                {{ fwd.local_host }}:{{ fwd.local_port }}
              </div>
              <div class="text-[11px] text-[var(--text-secondary)] truncate">
                <span class="text-[var(--text-primary)] opacity-70">{{ t('portForward.remote') }}</span>
                {{ fwd.remote_host }}:{{ fwd.remote_port }}
              </div>
              <div class="flex items-center gap-1 mt-0.5">
                <NTag :bordered="false" size="tiny" :type="running.has(fwd.id) ? 'success' : 'default'">
                  {{ running.has(fwd.id) ? t('portForward.running') : t('portForward.stopped') }}
                </NTag>
              </div>
            </div>
            <div class="flex gap-[2px] shrink-0 opacity-0 transition-opacity duration-150 group-hover:opacity-100">
              <NTooltip v-if="!running.has(fwd.id)">
                <template #trigger>
                  <NButton size="tiny" quaternary @click="handleStart(fwd)">
                    <template #icon><IconPlay :width="14" :height="14" /></template>
                  </NButton>
                </template>
                {{ t('portForward.start') }}
              </NTooltip>
              <NTooltip v-else>
                <template #trigger>
                  <NButton size="tiny" quaternary @click="handleStop(fwd)">
                    <template #icon><IconSquare :width="14" :height="14" /></template>
                  </NButton>
                </template>
                {{ t('portForward.stop') }}
              </NTooltip>
              <NTooltip>
                <template #trigger>
                  <NButton size="tiny" quaternary @click="openEdit(fwd)">
                    <template #icon><IconPencil :width="14" :height="14" /></template>
                  </NButton>
                </template>
                {{ t('common.edit') }}
              </NTooltip>
              <NPopconfirm @positive-click="handleDelete(fwd)">
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
                {{ t('portForward.deleteContent', { name: fwd.name }) }}
              </NPopconfirm>
            </div>
          </div>
        </NCollapseItem>
      </NCollapse>
    </div>

    <!-- Create / Edit Modal -->
    <NModal v-model:show="showModal" preset="card" :title="isEdit ? t('portForward.edit') : t('portForward.add')" style="width: 480px" :mask-closable="false">
      <NForm label-placement="left" label-width="100">
        <NFormItem :label="t('portForward.name')">
          <NInput v-model:value="form.name" :placeholder="t('portForward.namePlaceholder')" />
        </NFormItem>
        <NFormItem :label="t('portForward.connection')">
          <NSelect v-model:value="form.connectionID" :options="connectionOptions" :placeholder="t('portForward.connectionPlaceholder')" />
        </NFormItem>
        <NFormItem :label="t('portForward.servicePreset')">
          <NSelect
            v-model:value="form.preset"
            :options="presetOptions"
            :placeholder="t('portForward.servicePresetPlaceholder')"
            clearable
            @update:value="applyPreset"
          />
        </NFormItem>
        <NFormItem :label="t('portForward.type')">
          <NSelect v-model:value="form.type" :options="typeOptions" />
        </NFormItem>
        <NFormItem :label="t('portForward.localHost')">
          <NInput v-model:value="form.localHost" placeholder="127.0.0.1" />
        </NFormItem>
        <NFormItem :label="t('portForward.localPort')">
          <NInput v-model:value="form.localPort" placeholder="8080" />
        </NFormItem>
        <NFormItem :label="t('portForward.remoteHost')">
          <NInput v-model:value="form.remoteHost" placeholder="127.0.0.1" />
        </NFormItem>
        <NFormItem :label="t('portForward.remotePort')">
          <NInput v-model:value="form.remotePort" placeholder="80" />
        </NFormItem>
        <NFormItem label=" ">
          <NCheckbox v-model:checked="form.autoStart">{{ t('portForward.autoStart') }}</NCheckbox>
        </NFormItem>
      </NForm>
      <template #footer>
        <NSpace justify="end">
          <NButton @click="showModal = false">{{ t('common.cancel') }}</NButton>
          <NButton type="primary" :loading="saving" @click="handleSave">{{ isEdit ? t('common.update') : t('common.save') }}</NButton>
        </NSpace>
      </template>
    </NModal>
  </div>
</template>

