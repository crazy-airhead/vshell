<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import * as monaco from 'monaco-editor'
import { WriteSSHConfigRaw } from '../../../bindings/vshell/internal/app/appservice'
import { useSSHConfigStore } from '../../stores/sshconfig'
import { useTerminalStore } from '../../stores/terminal'
import type { TerminalTab } from '../../stores/terminal'

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

  editor = monaco.editor.create(editorContainer.value, {
    value: props.tab.editorContent || '',
    language: 'plaintext',
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
    id: 'save-ssh-config',
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
})

async function handleSave() {
  if (!editor) return
  const content = editor.getValue()
  try {
    await WriteSSHConfigRaw(content)
    originalContent = content
    terminalStore.markTabDirty(props.tab.id, false)
    sshConfigStore.loadEntries()
  } catch (e: any) {
    console.error('Failed to save SSH config:', e)
  }
}

onUnmounted(() => {
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
