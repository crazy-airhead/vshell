<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  NButton,
  NEmpty,
  NInput,
  NInputGroup,
  NModal,
  useDialog,
  useMessage,
} from 'naive-ui'
import { Events } from '@wailsio/runtime'
import IconBraces from '~icons/lucide/braces'
import IconPlay from '~icons/lucide/play'
import IconPencil from '~icons/lucide/pencil'
import IconTrash2 from '~icons/lucide/trash-2'
import IconPlus from '~icons/lucide/plus'
import { useSnippetsStore } from '../../stores/snippets'
import { useTerminalStore } from '../../stores/terminal'
import type { QuickCommand } from '../../types'

const { t } = useI18n()
const snippetsStore = useSnippetsStore()
const terminalStore = useTerminalStore()
const message = useMessage()
const dialog = useDialog()

const showModal = ref(false)
const editing = ref<QuickCommand | null>(null)
const form = reactive({
  name: '',
  command: '',
})

const modalTitle = computed(() => editing.value ? t('snippets.edit') : t('snippets.new'))
const activeTerminal = computed(() => {
  const id = terminalStore.activeTabID
  if (!id) return null
  const tab = terminalStore.tabs.find(item => item.id === id)
  if (!tab || tab.type === 'editor' || tab.connected === false) return null
  return tab
})

onMounted(async () => {
  try {
    await snippetsStore.loadSnippets()
  } catch (e: any) {
    message.error(t('snippets.loadFailed', { error: e }))
  }
})

function resetForm() {
  editing.value = null
  form.name = ''
  form.command = ''
}

function openNew() {
  resetForm()
  showModal.value = true
}

function openEdit(snippet: QuickCommand) {
  editing.value = snippet
  form.name = snippet.name
  form.command = snippet.command
  showModal.value = true
}

async function saveSnippet() {
  const name = form.name.trim()
  const command = form.command.trim()
  if (!name) {
    message.warning(t('snippets.nameRequired'))
    return
  }
  if (!command) {
    message.warning(t('snippets.commandRequired'))
    return
  }

  try {
    if (editing.value) {
      await snippetsStore.updateSnippet({
        id: editing.value.id,
        name,
        command,
        sort_order: editing.value.sort_order,
      })
      message.success(t('snippets.updated'))
    } else {
      await snippetsStore.createSnippet({
        id: crypto.randomUUID(),
        name,
        command,
        sort_order: snippetsStore.snippets.length,
      })
      message.success(t('snippets.created'))
    }
    showModal.value = false
    resetForm()
  } catch (e: any) {
    message.error(t('snippets.saveFailed', { error: e }))
  }
}

function deleteSnippet(snippet: QuickCommand) {
  dialog.warning({
    title: t('snippets.deleteTitle'),
    content: t('snippets.deleteContent', { name: snippet.name }),
    positiveText: t('common.delete'),
    negativeText: t('common.cancel'),
    onPositiveClick: async () => {
      try {
        await snippetsStore.deleteSnippet(snippet.id)
        message.success(t('snippets.deleted'))
      } catch (e: any) {
        message.error(t('snippets.deleteFailed', { error: e }))
      }
    },
  })
}

function runSnippet(snippet: QuickCommand) {
  const tab = activeTerminal.value
  if (!tab) {
    message.warning(t('snippets.noTerminal'))
    return
  }
  Events.Emit('terminal:stdin', {
    sessionID: tab.id,
    data: snippet.command.endsWith('\n') ? snippet.command : `${snippet.command}\r`,
  })
}

function previewCommand(command: string) {
  return command.replace(/\s+/g, ' ').trim()
}
</script>

