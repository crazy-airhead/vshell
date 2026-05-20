<script setup lang="ts">
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

function onMouseDown(e: MouseEvent) {
  e.preventDefault()
  startPos = props.direction === 'horizontal' ? e.clientY : e.clientX
  startSize = props.modelValue
  document.addEventListener('mousemove', onMouseMove)
  document.addEventListener('mouseup', onMouseUp)
}

function onMouseMove(e: MouseEvent) {
  const current = props.direction === 'horizontal' ? e.clientY : e.clientX
  const delta = (current - startPos) * (props.invert ? -1 : 1)
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
    class="shrink-0 bg-transparent transition-colors duration-150 hover:bg-[var(--color-primary)]"
    :class="direction === 'horizontal' ? 'h-[6px] w-full cursor-ns-resize rounded-[3px]' : 'w-[6px] h-full cursor-ew-resize rounded-[3px]'"
    @mousedown="onMouseDown"
  ></div>
</template>
