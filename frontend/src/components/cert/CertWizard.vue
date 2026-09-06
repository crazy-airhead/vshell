<script setup lang="ts">
import { computed, onUnmounted, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  NButton,
  NCheckbox,
  NCollapse,
  NCollapseItem,
  NForm,
  NFormItem,
  NInput,
  NInputNumber,
  NModal,
  NSelect,
  NSpace,
  NSteps,
  NStep,
  NTag,
  useMessage,
} from 'naive-ui'
import IconCheck from '~icons/lucide/check'
import IconLoader2 from '~icons/lucide/loader-2'
import { Events } from '@wailsio/runtime'
import { useCertStore } from '../../stores/cert'
import { useConnectionStore } from '../../stores/connection'
import type { CertTask, DNSProvider } from '../../types'
import CertLogView from './CertLogView.vue'

const props = defineProps<{ show: boolean }>()
const emit = defineEmits<{ (e: 'update:show', value: boolean): void; (e: 'closed', connectionID?: string): void }>()

const { t } = useI18n()
const message = useMessage()
const certStore = useCertStore()
const connectionStore = useConnectionStore()

const EMAIL_KEY = 'vshell:cert-email'

const visible = computed({
  get: () => props.show,
  set: (v: boolean) => emit('update:show', v),
})

const step = ref(1)
const detecting = ref(false)
const installOpID = ref('')
const installing = ref(false)
const email = ref(localStorage.getItem(EMAIL_KEY) ?? '')
const detectError = ref('')

const connectionOptions = computed(() =>
  connectionStore.connections.map(c => ({
    label: `${c.name} (${c.host})`,
    value: c.id,
  }))
)

const connectionID = ref('')
const env = computed(() => (connectionID.value ? certStore.envs.get(connectionID.value) ?? null : null))

const providerOptions = computed(() =>
  certStore.providers.map(p => ({ label: p.name, value: p.id }))
)
const selectedProvider = computed<DNSProvider | null>(() =>
  certStore.providers.find(p => p.id === form.dnsProvider) ?? null
)

const reloadPresets = computed(() => [
  { label: t('certs.reloadNone'), value: 'none' },
  { label: t('certs.reloadNginx'), value: 'nginx' },
  { label: t('certs.reloadApache'), value: 'apache' },
  { label: t('certs.reloadCaddy'), value: 'caddy' },
  { label: t('certs.reloadCustom'), value: 'custom' },
])

const reloadCommands: Record<string, string> = {
  nginx: 'systemctl reload nginx',
  apache: 'systemctl reload apache2',
  caddy: 'systemctl reload caddy',
}

interface KVPair { key: string; value: string }

const form = reactive({
  name: '',
  primaryDomain: '',
  sanDomainsText: '',
  dnsProvider: '',
  creds: {} as Record<string, string>,
  customPlugin: '',
  customPairs: [] as KVPair[],
  autoInstall: true,
  certDir: '/etc/nginx/ssl',
  keyFile: '',
  fullchainFile: '',
  reloadPreset: 'nginx',
  reloadCmd: 'systemctl reload nginx',
  keyLength: 'ec-256',
  dnsSleep: 120,
  testMode: true,
})

const keyLengthOptions = computed(() => [
  { label: t('certs.keyLengthEc256'), value: 'ec-256' },
  { label: t('certs.keyLengthRsa2048'), value: '2048' },
])

// Run state (step 4)
const runTaskID = ref('')
const runFinished = ref<'success' | 'failed' | ''>('')

const runningStage = computed(() => certStore.running.get(runTaskID.value)?.stage ?? '')
const currentTask = computed<CertTask | null>(() =>
  runTaskID.value ? certStore.tasks.find(t => t.id === runTaskID.value) ?? null : null
)

const stages = computed(() =>
  form.autoInstall
    ? ['detect', 'issue', 'install-cert', 'cron', 'done']
    : ['detect', 'issue', 'cron', 'done']
)

function stageLabel(stage: string): string {
  const map: Record<string, string> = {
    detect: t('certs.stageDetect'),
    install: t('certs.stageInstall'),
    issue: t('certs.stageIssue'),
    renew: t('certs.stageRenew'),
    'install-cert': t('certs.stageInstallCert'),
    cron: t('certs.stageCron'),
    remove: t('certs.stageRemove'),
    done: t('certs.stageDone'),
  }
  return map[stage] ?? stage
}

