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
  useMessage,
} from 'naive-ui'
import { useConnectionStore, newFormData, AuthType } from '../../stores/connection'
import type { ConnectionFormData } from '../../stores/connection'
import type { Connection } from '../../types'

const props = defineProps<{
  show: boolean
  editConnection?: Connection | null
}>()
const emit = defineEmits<{ (e: 'update:show', val: boolean): void }>()

const { t } = useI18n()
const store = useConnectionStore()
const message = useMessage()

const isEdit = computed(() => !!props.editConnection)

const form = ref<ConnectionFormData>(newFormData())
const saving = ref(false)

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

const groupOptions = computed(() => {
  const opts = [{ label: '-', value: '' }]
  for (const g of store.groups) {
    opts.push({ label: g.name, value: g.id })
  }
  return opts
})

watch(
  () => props.show,
  (visible) => {
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
        }
      } else {
        form.value = newFormData()
      }
    }
  },
)

function handleClose() {
  showModal.value = false
}

async function handleSave() {
  const f = form.value
  if (!f.name.trim()) { message.warning(t('connection.nameRequired')); return }
  if (!f.host.trim()) { message.warning(t('connection.hostRequired')); return }

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
      <NDivider style="margin: 8px 0" />
      <NFormItem :label="t('connection.authType')">
        <NSelect v-model:value="form.authType" :options="authTypeOptions" />
      </NFormItem>
      <NFormItem v-if="form.authType === 'password' || form.authType === 'interactive'" :label="isEdit ? t('connection.newPassword') : t('common.password')">
        <NInput v-model:value="form.password" type="password" show-password-on="click" :placeholder="isEdit ? t('connection.passwordEditPlaceholder') : t('connection.passwordPlaceholder')" />
      </NFormItem>
      <template v-if="form.authType === 'private_key'">
        <NFormItem :label="isEdit ? t('connection.newPrivateKey') : t('connection.authPrivateKey')">
          <NInput
            v-model:value="form.privateKey"
            type="textarea"
            :rows="4"
            :placeholder="isEdit ? t('connection.keyEditPlaceholder') : t('connection.keyPlaceholder')"
          />
        </NFormItem>
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
</template>
