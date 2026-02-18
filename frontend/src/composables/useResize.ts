import { Ref } from "vue"

type NullableElementRef = Ref<HTMLElement | null>

/**
 * Create a horizontal resizer bound to a container element.
 * - sizeRef is mutated with the new width (px)
 * - containerRef is used to compute bounds
 * - min: minimum width in px
 * - minOther: used to compute the maximum (container.width - minOther)
 */
export function createHorizontalResizer(opts: {
  containerRef: NullableElementRef
  sizeRef: Ref<number>
  min?: number
  minOther?: number
  max?: number
}) {
  const { containerRef, sizeRef } = opts
  const min = Math.max(0, opts.min ?? 200)
  const minOther = Math.max(0, opts.minOther ?? 200)

  let moveHandler: ((ev: PointerEvent) => void) | null = null
  let upHandler: ((ev: PointerEvent) => void) | null = null

  function getBounds() {
    const rect = containerRef.value?.getBoundingClientRect()
    const computedMax = rect ? Math.max(200, rect.width - minOther) : Infinity
    const max = typeof opts.max === "number" ? opts.max : computedMax
    return { min, max }
  }

  function clamp() {
    const { min: lo, max: hi } = getBounds()
    if (Number.isFinite(hi) && sizeRef.value > hi) sizeRef.value = Math.min(sizeRef.value, hi)
    if (sizeRef.value < lo) sizeRef.value = lo
  }

  function start(ev: PointerEvent) {
    ev.preventDefault()

    moveHandler = (e: PointerEvent) => {
      const rect = containerRef.value?.getBoundingClientRect()
      if (!rect) return
      const { min: lo, max: hi } = getBounds()
      let newW = Math.round(e.clientX - rect.left)
      newW = Math.max(lo, Math.min(newW, hi))
      sizeRef.value = newW
    }

    upHandler = (e: PointerEvent) => {
      if (moveHandler) window.removeEventListener("pointermove", moveHandler)
      if (upHandler) window.removeEventListener("pointerup", upHandler)
      moveHandler = null
      upHandler = null
      try {
        // release pointer capture when available
        e.target && (e.target as Element).releasePointerCapture?.(e.pointerId)
      } catch (err) {
        /* ignore */
      }
    }

    window.addEventListener("pointermove", moveHandler)
    window.addEventListener("pointerup", upHandler)
    try {
      ev.target && (ev.target as Element).setPointerCapture?.(ev.pointerId)
    } catch (err) {
      /* ignore */
    }
  }

  function destroy() {
    if (moveHandler) window.removeEventListener("pointermove", moveHandler)
    if (upHandler) window.removeEventListener("pointerup", upHandler)
    moveHandler = null
    upHandler = null
  }

  return { start, clamp, destroy }
}

/**
 * Create a vertical resizer that manipulates a single numeric height ref.
 * - sizeRef is mutated with the new height (px)
 * - getMax (optional) returns the current maximum allowed height
 * - min: minimum height in px
 */
export function createVerticalResizer(opts: {
  sizeRef: Ref<number>
  min?: number
  getMax?: () => number
}) {
  const { sizeRef } = opts
  const min = Math.max(0, opts.min ?? 80)
  const getMax = opts.getMax ?? (() => window.innerHeight)

  let startY = 0
  let startSize = 0
  let moveHandler: ((ev: PointerEvent) => void) | null = null
  let upHandler: ((ev: PointerEvent) => void) | null = null

  function clamp() {
    const hi = Math.max(min, getMax())
    if (sizeRef.value > hi) sizeRef.value = Math.min(sizeRef.value, hi)
    if (sizeRef.value < min) sizeRef.value = min
  }

  function start(ev: PointerEvent) {
    ev.preventDefault()
    startY = ev.clientY
    startSize = sizeRef.value || 0

    moveHandler = (e: PointerEvent) => {
      const delta = startY - e.clientY
      const hi = Math.max(min, getMax())
      let newH = Math.round(startSize + delta)
      newH = Math.max(min, Math.min(newH, hi))
      sizeRef.value = newH
    }

    upHandler = (e: PointerEvent) => {
      if (moveHandler) window.removeEventListener("pointermove", moveHandler)
      if (upHandler) window.removeEventListener("pointerup", upHandler)
      moveHandler = null
      upHandler = null
      try {
        e.target && (e.target as Element).releasePointerCapture?.(e.pointerId)
      } catch (err) {
        /* ignore */
      }
    }

    window.addEventListener("pointermove", moveHandler)
    window.addEventListener("pointerup", upHandler)
    try {
      ev.target && (ev.target as Element).setPointerCapture?.(ev.pointerId)
    } catch (err) {
      /* ignore */
    }
  }

  function destroy() {
    if (moveHandler) window.removeEventListener("pointermove", moveHandler)
    if (upHandler) window.removeEventListener("pointerup", upHandler)
    moveHandler = null
    upHandler = null
  }

  return { start, clamp, destroy }
}