function stageState(stage: string): 'done' | 'active' | 'pending' {
  if (runFinished.value === 'success') return 'done'
  if (runFinished.value === 'failed') {
    // failed at some stage: mark stages before failure done, the failed one stays active
    const order = stages.value
    const failedAt = order.indexOf(runningStage.value)
    const idx = order.indexOf(stage)
    if (idx < 0) return 'pending'
    if (failedAt < 0) return idx < order.length - 1 ? 'done' : 'active'
    return idx < failedAt ? 'done' : idx === failedAt ? 'active' : 'pending'
  }
  const idx = stages.value.indexOf(stage)
  const cur = stages.value.indexOf(runningStage.value)
  if (cur < 0) return 'pending'
  return idx < cur ? 'done' : idx === cur ? 'active' : 'pending'
}

watch(() => props.show, (open) => {
  if (open) {
    step.value = 1
    resetRun()
    connectionID.value = ''
    detectError.value = ''
    installOpID.value = ''
    installing.value = false
    certStore.loadProviders()
  }
})

watch(connectionID, async (id) => {
  detectError.value = ''
  if (!id) return
  detecting.value = true
  const result = await certStore.detectEnv(id)
  detecting.value = false
  if (!result) {
    detectError.value = t('certs.detectFailed', { error: '' })
  }
})

watch(() => form.primaryDomain, (domain) => {
  form.keyFile = domain ? `${domain}.key` : ''
  form.fullchainFile = domain ? `${domain}.crt` : ''
})

watch(() => form.dnsProvider, () => {
  form.creds = {}
  form.customPairs = [{ key: '', value: '' }]
})

function applyReloadPreset(value: string) {
  if (value === 'none' || value === 'custom') return
  form.reloadCmd = reloadCommands[value] ?? ''
}

async function installAcme() {
  if (!email.value || !connectionID.value) return
  installing.value = true
  const opID = await certStore.installAcmeSh(connectionID.value, email.value)
  if (!opID) {
    installing.value = false
    message.error(t('certs.installFailed'))
    return
  }
  installOpID.value = opID
  localStorage.setItem(EMAIL_KEY, email.value)
}

// Wait for install completion
const offOpDone = Events.On('cert:op-done', async (ev: any) => {
  const d = ev?.data
  if (!d?.opID || d.opID !== installOpID.value) return
  installing.value = false
  if (d.error) {
    message.error(`${t('certs.installFailed')}: ${d.error}`)
  } else {
    message.success(t('certs.installDone'))
  }
  await certStore.detectEnv(connectionID.value)
})

onUnmounted(() => {
  offOpDone()
})

function validateStep1(): boolean {
  if (!connectionID.value) {
    message.warning(t('certs.errConnRequired'))
    return false
  }
  if (!env.value?.installed) {
    message.warning(t('certs.notInstalled'))
    return false
  }
  return true
}

function collectCredentials(): Record<string, string> {
  const creds: Record<string, string> = {}
  for (const [k, v] of Object.entries(form.creds)) {
    if (k && v) creds[k] = v
  }
  if (form.dnsProvider === 'custom') {
    for (const pair of form.customPairs) {
      if (pair.key && pair.value) creds[pair.key] = pair.value
    }
  }
  return creds
}

function validateStep2(): boolean {
  if (!form.primaryDomain.trim()) {
    message.warning(t('certs.errDomainRequired'))
    return false
  }
  if (!form.dnsProvider) {
    message.warning(t('certs.dnsProviderPlaceholder'))
    return false
  }
  if (form.dnsProvider === 'custom') {
    if (!/^dns_[a-z0-9_]+$/.test(form.customPlugin)) {
      message.warning(t('certs.errPluginRequired'))
      return false
    }
  } else if (selectedProvider.value) {
    for (const f of selectedProvider.value.fields) {
      if (f.required && !form.creds[f.key]) {
        message.warning(t('certs.errCredsRequired'))
        return false
      }
    }
  }
  return true
}

function next() {
  if (step.value === 1 && !validateStep1()) return
  if (step.value === 2 && !validateStep2()) return
  step.value++
}

function prev() {
  step.value--
}

function resetRun() {
  runTaskID.value = ''
  runFinished.value = ''
}