<template>
  <div class="flex flex-col h-full overflow-hidden bg-[var(--bg-secondary)]">
    <div class="px-3 py-[10px] bg-[var(--bg-tertiary)] flex items-center justify-between shrink-0 thin-border-b">
      <span class="text-[var(--font-size-base)] font-semibold text-[var(--text-primary)]">{{ t('snippets.title') }}</span>
      <NButton size="tiny" quaternary :title="t('snippets.new')" @click="openNew">
        <IconPlus :width="14" :height="14" />
      </NButton>
    </div>

    <div class="flex-1 overflow-y-auto p-3">
      <NEmpty v-if="!snippetsStore.loading && snippetsStore.snippets.length === 0" :description="t('snippets.empty')" size="small" />
      <div v-else class="snippet-grid">
        <div v-for="snippet in snippetsStore.snippets" :key="snippet.id" class="snippet-card" @dblclick="runSnippet(snippet)">
          <button class="snippet-icon" :title="t('snippets.run')" @click="runSnippet(snippet)">
            <IconBraces :width="18" :height="18" />
          </button>
          <div class="min-w-0 flex-1">
            <div class="snippet-name">{{ snippet.name }}</div>
            <div class="snippet-command">{{ previewCommand(snippet.command) }}</div>
          </div>
          <div class="snippet-actions">
            <button class="snippet-action-btn" :title="t('snippets.run')" @click="runSnippet(snippet)">
              <IconPlay :width="13" :height="13" />
            </button>
            <button class="snippet-action-btn" :title="t('common.edit')" @click="openEdit(snippet)">
              <IconPencil :width="13" :height="13" />
            </button>
            <button class="snippet-action-btn danger" :title="t('common.delete')" @click="deleteSnippet(snippet)">
              <IconTrash2 :width="13" :height="13" />
            </button>
          </div>
        </div>
      </div>
    </div>

    <NModal v-model:show="showModal" preset="card" :title="modalTitle" style="width: 560px" :mask-closable="false">
      <div class="flex flex-col gap-3">
        <NInput
          v-model:value="form.name"
          :placeholder="t('snippets.namePlaceholder')"
          @keyup.enter="saveSnippet"
        />
        <NInput
          v-model:value="form.command"
          type="textarea"
          :autosize="{ minRows: 8, maxRows: 16 }"
          :placeholder="t('snippets.commandPlaceholder')"
        />
      </div>
      <template #footer>
        <NInputGroup class="justify-end">
          <NButton @click="showModal = false">{{ t('common.cancel') }}</NButton>
          <NButton type="primary" @click="saveSnippet">{{ t('common.save') }}</NButton>
        </NInputGroup>
      </template>
    </NModal>
  </div>
</template>

<style scoped>
.snippet-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(240px, 1fr));
  gap: 10px;
}

.snippet-card {
  display: flex;
  align-items: center;
  gap: 10px;
  min-height: 60px;
  padding: 10px;
  border-radius: 8px;
  background: var(--bg-tertiary);
  border: 1px solid transparent;
  cursor: default;
  transition: border-color 0.15s ease, background 0.15s ease;
}

.snippet-card:hover {
  background: var(--hover-overlay);
  border-color: var(--border-color);
}

.snippet-icon {
  width: 38px;
  height: 38px;
  display: flex;
  align-items: center;
  justify-content: center;
  border: 0;
  border-radius: 8px;
  color: white;
  background: #0572a8;
  cursor: pointer;
  flex-shrink: 0;
}

.snippet-name {
  color: var(--text-primary);
  font-size: var(--font-size-base);
  font-weight: 600;
  line-height: 1.35;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.snippet-command {
  margin-top: 2px;
  color: var(--text-secondary);
  font-size: var(--font-size-sm);
  line-height: 1.3;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.snippet-actions {
  display: flex;
  align-items: center;
  gap: 2px;
  opacity: 0;
  transition: opacity 0.15s ease;
}

.snippet-card:hover .snippet-actions {
  opacity: 1;
}

.snippet-action-btn {
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

.snippet-action-btn:hover {
  color: var(--text-primary);
  background: var(--hover-overlay);
}

.snippet-action-btn.danger:hover {
  color: var(--color-error);
}
</style>
