<script setup lang="ts">
import { ref, onUnmounted } from 'vue'

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

const handleEl = ref<HTMLElement>()

let startPos = 0
let startSize = 0
let active = false
// Full-viewport overlay shown only while dragging. It carries the resize cursor
// so the whole screen shows it during the drag. On mouseup we make it pointer-events:none
// and clear its cursor, then remove it on the next tick — this gives the browser time
// to re-evaluate the cursor from underlying elements before the overlay disappears.
let overlay: HTMLDivElement | null = null
let cursorRestoreTimer: ReturnType<typeof setTimeout> | null = null

function createOverlay() {
  // Remove any stale overlay from a previous interrupted drag
  if (overlay) {
    overlay.remove()
    overlay = null
  }
  overlay = document.createElement('div')
  overlay.style.cssText = `position:fixed;inset:0;z-index:9999;cursor:${props.direction === 'horizontal' ? 'ns-resize' : 'ew-resize'}`
  document.body.appendChild(overlay)
}

function cleanup() {
  if (!active) return
  active = false
  document.removeEventListener('mousemove', onMouseMove)
  document.removeEventListener('mouseup', onMouseUp)
  window.removeEventListener('mouseup', onWindowMouseUp, true)
  window.removeEventListener('blur', onWindowBlur)
  window.removeEventListener('mousedown', onNextMouseDown, true)
  document.body.style.userSelect = ''
  // Make overlay transparent to pointer events and clear its cursor so the
  // browser re-evaluates from elements below before we remove it on the next tick
  if (overlay) {
    overlay.style.pointerEvents = 'none'
    overlay.style.cursor = ''
    const old = overlay
    overlay = null
    setTimeout(() => old.remove(), 0)
  }
  // Briefly suppress the handle's own resize cursor so it doesn't immediately
  // re-apply when the mouse is released over the handle
  if (handleEl.value) {
    handleEl.value.style.cursor = 'default'
    if (cursorRestoreTimer) clearTimeout(cursorRestoreTimer)
    cursorRestoreTimer = setTimeout(() => {
      if (handleEl.value) {
        handleEl.value.style.cursor = '' // remove inline style, restore class cursor
      }
      cursorRestoreTimer = null
    }, 100)
  }
}

function onMouseDown(e: MouseEvent) {
  e.preventDefault()
  // Clean up stale listeners from a previous interrupted drag
  cleanup()
  active = true
  startPos = props.direction === 'horizontal' ? e.clientY : e.clientX
  startSize = props.modelValue
  createOverlay()
  document.addEventListener('mousemove', onMouseMove)
  document.addEventListener('mouseup', onMouseUp)
  window.addEventListener('mouseup', onWindowMouseUp, true)
  window.addEventListener('blur', onWindowBlur)
  window.addEventListener('mousedown', onNextMouseDown, { once: true, capture: true })
  document.body.style.userSelect = 'none'
}

function onNextMouseDown() {
  cleanup()
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
    ref="handleEl"
    class="shrink-0 relative bg-transparent transition-colors duration-150 hover:bg-[var(--color-primary)] resize-handle"
    :class="direction === 'horizontal' ? 'h-[3px] w-full cursor-ns-resize rounded-[1.5px] resize-handle-h' : 'w-[3px] h-full cursor-ew-resize rounded-[1.5px] resize-handle-v'"
    @mousedown="onMouseDown"
  ></div>
</template>

<style scoped>
</style>
