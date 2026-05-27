import { ref, onUnmounted, type Ref } from 'vue'

// --- Module-level singleton state ---
interface DragPayload {
  source: 'remote' | 'local'
  paths: string[]
  label: string
}

interface SourceOptions {
  source: 'remote' | 'local'
  getSelectedPaths: () => Set<string>
  getFilePath: (item: any) => string
  getFileLabel: (item: any) => string
}

interface DropConfig {
  acceptedSource: 'remote' | 'local'
  isDragOver: Ref<boolean>
  onDrop: (paths: string[]) => void
}

const DRAG_THRESHOLD = 5

let active = false
let dragging = false
let payload: DragPayload | null = null
let ghostEl: HTMLElement | null = null
let overlayEl: HTMLElement | null = null
let startX = 0
let startY = 0
let sourceItem: any = null
let sourceOptions: SourceOptions | null = null

const dropTargets = new Map<HTMLElement, DropConfig>()

// --- Overlay (blocks all interaction from mousedown) ---
function createOverlay(): HTMLElement {
  const el = document.createElement('div')
  el.style.cssText = 'position:fixed;inset:0;z-index:99998;'
  document.body.appendChild(el)
  return el
}

function removeOverlay() {
  if (overlayEl) {
    overlayEl.remove()
    overlayEl = null
  }
}

// --- Ghost (created on drag threshold) ---
function createGhost(label: string, count: number): HTMLElement {
  const el = document.createElement('div')
  el.className = 'drag-ghost'
  el.style.pointerEvents = 'none'
  const icon = document.createElement('span')
  icon.className = 'drag-ghost-icon'
  icon.textContent = '\u{1F4C4}'
  el.appendChild(icon)
  const lbl = document.createElement('span')
  lbl.className = 'drag-ghost-label'
  lbl.textContent = label
  el.appendChild(lbl)
  if (count > 1) {
    const badge = document.createElement('span')
    badge.className = 'drag-ghost-count'
    badge.textContent = `+${count - 1}`
    el.appendChild(badge)
  }
  document.body.appendChild(el)
  return el
}

function updateGhost(x: number, y: number) {
  if (ghostEl) {
    ghostEl.style.left = x + 12 + 'px'
    ghostEl.style.top = y + 12 + 'px'
  }
}

function removeGhost() {
  if (ghostEl) {
    ghostEl.remove()
    ghostEl = null
  }
}

function hitTest(cx: number, cy: number): DropConfig | null {
  for (const [el, cfg] of dropTargets) {
    const r = el.getBoundingClientRect()
    if (cx >= r.left && cx <= r.right && cy >= r.top && cy <= r.bottom) {
      return cfg
    }
  }
  return null
}

function resetHighlights() {
  for (const [, cfg] of dropTargets) {
    cfg.isDragOver.value = false
  }
}

// --- Safety net: cleanup on blur / escape / next mousedown ---
function onWindowBlur() {
  if (active) cleanupDrag()
}

function onKeyDown(e: KeyboardEvent) {
  if (e.key === 'Escape' && active) {
    cleanupDrag()
  }
}

function onSafetyMouseDown() {
  if (active) cleanupDrag()
}

// --- Global mouse handlers ---
function onGlobalMouseMove(e: MouseEvent) {
  if (!active) return
  const dx = e.clientX - startX
  const dy = e.clientY - startY

  if (!dragging) {
    if (Math.sqrt(dx * dx + dy * dy) < DRAG_THRESHOLD) return
    dragging = true

    const opts = sourceOptions!
    const selectedPaths = opts.getSelectedPaths()
    const itemPath = opts.getFilePath(sourceItem)
    let paths: string[]
    if (selectedPaths.size > 0 && selectedPaths.has(itemPath)) {
      paths = Array.from(selectedPaths)
    } else {
      paths = [itemPath]
    }
    const label = paths.length === 1 ? opts.getFileLabel(sourceItem) : `${paths.length} files`
    payload = { source: opts.source, paths, label }
    ghostEl = createGhost(label, paths.length)

    document.body.style.cursor = 'grabbing'
    document.body.style.userSelect = 'none'
  }

  updateGhost(e.clientX, e.clientY)

  if (dragging) {
    resetHighlights()
    const hit = hitTest(e.clientX, e.clientY)
    if (hit && payload && hit.acceptedSource === payload.source) {
      hit.isDragOver.value = true
    }
  }
}

function onGlobalMouseUp(e: MouseEvent) {
  try {
    if (dragging && payload) {
      const hit = hitTest(e.clientX, e.clientY)
      if (hit && hit.acceptedSource === payload.source) {
        hit.onDrop(payload.paths)
      }
      window.addEventListener('click', suppressClick, true)
    }

    const wasDrag = dragging

    cleanupDrag()

    // If was a plain click (no drag), re-dispatch click on whatever element
    // was under the cursor at mousedown (overlay blocked original click)
    if (!wasDrag) {
      const el = document.elementFromPoint(startX, startY)
      if (el) {
        el.dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true }))
      }
    }
  } catch (err) {
    console.error('[useDragTransfer] onGlobalMouseUp error:', err)
    cleanupDrag()
  }
}

function suppressClick(e: MouseEvent) {
  e.stopPropagation()
  e.preventDefault()
  window.removeEventListener('click', suppressClick, true)
}

function cleanupDrag() {
  removeGhost()
  removeOverlay()
  resetHighlights()
  document.body.style.cursor = ''
  document.body.style.userSelect = ''
  window.removeEventListener('mousemove', onGlobalMouseMove, true)
  window.removeEventListener('mouseup', onGlobalMouseUp, true)
  window.removeEventListener('blur', onWindowBlur)
  window.removeEventListener('keydown', onKeyDown, true)
  window.removeEventListener('mousedown', onSafetyMouseDown, true)
  window.removeEventListener('click', suppressClick, true)
  active = false
  dragging = false
  payload = null
  sourceItem = null
}

// --- Exported hooks ---

export function useDragSource(options: SourceOptions) {
  function onRowMouseDown(e: MouseEvent, item: any) {
    if (e.button !== 0) return
    startX = e.clientX
    startY = e.clientY
    active = true
    dragging = false
    sourceItem = item
    sourceOptions = options

    // Overlay immediately to block ALL interaction during potential drag
    overlayEl = createOverlay()

    // Use window capture phase for reliable event delivery in WKWebView
    window.addEventListener('mousemove', onGlobalMouseMove, true)
    window.addEventListener('mouseup', onGlobalMouseUp, true)
    window.addEventListener('blur', onWindowBlur)
    window.addEventListener('keydown', onKeyDown, true)
    window.addEventListener('mousedown', onSafetyMouseDown, true)
  }

  function cleanup() {
    if (active && sourceOptions === options) {
      cleanupDrag()
    }
  }

  return { onRowMouseDown, cleanup }
}

export function useDropTarget(options: {
  acceptedSource: 'remote' | 'local'
  onDrop: (paths: string[]) => void
}) {
  const targetRef = ref<HTMLElement | null>(null)
  const isDragOver = ref(false)

  const cfg: DropConfig = {
    acceptedSource: options.acceptedSource,
    isDragOver,
    onDrop: options.onDrop,
  }

  function register(el?: HTMLElement) {
    const t = el || targetRef.value
    if (t) {
      dropTargets.set(t, cfg)
    }
  }

  function unregister() {
    if (targetRef.value) {
      dropTargets.delete(targetRef.value)
    }
  }

  onUnmounted(unregister)

  return { targetRef, isDragOver, register, unregister }
}
