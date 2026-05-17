<script setup lang="ts">
import { ref, computed, watch } from 'vue'
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

const store = useConnectionStore()
const message = useMessage()

const isEdit = computed(() => !!props.editConnection)

const form = ref<ConnectionFormData>(newFormData())
const saving = ref(false)

const showModal = computed({
  get: () => props.show,
  set: (v) => emit('update:show', v),
})

const authTypeOptions = [
  { label: 'Password', value: AuthType.AuthPassword },
  { label: 'Private Key', value: AuthType.AuthPrivateKey },
  { label: 'SSH Agent', value: AuthType.AuthAgent },
  { label: 'Interactive', value: AuthType.AuthInteractive },
]

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
  if (!f.name.trim()) { message.warning('Name is required'); return }
  if (!f.host.trim()) { message.warning('Host is required'); return }

  saving.value = true
  try {
    if (isEdit.value) {
      await store.updateConnection(f)
      message.success('Connection updated')
    } else {
      await store.createConnection(f)
      message.success('Connection created')
    }
    showModal.value = false
  } catch (e: any) {
    message.error(`Failed: ${e}`)
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <NModal v-model:show="showModal" preset="card" :title="isEdit ? 'Edit Connection' : 'New SSH Connection'" style="width: 480px" :mask-closable="false">
    <NForm label-placement="left" label-width="90">
      <NFormItem label="Name">
        <NInput v-model:value="form.name" placeholder="My Server" />
      </NFormItem>
      <NFormItem label="Host">
        <NInput v-model:value="form.host" placeholder="192.168.1.1 or example.com" />
      </NFormItem>
      <NFormItem label="Port">
        <NInputNumber v-model:value="form.port" :min="1" :max="65535" style="width: 100%" />
      </NFormItem>
      <NFormItem label="Username">
        <NInput v-model:value="form.username" placeholder="root" />
      </NFormItem>
      <NDivider style="margin: 8px 0" />
      <NFormItem label="Auth Type">
        <NSelect v-model:value="form.authType" :options="authTypeOptions" />
      </NFormItem>
      <NFormItem v-if="form.authType === 'password' || form.authType === 'interactive'" :label="isEdit ? 'New Password' : 'Password'">
        <NInput v-model:value="form.password" type="password" show-password-on="click" :placeholder="isEdit ? 'Leave blank to keep current' : 'Password'" />
      </NFormItem>
      <template v-if="form.authType === 'private_key'">
        <NFormItem :label="isEdit ? 'New Private Key' : 'Private Key'">
          <NInput
            v-model:value="form.privateKey"
            type="textarea"
            :rows="4"
            :placeholder="isEdit ? 'Leave blank to keep current' : '-----BEGIN OPENSSH PRIVATE KEY-----\n...\n-----END OPENSSH PRIVATE KEY-----'"
          />
        </NFormItem>
        <NFormItem :label="isEdit ? 'New Passphrase' : 'Passphrase'">
          <NInput v-model:value="form.keyPassphrase" type="password" show-password-on="click" :placeholder="isEdit ? 'Leave blank to keep current' : 'Optional'" />
        </NFormItem>
      </template>
    </NForm>
    <template #footer>
      <NSpace justify="end">
        <NButton @click="handleClose">Cancel</NButton>
        <NButton type="primary" :loading="saving" @click="handleSave">{{ isEdit ? 'Update' : 'Save' }}</NButton>
      </NSpace>
    </template>
  </NModal>
</template>
