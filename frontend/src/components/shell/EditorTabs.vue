<script setup lang="ts">
import { ref, computed, h, watch, onMounted, onBeforeUnmount, nextTick } from 'vue'
import { useI18n } from 'vue-i18n'
import { NTooltip, NDropdown } from 'naive-ui'
import type { DropdownOption } from 'naive-ui'
import IconX from '~icons/lucide/x'
import IconChevronDown from '~icons/lucide/chevron-down'
import { useTerminalStore } from '../../stores/terminal'
import { useConnectionStore } from '../../stores/connection'
import { useSFTPStore } from '../../stores/sftp'

type Tab = ReturnType<typeof useTerminalStore>['tabs'][number]

const { t } = useI18n()
const terminalStore = useTerminalStore()
const connectionStore = useConnectionStore()
const sftpStore = useSFTPStore()

const hoveredTab = ref<string | null>(null)
const ctxTabID = ref<string | null>(null)
const ctxX = ref(0)
const ctxY = ref(0)

/* ── 关闭/重连/断开/复制（原样移植自 TerminalPane，携带 sftpStore 清理语义）── */

async function handleClose(id: string) {
  const tab = terminalStore.tabs.find((t) => t.id === id)
  if (tab && tab.type !== 'editor' && tab.connectionID) {
    await connectionStore.disconnectSession(tab.id, tab.connectionID)
    const remaining = terminalStore.tabs.filter(t => t.connectionID === tab.connectionID && t.id !== id)
    if (remaining.length === 0) {
      sftpStore.closePanel(tab.connectionID)
    }
  }
  terminalStore.removeTab(id)
}

async function handleReconnect(tab: Tab) {
  if (tab.type === 'editor' || !tab.connectionID) return
  await connectionStore.disconnectSession(tab.id, tab.connectionID)
  terminalStore.markTabDisconnected(tab.id)
  try {
    const sessionID = await connectionStore.connect(tab.connectionID)
    terminalStore.removeTab(tab.id)
    terminalStore.addTab({
      id: sessionID,
      connectionID: tab.connectionID,
      title: tab.title,
      connected: true,
    })
  } catch {
    // Tab stays but shows as disconnected
  }
}

async function handleDisconnectSession(tab: Tab) {
  if (tab.type === 'editor' || !tab.connectionID) return
  await connectionStore.disconnectSession(tab.id, tab.connectionID)
  terminalStore.markTabDisconnected(tab.id)
}

async function handleDuplicate(tab: Tab) {
  if (tab.type === 'editor' || !tab.connectionID) return
  try {
    const sessionID = await connectionStore.connect(tab.connectionID)
    terminalStore.addTab({
      id: sessionID,
      connectionID: tab.connectionID,
      title: tab.title,
      connected: true,
    })
  } catch {
    // Failed silently
  }
}

function handleCloseOthers(id: string) {
  const tabsToClose = terminalStore.tabs.filter(t => t.id !== id)
  for (const tab of tabsToClose) {
    if (tab.type !== 'editor' && tab.connectionID) {
      connectionStore.disconnectSession(tab.id, tab.connectionID)
    }
  }
  const keepTab = terminalStore.tabs.find(t => t.id === id)
  for (const tab of tabsToClose) {
    if (tab.connectionID && tab.connectionID !== keepTab?.connectionID) {
      sftpStore.closePanel(tab.connectionID)
    }
  }
  terminalStore.closeOtherTabs(id)
}

function handleCloseAll() {
  const connectionIDs = new Set<string>()
  for (const tab of terminalStore.tabs) {
    if (tab.type !== 'editor' && tab.connectionID) {
      connectionStore.disconnectSession(tab.id, tab.connectionID)
      connectionIDs.add(tab.connectionID)
    }
  }
  for (const connID of connectionIDs) {
    sftpStore.closePanel(connID)
  }
  terminalStore.closeAllTabs()
}

/* ── 右键菜单 ── */

function onTabContextMenu(e: MouseEvent, tab: Tab) {
  if (tab.type === 'editor') return
  e.preventDefault()
  ctxX.value = e.clientX
  ctxY.value = e.clientY
  ctxTabID.value = tab.id
}

function getContextOptions(): DropdownOption[] {
  const tab = terminalStore.tabs.find(t => t.id === ctxTabID.value)
  if (!tab) return []
  const disconnected = !tab.connected
  return [
    { label: t('tab.duplicate'), key: 'duplicate' },
    { label: t('tab.reconnect'), key: 'reconnect', disabled: !disconnected },
    { label: t('tab.disconnect'), key: 'disconnect', disabled: disconnected },
    { type: 'divider', key: 'd1' },
    { label: t('tab.close'), key: 'close' },
    { label: t('tab.closeOthers'), key: 'closeOthers' },
    { label: t('tab.closeAll'), key: 'closeAll' },
  ]
}

