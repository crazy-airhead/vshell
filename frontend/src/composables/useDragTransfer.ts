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
let capturedPointerId: number | null = null

const dropTargets = new Map<HTMLElement, DropConfig>()

// --- Overlay (blocks all interaction from mousedown) ---
function createOverlay(): HTMLElement {
  if (overlayEl) {
    overlayEl.remove()
    overlayEl = null
  }
  const el = document.createElement('div')
  el.style.cssText = 'position:fixed;inset:0;z-index:99998;touch-action:none;'

  // Primary drop handler — pointerup fires reliably on the overlay
  // because setPointerCapture redirects all pointer events to it
  el.addEventListener('pointerup', onPointerUp)
  el.addEventListener('pointermove', onPointerMove)

  // Fallback: if pointer capture is lost without pointerup (rare WKWebView edge case),
  // a subsequent click on the overlay still triggers the drop
  el.addEventListener('click', (e: MouseEvent) => {
    if (dragging && payload) {
      const hit = hitTest(e.clientX, e.clientY)
      if (hit && hit.acceptedSource === payload.source) {
        hit.onDrop(payload.paths)
      }
    }
    cleanupDrag()
  })

  // Safety: cleanup if pointer capture is lost unexpectedly
  el.addEventListener('lostpointercapture', () => {
    if (active) cleanupDrag()
  })

  document.body.appendChild(el)
  return el
}

function removeOverlay() {
  if (overlayEl) {
    overlayEl.removeEventListener('pointerup', onPointerUp)
    overlayEl.removeEventListener('pointermove', onPointerMove)
    overlayEl.remove()
    overlayEl = null
  }
}

// --- Ghost (created on drag threshold, placed BELOW overlay so it never blocks events) ---
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

// --- Pointer event handlers (on overlay, via pointer capture) ---
function onPointerMove(e: PointerEvent) {
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

function onPointerUp(e: PointerEvent) {
  try {
    if (dragging && payload) {
      const hit = hitTest(e.clientX, e.clientY)
      if (hit && hit.acceptedSource === payload.source) {
        hit.onDrop(payload.paths)
      }
      window.addEventListener('click', suppressClick, true)
    }

    const wasDrag = dragging

    if (overlayEl && capturedPointerId !== null) {
      try { overlayEl.releasePointerCapture(capturedPointerId) } catch { /* ignore */ }
    }
    cleanupDrag()

    // Plain click re-dispatch (no drag happened)
    if (!wasDrag) {
      const el = document.elementFromPoint(startX, startY)
      if (el) {
        el.dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true }))
      }
    }
  } catch (err) {
    console.error('[useDragTransfer] onPointerUp error:', err)
    cleanupDrag()
  }
}

function suppressClick(e: MouseEvent) {
  e.stopPropagation()
  e.stopImmediatePropagation()
  window.removeEventListener('click', suppressClick, true)
}

// --- Safety: keyboard / blur cleanup ---
function onWindowBlur() {
  if (active) cleanupDrag()
}

function onKeyDown(e: KeyboardEvent) {
  if (e.key === 'Escape' && active) {
    cleanupDrag()
  }
}

// --- Fallback mouseup on window capture (safety net for WKWebView) ---
function onWindowMouseUp(e: MouseEvent) {
  if (!active) return
  if (dragging && payload) {
    const hit = hitTest(e.clientX, e.clientY)
    if (hit && hit.acceptedSource === payload.source) {
      hit.onDrop(payload.paths)
    }
  }
  cleanupDrag()
}

function cleanupDrag() {
  if (!active) return
  active = false
  capturedPointerId = null
  removeGhost()
  removeOverlay()
  resetHighlights()
  document.body.style.cursor = ''
  document.body.style.userSelect = ''
  window.removeEventListener('mouseup', onWindowMouseUp, true)
  window.removeEventListener('blur', onWindowBlur)
  window.removeEventListener('keydown', onKeyDown, true)
  window.removeEventListener('click', suppressClick, true)
  document.removeEventListener('pointermove', onDocumentPointerMove, true)
  document.removeEventListener('pointerup', onDocumentPointerUp, true)
  dragging = false
  payload = null
  sourceItem = null
  sourceOptions = null
}

// --- Exported hooks ---

function onDocumentPointerMove(e: PointerEvent) {
  onPointerMove(e)
}

function onDocumentPointerUp(e: PointerEvent) {
  onPointerUp(e)
}

export function useDragSource(options: SourceOptions) {
  function onRowPointerDown(e: PointerEvent, item: any) {
    if (e.button !== 0) return
    if (e.pointerType !== 'mouse') return

    // Prevent WKWebView from interpreting the drag as a native gesture
    // (swipe, force-click, etc.) which would swallow subsequent events.
    e.preventDefault()

    startX = e.clientX
    startY = e.clientY
    active = true
    dragging = false
    sourceItem = item
    sourceOptions = options
    capturedPointerId = e.pointerId

    overlayEl = createOverlay()
    overlayEl.setPointerCapture(e.pointerId)

    // Verify pointer capture was granted. In some WKWebView configurations
    // (secondary displays, certain macOS versions), setPointerCapture may
    // silently fail. Fall back to document-level pointer listeners.
    const captureOk = overlayEl.hasPointerCapture(e.pointerId)
    if (!captureOk) {
      console.warn('[useDragTransfer] setPointerCapture failed — using document-level fallback')
      document.addEventListener('pointermove', onDocumentPointerMove, true)
      document.addEventListener('pointerup', onDocumentPointerUp, true)
    }

    // Safety nets for WKWebView edge cases
    window.addEventListener('mouseup', onWindowMouseUp, true)
    window.addEventListener('blur', onWindowBlur)
    window.addEventListener('keydown', onKeyDown, true)
  }

  function cleanup() {
    if (active && sourceOptions === options) {
      cleanupDrag()
    }
  }

  return { onRowPointerDown, cleanup }
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
