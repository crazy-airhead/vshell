<script setup lang="ts">
import { h } from 'vue'
import { useI18n } from 'vue-i18n'
import { NTabs, NTabPane, NEmpty, NTooltip } from 'naive-ui'
import { useTerminalStore } from '../../stores/terminal'
import { useConnectionStore } from '../../stores/connection'
import XTerminal from './XTerminal.vue'
import EditorTab from './EditorTab.vue'

const { t } = useI18n()
const terminalStore = useTerminalStore()
const connectionStore = useConnectionStore()

function getActiveTab(): string | undefined {
  return terminalStore.activeTabID ?? undefined
}

function renderTabTitle(tab: typeof terminalStore.tabs[number]) {
  if (tab.type !== 'editor') {
    return tab.title
  }

  const children: (string | ReturnType<typeof h>)[] = []
  if (tab.isRemote === true) {
    children.push(h('span', { class: 'tab-tag-remote' }, `[${t('sftp.remotePrefix')}] `))
  } else if (tab.isRemote === false) {
    children.push(h('span', { class: 'tab-tag-local' }, `[${t('sftp.localPrefix')}] `))
  }

  children.push(tab.title)

  if (tab.dirty) {
    children.push(h('span', { class: 'tab-dirty' }, ' •'))
  }

  return h(NTooltip, { delay: 0, placement: 'bottom' }, {
    trigger: () => h('span', children),
    default: () => tab.tooltip || tab.title,
  })
}

function handleClose(id: string) {
  const tab = terminalStore.tabs.find((t) => t.id === id)
  if (tab && tab.type !== 'editor' && tab.connectionID) {
    connectionStore.disconnect(tab.connectionID)
  }
  terminalStore.removeTab(id)
}
</script>

<template>
  <div class="flex-1 h-full flex flex-col overflow-hidden">
    <div v-if="terminalStore.tabs.length === 0" class="flex-1 flex-center">
      <NEmpty :description="t('terminal.empty')" />
    </div>
    <div v-else class="flex-1 flex flex-col overflow-hidden min-h-0 terminal-tabs">
      <NTabs
        :value="getActiveTab()"
        type="card"
        closable
        @close="handleClose"
        @update:value="(v: string | number) => { terminalStore.activeTabID = String(v) }"
        size="small"
      >
        <NTabPane
          v-for="tab in terminalStore.tabs"
          :key="tab.id"
          :name="tab.id"
          :tab="renderTabTitle(tab)"
        />
      </NTabs>
      <div class="flex-1 relative min-h-0">
        <div
          v-for="tab in terminalStore.tabs"
          :key="tab.id"
          class="absolute inset-0 invisible pointer-events-none"
          :class="{ '!visible !pointer-events-auto': tab.id === terminalStore.activeTabID }"
        >
          <XTerminal v-if="tab.type !== 'editor'" :sessionID="tab.id" />
          <EditorTab v-else :tab="tab" />
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.terminal-tabs :deep(.n-tabs) {
  display: flex;
  flex-direction: column;
  min-height: 0;
}

.terminal-tabs :deep(.n-tabs-nav) {
  flex-shrink: 0;
}

.terminal-tabs :deep(.n-tabs-content) {
  display: none;
}
</style>

<style>
.tab-tag-remote {
  color: var(--color-info);
  font-weight: 600;
}
.tab-tag-local {
  color: var(--color-success);
  font-weight: 600;
}
.tab-dirty {
  color: var(--color-warning);
}
</style>
