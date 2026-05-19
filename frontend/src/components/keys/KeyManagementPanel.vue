<script setup lang="ts">
import { ref, reactive, computed, onMounted, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  NModal,
  NForm,
  NFormItem,
  NInput,
  NInputNumber,
  NRadioGroup,
  NRadio,
  NButton,
  NSpace,
  NPopconfirm,
  NDropdown,
  useMessage,
} from 'naive-ui'
import { useSSHKeyStore } from '../../stores/sshkey'
import type { SSHKeyInfo } from '../../types'

const { t } = useI18n()
const store = useSSHKeyStore()
const message = useMessage()

// Create modal (paste key)
const showCreateModal = ref(false)
const createForm = reactive({ name: '', privateKey: '', publicKey: '' })
const creating = ref(false)

// Generate modal
const showGenerateModal = ref(false)
const genForm = reactive({ name: 'id_ed25519', keyType: 'ed25519', bits: 256, comment: '', passphrase: '' })
const generating = ref(false)

watch(() => genForm.keyType, (type) => {
  if (type === 'rsa') { genForm.bits = 4096; genForm.name = 'id_rsa' }
  else if (type === 'ecdsa') { genForm.bits = 256; genForm.name = 'id_ecdsa' }
  else { genForm.name = 'id_ed25519' }
})

const keyTypeOptions = [
  { label: 'Ed25519', value: 'ed25519' },
  { label: 'RSA', value: 'rsa' },
  { label: 'ECDSA', value: 'ecdsa' },
]

const rsaBitsOptions = [
  { label: '2048', value: 2048 },
  { label: '3072', value: 3072 },
  { label: '4096', value: 4096 },
]

const ecdsaBitsOptions = [
  { label: '256', value: 256 },
  { label: '384', value: 384 },
  { label: '521', value: 521 },
]

const showBits = computed(() => genForm.keyType === 'rsa' || genForm.keyType === 'ecdsa')
const bitsOptions = computed(() => genForm.keyType === 'rsa' ? rsaBitsOptions : ecdsaBitsOptions)

// Rename modal
const showRenameModal = ref(false)
const renameOldName = ref('')
const renameNewName = ref('')
const renaming = ref(false)

// Context menu
const ctxKey = ref('')
const ctxX = ref(0)
const ctxY = ref(0)

onMounted(() => {
  store.loadKeys()
})

function openCreate() {
  Object.assign(createForm, { name: '', privateKey: '', publicKey: '' })
  showCreateModal.value = true
}

function openGenerate() {
  Object.assign(genForm, { name: 'id_ed25519', keyType: 'ed25519', bits: 256, comment: '', passphrase: '' })
  showGenerateModal.value = true
}

function openRename(key: SSHKeyInfo) {
  renameOldName.value = key.name
  renameNewName.value = key.name
  showRenameModal.value = true
}

async function handleCreate() {
  const f = createForm
  if (!f.name.trim()) { message.warning(t('keys.nameRequired')); return }
  if (!f.privateKey.trim()) { message.warning(t('keys.keyRequired')); return }
  creating.value = true
  try {
    await store.saveKey(f.name, f.privateKey, f.publicKey)
    message.success(t('keys.created'))
    showCreateModal.value = false
  } catch (e: any) {
    message.error(t('keys.failed', { error: e }))
  } finally {
    creating.value = false
  }
}

async function handleGenerate() {
  const f = genForm
  if (!f.name.trim()) { message.warning(t('keys.nameRequired')); return }
  generating.value = true
  try {
    await store.generateKey(f.name, f.keyType, f.bits, f.comment, f.passphrase)
    message.success(t('keys.generated'))
    showGenerateModal.value = false
  } catch (e: any) {
    message.error(t('keys.failed', { error: e }))
  } finally {
    generating.value = false
  }
}

async function handleRename() {
  const newName = renameNewName.value.trim()
  if (!newName) { message.warning(t('keys.nameRequired')); return }
  renaming.value = true
  try {
    await store.renameKey(renameOldName.value, newName)
    message.success(t('keys.renamed'))
    showRenameModal.value = false
  } catch (e: any) {
    message.error(t('keys.failed', { error: e }))
  } finally {
    renaming.value = false
  }
}