async function startIssue() {
  const creds = collectCredentials()
  const sanDomains = form.sanDomainsText
    .split('\n')
    .map(s => s.trim())
    .filter(s => s !== '')
  try {
    const task = await certStore.createTask({
      id: '',
      connection_id: connectionID.value,
      name: form.name.trim() || form.primaryDomain.trim(),
      primary_domain: form.primaryDomain.trim(),
      san_domains: sanDomains,
      dns_provider: form.dnsProvider,
      dns_plugin: form.dnsProvider === 'custom' ? form.customPlugin : '',
      dns_credentials: creds,
      key_length: form.keyLength,
      dns_sleep: form.dnsSleep,
      test_mode: form.testMode,
      auto_install: form.autoInstall,
      cert_dir: form.autoInstall ? form.certDir.trim() : '',
      key_file: form.autoInstall ? form.keyFile.trim() : '',
      fullchain_file: form.autoInstall ? form.fullchainFile.trim() : '',
      reload_cmd: form.autoInstall && form.reloadPreset !== 'none' ? form.reloadCmd.trim() : '',
    })
    if (!task) return
    runTaskID.value = task.id
    await certStore.startIssue(task.id, email.value)
  } catch (e: any) {
    message.error(t('certs.failed', { error: String(e) }))
  }
}

async function retry() {
  if (!runTaskID.value) return
  try {
    runFinished.value = ''
    await certStore.startIssue(runTaskID.value, email.value)
  } catch (e: any) {
    message.error(t('certs.failed', { error: String(e) }))
  }
}

// Completion detection: the store drops the running entry when the task
// finishes; the persisted last_status tells success from failure.
watch(
  () => [certStore.isRunning(runTaskID.value), currentTask.value?.last_status] as const,
  ([isRun, status]) => {
    if (!runTaskID.value || runFinished.value) return
    if (!isRun) {
      if (status === 'issued') runFinished.value = 'success'
      else if (status === 'failed') runFinished.value = 'failed'
    }
  },
)

function finish() {
  const connID = connectionID.value || undefined
  visible.value = false
  emit('closed', connID)
}

async function cancelRun() {
  await certStore.cancelOp(runTaskID.value)
  runFinished.value = 'failed'
}
</script>

