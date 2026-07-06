<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  NModal,
  NForm,
  NFormItem,
  NInput,
  NInputNumber,
  NSelect,
  NButton,
  NSpace,
  NDivider,
  NRadioGroup,
  NRadio,
  NCheckbox,
  useMessage,
} from 'naive-ui'
import { useConnectionStore, newFormData, AuthType } from '../../stores/connection'
import { useSSHKeyStore } from '../../stores/sshkey'
import { useProxyStore, newProxyFormData, type ProxyFormData } from '../../stores/proxy'
import { GetPassword } from '../../../bindings/vshell/internal/app/appservice'
import IconKey from '~icons/lucide/key'
import IconPlus from '~icons/lucide/plus'
import type { ConnectionFormData } from '../../stores/connection'
import type { Connection } from '../../types'

const props = defineProps<{
  show: boolean
  editConnection?: Connection | null
  defaultGroupID?: string | null
}>()
const emit = defineEmits<{ (e: 'update:show', val: boolean): void }>()

const { t } = useI18n()
const store = useConnectionStore()
const sshKeyStore = useSSHKeyStore()
const proxyStore = useProxyStore()
const message = useMessage()

const isEdit = computed(() => !!props.editConnection)

const form = ref<ConnectionFormData>(newFormData())
const saving = ref(false)
const keySource = ref<'managed' | 'manual'>('managed')
const selectedKeyName = ref<string | null>(null)
const loadingKey = ref(false)
const revealingPassword = ref(false)
const passwordVisible = ref(false)
const useProxy = ref(false)
const showProxyModal = ref(false)
const proxyForm = ref<ProxyFormData>(newProxyFormData())

const showModal = computed({
  get: () => props.show,
  set: (v) => emit('update:show', v),
})

const authTypeOptions = computed(() => [
  { label: t('connection.authPassword'), value: AuthType.AuthPassword },
  { label: t('connection.authPrivateKey'), value: AuthType.AuthPrivateKey },
  { label: t('connection.authAgent'), value: AuthType.AuthAgent },
  { label: t('connection.authInteractive'), value: AuthType.AuthInteractive },
])

const managedKeyOptions = computed(() => {
  if (!sshKeyStore.keys.length) return []
  return sshKeyStore.keys.map((k) => ({
    label: `${k.name} (${k.type}${k.comment ? ' - ' + k.comment : ''})`,
    value: k.name,
  }))
})

const selectedKeyHasPassphrase = computed(() => {
  if (!selectedKeyName.value) return false
  const key = sshKeyStore.keys.find((k) => k.name === selectedKeyName.value)
  return key?.has_passphrase ?? false
})

const groupOptions = computed(() => {
  const opts = [{ label: '-', value: '' }]
  for (const g of store.groups) {
    opts.push({ label: g.name, value: g.id })
  }
  return opts
})

const proxyOptions = computed(() => proxyStore.proxies.map((proxy) => ({
  label: `${proxy.name} (${proxy.type.toUpperCase()} ${proxy.host}:${proxy.port})`,
  value: proxy.id,
})))

const proxyTypeOptions = computed(() => [
  { label: 'HTTP', value: 'http' },
  { label: 'SOCKS5', value: 'socks5' },
])

watch(
  () => props.show,
  async (visible) => {
    if (visible) {
      if (props.editConnection) {
        const c = props.editConnection
        form.value = {
          id: c.id,
          name: c.name,
          host: c.host,
          port: c.port,
          username: c.username,
          authType: c.auth_type || AuthType.AuthPassword,
          password: '',
          privateKey: '',
          keyPassphrase: '',
          groupID: c.group_id || null,
          proxyID: c.proxy_addr || null,
        }
        useProxy.value = !!c.proxy_addr
        keySource.value = 'manual'
        selectedKeyName.value = null
      } else {
        form.value = newFormData()
        if (props.defaultGroupID) {
          form.value.groupID = props.defaultGroupID
        }
        useProxy.value = false
        keySource.value = 'managed'
        selectedKeyName.value = null
      }
      await Promise.all([sshKeyStore.loadKeys(), proxyStore.loadProxies()])
    }
  },
)

watch(useProxy, (enabled) => {
  if (!enabled) {
    form.value.proxyID = null
  }
})