function handleContextSelect(action: string) {
  const tab = terminalStore.tabs.find(t => t.id === ctxTabID.value)
  if (!tab) return
  switch (action) {
    case 'reconnect':
      handleReconnect(tab)
      break
    case 'disconnect':
      handleDisconnectSession(tab)
      break
    case 'duplicate':
      handleDuplicate(tab)
      break
    case 'close':
      handleClose(tab.id)
      break
    case 'closeOthers':
      handleCloseOthers(tab.id)
      break
    case 'closeAll':
      handleCloseAll()
      break
  }
  ctxTabID.value = null
}

function statusDotColor(tab: Tab): string {
  return tab.connected ? 'var(--status-online)' : 'var(--status-offline)'
}

/* ── 超宽折叠（GMark GEditorTabs 0104 同款移植，IDEA「放不下即隐藏」）：
      标签保自然宽度不缩水，放不下的按 LRU 折叠进右端 chevron-down 按钮
      （NDropdown 列被折叠标签，点击激活并回带）；激活标签恒可见；
      滚轮 = 按最近使用序平移可见窗口（skip） ── */

const viewportRef = ref<HTMLElement | null>(null)
const overflowWrapRef = ref<HTMLElement | null>(null)
/** 标签元素注册表（测量用，非响应式；每个元素单独进 RO，标题变化即触发） */
const tabEls = new Map<string, HTMLElement>()
/** 访问时间戳（LRU 折叠策略） */
const lastUsed = new Map<string, number>()
/** 滚轮窗口：按最近使用序跳过的非激活项数 */
const skip = ref(0)
/** 可见集；null = 尚未测量（首帧全显，onMounted 同步测量收敛后再裁剪） */
const visibleIds = ref<Set<string> | null>(null)

const overflowTabs = computed(() => {
  const set = visibleIds.value
  if (!set) return []
  return terminalStore.tabs.filter(tb => !set.has(tb.id))
})
const overflowCount = computed(() => overflowTabs.value.length)

function isHidden(id: string): boolean {
  const set = visibleIds.value
  return set !== null && !set.has(id)
}

let stripRO: ResizeObserver | null = null
let measureScheduled = false

function scheduleMeasure(): void {
  if (measureScheduled) return
  measureScheduled = true
  void nextTick(() => {
    measureScheduled = false
    measure()
  })
}

function setTabEl(id: string, el: unknown): void {
  const node = el instanceof HTMLElement ? el : null
  const prev = tabEls.get(id)
  if (prev) stripRO?.unobserve(prev)
  if (node) {
    tabEls.set(id, node)
    stripRO?.observe(node)
  } else {
    tabEls.delete(id)
  }
}

/** 按优先级（激活 → 最近使用 → 原始序）贪心装入可用宽度；装不下折叠（LRU） */
function measure(): void {
  const row = viewportRef.value
  if (!row) return
  const tabs = terminalStore.tabs
  const ids = tabs.map(tb => tb.id)
  const cs = getComputedStyle(row)
  const gap = parseFloat(cs.columnGap) || 0
  const pad = parseFloat(cs.paddingLeft) + parseFloat(cs.paddingRight)
  // 溢出按钮未显示时宽 0（v-show）；显示后 watch overflowCount 补测收敛
  const btnW = overflowWrapRef.value?.offsetWidth ?? 0
  const fixed = btnW > 0 ? btnW + gap : 0
  const avail = row.clientWidth - pad - fixed
  // 极窄行时标签随行收缩（≥100px，标签内截断），防 300px 顶满把按钮挤出
  row.style.setProperty('--v-tab-max', `${Math.max(100, Math.min(300, avail))}px`)
  const width = (id: string): number => tabEls.get(id)?.offsetWidth ?? 0

  const totalAll = ids.reduce((s, id) => s + width(id), 0) + gap * Math.max(0, ids.length - 1)
  if (totalAll <= avail) {
    skip.value = 0
    visibleIds.value = new Set(ids)
    return
  }

  const rest = tabs
    .filter(tb => tb.id !== terminalStore.activeTabID)
    .sort((a, b) => (lastUsed.get(b.id) ?? -1) - (lastUsed.get(a.id) ?? -1))
    .map(tb => tb.id)
  const maxSkip = Math.max(0, rest.length - 1)
  if (skip.value > maxSkip) skip.value = maxSkip
  const pool = [
    ...(terminalStore.activeTabID && ids.includes(terminalStore.activeTabID) ? [terminalStore.activeTabID] : []),
    ...rest.slice(skip.value),
  ]

  const set = new Set<string>()
  let used = 0
  for (const id of pool) {
    const w = width(id)
    const extra = set.size > 0 ? gap : 0
    if (set.size > 0 && used + extra + w > avail) break
    set.add(id)
    used += extra + w
  }
  if (set.size === 0 && pool.length > 0) set.add(pool[0]) // 极窄兜底：至少留激活项
  visibleIds.value = set
}