async function handleDelete(key: SSHKeyInfo) {
  try {
    await store.deleteKey(key.name)
    message.success(t('keys.deleted', { name: key.name }))
  } catch (e: any) {
    message.error(t('keys.deleteFailed', { error: e }))
  }
}

async function copyKey(key: SSHKeyInfo, kind: string) {
  try {
    const content = await store.readContent(key.name, kind)
    await navigator.clipboard.writeText(content)
    message.success(t('keys.copied'))
  } catch (e: any) {
    message.error(t('keys.copyFailed'))
  }
}

function getContextMenu(key: SSHKeyInfo) {
  return [
    { label: t('keys.copyPub'), key: 'copyPub' },
    { label: t('keys.copyPrivate'), key: 'copyPrivate' },
    { type: 'divider', key: 'd1' },
    { label: t('keys.rename'), key: 'rename' },
    { label: t('common.delete'), key: 'delete' },
  ]
}

function handleContextMenu(key: SSHKeyInfo, action: string) {
  switch (action) {
    case 'copyPub': copyKey(key, 'pub'); break
    case 'copyPrivate': copyKey(key, 'private'); break
    case 'rename': openRename(key); break
    case 'delete': handleDelete(key); break
  }
}

function formatType(keyType: string): string {
  const map: Record<string, string> = {
    'ssh-rsa': 'RSA',
    'ssh-ed25519': 'Ed25519',
    'ecdsa-sha2-nistp256': 'ECDSA',
    'ecdsa-sha2-nistp384': 'ECDSA',
    'ecdsa-sha2-nistp521': 'ECDSA',
    'ssh-dss': 'DSA',
  }
  return map[keyType] || keyType
}
</script>

