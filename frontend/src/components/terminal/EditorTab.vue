<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import * as monaco from 'monaco-editor'
import { Events } from '@wailsio/runtime'
import { WriteSSHConfigRaw, SFTPWriteFileContent, WriteLocalFileContent } from '../../../bindings/vshell/internal/app/appservice'
import { useSSHConfigStore } from '../../stores/sshconfig'
import { useTerminalStore } from '../../stores/terminal'
import type { TerminalTab } from '../../stores/terminal'
import { detectLanguage } from '../../utils/fileType'

const props = defineProps<{ tab: TerminalTab }>()

const editorContainer = ref<HTMLElement | null>(null)
const sshConfigStore = useSSHConfigStore()
const terminalStore = useTerminalStore()

let editor: monaco.editor.IStandaloneCodeEditor | null = null
let resizeObserver: ResizeObserver | null = null
let originalContent = ''

onMounted(() => {
  if (!editorContainer.value) return

  const isDark = document.documentElement.getAttribute('data-theme')?.includes('dark') !== false
  const language = props.tab.filePath ? detectLanguage(props.tab.filePath) : 'plaintext'

  editor = monaco.editor.create(editorContainer.value, {
    value: props.tab.editorContent || '',
    language,
    theme: isDark ? 'vs-dark' : 'vs',
    minimap: { enabled: false },
    wordWrap: 'on',
    fontSize: 13,
    lineNumbers: 'on',
    scrollBeyondLastLine: false,
    automaticLayout: false,
    padding: { top: 8 },
    renderLineHighlight: 'line',
    smoothScrolling: true,
  })

  originalContent = props.tab.editorContent || ''

  editor.onDidChangeModelContent(() => {
    const content = editor?.getValue() || ''
    const dirty = content !== originalContent
    terminalStore.markTabDirty(props.tab.id, dirty)
    terminalStore.updateTabContent(props.tab.id, content)
  })

  editor.addAction({
    id: 'save-file',
    label: 'Save',
    keybindings: [monaco.KeyMod.CtrlCmd | monaco.KeyCode.KeyS],
    run: async () => {
      await handleSave()
    },
  })

  resizeObserver = new ResizeObserver(() => {
    editor?.layout()
  })
  resizeObserver.observe(editorContainer.value)

  Events.On('menu:save', () => {
    if (terminalStore.activeTabID === props.tab.id) {
      handleSave()
    }
  })
})

async function handleSave() {
  if (!editor) return
  const content = editor.getValue()
  try {
    switch (props.tab.editorMode) {
      case 'remote-sftp':
        if (!props.tab.connectionID || !props.tab.filePath) break
        await SFTPWriteFileContent(props.tab.connectionID, props.tab.filePath, content)
        break
      case 'local-file':
        if (!props.tab.filePath) break
        await WriteLocalFileContent(props.tab.filePath, content)
        break
      case 'ssh-config':
      default:
        await WriteSSHConfigRaw(content)
        sshConfigStore.loadEntries()
        break
    }
    originalContent = content
    terminalStore.markTabDirty(props.tab.id, false)
  } catch (e: any) {
    console.error('Failed to save file:', e)
  }
}

onUnmounted(() => {
  Events.Off('menu:save')
  resizeObserver?.disconnect()
  editor?.dispose()
  editor = null
})
</script>

<template>
  <div ref="editorContainer" class="editor-container" />
</template>

<style scoped>
.editor-container {
  width: 100%;
  height: 100%;
}
</style>
