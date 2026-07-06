<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  NButton,
  NEmpty,
  NForm,
  NFormItem,
  NInput,
  NInputNumber,
  NModal,
  NSelect,
  NSpace,
  useDialog,
  useMessage,
} from 'naive-ui'
import IconNetwork from '~icons/lucide/network'
import IconPencil from '~icons/lucide/pencil'
import IconPlus from '~icons/lucide/plus'
import IconTrash2 from '~icons/lucide/trash-2'
import { newProxyFormData, useProxyStore, type ProxyFormData } from '../../stores/proxy'
import type { ProxyConfig } from '../../types'

const { t } = useI18n()
const proxyStore = useProxyStore()
const message = useMessage()
const dialog = useDialog()

const showModal = ref(false)
const editingID = ref<string | null>(null)
const form = reactive<ProxyFormData>(newProxyFormData())

const proxyTypeOptions = computed(() => [
  { label: 'HTTP', value: 'http' },
  { label: 'SOCKS5', value: 'socks5' },
])

const modalTitle = computed(() => editingID.value ? t('proxy.editProxy') : t('proxy.newProxy'))

onMounted(async () => {
  try {
    await proxyStore.loadProxies()
  } catch (e: any) {
    message.error(t('proxy.loadFailed', { error: e }))
  }
})

function resetForm() {
  const fresh = newProxyFormData()
  Object.assign(form, fresh)
  editingID.value = null
}

function openNew() {
  resetForm()
  showModal.value = true
}

function openEdit(proxy: ProxyConfig) {
  editingID.value = proxy.id
  form.id = proxy.id
  form.name = proxy.name
  form.type = proxy.type
  form.host = proxy.host
  form.port = proxy.port
  form.username = proxy.username
  form.password = ''
  showModal.value = true
}

async function saveProxy() {
  if (!form.name.trim()) { message.warning(t('proxy.nameRequired')); return }
  if (!form.host.trim()) { message.warning(t('proxy.hostRequired')); return }

  const data = {
    ...form,
    name: form.name.trim(),
    host: form.host.trim(),
    username: form.username.trim(),
  }

  try {
    if (editingID.value) {
      await proxyStore.updateProxy(data)
      message.success(t('proxy.updated'))
    } else {
      await proxyStore.createProxy(data)
      message.success(t('proxy.created'))
    }
    showModal.value = false
    resetForm()
  } catch (e: any) {
    message.error(t('proxy.saveFailed', { error: e }))
  }
}

function deleteProxy(proxy: ProxyConfig) {
  dialog.warning({
    title: t('proxy.deleteTitle'),
    content: t('proxy.deleteContent', { name: proxy.name }),
    positiveText: t('common.delete'),
    negativeText: t('common.cancel'),
    onPositiveClick: async () => {
      try {
        await proxyStore.deleteProxy(proxy.id)
        message.success(t('proxy.deleted'))
      } catch (e: any) {
        message.error(t('proxy.deleteFailed', { error: e }))
      }
    },
  })
}
</script>

