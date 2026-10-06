<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import IconServer from '~icons/lucide/server'
import IconKeyRound from '~icons/lucide/key-round'
import IconFileText from '~icons/lucide/file-text'
import IconArrowLeftRight from '~icons/lucide/arrow-left-right'
import IconShieldCheck from '~icons/lucide/shield-check'
import IconFolder from '~icons/lucide/folder'
import IconActivity from '~icons/lucide/activity'
import IconSettings from '~icons/lucide/settings'
import { useLayoutStore, type SidebarView, type BottomTool } from '../../stores/layout'

const { t } = useI18n()
const emit = defineEmits<{ (e: 'openSettings'): void }>()
const layout = useLayoutStore()

const sidebarEntries: { view: SidebarView; icon: typeof IconServer; label: string }[] = [
  { view: 'connections', icon: IconServer, label: 'connection.title' },
  { view: 'keys', icon: IconKeyRound, label: 'keys.title' },
  { view: 'ssh-config', icon: IconFileText, label: 'sshConfig.title' },
  { view: 'port-forward', icon: IconArrowLeftRight, label: 'portForward.title' },
  { view: 'certs', icon: IconShieldCheck, label: 'certs.title' },
]

function isSidebarActive(view: SidebarView): boolean {
  return layout.leftPanelVisible && layout.activeSidebar === view
}

function isBottomActive(tool: BottomTool): boolean {
  return layout.bottomTool === tool
}
</script>

<template>
  <!-- 左条：常驻唤回入口（设计语言 §4.2 / §3.2-2） -->
  <nav class="island-bar shrink-0 flex flex-col items-center py-[6px] overflow-hidden" style="width: 40px">
    <div class="flex flex-col items-center gap-[2px]">
      <button
        v-for="entry in sidebarEntries"
        :key="entry.view"
        class="stripe-btn"
        :class="{ 'stripe-btn-active': isSidebarActive(entry.view) }"
        :title="t(entry.label)"
        @click="layout.toggleSidebar(entry.view)"
      >
        <component :is="entry.icon" :size="20" />
      </button>
    </div>

    <div class="flex-1" />

    <div class="flex flex-col items-center gap-[2px]">
      <button
        class="stripe-btn"
        :class="{ 'stripe-btn-active': isBottomActive('sftp') }"
        :title="t('sftp.title')"
        @click="layout.toggleBottomTool('sftp')"
      ><IconFolder :size="20" /></button>
      <button
        class="stripe-btn"
        :class="{ 'stripe-btn-active': isBottomActive('monitor') }"
        :title="t('monitor.title')"
        @click="layout.toggleBottomTool('monitor')"
      ><IconActivity :size="20" /></button>
      <button
        class="stripe-btn"
        :title="t('settings.title')"
        @click="emit('openSettings')"
      ><IconSettings :size="20" /></button>
    </div>
  </nav>
</template>

<style scoped>
.stripe-btn-active {
  background: var(--color-primary);
  color: #fff;
}
.stripe-btn-active:hover {
  background: var(--color-primary);
  color: #fff;
}
</style>
