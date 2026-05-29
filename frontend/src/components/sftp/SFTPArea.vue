<script setup lang="ts">
import { computed, watch, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { NEmpty } from 'naive-ui'
import { useTerminalStore } from '../../stores/terminal'
import { useSFTPStore } from '../../stores/sftp'
import { useLayoutStore } from '../../stores/layout'
import SFTPPanel from './SFTPPanel.vue'

const { t } = useI18n()
const terminalStore = useTerminalStore()
const sftpStore = useSFTPStore()
const layoutStore = useLayoutStore()

const lastConnectionID = ref<string | null>(null)

const activeConnectionID = computed(() => {
  const tab = terminalStore.tabs.find(t => t.id === terminalStore.activeTabID)
  const connID = tab?.connectionID || null
  if (connID) {
    lastConnectionID.value = connID
  }
  const resolved = connID || lastConnectionID.value
  if (resolved && !terminalStore.tabs.some(t => t.connectionID === resolved)) {
    lastConnectionID.value = null
    sftpStore.closePanel(resolved)
    if (layoutStore.activeBottomTool === 'sftp') {
      layoutStore.activeBottomTool = null
    }
    return null
  }
  return resolved
})

watch(activeConnectionID, (newID) => {
  if (newID) {
    const p = sftpStore.getPanel(newID)
    if (p.files.length === 0 && !p.loading) {
      sftpStore.navigateToDir(newID, p.currentPath)
    }
  }
}, { immediate: true })
</script>

<template>
  <div class="h-full overflow-hidden bg-[var(--bg-secondary)]">
    <div v-if="!activeConnectionID" class="h-full flex-center">
      <NEmpty :description="t('sftp.noSession')" size="small" />
    </div>
    <SFTPPanel v-else :connectionID="activeConnectionID" />
  </div>
</template>
