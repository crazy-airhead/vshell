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
  useMessage,
} from 'naive-ui'
import { useConnectionStore, newFormData, AuthType } from '../../stores/connection'
import { useSSHKeyStore } from '../../stores/sshkey'
import { GetPassword, GetPrivateKey, GetKeyPassphrase } from '../../../bindings/vshell/internal/app/appservice'
import IconKey from '~icons/lucide/key'
import type { ConnectionFormData } from '../../stores/connection'
import type { Connection } from '../../types'

const props = defineProps<{
  show: boolean
  editConnection?: Connection | null
  defaultGroupID?: string | null
  prefillData?: ConnectionFormData | null
}>()
const emit = defineEmits<{ (e: 'update:show', val: boolean): void }>()

const { t } = useI18n()
const store = useConnectionStore()
const sshKeyStore = useSSHKeyStore()
const message = useMessage()

const isEdit = computed(() => !!props.editConnection)

const form = ref<ConnectionFormData>(newFormData())
const saving = ref(false)
const keySource = ref<'managed' | 'manual'>('managed')
const selectedKeyName = ref<string | null>(null)
const loadingKey = ref(false)
const revealingPassword = ref(false)
const passwordVisible = ref(false)
const revealingPrivateKey = ref(false)
const privateKeyVisible = ref(false)
const revealingPassphrase = ref(false)
const passphraseVisible = ref(false)
// Which managed key (if any) the stored private key matches — checked after reveal
const keyMatchChecked = ref(false)
const keyMatchName = ref<string | null>(null)

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

watch(
  () => props.show,
  async (visible) => {
    if (visible) {
      passwordVisible.value = false
      privateKeyVisible.value = false
      passphraseVisible.value = false
      keyMatchChecked.value = false
      keyMatchName.value = null
      // Load keys first: the edit branch needs the key store to validate key_name
      await sshKeyStore.loadKeys()
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
          keyName: c.key_name || null,
          groupID: c.group_id || null,
        }
        // Restore the managed-key source when the stored key_name still exists
        // in the key store; otherwise fall back to manual (round 1's reveal +
        // content matching still answers "which key" for legacy rows).
        if (c.auth_type === 'private_key' && c.key_name && sshKeyStore.keys.some((k) => k.name === c.key_name)) {
          keySource.value = 'managed'
          selectedKeyName.value = c.key_name
        } else {
          keySource.value = 'manual'
          selectedKeyName.value = null
        }
      } else if (props.prefillData) {
        form.value = { ...props.prefillData }
        keySource.value = 'manual'
        selectedKeyName.value = null
      } else {
        form.value = newFormData()
        if (props.defaultGroupID) {
          form.value.groupID = props.defaultGroupID
        }
        keySource.value = 'managed'
        selectedKeyName.value = null
      }
    }
  },
)