async function onManagedKeyChange(name: string | null) {
  if (!name) {
    form.value.privateKey = ''
    selectedKeyName.value = null
    return
  }
  selectedKeyName.value = name
  loadingKey.value = true
  try {
    const content = await sshKeyStore.readContent(name, 'priv')
    form.value.privateKey = content
  } catch (e: any) {
    message.error(t('connection.keyReadFailed', { error: e }))
  } finally {
    loadingKey.value = false
  }
}

async function toggleRevealPassword() {
  if (passwordVisible.value) {
    form.value.password = ''
    passwordVisible.value = false
    return
  }
  revealingPassword.value = true
  try {
    form.value.password = await GetPassword(form.value.id)
    passwordVisible.value = true
  } catch (e: any) {
    message.error(t('connection.failed', { error: e }))
  } finally {
    revealingPassword.value = false
  }
}

function openProxyModal() {
  proxyForm.value = newProxyFormData()
  showProxyModal.value = true
}

async function saveProxy() {
  const p = proxyForm.value
  if (!p.name.trim()) { message.warning(t('proxy.nameRequired')); return }
  if (!p.host.trim()) { message.warning(t('proxy.hostRequired')); return }

  try {
    await proxyStore.createProxy({
      ...p,
      name: p.name.trim(),
      host: p.host.trim(),
      username: p.username.trim(),
    })
    form.value.proxyID = p.id
    useProxy.value = true
    showProxyModal.value = false
    message.success(t('proxy.created'))
  } catch (e: any) {
    message.error(t('proxy.saveFailed', { error: e }))
  }
}

function handleClose() {
  passwordVisible.value = false
  form.value.password = ''
  showModal.value = false
}

