<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import IconTerminal from '~icons/lucide/terminal'
import { useTerminalStore } from '../../stores/terminal'
import XTerminal from '../terminal/XTerminal.vue'
import EditorTab from '../terminal/EditorTab.vue'
import EditorTabs from './EditorTabs.vue'

const { t } = useI18n()
const terminalStore = useTerminalStore()
</script>

<template>
  <!-- 中心岛：永不收起，终端会话是一等公民（设计语言 §3.1 / §5） -->
  <main class="island flex-1 min-w-0 flex flex-col overflow-hidden">
    <template v-if="terminalStore.tabs.length > 0">
      <EditorTabs />

      <!-- 保活堆叠：所有会话同时挂载，visibility 切换（禁 v-if/display:none） -->
      <div class="flex-1 relative min-h-0">
        <div
          v-for="tab in terminalStore.tabs"
          :key="tab.id"
          class="absolute inset-0 flex invisible pointer-events-none"
          :class="{ '!visible !pointer-events-auto': tab.id === terminalStore.activeTabID }"
        >
          <XTerminal v-if="tab.type !== 'editor'" :sessionID="tab.id" class="flex-1 min-width-0" />
          <EditorTab v-else :tab="tab" class="flex-1 min-width-0" />
        </div>
      </div>
    </template>

    <!-- 富空态（设计语言 §11） -->
    <div v-else class="flex-1 flex-col-center gap-[var(--space-4)] text-center">
      <IconTerminal :size="40" class="text-[var(--text-muted)]" />
      <h3 class="text-[var(--font-size-base)] font-semibold text-[var(--text-primary)]">{{ t('terminal.emptyTitle') }}</h3>
      <p class="text-[var(--font-size-sm)] text-[var(--text-muted)]">{{ t('terminal.emptyHint') }}</p>
    </div>
  </main>
</template>

<style scoped>
/* xterm/Monaco 内容视图必须可收缩（设计语言 §5：防按内容收缩致 fit 列数错） */
.min-width-0 {
  min-width: 0;
}
</style>
