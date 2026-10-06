<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
// eslint-disable-next-line @typescript-eslint/ban-ts-comment
// @ts-ignore: Window is exported at runtime
import { Window } from '@wailsio/runtime'
import IconSun from '~icons/lucide/sun'
import IconMoon from '~icons/lucide/moon'
import { useSettingsStore } from '../../stores/settings'
import { useConnectionStore } from '../../stores/connection'
import type { LocaleCode } from '../../stores/settings'

const { t, locale } = useI18n()

const settings = useSettingsStore()
const connectionStore = useConnectionStore()

const themeIcon = computed(() => settings.isDark ? IconMoon : IconSun)
const localeLabel = computed(() => settings.localeCode === 'zh-CN' ? 'EN' : '中')

function toggleLocale() {
  const next: LocaleCode = settings.localeCode === 'zh-CN' ? 'en' : 'zh-CN'
  settings.setLocale(next)
  locale.value = next
}
</script>

<template>
  <!-- 主工具栏岛：唯一窗口拖拽区（设计语言 §3.5） -->
  <div
    class="island-bar shrink-0 relative overflow-hidden flex items-center"
    style="-webkit-app-region: drag"
    :style="{ height: 'var(--toolbar-h)' }"
    @dblclick="Window.ToggleMaximise()"
  >
    <!-- 红绿灯留位（76px 内不放任何交互元素） -->
    <div class="shrink-0" :style="{ width: 'var(--traffic-light-w)' }" />

    <!-- 居中字标 -->
    <span class="absolute inset-x-0 text-center pointer-events-none select-none text-[var(--font-size-base)] font-semibold text-[var(--text-secondary)]">
      vShell
    </span>

    <!-- 右侧动作 -->
    <div class="ml-auto flex items-center gap-[2px] pr-[6px]" style="-webkit-app-region: no-drag">
      <button
        class="icon-btn w-6 h-6"
        :title="settings.isDark ? t('settings.light') : t('settings.dark')"
        @click="settings.toggleTheme()"
      ><component :is="themeIcon" :size="14" /></button>
      <button
        class="icon-btn w-6 h-6 text-[var(--font-size-sm)]"
        :title="t('settings.language')"
        @click="toggleLocale()"
      >{{ localeLabel }}</button>
    </div>

    <!-- 连接进度细条（底缘，红绿灯留位右侧起） -->
    <div
      v-if="connectionStore.connecting"
      class="absolute bottom-0 h-[2px] animate-connecting-bar"
      :style="{ left: 'var(--traffic-light-w)', right: 0 }"
    />
  </div>
</template>