function onWheel(e: WheelEvent): void {
  if (overflowCount.value === 0) return
  e.preventDefault()
  const d = e.deltaY + e.deltaX > 0 ? 1 : -1
  skip.value = Math.max(0, skip.value + d)
}

function hiddenTabOptions(): DropdownOption[] {
  return overflowTabs.value.map((tab) => ({
    key: tab.id,
    label: () => h('div', { style: 'display:flex;align-items:center;gap:6px;min-width:120px;max-width:260px' }, [
      tab.type !== 'editor'
        ? h('span', { style: `width:7px;height:7px;border-radius:50%;flex-shrink:0;background:${statusDotColor(tab)}` })
        : h('span', { style: 'font-size:11px;font-weight:600;flex-shrink:0;color:var(--color-info)' }, tab.isRemote ? t('sftp.remotePrefix') : t('sftp.localPrefix')),
      h('span', { style: 'overflow:hidden;text-overflow:ellipsis;white-space:nowrap' }, tab.title),
      tab.dirty ? h('span', { style: 'width:8px;height:8px;border-radius:50%;flex-shrink:0;background:var(--warning-accent)' }) : null,
    ]),
  }))
}

function selectHidden(key: string | number) {
  terminalStore.activeTabID = String(key)
}

watch(() => terminalStore.tabs.map(tb => tb.id).join('\u0001'), () => {
  const alive = new Set(terminalStore.tabs.map(tb => tb.id))
  for (const id of [...lastUsed.keys()]) {
    if (!alive.has(id)) lastUsed.delete(id)
  }
  scheduleMeasure()
})
watch(() => terminalStore.activeTabID, id => {
  if (id) lastUsed.set(id, Date.now())
  scheduleMeasure()
}, { immediate: true })
watch(skip, () => scheduleMeasure())
// 溢出按钮显隐改变可用宽度，显隐后补测一轮收敛
watch(overflowCount, () => scheduleMeasure())

onMounted(() => {
  if (typeof ResizeObserver !== 'undefined') {
    stripRO = new ResizeObserver(() => scheduleMeasure())
    if (viewportRef.value) stripRO.observe(viewportRef.value)
    for (const el of tabEls.values()) stripRO.observe(el)
  }
  // 首帧同步收敛，避免先全显再裁剪闪一下
  measure()
  if (typeof document !== 'undefined' && 'fonts' in document) {
    document.fonts.ready.then(() => scheduleMeasure()).catch(() => {})
  }
})
onBeforeUnmount(() => {
  stripRO?.disconnect()
  stripRO = null
})
</script>