<template>
  <div class="flex flex-col h-full overflow-hidden bg-[var(--bg-secondary)]">
    <div class="px-3 py-[10px] bg-[var(--bg-tertiary)] flex items-center justify-between shrink-0 thin-border-b">
      <span class="text-[var(--font-size-base)] font-semibold text-[var(--text-primary)]">{{ t('proxy.title') }}</span>
      <NButton size="tiny" quaternary :title="t('proxy.newProxy')" @click="openNew">
        <IconPlus :width="14" :height="14" />
      </NButton>
    </div>

    <div class="flex-1 overflow-y-auto p-3">
      <NEmpty v-if="!proxyStore.loading && proxyStore.proxies.length === 0" :description="t('proxy.empty')" size="small" />
      <div v-else class="proxy-list">
        <div v-for="proxy in proxyStore.proxies" :key="proxy.id" class="proxy-card">
          <div class="proxy-icon">
            <IconNetwork :width="18" :height="18" />
          </div>
          <div class="min-w-0 flex-1">
            <div class="proxy-name">{{ proxy.name }}</div>
            <div class="proxy-addr">{{ proxy.type.toUpperCase() }} {{ proxy.host }}:{{ proxy.port }}</div>
            <div v-if="proxy.username" class="proxy-user">{{ proxy.username }}</div>
          </div>
          <div class="proxy-actions">
            <button class="proxy-action-btn" :title="t('common.edit')" @click="openEdit(proxy)">
              <IconPencil :width="13" :height="13" />
            </button>
            <button class="proxy-action-btn danger" :title="t('common.delete')" @click="deleteProxy(proxy)">
              <IconTrash2 :width="13" :height="13" />
            </button>
          </div>
        </div>
      </div>
    </div>

    <NModal v-model:show="showModal" preset="card" :title="modalTitle" style="width: 420px" :mask-closable="false">
      <NForm label-placement="left" label-width="90">
        <NFormItem :label="t('common.name')">
          <NInput v-model:value="form.name" :placeholder="t('proxy.namePlaceholder')" />
        </NFormItem>
        <NFormItem label="Type">
          <NSelect v-model:value="form.type" :options="proxyTypeOptions" />
        </NFormItem>
        <NFormItem :label="t('common.host')">
          <NInput v-model:value="form.host" :placeholder="t('proxy.hostPlaceholder')" />
        </NFormItem>
        <NFormItem :label="t('common.port')">
          <NInputNumber v-model:value="form.port" :min="1" :max="65535" style="width: 100%" />
        </NFormItem>
        <NFormItem :label="t('common.username')">
          <NInput v-model:value="form.username" :placeholder="t('proxy.usernamePlaceholder')" />
        </NFormItem>
        <NFormItem :label="t('common.password')">
          <NInput v-model:value="form.password" type="password" show-password-on="click" :placeholder="editingID ? t('proxy.passwordEditPlaceholder') : t('proxy.passwordPlaceholder')" />
        </NFormItem>
      </NForm>
      <template #footer>
        <NSpace justify="end">
          <NButton @click="showModal = false">{{ t('common.cancel') }}</NButton>
          <NButton type="primary" @click="saveProxy">{{ t('common.save') }}</NButton>
        </NSpace>
      </template>
    </NModal>
  </div>
</template>

<style scoped>
.proxy-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.proxy-card {
  display: flex;
  align-items: center;
  gap: 10px;
  min-height: 62px;
  padding: 10px;
  border-radius: 8px;
  background: var(--bg-tertiary);
  border: 1px solid transparent;
  transition: border-color 0.15s ease, background 0.15s ease;
}

.proxy-card:hover {
  background: var(--hover-overlay);
  border-color: var(--border-color);
}

.proxy-icon {
  width: 38px;
  height: 38px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 8px;
  color: white;
  background: #4f46e5;
  flex-shrink: 0;
}

.proxy-name {
  color: var(--text-primary);
  font-size: var(--font-size-base);
  font-weight: 600;
  line-height: 1.35;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.proxy-addr,
.proxy-user {
  margin-top: 2px;
  color: var(--text-secondary);
  font-size: var(--font-size-sm);
  line-height: 1.3;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.proxy-actions {
  display: flex;
  align-items: center;
  gap: 2px;
  opacity: 0;
  transition: opacity 0.15s ease;
}

.proxy-card:hover .proxy-actions {
  opacity: 1;
}

.proxy-action-btn {
  width: 24px;
  height: 24px;
  display: flex;
  align-items: center;
  justify-content: center;
  border: 0;
  border-radius: 5px;
  background: transparent;
  color: var(--text-secondary);
  cursor: pointer;
}

.proxy-action-btn:hover {
  color: var(--text-primary);
  background: var(--hover-overlay);
}

.proxy-action-btn.danger:hover {
  color: var(--color-error);
}
</style>
