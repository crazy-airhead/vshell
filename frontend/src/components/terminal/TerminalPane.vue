<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { NTabs, NTabPane, NEmpty } from 'naive-ui'
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

function getTabTitle(tab: { title: string; type?: string; dirty?: boolean }): string {
  if (tab.type === 'editor' && tab.dirty) {
    return tab.title + ' •'
  }
  return tab.title
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
  <div class="terminal-pane">
    <div v-if="terminalStore.tabs.length === 0" class="terminal-empty">
      <NEmpty :description="t('terminal.empty')" />
    </div>
    <div v-else class="terminal-tabs">
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
          :tab="getTabTitle(tab)"
        />
      </NTabs>
      <div class="terminals-container">
        <div
          v-for="tab in terminalStore.tabs"
          :key="tab.id"
          class="terminal-instance"
          :class="{ active: tab.id === terminalStore.activeTabID }"
        >
          <XTerminal v-if="tab.type !== 'editor'" :sessionID="tab.id" />
          <EditorTab v-else :tab="tab" />
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.terminal-pane {
  flex: 1;
  height: 100%;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.terminal-empty {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
}

.terminal-tabs {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  min-height: 0;
}

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

.terminals-container {
  flex: 1;
  position: relative;
  min-height: 0;
}

.terminal-instance {
  position: absolute;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  visibility: hidden;
  pointer-events: none;
}

.terminal-instance.active {
  visibility: visible;
  pointer-events: auto;
}
</style>
