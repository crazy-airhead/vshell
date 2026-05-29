<script setup lang="ts">
import { onUnmounted } from 'vue'

const props = withDefaults(defineProps<{
  modelValue: number
  min?: number
  max?: number
  direction?: 'horizontal' | 'vertical'
  invert?: boolean
}>(), {
  min: 100,
  max: 600,
  direction: 'horizontal',
  invert: false,
})

const emit = defineEmits<{ (e: 'update:modelValue', val: number): void }>()

let startPos = 0
let startSize = 0
let active = false

function onNextMouseDown() {
  cleanup()
}

function cleanup() {
  if (!active) return
  active = false
  document.removeEventListener('mousemove', onMouseMove)
  document.removeEventListener('mouseup', onMouseUp)
  window.removeEventListener('mouseup', onWindowMouseUp, true)
  window.removeEventListener('blur', onWindowBlur)
  window.removeEventListener('mousedown', onNextMouseDown, true)
  document.body.style.cursor = ''
  document.body.style.userSelect = ''
}

function onMouseDown(e: MouseEvent) {
  e.preventDefault()
  // Clean up stale listeners from a previous interrupted drag
  cleanup()
  active = true
  startPos = props.direction === 'horizontal' ? e.clientY : e.clientX
  startSize = props.modelValue
  document.addEventListener('mousemove', onMouseMove)
  document.addEventListener('mouseup', onMouseUp)
  window.addEventListener('mouseup', onWindowMouseUp, true)
  window.addEventListener('blur', onWindowBlur)
  window.addEventListener('mousedown', onNextMouseDown, { once: true, capture: true })
  document.body.style.cursor = props.direction === 'horizontal' ? 'ns-resize' : 'ew-resize'
  document.body.style.userSelect = 'none'
}

function onMouseMove(e: MouseEvent) {
  if (!active) return
  const current = props.direction === 'horizontal' ? e.clientY : e.clientX
  const delta = (current - startPos) * (props.invert ? -1 : 1)
  const s = Math.max(props.min, Math.min(props.max, startSize + delta))
  emit('update:modelValue', s)
}

function onMouseUp() {
  cleanup()
}

function onWindowMouseUp() {
  cleanup()
}

function onWindowBlur() {
  cleanup()
}

onUnmounted(() => {
  cleanup()
})
</script>

<template>
  <div
    class="shrink-0 relative bg-transparent transition-colors duration-150 hover:bg-[var(--color-primary)] resize-handle"
    :class="direction === 'horizontal' ? 'h-[3px] w-full cursor-ns-resize rounded-[1.5px] resize-handle-h' : 'w-[3px] h-full cursor-ew-resize rounded-[1.5px] resize-handle-v'"
    @mousedown="onMouseDown"
  ></div>
</template>

<style scoped>
.resize-handle::before {
  content: '';
  position: absolute;
}
.resize-handle-h::before {
  left: 0;
  right: 0;
  top: -3px;
  bottom: -3px;
}
.resize-handle-v::before {
  top: 0;
  bottom: 0;
  left: -3px;
  right: -3px;
}
</style>
