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
import IconRefreshCw from '~icons/lucide/refresh-cw'
import IconKeyRound from '~icons/lucide/key-round'
import IconPlus from '~icons/lucide/plus'
import IconLock from '~icons/lucide/lock'
import IconCopy from '~icons/lucide/copy'
import IconPencil from '~icons/lucide/pencil'
import IconTrash2 from '~icons/lucide/trash-2'
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
  <div class="flex flex-col h-full overflow-hidden bg-[var(--bg-secondary)]">
    <div class="px-3 py-[10px] bg-[var(--bg-tertiary)] flex items-center justify-between shrink-0">
      <span class="text-[var(--font-size-base)] font-semibold text-[var(--text-primary)]">{{ t('keys.title') }}</span>
      <div class="flex items-center gap-[2px]">
        <button class="panel-action-btn" @click="store.loadKeys()" :title="t('common.refresh')">
          <IconRefreshCw :width="14" :height="14" />
        </button>
        <button class="panel-action-btn" @click="openGenerate" :title="t('keys.generateKey')">
          <IconKeyRound :width="14" :height="14" />
        </button>
        <button class="panel-action-btn" @click="openCreate" :title="t('keys.newKey')">
          <IconPlus :width="14" :height="14" />
        </button>
      </div>
    </div>

    <div class="flex-1 overflow-y-auto py-1">
      <div v-if="store.keys.length === 0" class="px-3 py-10 flex-center">
        <span class="text-[var(--font-size-sm)] text-[var(--text-secondary)]">{{ t('keys.noKeys') }}</span>
      </div>

      <div
        v-for="key in store.keys"
        :key="key.name"
        class="group flex items-center justify-between px-3 py-2 gap-2 cursor-default transition-colors duration-150 hover:bg-[var(--hover-overlay)]"
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
          <div class="flex-1 min-w-0">
            <div class="flex items-center gap-[6px]">
              <span v-if="key.type" class="text-[10px] py-[1px] px-[5px] rounded-[3px] bg-[var(--color-primary)] text-white font-semibold leading-[1.4] shrink-0">{{ formatType(key.type) }}</span>
              <span class="text-[var(--font-size-base)] text-[var(--text-primary)] font-medium whitespace-nowrap overflow-hidden text-ellipsis">{{ key.name }}</span>
              <span v-if="key.has_passphrase" class="text-[var(--text-secondary)] shrink-0 flex items-center" :title="t('keys.passphraseProtected')">
                <IconLock :width="10" :height="10" />
              </span>
            </div>
            <div class="flex flex-col gap-[1px] mt-[2px]">
              <span v-if="key.fingerprint" class="text-[11px] text-[var(--text-secondary)] whitespace-nowrap overflow-hidden text-ellipsis opacity-70">{{ key.fingerprint }}</span>
              <span v-if="key.comment" class="text-[11px] text-[var(--text-secondary)] whitespace-nowrap overflow-hidden text-ellipsis">{{ key.comment }}</span>
            </div>
          </div>
          <div class="flex gap-[2px] shrink-0 opacity-0 transition-opacity duration-150 group-hover:opacity-100" :class="{ '!opacity-100': ctxKey === key.name }">
            <button class="key-action-btn" :title="t('keys.copyPub')" @click.stop="copyKey(key, 'pub')">
              <IconCopy :width="12" :height="12" />
            </button>
            <button class="key-action-btn" @click.stop="openRename(key)" :title="t('keys.rename')">
              <IconPencil :width="12" :height="12" />
            </button>
            <NPopconfirm @positive-click="handleDelete(key)">
              <template #trigger>
                <button class="key-action-btn key-action-danger" :title="t('common.delete')" @click.stop>
                  <IconTrash2 :width="12" :height="12" />
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
  transition: color 0.15s, background 0.15s;
}
.panel-action-btn:hover {
  color: var(--text-primary);
  background: var(--hover-overlay);
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
  transition: color 0.15s, background 0.15s;
}
.key-action-btn:hover {
  color: var(--text-primary);
  background: var(--hover-overlay);
}
.key-action-danger:hover {
  color: var(--color-error);
}
</style>