async function onManagedKeyChange(name: string | null) {
  if (!name) {
    form.value.privateKey = ''
    selectedKeyName.value = null
    return
  }
  selectedKeyName.value = name
  // The reveal-match hint describes the previously stored key; it's stale once
  // a different key is picked.
  keyMatchChecked.value = false
  keyMatchName.value = null
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

async function toggleRevealPrivateKey() {
  if (privateKeyVisible.value) {
    form.value.privateKey = ''
    privateKeyVisible.value = false
    keyMatchChecked.value = false
    keyMatchName.value = null
    return
  }
  revealingPrivateKey.value = true
  try {
    const content = await GetPrivateKey(form.value.id)
    form.value.privateKey = content
    privateKeyVisible.value = true
    await detectManagedKeyMatch(content)
  } catch (e: any) {
    message.error(t('connection.failed', { error: e }))
  } finally {
    revealingPrivateKey.value = false
  }
}

async function toggleRevealPassphrase() {
  if (passphraseVisible.value) {
    form.value.keyPassphrase = ''
    passphraseVisible.value = false
    return
  }
  revealingPassphrase.value = true
  try {
    form.value.keyPassphrase = await GetKeyPassphrase(form.value.id)
    passphraseVisible.value = true
  } catch (e: any) {
    message.error(t('connection.failed', { error: e }))
  } finally {
    revealingPassphrase.value = false
  }
}

// Compare the stored key against managed keys (~/.ssh) by content, so the
// reveal can also answer "which key file is this connection using".
async function detectManagedKeyMatch(stored: string) {
  keyMatchChecked.value = false
  keyMatchName.value = null
  try {
    if (!sshKeyStore.keys.length) await sshKeyStore.loadKeys()
    const target = stored.trim()
    for (const k of sshKeyStore.keys) {
      try {
        const content = await sshKeyStore.readContent(k.name, 'private')
        if (content.trim() === target) {
          keyMatchName.value = k.name
          break
        }
      } catch {
        // skip unreadable key
      }
    }
  } finally {
    keyMatchChecked.value = true
  }
}

function handleClose() {
  passwordVisible.value = false
  privateKeyVisible.value = false
  passphraseVisible.value = false
  keyMatchChecked.value = false
  keyMatchName.value = null
  form.value.password = ''
  showModal.value = false
}

async function handleSave() {
  const f = form.value
  if (!f.name.trim()) { message.warning(t('connection.nameRequired')); return }
  if (!f.host.trim()) { message.warning(t('connection.hostRequired')); return }
  if (!isEdit.value && f.authType === 'private_key' && !f.privateKey.trim()) {
    message.warning(t('connection.keyRequired'))
    return
  }
  // The managed-key association follows the key source UI at save time:
  // managed + selected name keeps/updates it, anything else clears it.
  f.keyName = keySource.value === 'managed' && selectedKeyName.value ? selectedKeyName.value : null

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
        <NInput v-model:value="form.name" :placeholder="t('connection.namePlaceholder')" :input-props="{ autocapitalize: 'off', autocomplete: 'off', autocorrect: 'off', spellcheck: false }" />
      </NFormItem>
      <NFormItem :label="t('connection.group')">
        <NSelect v-model:value="form.groupID" :options="groupOptions" clearable />
      </NFormItem>
      <NFormItem :label="t('common.host')">
        <NInput v-model:value="form.host" :placeholder="t('connection.hostPlaceholder')" :input-props="{ autocapitalize: 'off', autocomplete: 'off', autocorrect: 'off', spellcheck: false }" />
      </NFormItem>
      <NFormItem :label="t('common.port')">
        <NInputNumber v-model:value="form.port" :min="1" :max="65535" style="width: 100%" />
      </NFormItem>
      <NFormItem :label="t('common.username')">
        <NInput v-model:value="form.username" :placeholder="t('connection.usernamePlaceholder')" :input-props="{ autocapitalize: 'off', autocomplete: 'off', autocorrect: 'off', spellcheck: false }" />
      </NFormItem>
      <NDivider style="margin: 8px 0" />
      <NFormItem :label="t('connection.authType')">
        <NSelect v-model:value="form.authType" :options="authTypeOptions" />
      </NFormItem>
      <NFormItem v-if="form.authType === 'password' || form.authType === 'interactive'" :label="isEdit ? t('connection.newPassword') : t('common.password')">
        <div style="display: flex; align-items: center; gap: 4px; width: 100%">
          <NInput v-model:value="form.password" type="password" show-password-on="click" :placeholder="isEdit ? t('connection.passwordEditPlaceholder') : t('connection.passwordPlaceholder')" :input-props="{ autocapitalize: 'off', autocomplete: 'off', autocorrect: 'off', spellcheck: false }" style="flex: 1" />
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
            <div style="display: flex; align-items: flex-start; gap: 4px; width: 100%">
              <NInput
                v-model:value="form.privateKey"
                type="textarea"
                :rows="4"
                :placeholder="isEdit ? t('connection.keyEditPlaceholder') : t('connection.keyPlaceholder')"
                :input-props="{ autocapitalize: 'off', autocomplete: 'off', autocorrect: 'off', spellcheck: false }"
                style="flex: 1"
              />
              <NButton v-if="isEdit" :type="privateKeyVisible ? 'primary' : 'default'" :ghost="true" :loading="revealingPrivateKey" @click="toggleRevealPrivateKey" style="flex-shrink: 0; height: 34px; width: 34px">
                <template #icon>
                  <IconKey :width="16" :height="16" />
                </template>
              </NButton>
            </div>
          </NFormItem>
          <NFormItem v-if="privateKeyVisible && keyMatchChecked" label=" " style="margin-top: -16px">
            <span style="color: var(--text-secondary); font-size: 12px">
              {{ keyMatchName ? t('connection.keyMatchFound', { name: keyMatchName }) : t('connection.keyMatchNone') }}
            </span>
          </NFormItem>
        </template>
        <NFormItem :label="isEdit ? t('connection.newPassphrase') : t('connection.passphrase')">
          <div style="display: flex; align-items: center; gap: 4px; width: 100%">
            <NInput v-model:value="form.keyPassphrase" type="password" show-password-on="click" :placeholder="isEdit ? t('connection.passphraseEditPlaceholder') : t('connection.passphrasePlaceholder')" :input-props="{ autocapitalize: 'off', autocomplete: 'off', autocorrect: 'off', spellcheck: false }" style="flex: 1" />
            <NButton v-if="isEdit" :type="passphraseVisible ? 'primary' : 'default'" :ghost="true" :loading="revealingPassphrase" @click="toggleRevealPassphrase" style="flex-shrink: 0; height: 34px; width: 34px">
              <template #icon>
                <IconKey :width="16" :height="16" />
              </template>
            </NButton>
          </div>
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
</template>
