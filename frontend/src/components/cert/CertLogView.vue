<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { NButton, NTooltip, useMessage } from 'naive-ui'
import IconCopy from '~icons/lucide/copy'
import { useCertStore } from '../../stores/cert'
import { writeClipboard } from '../../utils/clipboard'

const props = defineProps<{ logKey: string }>()

const { t } = useI18n()
const message = useMessage()
const certStore = useCertStore()

const containerRef = ref<HTMLElement | null>(null)

const lines = computed(() => certStore.getLog(props.logKey))

watch(
  () => lines.value.length,
  async () => {
    await nextTick()
    const el = containerRef.value
    if (el) el.scrollTop = el.scrollHeight
  },
)

async function copyLog() {
  if (await writeClipboard(lines.value.join('\n'))) {
    message.success(t('certs.copyDone'))
  } else {
    message.error(t('certs.copyFailed'))
  }
}
</script>

<template>
  <div class="flex flex-col min-h-0 h-full">
    <div class="flex justify-end pb-1.5">
      <NTooltip>
        <template #trigger>
          <NButton size="tiny" quaternary @click="copyLog">
            <template #icon><IconCopy :width="14" :height="14" /></template>
          </NButton>
        </template>
        {{ t('certs.copyLog') }}
      </NTooltip>
    </div>
    <div
      ref="containerRef"
      class="flex-1 min-h-0 overflow-y-auto rounded-[var(--border-radius)] bg-[var(--bg-primary)] border border-[var(--border-color)] px-2 py-1.5"
    >
      <div v-if="lines.length === 0" class="text-[11px] text-[var(--text-secondary)] py-2 flex-center">
        {{ t('certs.logEmpty') }}
      </div>
      <pre
        v-else
        class="font-mono text-[11px] leading-[1.5] text-[var(--text-primary)] whitespace-pre-wrap break-all m-0"
      >{{ lines.join('\n') }}</pre>
    </div>
  </div>
</template>
