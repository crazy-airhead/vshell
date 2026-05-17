<script setup lang="ts">
const props = withDefaults(defineProps<{
  modelValue: number
  min?: number
  max?: number
  direction?: 'horizontal' | 'vertical'
}>(), {
  min: 100,
  max: 600,
  direction: 'horizontal',
})

const emit = defineEmits<{ (e: 'update:modelValue', val: number): void }>()

let startPos = 0
let startSize = 0

function onMouseDown(e: MouseEvent) {
  e.preventDefault()
  startPos = props.direction === 'horizontal' ? e.clientY : e.clientX
  startSize = props.modelValue
  document.addEventListener('mousemove', onMouseMove)
  document.addEventListener('mouseup', onMouseUp)
}

function onMouseMove(e: MouseEvent) {
  const current = props.direction === 'horizontal' ? e.clientY : e.clientX
  const delta = props.direction === 'horizontal'
    ? startPos - current
    : current - startPos
  const s = Math.max(props.min, Math.min(props.max, startSize + delta))
  emit('update:modelValue', s)
}

function onMouseUp() {
  document.removeEventListener('mousemove', onMouseMove)
  document.removeEventListener('mouseup', onMouseUp)
}
</script>

<template>
  <div
    class="drag-divider"
    :class="[direction]"
    @mousedown="onMouseDown"
  ></div>
</template>

<style scoped>
.drag-divider {
  flex-shrink: 0;
  background: transparent;
  transition: background 0.15s;
}
.drag-divider:hover {
  background: #59a8f5;
}
.drag-divider.horizontal {
  height: 6px;
  width: 100%;
  cursor: ns-resize;
  border-radius: 3px;
}
.drag-divider.vertical {
  width: 6px;
  height: 100%;
  cursor: ew-resize;
  border-radius: 3px;
}
</style>