<template>
  <div class="key-panel">
    <div class="panel-header">
      <span class="panel-title">{{ t('keys.title') }}</span>
      <div class="panel-header-actions">
        <button class="panel-action-btn" @click="store.loadKeys()" :title="t('common.refresh')">
          <svg width="14" height="14" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5">
            <path d="M13.65 2.35A7.5 7.5 0 1 0 15.5 8.5" stroke-linecap="round" />
            <path d="M13.65 0.5v2.5h2.5" stroke-linecap="round" stroke-linejoin="round" />
          </svg>
        </button>
        <button class="panel-action-btn" @click="openGenerate" :title="t('keys.generateKey')">
          <svg width="14" height="14" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5">
            <circle cx="6" cy="6" r="4" />
            <path d="M9 9l5.5 5.5" stroke-linecap="round" />
            <path d="M12 14.5l1.5-1.5" stroke-linecap="round" />
            <path d="M5 5h2M6 4v2" stroke-linecap="round" />
          </svg>
        </button>
        <button class="panel-action-btn" @click="openCreate" :title="t('keys.newKey')">
          <svg width="14" height="14" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5">
            <path d="M2 8h12M8 2v12" stroke-linecap="round" />
          </svg>
        </button>
      </div>
    </div>

    <div class="key-list">
      <div v-if="store.keys.length === 0" class="panel-body-empty">
        <span class="empty-text">{{ t('keys.noKeys') }}</span>
      </div>

      <div
        v-for="key in store.keys"
        :key="key.name"
        class="key-item"
        @contextmenu.prevent="(e: MouseEvent) => { ctxX = e.clientX; ctxY = e.clientY; ctxKey = key.name }"
      >
        <NDropdown
          :options="getContextMenu(key)"
          trigger="manual"
          :x="ctxX"
          :y="ctxY"
          :show="ctxKey === key.name"
          @select="(action: string) => handleContextMenu(key, action)"
          @clickoutside="ctxKey = ''"
          placement="bottom-start"
        />
          <div class="key-item-main">
            <div class="key-item-row">
              <span v-if="key.type" class="key-badge">{{ formatType(key.type) }}</span>
              <span class="key-item-name">{{ key.name }}</span>
              <span v-if="key.has_passphrase" class="key-lock" :title="t('keys.passphraseProtected')">
                <svg width="10" height="10" viewBox="0 0 16 16" fill="currentColor">
                  <path d="M8 1a3 3 0 00-3 3v2H4a1 1 0 00-1 1v7a1 1 0 001 1h8a1 1 0 001-1V7a1 1 0 00-1-1h-1V4a3 3 0 00-3-3zm2 5H6V4a2 2 0 114 0v2z"/>
                </svg>
              </span>
            </div>
            <div class="key-item-meta">
              <span v-if="key.fingerprint" class="key-fingerprint">{{ key.fingerprint }}</span>
              <span v-if="key.comment" class="key-comment">{{ key.comment }}</span>
            </div>
          </div>
          <div class="key-item-actions">
            <button class="key-action-btn" :title="t('keys.copyPub')" @click.stop="copyKey(key, 'pub')">
              <svg width="12" height="12" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5">
                <rect x="5" y="5" width="9" height="9" rx="1" />
                <path d="M3 11V3a1 1 0 011-1h8" stroke-linecap="round" />
              </svg>
            </button>
            <button class="key-action-btn" @click.stop="openRename(key)" :title="t('keys.rename')">
              <svg width="12" height="12" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5">
                <path d="M11.5 1.5l3 3L5 14H2v-3L11.5 1.5z" />
              </svg>
            </button>
            <NPopconfirm @positive-click="handleDelete(key)">
              <template #trigger>
                <button class="key-action-btn key-action-danger" :title="t('common.delete')" @click.stop>
                  <svg width="12" height="12" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5">
                    <path d="M3 4h10M5 4V3a1 1 0 011-1h4a1 1 0 011 1v1M6 7v4M10 7v4M4 4l.7 9.4a1 1 0 001 .6h4.6a1 1 0 001-.6L12 4" stroke-linecap="round" stroke-linejoin="round" />
                  </svg>
                </button>
              </template>
              {{ t('keys.deleteContent', { name: key.name }) }}
            </NPopconfirm>
          </div>
        </div>
      </div>

    <!-- Create (paste) key modal -->
    <NModal v-model:show="showCreateModal" preset="card" :title="t('keys.newKey')" style="width: 480px" :mask-closable="false">
      <NForm label-placement="left" label-width="90">
        <NFormItem :label="t('keys.fileName')">
          <NInput v-model:value="createForm.name" :placeholder="t('keys.fileNamePlaceholder')" />
        </NFormItem>
        <NFormItem :label="t('keys.privateKey')">
          <NInput v-model:value="createForm.privateKey" type="textarea" :rows="6" :placeholder="t('keys.keyPlaceholder')" />
        </NFormItem>
        <NFormItem :label="t('keys.publicKey')">
          <NInput v-model:value="createForm.publicKey" type="textarea" :rows="2" :placeholder="'ssh-ed25519 AAAA...'" />
        </NFormItem>
      </NForm>
      <template #footer>
        <NSpace justify="end">
          <NButton @click="showCreateModal = false">{{ t('common.cancel') }}</NButton>
          <NButton type="primary" :loading="creating" @click="handleCreate">{{ t('common.save') }}</NButton>
        </NSpace>
      </template>
    </NModal>

    <!-- Generate key modal -->
    <NModal v-model:show="showGenerateModal" preset="card" :title="t('keys.generateKey')" style="width: 480px" :mask-closable="false">
      <NForm label-placement="left" label-width="90">
        <NFormItem :label="t('keys.keyType')">
          <NRadioGroup v-model:value="genForm.keyType">
            <NSpace>
              <NRadio value="ed25519">{{ t('keys.keyTypeEd25519') }}</NRadio>
              <NRadio value="rsa">{{ t('keys.keyTypeRsa') }}</NRadio>
              <NRadio value="ecdsa">{{ t('keys.keyTypeEcdsa') }}</NRadio>
            </NSpace>
          </NRadioGroup>
        </NFormItem>
        <NFormItem v-if="showBits" :label="t('keys.bits')">
          <NRadioGroup v-model:value="genForm.bits">
            <NSpace>
              <NRadio v-for="opt in bitsOptions" :key="opt.value" :value="opt.value">{{ opt.label }}</NRadio>
            </NSpace>
          </NRadioGroup>
        </NFormItem>
        <NFormItem :label="t('keys.fileName')">
          <NInput v-model:value="genForm.name" :placeholder="t('keys.fileNamePlaceholder')" />
        </NFormItem>
        <NFormItem :label="t('keys.comment')">
          <NInput v-model:value="genForm.comment" :placeholder="t('keys.commentPlaceholder')" />
        </NFormItem>
        <NFormItem :label="t('keys.passphrase')">
          <NInput v-model:value="genForm.passphrase" type="password" show-password-on="click" :placeholder="t('keys.passphrasePlaceholder')" />
        </NFormItem>
      </NForm>
      <template #footer>
        <NSpace justify="end">
          <NButton @click="showGenerateModal = false">{{ t('common.cancel') }}</NButton>
          <NButton type="primary" :loading="generating" @click="handleGenerate">{{ t('keys.generateKey') }}</NButton>
        </NSpace>
      </template>
    </NModal>

    <!-- Rename modal -->
    <NModal v-model:show="showRenameModal" preset="card" :title="t('keys.rename')" style="width: 360px" :mask-closable="false">
      <NForm label-placement="left" label-width="90">
        <NFormItem :label="t('keys.fileName')">
          <NInput v-model:value="renameNewName" :placeholder="t('keys.fileNamePlaceholder')" @keydown.enter="handleRename" />
        </NFormItem>
      </NForm>
      <template #footer>
        <NSpace justify="end">
          <NButton @click="showRenameModal = false">{{ t('common.cancel') }}</NButton>
          <NButton type="primary" :loading="renaming" @click="handleRename">{{ t('common.save') }}</NButton>
        </NSpace>
      </template>
    </NModal>
  </div>
