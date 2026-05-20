<script setup lang="ts">
import { computed, watch, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { NEmpty } from 'naive-ui'
import { useTerminalStore } from '../../stores/terminal'
import { useSFTPStore } from '../../stores/sftp'
import SFTPPanel from './SFTPPanel.vue'

const { t } = useI18n()
const terminalStore = useTerminalStore()
const sftpStore = useSFTPStore()

const lastConnectionID = ref<string | null>(null)

const activeConnectionID = computed(() => {
  const tab = terminalStore.tabs.find(t => t.id === terminalStore.activeTabID)
  const connID = tab?.connectionID || null
  if (connID) {
    lastConnectionID.value = connID
  }
  return connID || lastConnectionID.value
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
  <div class="sftp-area">
    <div v-if="!activeConnectionID" class="sftp-empty">
      <NEmpty :description="t('sftp.noSession')" size="small" />
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