<template>
  <NModal v-model:show="visible" preset="card" :title="t('certs.wizardTitle')" style="width: 640px" :mask-closable="false">
    <!-- Fixed-height content: log areas must be bounded so they scroll
         instead of stretching the modal as streamed lines accumulate. -->
    <div class="flex flex-col gap-3 h-[520px]">
      <NSteps :current="step" size="small">
        <NStep :title="t('certs.stepConnection')" />
        <NStep :title="t('certs.stepDomains')" />
        <NStep :title="t('certs.stepInstall')" />
        <NStep :title="t('certs.stepRun')" />
      </NSteps>

      <!-- Step 1: server + environment -->
      <div v-if="step === 1" class="flex flex-col gap-3 flex-1 min-h-0 overflow-y-auto pr-1">
        <NFormItem :label="t('portForward.connection')" label-placement="top">
          <NSelect v-model:value="connectionID" :options="connectionOptions" :placeholder="t('certs.connectionPlaceholder')" filterable />
        </NFormItem>
        <div v-if="detecting" class="text-[11px] text-[var(--text-secondary)] flex items-center gap-1.5">
          <IconLoader2 :width="12" :height="12" class="animate-spin" />
          {{ t('certs.detecting') }}
        </div>
        <template v-else-if="env">
          <div class="flex items-center gap-1.5 flex-wrap">
            <NTag :bordered="false" size="small" :type="env.installed ? 'success' : 'warning'">
              {{ env.installed ? t('certs.installed') : t('certs.notInstalled') }}
            </NTag>
            <NTag :bordered="false" size="small" :type="env.cron_present ? 'success' : 'warning'">
              {{ env.cron_present ? t('certs.cronOk') : t('certs.cronMissing') }}
            </NTag>
            <NTag v-if="!env.curl_present" :bordered="false" size="small" type="error">{{ t('certs.curlMissing') }}</NTag>
          </div>
          <div class="text-[11px] text-[var(--text-secondary)] break-all">{{ t('certs.homePath', { path: env.acme_sh_path }) }}</div>
        </template>
        <div v-if="detectError" class="text-[11px] text-[var(--color-error)]">{{ detectError }}</div>

        <template v-if="env && !env.installed">
          <NFormItem :label="t('certs.accountEmail')" label-placement="top">
            <NInput v-model:value="email" :placeholder="t('certs.accountEmailPlaceholder')"
              :input-props="{ autocapitalize: 'off', autocomplete: 'off', spellcheck: false }" />
          </NFormItem>
          <div>
            <NButton size="small" type="primary" :loading="installing" :disabled="!email" @click="installAcme">
              {{ installing ? t('certs.installing') : t('certs.installAcmeSh') }}
            </NButton>
          </div>
          <div v-if="installOpID" class="h-[140px]">
            <CertLogView :log-key="installOpID" />
          </div>
        </template>
      </div>

      <!-- Step 2: domains + DNS -->
      <div v-else-if="step === 2" class="flex flex-col gap-1 flex-1 min-h-0 overflow-y-auto pr-1">
        <NFormItem :label="t('certs.taskName')" label-placement="top">
          <NInput v-model:value="form.name" :placeholder="t('certs.taskNamePlaceholder')" />
        </NFormItem>
        <NFormItem :label="t('certs.primaryDomain')" label-placement="top">
          <NInput v-model:value="form.primaryDomain" :placeholder="t('certs.primaryDomainPlaceholder')"
            :input-props="{ autocapitalize: 'off', autocomplete: 'off', autocorrect: 'off', spellcheck: false }" />
        </NFormItem>
        <NFormItem :label="t('certs.sanDomains')" label-placement="top">
          <NInput v-model:value="form.sanDomainsText" type="textarea" :rows="2" :placeholder="t('certs.sanDomainsPlaceholder')" />
        </NFormItem>
        <NFormItem :label="t('certs.dnsProvider')" label-placement="top">
          <NSelect v-model:value="form.dnsProvider" :options="providerOptions" :placeholder="t('certs.dnsProviderPlaceholder')" />
        </NFormItem>
        <template v-if="form.dnsProvider === 'custom'">
          <NFormItem :label="t('certs.dnsPlugin')" label-placement="top">
            <NInput v-model:value="form.customPlugin" :placeholder="t('certs.dnsPluginPlaceholder')"
              :input-props="{ autocapitalize: 'off', autocomplete: 'off', autocorrect: 'off', spellcheck: false }" />
          </NFormItem>
          <div class="text-[11px] text-[var(--text-secondary)] mb-1">{{ t('certs.customPairs') }}</div>
          <div v-for="(pair, i) in form.customPairs" :key="i" class="flex items-center gap-1.5 mb-1.5">
            <NInput v-model:value="pair.key" :placeholder="t('certs.envKey')" size="small" class="w-[160px]"
              :input-props="{ autocapitalize: 'off', autocomplete: 'off', spellcheck: false }" />
            <NInput v-model:value="pair.value" :placeholder="t('certs.envValue')" size="small" type="password" show-password-on="click" />
            <NButton size="tiny" quaternary @click="form.customPairs.splice(i, 1)">✕</NButton>
          </div>
          <NButton size="tiny" dashed @click="form.customPairs.push({ key: '', value: '' })">{{ t('certs.addPair') }}</NButton>
        </template>
        <template v-else-if="selectedProvider">
          <NFormItem v-for="f in selectedProvider.fields" :key="f.key" :label="f.label + (f.required ? ' *' : '')" label-placement="top">
            <NInput v-model:value="form.creds[f.key]" :type="f.secret ? 'password' : 'text'"
              :show-password-on="f.secret ? 'click' : undefined" :placeholder="f.placeholder"
              :input-props="{ autocapitalize: 'off', autocomplete: 'off', autocorrect: 'off', spellcheck: false }" />
          </NFormItem>
        </template>
      </div>

      <!-- Step 3: deployment -->
      <div v-else-if="step === 3" class="flex flex-col gap-1 flex-1 min-h-0 overflow-y-auto pr-1">
        <NCheckbox v-model:checked="form.autoInstall">{{ t('certs.autoInstall') }}</NCheckbox>
        <template v-if="form.autoInstall">
          <NFormItem :label="t('certs.certDir')" label-placement="top">
            <NInput v-model:value="form.certDir" :placeholder="t('certs.certDirPlaceholder')" />
          </NFormItem>
          <NFormItem :label="t('certs.keyFile')" label-placement="top">
            <NInput v-model:value="form.keyFile" :input-props="{ autocapitalize: 'off', autocomplete: 'off', spellcheck: false }" />
          </NFormItem>
          <NFormItem :label="t('certs.fullchainFile')" label-placement="top">
            <NInput v-model:value="form.fullchainFile" :input-props="{ autocapitalize: 'off', autocomplete: 'off', spellcheck: false }" />
          </NFormItem>
          <NFormItem :label="t('certs.reloadPreset')" label-placement="top">
            <NSelect v-model:value="form.reloadPreset" :options="reloadPresets" @update:value="applyReloadPreset" />
          </NFormItem>
          <NFormItem v-if="form.reloadPreset !== 'none'" :label="t('certs.reloadCmd')" label-placement="top">
            <NInput v-model:value="form.reloadCmd" :disabled="form.reloadPreset !== 'custom'"
              :input-props="{ autocapitalize: 'off', autocomplete: 'off', spellcheck: false }" />
          </NFormItem>
        </template>
        <NCollapse class="mt-1">
          <NCollapseItem :title="t('certs.advanced')" name="advanced">
            <NFormItem :label="t('certs.keyLength')" label-placement="top">
              <NSelect v-model:value="form.keyLength" :options="keyLengthOptions" />
            </NFormItem>
            <NFormItem :label="t('certs.dnsSleep')" label-placement="top">
              <NInputNumber v-model:value="form.dnsSleep" :min="0" class="w-full" />
            </NFormItem>
            <NCheckbox v-model:checked="form.testMode">{{ t('certs.testMode') }}</NCheckbox>
          </NCollapseItem>
        </NCollapse>
      </div>

      <!-- Step 4: run -->
      <div v-else class="flex flex-col gap-2 flex-1 min-h-0">
        <template v-if="runTaskID">
          <div class="flex items-center gap-2 flex-wrap">
            <template v-if="runFinished === 'success'">
              <NTag :bordered="false" size="small" type="success">{{ t('certs.issued') }}</NTag>
            </template>
            <template v-else-if="runFinished === 'failed'">
              <NTag :bordered="false" size="small" type="error">{{ t('certs.statusFailed') }}</NTag>
            </template>
            <template v-else>
              <IconLoader2 :width="14" :height="14" class="animate-spin text-[var(--color-primary)]" />
              <span class="text-[var(--font-size-sm)] text-[var(--text-primary)]">{{ t('certs.issuing') }}</span>
              <NButton size="tiny" quaternary @click="cancelRun">{{ t('common.cancel') }}</NButton>
            </template>
          </div>
          <div class="flex flex-col gap-1 py-1">
            <div v-for="s in stages" :key="s" class="flex items-center gap-1.5 text-[11px]">
              <IconCheck v-if="stageState(s) === 'done'" :width="12" :height="12" class="text-[var(--color-success)]" />
              <IconLoader2 v-else-if="stageState(s) === 'active'" :width="12" :height="12" class="animate-spin text-[var(--color-primary)]" />
              <span v-else class="inline-block w-[12px] h-[12px] rounded-full border border-[var(--border-color)]" />
              <span :class="stageState(s) === 'pending' ? 'text-[var(--text-secondary)]' : 'text-[var(--text-primary)]'">
                {{ stageLabel(s) }}
                <span v-if="stageState(s) === 'active' && runFinished === 'failed'" class="text-[var(--color-error)]">✕</span>
              </span>
            </div>
          </div>
          <div class="flex-1 min-h-0">
            <CertLogView :log-key="runTaskID" />
          </div>
          <div v-if="currentTask?.last_error && runFinished === 'failed'" class="text-[11px] text-[var(--color-error)] whitespace-pre-wrap break-all max-h-[80px] overflow-y-auto">
            {{ currentTask.last_error }}
          </div>
        </template>
        <div v-else class="flex-center flex-1">
          <NButton type="primary" @click="startIssue">{{ t('certs.start') }}</NButton>
        </div>
      </div>
    </div>

    <template #footer>
      <NSpace justify="space-between">
        <NButton v-if="step > 1 && step < 4" @click="prev">{{ t('certs.prev') }}</NButton>
        <span v-else />
        <NSpace>
          <NButton v-if="step < 4" type="primary" @click="next">{{ t('certs.next') }}</NButton>
          <template v-if="step === 4">
            <NButton v-if="runFinished === 'failed'" type="primary" @click="retry">{{ t('certs.retry') }}</NButton>
            <NButton v-if="runFinished" type="primary" @click="finish">{{ t('certs.finish') }}</NButton>
            <NButton v-else @click="visible = false">{{ t('common.cancel') }}</NButton>
          </template>
        </NSpace>
      </NSpace>
    </template>
  </NModal>
</template>
