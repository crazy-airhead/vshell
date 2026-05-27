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
    class="shrink-0 bg-transparent transition-colors duration-150 hover:bg-[var(--color-primary)]"
    :class="direction === 'horizontal' ? 'h-[3px] w-full cursor-ns-resize rounded-[1.5px]' : 'w-[3px] h-full cursor-ew-resize rounded-[1.5px]'"
    @mousedown="onMouseDown"
  ></div>
</template>
