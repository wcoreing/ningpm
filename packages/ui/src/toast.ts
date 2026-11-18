import { reactive } from "vue"

export type ToastTone = "success" | "error" | "info" | "warning"

export type ToastItem = {
  id: string
  message: string
  tone: ToastTone
}

type ToastState = {
  items: ToastItem[]
}

const state = reactive<ToastState>({ items: [] })

let seq = 0

function nextId() {
  seq += 1
  return `np-toast-${seq}-${Date.now()}`
}

/** Dismiss a toast by id. */
export function dismissToast(id: string) {
  const i = state.items.findIndex((t) => t.id === id)
  if (i >= 0) state.items.splice(i, 1)
}

/**
 * Show a transient result tip.
 * Prefer helpers: toast.success / toast.error / toast.info / toast.warning.
 */
export function showToast(message: string, tone: ToastTone = "info", duration = 2800) {
  const text = String(message || "").trim()
  if (!text) return ""
  const id = nextId()
  state.items.push({ id, message: text, tone })
  if (duration > 0) {
    window.setTimeout(() => dismissToast(id), duration)
  }
  return id
}

export const toast = {
  show: showToast,
  success: (message: string, duration?: number) => showToast(message, "success", duration ?? 2800),
  error: (message: string, duration?: number) => showToast(message, "error", duration ?? 3600),
  info: (message: string, duration?: number) => showToast(message, "info", duration ?? 2800),
  warning: (message: string, duration?: number) => showToast(message, "warning", duration ?? 3200),
  dismiss: dismissToast,
}

/** Reactive toast queue for NpToastHost. */
export function useToastState() {
  return state
}
