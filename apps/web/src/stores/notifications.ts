import { defineStore } from "pinia"
import { computed, ref } from "vue"
import { api } from "../lib/api"
import type { AppNotification } from "../types"

export const useNotificationsStore = defineStore("notifications", () => {
  const items = ref<AppNotification[]>([])
  const unread = ref(0)
  const loading = ref(false)

  const unreadLabel = computed(() => (unread.value > 99 ? "99+" : String(unread.value)))

  async function fetchCount() {
    try {
      const data = await api<{ count: number }>("/api/notifications/unread-count")
      unread.value = data.count
    } catch {
      /* ignore poll errors */
    }
  }

  async function fetchList() {
    loading.value = true
    try {
      items.value = await api<AppNotification[]>("/api/notifications")
      unread.value = items.value.filter((n) => !n.readAt).length
    } catch {
      items.value = []
    } finally {
      loading.value = false
    }
  }

  async function markAll() {
    await api("/api/notifications/read-all", { method: "POST" })
    items.value = items.value.map((n) => ({
      ...n,
      readAt: n.readAt || new Date().toISOString(),
    }))
    unread.value = 0
  }

  async function markRead(id: string) {
    await api(`/api/notifications/${id}/read`, { method: "POST" })
    const n = items.value.find((x) => x.id === id)
    if (n && !n.readAt) {
      n.readAt = new Date().toISOString()
      unread.value = Math.max(0, unread.value - 1)
    }
  }

  return {
    items,
    unread,
    loading,
    unreadLabel,
    fetchCount,
    fetchList,
    markAll,
    markRead,
  }
})
