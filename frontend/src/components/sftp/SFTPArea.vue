<script setup lang="ts">
import { computed, watch } from 'vue'
import { NEmpty } from 'naive-ui'
import { useTerminalStore } from '../../stores/terminal'
import { useSFTPStore } from '../../stores/sftp'
import SFTPPanel from './SFTPPanel.vue'

const terminalStore = useTerminalStore()
const sftpStore = useSFTPStore()

const activeConnectionID = computed(() => {
  const tab = terminalStore.tabs.find(t => t.id === terminalStore.activeTabID)
  return tab?.connectionID ?? null
})

// Auto-load directory when active connection changes
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
  <div class="sftp-area">
    <div v-if="!activeConnectionID" class="sftp-empty">
      <NEmpty description="No active session. Connect to a server to browse files." size="small" />
    </div>
    <SFTPPanel v-else :connectionID="activeConnectionID" />
  </div>
</template>

<style scoped>
.sftp-area {
  height: 100%;
  overflow: hidden;
  background: var(--bg-secondary);
}

.sftp-empty {
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
}
</style>
