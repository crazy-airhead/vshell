<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { NModal, NButton, NSpace, NTabs, NTabPane, NTooltip, useMessage } from 'naive-ui'
import IconCopy from '~icons/lucide/copy'
import { useCertStore } from '../../stores/cert'
import { writeClipboard } from '../../utils/clipboard'
import type { CertTask } from '../../types'
import CertLogView from './CertLogView.vue'

const props = defineProps<{
  show: boolean
  task: CertTask | null
}>()

const emit = defineEmits<{ (e: 'update:show', value: boolean): void }>()

const { t } = useI18n()
const message = useMessage()
const certStore = useCertStore()

const tab = ref('op')
const serverLog = ref('')
const loadingServerLog = ref(false)

const visible = computed({
  get: () => props.show,
  set: (v: boolean) => emit('update:show', v),
})

watch(
  () => props.show,
  async (open) => {
    tab.value = 'op'
    serverLog.value = ''
    if (open && props.task) {
      certStore.ensureOpLog(props.task.id)
      loadingServerLog.value = true
      try {
        serverLog.value = await certStore.readServerLog(props.task.connection_id)
      } catch (e) {
        serverLog.value = String(e)
      } finally {
        loadingServerLog.value = false
      }
    }
  },
)

async function copyServerLog() {
  if (await writeClipboard(serverLog.value)) {
    message.success(t('certs.copyDone'))
  } else {
    message.error(t('certs.copyFailed'))
  }
}
</script>

<template>
  <NModal
    v-model:show="visible"
    preset="card"
    :title="task ? t('certs.logTitle', { name: task.primary_domain }) : t('certs.logs')"
    style="width: 680px"
    :mask-closable="true"
  >
    <div class="h-[420px]">
      <NTabs v-model:value="tab" type="line" size="small" class="h-full" pane-class="!h-[calc(100%-36px)]">
        <NTabPane name="op" :tab="t('certs.logs')">
          <div v-if="task" class="h-full">
            <CertLogView :log-key="task.id" />
          </div>
        </NTabPane>
        <NTabPane name="server" :tab="t('certs.serverLog')">
          <div class="h-full flex flex-col min-h-0">
            <div class="flex justify-end pb-1.5">
              <NTooltip>
                <template #trigger>
                  <NButton size="tiny" quaternary :disabled="!serverLog" @click="copyServerLog">
                    <template #icon><IconCopy :width="14" :height="14" /></template>
                  </NButton>
                </template>
                {{ t('certs.copyLog') }}
              </NTooltip>
            </div>
            <div class="flex-1 min-h-0 overflow-y-auto rounded-[var(--border-radius)] bg-[var(--bg-primary)] border border-[var(--border-color)] px-2 py-1.5">
              <pre class="font-mono text-[11px] leading-[1.5] text-[var(--text-primary)] whitespace-pre-wrap break-all m-0">{{ serverLog || t('certs.logEmpty') }}</pre>
            </div>
          </div>
        </NTabPane>
      </NTabs>
    </div>
    <template #footer>
      <NSpace justify="end">
        <NButton @click="visible = false">{{ t('common.cancel') }}</NButton>
      </NSpace>
    </template>
  </NModal>
</template>
