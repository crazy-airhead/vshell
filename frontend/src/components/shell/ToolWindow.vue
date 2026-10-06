<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import IconMinus from '~icons/lucide/minus'

/**
 * 通用工具窗岛壳（设计语言 §4.3）。
 * 收起 = 尺寸到 0 + 负 margin 吃掉自身 gap 槽（§3.2-3），
 * 300ms 过渡，永不 display:none / 卸载。
 */
const props = withDefaults(defineProps<{
  placement: 'left' | 'bottom'
  /** 底部工具窗可无标题（纯标签头部） */
  title?: string
  visible: boolean
  /** 展开时的尺寸（left=宽 / bottom=高），px */
  size: number
}>(), {
  title: '',
})

const emit = defineEmits<{ (e: 'hide'): void }>()
const { t } = useI18n()

const style = computed<Record<string, string>>(() => {
  if (props.placement === 'left') {
    const s: Record<string, string> = {
      width: props.visible ? `${props.size}px` : '0px',
      marginRight: props.visible ? '0px' : 'calc(-1 * var(--island-bw))',
      transition: 'width 300ms ease, margin-right 300ms ease',
    }
    return s
  }
  const s: Record<string, string> = {
    height: props.visible ? `${props.size}px` : '0px',
    marginTop: props.visible ? '0px' : 'calc(-1 * var(--island-bw))',
    flexShrink: '0',
    transition: 'height 300ms ease, margin-top 300ms ease',
  }
  return s
})
</script>

<template>
  <section
    class="island shrink-0 flex overflow-hidden"
    :class="placement === 'left' ? 'flex-col' : 'flex-col'"
    :style="style"
  >
    <!-- 头 41：标题 + 标签槽 + 动作槽 + 隐藏钮（短横线，Hide 语义） -->
    <header
      class="shrink-0 flex items-center gap-[var(--space-3)] pl-[12px] pr-[6px] border-b border-[var(--border-color)] select-none"
      :style="{ height: 'var(--toolwindow-header-h)' }"
    >
      <!-- 标题：IDEA medium 档 = 基准−1（相对偏移，跟随用户字号缩放） -->
      <h2 v-if="title" class="tw-title text-[var(--text-secondary)] font-normal whitespace-nowrap">{{ title }}</h2>

      <div v-if="$slots.tabs" class="flex items-center gap-[var(--space-3)] min-w-0">
        <slot name="tabs" />
      </div>

      <div class="ml-auto flex items-center gap-[2px]">
        <slot name="actions" />
        <button
          class="icon-btn w-6 h-6"
          :title="t('common.hide')"
          @click="emit('hide')"
        ><IconMinus :size="14" /></button>
      </div>
    </header>

    <!-- 身体 -->
    <div class="flex-1 min-h-0 min-w-0 overflow-hidden relative">
      <slot />
    </div>
  </section>
</template>

<style scoped>
.tw-title {
  font-size: calc(var(--font-size-base) - 1px);
}
</style>
