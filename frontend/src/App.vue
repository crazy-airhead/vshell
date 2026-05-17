<script setup lang="ts">
import { ref } from 'vue'
import { NConfigProvider, darkTheme, NMessageProvider, NDialogProvider } from 'naive-ui'
import ConnectionTree from './components/sidebar/ConnectionTree.vue'
import MonitorPanel from './components/monitor/MonitorPanel.vue'
import TerminalPane from './components/terminal/TerminalPane.vue'
import SFTPArea from './components/sftp/SFTPArea.vue'
import DraggableDivider from './components/common/DraggableDivider.vue'

const sidebarWidth = ref(280)
const sidebarTreeHeight = ref(300)
const sftpPanelHeight = ref(220)
</script>

<template>
  <NConfigProvider :theme="darkTheme">
    <NMessageProvider>
      <NDialogProvider>
        <div class="app-layout">
          <!-- Left Column: Connection Tree + Monitor -->
          <div class="left-column" :style="{ width: sidebarWidth + 'px' }">
            <div class="tree-zone" :style="{ height: sidebarTreeHeight + 'px', flexShrink: 0 }">
              <ConnectionTree />
            </div>
            <DraggableDivider
              direction="horizontal"
              :modelValue="sidebarTreeHeight"
              @update:modelValue="(v: number) => sidebarTreeHeight = v"
              :min="100"
              :max="800"
            />
            <div class="monitor-zone">
              <MonitorPanel />
            </div>
          </div>

          <!-- Vertical Divider: Left | Right -->
          <DraggableDivider
            direction="vertical"
            :modelValue="sidebarWidth"
            @update:modelValue="(v: number) => sidebarWidth = v"
            :min="200"
            :max="500"
          />

          <!-- Right Column: Terminal + SFTP -->
          <div class="right-column">
            <div class="terminal-zone">
              <TerminalPane />
            </div>
            <DraggableDivider
              direction="horizontal"
              :modelValue="sftpPanelHeight"
              @update:modelValue="(v: number) => sftpPanelHeight = v"
              :min="80"
              :max="600"
            />
            <div class="sftp-zone" :style="{ height: sftpPanelHeight + 'px', flexShrink: 0 }">
              <SFTPArea />
            </div>
          </div>
        </div>
      </NDialogProvider>
    </NMessageProvider>
  </NConfigProvider>
</template>

<style scoped>
.app-layout {
  display: flex;
  width: 100vw;
  height: 100vh;
  overflow: hidden;
  background: var(--bg-primary);
  padding: 36px 6px 6px 6px;
}

.left-column {
  display: flex;
  flex-direction: column;
  flex-shrink: 0;
  min-width: 0;
  overflow: hidden;
}

.tree-zone {
  overflow: hidden;
  border-radius: 8px;
  background: var(--bg-secondary);
}

.monitor-zone {
  flex: 1;
  min-height: 0;
  overflow: hidden;
  border-radius: 8px;
  background: var(--bg-secondary);
}

.right-column {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-width: 0;
  overflow: hidden;
}

.terminal-zone {
  flex: 1;
  min-height: 0;
  overflow: hidden;
  border-radius: 8px;
  background: var(--bg-secondary);
}

.sftp-zone {
  overflow: hidden;
  border-radius: 8px;
  background: var(--bg-secondary);
}
</style>
