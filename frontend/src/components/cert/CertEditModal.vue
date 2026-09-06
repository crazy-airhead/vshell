<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
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
  useMessage,
} from 'naive-ui'
import IconEye from '~icons/lucide/eye'
import { useCertStore } from '../../stores/cert'
import type { CertTask, DNSProvider } from '../../types'

const props = defineProps<{
  show: boolean
  task: CertTask | null
}>()

const emit = defineEmits<{ (e: 'update:show', value: boolean): void }>()

const { t } = useI18n()
const message = useMessage()
const certStore = useCertStore()

const visible = computed({
  get: () => props.show,
  set: (v: boolean) => emit('update:show', v),
})

const saving = ref(false)
const revealing = ref(false)

const selectedProvider = computed<DNSProvider | null>(() =>
  certStore.providers.find(p => p.id === form.dnsProvider) ?? null
)

const keyLengthOptions = computed(() => [
  { label: t('certs.keyLengthEc256'), value: 'ec-256' },
  { label: t('certs.keyLengthRsa2048'), value: '2048' },
])

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
  certDir: '',
  keyFile: '',
  fullchainFile: '',
  reloadCmd: '',
  keyLength: 'ec-256',
  dnsSleep: 120,
  testMode: true,
})

watch(
  () => props.show,
  (open) => {
    if (!open || !props.task) return
    const task = props.task
    form.name = task.name
    form.primaryDomain = task.primary_domain
    form.sanDomainsText = (task.san_domains ?? []).join('\n')
    form.dnsProvider = task.dns_provider
    form.creds = {}
    form.customPlugin = task.dns_plugin ?? ''
    form.customPairs = [{ key: '', value: '' }]
    form.autoInstall = task.auto_install
    form.certDir = task.cert_dir ?? ''
    form.keyFile = task.key_file ?? ''
    form.fullchainFile = task.fullchain_file ?? ''
    form.reloadCmd = task.reload_cmd ?? ''
    form.keyLength = task.key_length
    form.dnsSleep = task.dns_sleep
    form.testMode = task.test_mode
    certStore.loadProviders()
  },
)

