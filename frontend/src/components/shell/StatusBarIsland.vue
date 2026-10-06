<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import IconPanelLeft from '~icons/lucide/panel-left'
import IconActivity from '~icons/lucide/activity'
import IconFolder from '~icons/lucide/folder'
import { useLayoutStore } from '../../stores/layout'
import { useTerminalStore } from '../../stores/terminal'
import { useConnectionStore } from '../../stores/connection'
import { useTransferStore } from '../../stores/transfers'

const { t } = useI18n()
const layout = useLayoutStore()
const terminalStore = useTerminalStore()
const connectionStore = useConnectionStore()
const transferStore = useTransferStore()

type ConnStatus = 'none' | 'connecting' | 'online' | 'offline'

/** 活动连接四态（设计语言 §11）：connecting / online / offline / 无连接 */
const status = computed<{ kind: ConnStatus; label: string; color: string }>(() => {
  if (connectionStore.connecting) {
    return { kind: 'connecting', label: t('status.connecting'), color: 'var(--status-connecting)' }
  }
  const activeTab = terminalStore.tabs.find(t => t.id === terminalStore.activeTabID)
  const connID = activeTab?.connectionID
  if (!connID) {
    return { kind: 'none', label: t('status.noConnection'), color: 'transparent' }
  }
  if (connectionStore.connectedIDs.has(connID)) {
    return { kind: 'online', label: t('status.online'), color: 'var(--status-online)' }
  }
  return { kind: 'offline', label: t('status.offline'), color: 'var(--status-offline)' }
})

const activeConnectionName = computed(() => {
  const activeTab = terminalStore.tabs.find(t => t.id === terminalStore.activeTabID)
  if (!activeTab?.connectionID) return null
  return connectionStore.connections.find(c => c.id === activeTab.connectionID)?.name ?? null
})

const sessionCount = computed(() => terminalStore.tabs.filter(t => t.type !== 'editor').length)

/** 传输摘要：取最近一条活跃传输（点击直达 SFTP） */
const activeTransfer = computed(() => {
  const active = transferStore.transfers.filter(x => !x.done)
  return active.length > 0 ? active[active.length - 1] : null
})

function openSftp() {
  if (layout.bottomTool !== 'sftp') layout.toggleBottomTool('sftp')
}
</script>

<template>
  <!-- 状态栏岛（设计语言 §4.5）：连接状态 · 传输摘要 · 面板开关 -->
  <footer
    class="island-bar shrink-0 flex items-center gap-[var(--space-3)] px-[10px] text-[var(--font-size-xs)] text-[var(--text-secondary)] select-none"
    :style="{ height: 'var(--statusbar-h)' }"
  >
    <!-- 左：活动连接状态（纯显示） -->
    <div class="flex items-center gap-[var(--space-2)] min-w-0">
      <span v-if="status.kind !== 'none'" class="w-[7px] h-[7px] rounded-full shrink-0" :style="{ background: status.color }" />
      <span class="truncate max-w-[220px]">{{ activeConnectionName ?? status.label }}</span>
      <template v-if="activeConnectionName && status.kind !== 'none'">
        <span class="text-[var(--text-muted)]">·</span>
        <span class="whitespace-nowrap">{{ status.label }}</span>
      </template>
      <template v-if="sessionCount > 0">
        <span class="text-[var(--text-muted)]">·</span>
        <span class="whitespace-nowrap">{{ t('status.sessions', sessionCount) }}</span>
      </template>
    </div>

    <!-- 中：传输摘要 -->
    <button
      v-if="activeTransfer"
      class="flex items-center gap-[var(--space-2)] mx-auto px-[var(--space-3)] h-[20px] rounded-[var(--radius-m)] bg-transparent border-none cursor-pointer whitespace-nowrap hover:bg-[var(--hover-overlay)] hover:text-[var(--text-primary)] transition-colors"
      :title="t('status.bottomSftp')"
      @click="openSftp"
    >
      <span
        class="w-[6px] h-[6px] rounded-full shrink-0"
        :style="{ background: activeTransfer.direction === 'upload' ? 'var(--transfer-up)' : 'var(--transfer-down)' }"
      />
      <span class="truncate max-w-[200px]">{{ activeTransfer.file_name }}</span>
      <span class="tabular-nums">{{ activeTransfer.percent }}%</span>
    </button>
    <div v-else class="mx-auto" />

    <!-- 右：面板快速开关（active 高亮） -->
    <div class="flex items-center gap-[2px]">
      <button
        class="status-toggle"
        :class="{ 'status-toggle-active': layout.leftPanelVisible }"
        :title="t('status.leftPanel')"
        @click="layout.toggleSidebar()"
      ><IconPanelLeft :size="14" /></button>
      <button
        class="status-toggle"
        :class="{ 'status-toggle-active': layout.bottomTool === 'monitor' }"
        :title="t('status.bottomMonitor')"
        @click="layout.toggleBottomTool('monitor')"
      ><IconActivity :size="14" /></button>
      <button
        class="status-toggle"
        :class="{ 'status-toggle-active': layout.bottomTool === 'sftp' }"
        :title="t('status.bottomSftp')"
        @click="layout.toggleBottomTool('sftp')"
      ><IconFolder :size="14" /></button>
    </div>
  </footer>
</template>

<style scoped>
.status-toggle {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 24px;
  height: 20px;
  padding: 0;
  border: none;
  border-radius: var(--radius-s);
  background: transparent;
  color: var(--text-secondary);
  cursor: pointer;
  transition: background-color 150ms ease, color 150ms ease;
}
.status-toggle:hover {
  background: var(--hover-overlay);
  color: var(--text-primary);
}
.status-toggle-active {
  color: var(--color-primary);
}
.status-toggle-active:hover {
  color: var(--color-primary);
}
</style>
