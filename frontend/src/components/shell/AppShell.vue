<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import ToolbarIsland from './ToolbarIsland.vue'
import LeftStripe from './LeftStripe.vue'
import ToolWindow from './ToolWindow.vue'
import EditorIsland from './EditorIsland.vue'
import StatusBarIsland from './StatusBarIsland.vue'
import DraggableDivider from '../common/DraggableDivider.vue'
import ConnectionTree from '../sidebar/ConnectionTree.vue'
import KeyManagementPanel from '../keys/KeyManagementPanel.vue'
import SSHConfigPanel from '../config/SSHConfigPanel.vue'
import PortForwardPanel from '../panels/PortForwardPanel.vue'
import CertPanel from '../cert/CertPanel.vue'
import MonitorPanel from '../monitor/MonitorPanel.vue'
import SFTPArea from '../sftp/SFTPArea.vue'
import IconFolderPlus from '~icons/lucide/folder-plus'
import IconPlus from '~icons/lucide/plus'
import IconRefreshCw from '~icons/lucide/refresh-cw'
import IconKeyRound from '~icons/lucide/key-round'
import IconPencil from '~icons/lucide/pencil'
import IconFileInput from '~icons/lucide/file-input'
import { useLayoutStore, type SidebarView, type BottomTool } from '../../stores/layout'

/**
 * 岛式画布（设计语言 §3）：
 * 画布级纵向 flex（gap = --island-bw，四周 padding 同值）；
 * 主区行 = 左条（通高，自主工具栏岛下缘到状态栏岛上缘，高度固定
 * 不随底部工具窗展开而压缩）+ 中央列；
 * 中央列内部 = 编辑行（左工具窗 + 竖热区 + 中心岛）+ 横热区 + 底部工具窗岛。
 * 热区在面板隐藏时随 v-if 移除——热区视觉与普通接缝无异，移除无跳变；
 * 收起面板自身以负 margin 吃掉 gap 槽（§3.2-3，见 ToolWindow）。
 */
const emit = defineEmits<{ (e: 'openSettings'): void }>()

const { t } = useI18n()
const layout = useLayoutStore()

const LEFT_TITLES: Record<SidebarView, () => string> = {
  connections: () => t('connection.title'),
  keys: () => t('keys.title'),
  'ssh-config': () => t('sshConfig.title'),
  'port-forward': () => t('portForward.title'),
  certs: () => t('certs.title'),
}

const leftTitle = computed(() => LEFT_TITLES[layout.activeSidebar]())

/* 各面板经 defineExpose 上来的头部动作（面板不再自带头部行，§4.3） */
const connTreeRef = ref<InstanceType<typeof ConnectionTree> | null>(null)
const keysRef = ref<InstanceType<typeof KeyManagementPanel> | null>(null)
const sshConfigRef = ref<InstanceType<typeof SSHConfigPanel> | null>(null)
const portForwardRef = ref<InstanceType<typeof PortForwardPanel> | null>(null)
const certRef = ref<InstanceType<typeof CertPanel> | null>(null)

const VIEW_ACTIONS: Record<SidebarView, () => { icon: typeof IconPlus; label: string; run: () => void }[]> = {
  connections: () => [
    { icon: IconFolderPlus, label: t('group.newGroup'), run: () => connTreeRef.value?.startNewGroup(null) },
    { icon: IconPlus, label: t('connection.newConnection'), run: () => connTreeRef.value?.handleNew() },
  ],
  keys: () => [
    { icon: IconRefreshCw, label: t('common.refresh'), run: () => keysRef.value?.refresh() },
    { icon: IconKeyRound, label: t('keys.generateKey'), run: () => keysRef.value?.openGenerate() },
    { icon: IconPlus, label: t('keys.newKey'), run: () => keysRef.value?.openCreate() },
  ],
  'ssh-config': () => [
    { icon: IconRefreshCw, label: t('common.refresh'), run: () => sshConfigRef.value?.refresh() },
    { icon: IconPencil, label: t('sshConfig.editRaw'), run: () => sshConfigRef.value?.handleEditRaw() },
    { icon: IconPlus, label: t('sshConfig.addHost'), run: () => sshConfigRef.value?.handleAdd() },
    { icon: IconFileInput, label: t('sshConfig.importHosts'), run: () => sshConfigRef.value?.openImportModal() },
  ],
  'port-forward': () => [
    { icon: IconRefreshCw, label: t('common.refresh'), run: () => portForwardRef.value?.refresh() },
    { icon: IconPlus, label: t('portForward.add'), run: () => portForwardRef.value?.openCreate() },
  ],
  certs: () => [
    { icon: IconRefreshCw, label: t('common.refresh'), run: () => certRef.value?.refresh() },
    { icon: IconPlus, label: t('certs.add'), run: () => certRef.value?.openWizard() },
  ],
}