async function revealCredentials() {
  if (!props.task) return
  revealing.value = true
  try {
    const creds = await certStore.revealCredentials(props.task.id)
    if (props.task.dns_provider === 'custom') {
      form.customPairs = Object.entries(creds).map(([key, value]) => ({ key, value }))
    } else {
      for (const [k, v] of Object.entries(creds)) {
        form.creds[k] = v
      }
    }
  } catch (e: any) {
    message.error(String(e))
  } finally {
    revealing.value = false
  }
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

async function handleSave() {
  if (!props.task) return
  if (!form.primaryDomain.trim()) {
    message.warning(t('certs.errDomainRequired'))
    return
  }
  if (form.dnsProvider === 'custom' && !/^dns_[a-z0-9_]+$/.test(form.customPlugin)) {
    message.warning(t('certs.errPluginRequired'))
    return
  }
  if (form.dnsProvider !== 'custom' && selectedProvider.value) {
    for (const f of selectedProvider.value.fields) {
      if (f.required && !form.creds[f.key]) {
        message.warning(t('certs.errCredsRequired'))
        return
      }
    }
  }

  saving.value = true
  try {
    const sanDomains = form.sanDomainsText.split('\n').map(s => s.trim()).filter(s => s !== '')
    await certStore.updateTask({
      id: props.task.id,
      connection_id: props.task.connection_id,
      name: form.name.trim() || form.primaryDomain.trim(),
      primary_domain: form.primaryDomain.trim(),
      san_domains: sanDomains,
      dns_provider: form.dnsProvider,
      dns_plugin: form.dnsProvider === 'custom' ? form.customPlugin : '',
      // empty credentials mean "keep stored ones"
      dns_credentials: collectCredentials(),
      key_length: form.keyLength,
      dns_sleep: form.dnsSleep,
      test_mode: form.testMode,
      auto_install: form.autoInstall,
      cert_dir: form.autoInstall ? form.certDir.trim() : '',
      key_file: form.autoInstall ? form.keyFile.trim() : '',
      fullchain_file: form.autoInstall ? form.fullchainFile.trim() : '',
      reload_cmd: form.autoInstall ? form.reloadCmd.trim() : '',
    })
    message.success(t('common.update'))
    visible.value = false
  } catch (e: any) {
    message.error(t('certs.failed', { error: String(e) }))
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <NModal v-model:show="visible" preset="card" :title="t('common.edit')" style="width: 520px" :mask-closable="false">
    <div class="max-h-[60vh] overflow-y-auto pr-1">
      <NForm label-placement="top">
        <NFormItem :label="t('certs.taskName')">
          <NInput v-model:value="form.name" :placeholder="t('certs.taskNamePlaceholder')" />
        </NFormItem>
        <NFormItem :label="t('certs.primaryDomain')">
          <NInput v-model:value="form.primaryDomain" :input-props="{ autocapitalize: 'off', autocomplete: 'off', autocorrect: 'off', spellcheck: false }" />
        </NFormItem>
        <NFormItem :label="t('certs.sanDomains')">
          <NInput v-model:value="form.sanDomainsText" type="textarea" :rows="2" :placeholder="t('certs.sanDomainsPlaceholder')" />
        </NFormItem>
        <NFormItem :label="t('certs.dnsProvider')">
          <div class="w-full">
            <NSelect v-model:value="form.dnsProvider" :options="certStore.providers.map(p => ({ label: p.name, value: p.id }))" />
            <NButton size="tiny" quaternary class="mt-1" :loading="revealing" @click="revealCredentials">
              <template #icon><IconEye :width="14" :height="14" /></template>
              {{ t('certs.reveal') }}
            </NButton>
          </div>
        </NFormItem>
        <template v-if="form.dnsProvider === 'custom'">
          <NFormItem :label="t('certs.dnsPlugin')">
            <NInput v-model:value="form.customPlugin" :placeholder="t('certs.dnsPluginPlaceholder')"
              :input-props="{ autocapitalize: 'off', autocomplete: 'off', autocorrect: 'off', spellcheck: false }" />
          </NFormItem>
          <div v-for="(pair, i) in form.customPairs" :key="i" class="flex items-center gap-1.5 mb-1.5">
            <NInput v-model:value="pair.key" :placeholder="t('certs.envKey')" size="small" class="w-[150px]"
              :input-props="{ autocapitalize: 'off', autocomplete: 'off', spellcheck: false }" />
            <NInput v-model:value="pair.value" :placeholder="t('certs.envValue')" size="small" type="password" show-password-on="click" />
            <NButton size="tiny" quaternary @click="form.customPairs.splice(i, 1)">✕</NButton>
          </div>
          <NButton size="tiny" dashed class="mb-2" @click="form.customPairs.push({ key: '', value: '' })">{{ t('certs.addPair') }}</NButton>
        </template>
        <template v-else-if="selectedProvider">
          <NFormItem v-for="f in selectedProvider.fields" :key="f.key" :label="f.label + (f.required ? ' *' : '')">
            <NInput v-model:value="form.creds[f.key]" :type="f.secret ? 'password' : 'text'"
              :show-password-on="f.secret ? 'click' : undefined" :placeholder="f.placeholder"
              :input-props="{ autocapitalize: 'off', autocomplete: 'off', autocorrect: 'off', spellcheck: false }" />
          </NFormItem>
        </template>

        <NCheckbox v-model:checked="form.autoInstall" class="mb-2">{{ t('certs.autoInstall') }}</NCheckbox>
        <template v-if="form.autoInstall">
          <NFormItem :label="t('certs.certDir')">
            <NInput v-model:value="form.certDir" :placeholder="t('certs.certDirPlaceholder')" />
          </NFormItem>
          <NFormItem :label="t('certs.keyFile')">
            <NInput v-model:value="form.keyFile" :input-props="{ autocapitalize: 'off', autocomplete: 'off', spellcheck: false }" />
          </NFormItem>
          <NFormItem :label="t('certs.fullchainFile')">
            <NInput v-model:value="form.fullchainFile" :input-props="{ autocapitalize: 'off', autocomplete: 'off', spellcheck: false }" />
          </NFormItem>
          <NFormItem :label="t('certs.reloadCmd')">
            <NInput v-model:value="form.reloadCmd" :input-props="{ autocapitalize: 'off', autocomplete: 'off', spellcheck: false }" />
          </NFormItem>
        </template>

        <NCollapse>
          <NCollapseItem :title="t('certs.advanced')" name="advanced">
            <NFormItem :label="t('certs.keyLength')">
              <NSelect v-model:value="form.keyLength" :options="keyLengthOptions" />
            </NFormItem>
            <NFormItem :label="t('certs.dnsSleep')">
              <NInputNumber v-model:value="form.dnsSleep" :min="0" class="w-full" />
            </NFormItem>
            <NCheckbox v-model:checked="form.testMode">{{ t('certs.testMode') }}</NCheckbox>
          </NCollapseItem>
        </NCollapse>
      </NForm>
    </div>
    <template #footer>
      <NSpace justify="end">
        <NButton @click="visible = false">{{ t('common.cancel') }}</NButton>
        <NButton type="primary" :loading="saving" @click="handleSave">{{ t('common.update') }}</NButton>
      </NSpace>
    </template>
  </NModal>
</template>