</template>

<style scoped>
.key-panel {
  display: flex;
  flex-direction: column;
  height: 100%;
  overflow: hidden;
  background: var(--bg-secondary);
}

.panel-header {
  padding: 10px 12px;
  background: var(--bg-tertiary);
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-shrink: 0;
}

.panel-header-actions {
  display: flex;
  align-items: center;
  gap: 2px;
}

.panel-title {
  font-size: var(--font-size-base);
  font-weight: 600;
  color: var(--text-primary);
}

.panel-action-btn {
  background: none;
  border: none;
  color: var(--text-secondary);
  font-size: 16px;
  cursor: pointer;
  padding: 2px 8px;
  border-radius: 3px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.panel-action-btn:hover {
  color: var(--text-primary);
  background: var(--hover-overlay);
}

.key-list {
  flex: 1;
  overflow-y: auto;
  padding: 4px 0;
}

.key-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 12px;
  gap: 8px;
  cursor: default;
  transition: background 0.15s;
}

.key-item:hover {
  background: var(--hover-overlay);
}

.key-item-main {
  flex: 1;
  min-width: 0;
}

.key-item-row {
  display: flex;
  align-items: center;
  gap: 6px;
}

.key-badge {
  font-size: 10px;
  padding: 1px 5px;
  border-radius: 3px;
  background: var(--accent-color);
  color: #fff;
  font-weight: 600;
  line-height: 1.4;
  flex-shrink: 0;
}

.key-item-name {
  font-size: var(--font-size-base);
  color: var(--text-primary);
  font-weight: 500;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.key-lock {
  color: var(--text-secondary);
  flex-shrink: 0;
  display: flex;
  align-items: center;
}

.key-item-meta {
  display: flex;
  flex-direction: column;
  gap: 1px;
  margin-top: 2px;
}

.key-comment {
  font-size: 11px;
  color: var(--text-secondary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.key-fingerprint {
  font-size: 11px;
  color: var(--text-secondary);
  font-family: var(--font-family);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  opacity: 0.7;
}

.key-item-actions {
  display: flex;
  gap: 2px;
  flex-shrink: 0;
  opacity: 0;
  transition: opacity 0.15s;
}

.key-item:hover .key-item-actions {
  opacity: 1;
}

.key-action-btn {
  background: none;
  border: none;
  color: var(--text-secondary);
  cursor: pointer;
  padding: 4px 6px;
  border-radius: 3px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.key-action-btn:hover {
  color: var(--text-primary);
  background: var(--hover-overlay);
}

.key-action-danger:hover {
  color: #e74c3c;
}

.panel-body-empty {
  padding: 40px 12px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.empty-text {
  font-size: var(--font-size-sm);
  color: var(--text-secondary);
}
</style>