const leftActions = computed(() => VIEW_ACTIONS[layout.activeSidebar]())

const bottomTabs: { tool: BottomTool; label: string }[] = [
  { tool: 'monitor', label: 'monitor.title' },
  { tool: 'sftp', label: 'sftp.title' },
]

function selectBottomTab(tool: BottomTool) {
  layout.bottomTool = tool
}
</script>

<template>
  <div
    class="flex flex-col w-screen h-screen overflow-hidden bg-[var(--bg-canvas)] select-none"
    :style="{ gap: 'var(--island-bw)', padding: 'var(--island-bw)' }"
  >
    <ToolbarIsland />

    <!-- 主区行：左条通高（高度固定，不被底部工具窗压缩）+ 中央列 -->
    <div class="flex flex-1 min-h-0 min-w-0" :style="{ gap: 'var(--island-bw)' }">
      <LeftStripe @open-settings="emit('openSettings')" />

      <!-- 中央列：编辑行 + 横热区 + 底部工具窗岛 -->
      <div class="flex flex-col flex-1 min-h-0 min-w-0" :style="{ gap: 'var(--island-bw)' }">
        <div class="flex flex-1 min-h-0 min-w-0" :style="{ gap: 'var(--island-bw)' }">
          <ToolWindow
            placement="left"
            :title="leftTitle"
            :visible="layout.leftPanelVisible"
            :size="layout.sidebarWidth"
            @hide="layout.toggleSidebar()"
          >
            <template #actions>
              <button
                v-for="action in leftActions"
                :key="action.label"
                class="icon-btn w-6 h-6"
                :title="action.label"
                @click="action.run()"
              ><component :is="action.icon" :size="14" /></button>
            </template>

            <ConnectionTree v-if="layout.activeSidebar === 'connections'" ref="connTreeRef" />
            <KeyManagementPanel v-else-if="layout.activeSidebar === 'keys'" ref="keysRef" />
            <SSHConfigPanel v-else-if="layout.activeSidebar === 'ssh-config'" ref="sshConfigRef" />
            <PortForwardPanel v-else-if="layout.activeSidebar === 'port-forward'" ref="portForwardRef" />
            <CertPanel v-else-if="layout.activeSidebar === 'certs'" ref="certRef" />
          </ToolWindow>

          <DraggableDivider
            v-if="layout.leftPanelVisible"
            direction="vertical"
            :min="150"
            :max="600"
            :model-value="layout.sidebarWidth"
            @update:model-value="layout.setSidebarWidth"
          />

          <EditorIsland />
        </div>

        <DraggableDivider
          v-if="layout.bottomVisible"
          direction="horizontal"
          :min="80"
          :max="600"
          invert
          :model-value="layout.bottomHeight"
          @update:model-value="layout.setBottomHeight"
        />

        <ToolWindow
          placement="bottom"
          :visible="layout.bottomVisible"
          :size="layout.bottomHeight"
          @hide="layout.bottomTool = null"
        >
          <template #tabs>
            <button
              v-for="tab in bottomTabs"
              :key="tab.tool"
              class="tool-tab"
              :class="{ 'tool-tab-active': layout.bottomTool === tab.tool }"
              @click="selectBottomTab(tab.tool)"
            >{{ t(tab.label) }}</button>
          </template>

          <!-- 两面板同时挂载、v-show 切换（§3.1：保监控选择态与 SFTP 浏览状态） -->
          <MonitorPanel v-show="layout.bottomTool === 'monitor'" />
          <SFTPArea v-show="layout.bottomTool === 'sftp'" />
        </ToolWindow>
      </div>
    </div>

    <StatusBarIsland />
  </div>
</template>

<style scoped>
/* 底部工具窗药丸标签（§4.3 #tabs 槽）：24 高 · 圆角 8 · 激活 --tab-selected-bg */
.tool-tab {
  height: 24px;
  padding: 0 10px;
  border: none;
  border-radius: var(--radius-m);
  background: transparent;
  color: var(--text-secondary);
  font-size: var(--font-size-xs);
  cursor: pointer;
  white-space: nowrap;
  transition: background-color 150ms ease, color 150ms ease;
}
.tool-tab:hover {
  background: var(--hover-overlay);
  color: var(--text-primary);
}
.tool-tab-active {
  background: var(--tab-selected-bg);
  color: var(--text-primary);
}
.tool-tab-active:hover {
  background: var(--tab-selected-bg);
}
</style>