async function handleSave() {
  const f = form.value
  if (!f.name.trim()) { message.warning(t('connection.nameRequired')); return }
  if (!f.host.trim()) { message.warning(t('connection.hostRequired')); return }
  if (f.authType === 'private_key' && !f.privateKey.trim()) {
    message.warning(t('connection.keyRequired'))
    return
  }
  if (useProxy.value && !f.proxyID) {
    message.warning(t('proxy.selectRequired'))
    return
  }

  saving.value = true
  try {
    if (isEdit.value) {
      await store.updateConnection(f)
      message.success(t('connection.updated'))
    } else {
      await store.createConnection(f)
      message.success(t('connection.created'))
    }
    showModal.value = false
  } catch (e: any) {
    message.error(t('connection.failed', { error: e }))
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <NModal v-model:show="showModal" preset="card" :title="isEdit ? t('connection.editConnection') : t('connection.newConnection')" style="width: 480px" :mask-closable="false">
    <NForm label-placement="left" label-width="90">
      <NFormItem :label="t('common.name')">
        <NInput v-model:value="form.name" :placeholder="t('connection.namePlaceholder')" />
      </NFormItem>
      <NFormItem :label="t('connection.group')">
        <NSelect v-model:value="form.groupID" :options="groupOptions" clearable />
      </NFormItem>
      <NFormItem :label="t('common.host')">
        <NInput v-model:value="form.host" :placeholder="t('connection.hostPlaceholder')" />
      </NFormItem>
      <NFormItem :label="t('common.port')">
        <NInputNumber v-model:value="form.port" :min="1" :max="65535" style="width: 100%" />
      </NFormItem>
      <NFormItem :label="t('common.username')">
        <NInput v-model:value="form.username" :placeholder="t('connection.usernamePlaceholder')" />
      </NFormItem>
      <NFormItem :label="t('proxy.useProxy')">
        <div style="display: flex; align-items: center; gap: 8px; width: 100%">
          <NCheckbox v-model:checked="useProxy" />
          <NSelect
            v-model:value="form.proxyID"
            :options="proxyOptions"
            :placeholder="t('proxy.selectProxy')"
            :disabled="!useProxy"
            :loading="proxyStore.loading"
            clearable
            filterable
            style="flex: 1"
          />
          <NButton size="small" quaternary :title="t('proxy.newProxy')" @click="openProxyModal">
            <IconPlus :width="14" :height="14" />
          </NButton>
        </div>
      </NFormItem>
      <NDivider style="margin: 8px 0" />
      <NFormItem :label="t('connection.authType')">
        <NSelect v-model:value="form.authType" :options="authTypeOptions" />
      </NFormItem>
      <NFormItem v-if="form.authType === 'password' || form.authType === 'interactive'" :label="isEdit ? t('connection.newPassword') : t('common.password')">
        <div style="display: flex; align-items: center; gap: 4px; width: 100%">
          <NInput v-model:value="form.password" type="password" show-password-on="click" :placeholder="isEdit ? t('connection.passwordEditPlaceholder') : t('connection.passwordPlaceholder')" style="flex: 1" />
          <NButton v-if="isEdit" :type="passwordVisible ? 'primary' : 'default'" :ghost="true" :loading="revealingPassword" @click="toggleRevealPassword" style="flex-shrink: 0; height: 34px; width: 34px">
            <template #icon>
              <IconKey :width="16" :height="16" />
            </template>
          </NButton>
        </div>
      </NFormItem>
      <template v-if="form.authType === 'private_key'">
        <NFormItem :label="t('connection.keySource')">
          <NRadioGroup v-model:value="keySource" name="keySource">
            <NSpace>
              <NRadio value="managed">{{ t('connection.keyFromManaged') }}</NRadio>
              <NRadio value="manual">{{ t('connection.keyManual') }}</NRadio>
            </NSpace>
          </NRadioGroup>
        </NFormItem>
        <template v-if="keySource === 'managed'">
          <NFormItem :label="t('connection.selectKey')">
            <NSelect
              v-model:value="selectedKeyName"
              :options="managedKeyOptions"
              :placeholder="t('connection.selectKeyPlaceholder')"
              :loading="sshKeyStore.keys.length === 0"
              clearable
              filterable
              @update:value="onManagedKeyChange"
            />
          </NFormItem>
          <NFormItem v-if="selectedKeyHasPassphrase" label=" " style="margin-top: -16px">
            <span style="color: var(--warning-color); font-size: 12px">
              {{ t('connection.keyHasPassphrase') }}
            </span>
          </NFormItem>
        </template>
        <template v-else>
          <NFormItem :label="isEdit ? t('connection.newPrivateKey') : t('connection.authPrivateKey')">
            <NInput
              v-model:value="form.privateKey"
              type="textarea"
              :rows="4"
              :placeholder="isEdit ? t('connection.keyEditPlaceholder') : t('connection.keyPlaceholder')"
            />
          </NFormItem>
        </template>
        <NFormItem :label="isEdit ? t('connection.newPassphrase') : t('connection.passphrase')">
          <NInput v-model:value="form.keyPassphrase" type="password" show-password-on="click" :placeholder="isEdit ? t('connection.passphraseEditPlaceholder') : t('connection.passphrasePlaceholder')" />
        </NFormItem>
      </template>
    </NForm>
    <template #footer>
      <NSpace justify="end">
        <NButton @click="handleClose">{{ t('common.cancel') }}</NButton>
        <NButton type="primary" :loading="saving" @click="handleSave">{{ isEdit ? t('common.update') : t('common.save') }}</NButton>
      </NSpace>
    </template>
  </NModal>

  <NModal v-model:show="showProxyModal" preset="card" :title="t('proxy.newProxy')" style="width: 420px" :mask-closable="false">
    <NForm label-placement="left" label-width="90">
      <NFormItem :label="t('common.name')">
        <NInput v-model:value="proxyForm.name" :placeholder="t('proxy.namePlaceholder')" />
      </NFormItem>
      <NFormItem label="Type">
        <NSelect v-model:value="proxyForm.type" :options="proxyTypeOptions" />
      </NFormItem>
      <NFormItem :label="t('common.host')">
        <NInput v-model:value="proxyForm.host" :placeholder="t('proxy.hostPlaceholder')" />
      </NFormItem>
      <NFormItem :label="t('common.port')">
        <NInputNumber v-model:value="proxyForm.port" :min="1" :max="65535" style="width: 100%" />
      </NFormItem>
      <NFormItem :label="t('common.username')">
        <NInput v-model:value="proxyForm.username" :placeholder="t('proxy.usernamePlaceholder')" />
      </NFormItem>
      <NFormItem :label="t('common.password')">
        <NInput v-model:value="proxyForm.password" type="password" show-password-on="click" :placeholder="t('proxy.passwordPlaceholder')" />
      </NFormItem>
    </NForm>
    <template #footer>
      <NSpace justify="end">
        <NButton @click="showProxyModal = false">{{ t('common.cancel') }}</NButton>
        <NButton type="primary" @click="saveProxy">{{ t('common.save') }}</NButton>
      </NSpace>
    </template>
  </NModal>
</template>