<template>
  <!-- 会话标签行（设计语言 §4.4）：40 行 · 32 药丸 · 圆角 8 · 间距 6；
       超宽折叠（GMark GEditorTabs 同款）：放不下的 LRU 折叠进右端下拉，
       滚轮按最近使用序平移可见窗口 -->
  <div
    ref="viewportRef"
    class="relative shrink-0 min-w-0 flex items-center gap-[var(--space-3)] px-[6px] overflow-hidden select-none"
    :style="{ height: 'var(--tab-row-h)' }"
    @wheel="onWheel"
  >
    <NTooltip v-for="tab in terminalStore.tabs" :key="tab.id" :delay="240" placement="bottom">
      <template #trigger>
        <div
          :ref="el => setTabEl(tab.id, el)"
          class="tab-pill flex items-center gap-[var(--space-3)] cursor-pointer"
          :class="{ 'tab-pill-active': tab.id === terminalStore.activeTabID, 'tab-pill-overflowed': isHidden(tab.id) }"
          role="tab"
          :data-tab-id="tab.id"
          :aria-selected="tab.id === terminalStore.activeTabID"
          @click="terminalStore.activeTabID = tab.id"
          @mouseenter="hoveredTab = tab.id"
          @mouseleave="hoveredTab = null"
          @contextmenu="onTabContextMenu($event, tab)"
          @mousedown.middle.prevent="handleClose(tab.id)"
        >
          <span v-if="tab.type !== 'editor'" class="tab-dot shrink-0" :style="{ background: statusDotColor(tab) }" />
          <template v-else>
            <span v-if="tab.isRemote === true" class="tab-tag-remote">{{ t('sftp.remotePrefix') }}</span>
            <span v-else-if="tab.isRemote === false" class="tab-tag-local">{{ t('sftp.localPrefix') }}</span>
          </template>

          <span class="tab-title truncate">{{ tab.title }}</span>

          <!-- 关闭钮/脏点同位叠放（G28）：脏且未悬停=点；悬停或激活未修改=关闭钮 -->
          <span class="tab-close-slot relative shrink-0">
            <button
              class="tab-close"
              :class="{ 'tab-close-visible': hoveredTab === tab.id || (tab.id === terminalStore.activeTabID && !tab.dirty) }"
              :title="t('tab.close')"
              @click.stop="handleClose(tab.id)"
            ><IconX :size="16" /></button>
            <span
              v-if="tab.dirty && hoveredTab !== tab.id"
              class="tab-dirty-dot"
            />
          </span>
        </div>
      </template>
      {{ tab.tooltip || tab.title }}
    </NTooltip>

    <!-- 溢出钮：行内右侧；v-show 隐形时 offsetWidth=0，显形后 watch overflowCount 补测收敛。
         注意：NDropdown 触发器放默认插槽（无 #trigger 具名槽，错放会让 VTarget 抛
         slot[default] should have exactly one child，整树挂载失败） -->
    <span v-show="overflowCount > 0" ref="overflowWrapRef" class="tab-overflow-btn">
      <NDropdown
        trigger="click"
        placement="bottom-end"
        :options="hiddenTabOptions()"
        @select="selectHidden"
      >
        <button class="icon-btn w-6 h-6" :title="t('tab.showAll')">
          <IconChevronDown :size="14" />
        </button>
      </NDropdown>
    </span>

    <NDropdown
      trigger="manual"
      :show="ctxTabID !== null"
      :x="ctxX"
      :y="ctxY"
      :options="getContextOptions()"
      @select="handleContextSelect"
      @clickoutside="ctxTabID = null"
      placement="bottom-start"
    />
  </div>
</template>

<style scoped>
/* 保自然宽度不缩水（flex:0 0 auto）；极窄行经 --v-tab-max 随行收缩（≥100px） */
.tab-pill {
  height: var(--tab-h);
  flex: 0 0 auto;
  max-width: var(--v-tab-max, 300px);
  padding: 0 var(--space-5);
  border-radius: var(--radius-m);
  opacity: 0.75;
  transition: background-color 150ms ease, opacity 150ms ease;
}
/* 被折叠的标签：脱离流但仍可测量（offsetWidth），供 measure() 计算自然宽度 */
.tab-pill-overflowed {
  position: absolute;
  left: -9999px;
  top: 0;
  visibility: hidden;
  pointer-events: none;
}
/* 溢出下拉钮：行内右侧 */
.tab-overflow-btn {
  flex: none;
  margin-left: auto;
  display: inline-flex;
  align-items: center;
}
.tab-pill:hover {
  background: var(--hover-gray);
  opacity: 1;
}
.tab-pill-active {
  background: var(--tab-selected-bg);
  opacity: 1;
  color: var(--text-primary);
}
.tab-pill-active:hover {
  background: var(--tab-selected-bg);
}

.tab-dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
}

.tab-title {
  font-size: var(--font-size-sm);
  white-space: nowrap;
}

/* 关闭钮：visibility 切换零抖动；20 命中 / 16 图标 / padding 0 */
.tab-close-slot {
  width: 20px;
  height: 20px;
}
.tab-close {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 0;
  border: none;
  background: transparent;
  color: var(--text-secondary);
  border-radius: var(--radius-s);
  cursor: pointer;
  visibility: hidden;
  transition: color 150ms ease;
}
.tab-close:hover {
  color: var(--text-primary);
}
.tab-close-visible {
  visibility: visible;
}

/* 脏点：与关闭钮同位，纯指示器不可点 */
.tab-dirty-dot {
  position: absolute;
  left: 50%;
  top: 50%;
  width: 8px;
  height: 8px;
  transform: translate(-50%, -50%);
  border-radius: 50%;
  background: var(--warning-accent);
  pointer-events: none;
}

.tab-tag-remote,
.tab-tag-local {
  font-size: var(--font-size-xs);
  font-weight: 600;
  white-space: nowrap;
}
.tab-tag-remote { color: var(--color-info); }
.tab-tag-local { color: var(--color-success); }
</style>
