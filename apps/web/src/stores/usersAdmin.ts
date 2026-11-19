import { acceptHMRUpdate, defineStore } from "pinia"
import { ref } from "vue"
import { api } from "../lib/api"

export type AdminUser = {
  id: string
  name: string
  account: string
  email?: string
  isAdmin: boolean
  avatarUrl?: string
  createdAt?: string
}

export const useUsersAdminStore = defineStore("usersAdmin", () => {
  const items = ref<AdminUser[]>([])
  const loading = ref(false)
  const error = ref("")

  async function fetchAll() {
    loading.value = true
    error.value = ""
    try {
      items.value = await api<AdminUser[]>("/api/admin/users")
    } catch (e) {
      error.value = e instanceof Error ? e.message : "加载失败"
      throw e
    } finally {
      loading.value = false
    }
  }

  async function create(payload: {
    name: string
    account: string
    password: string
    isAdmin: boolean
  }) {
    const u = await api<AdminUser>("/api/admin/users", {
      method: "POST",
      body: JSON.stringify(payload),
    })
    items.value = [u, ...items.value.filter((x) => x.id !== u.id)]
    return u
  }

  async function update(
    id: string,
    payload: { name?: string; account?: string; password?: string; isAdmin?: boolean },
  ) {
    const u = await api<AdminUser>(`/api/admin/users/${id}`, {
      method: "PATCH",
      body: JSON.stringify(payload),
    })
    items.value = items.value.map((x) => (x.id === id ? u : x))
    return u
  }

  async function remove(id: string) {
    await api<{ deleted: boolean }>(`/api/admin/users/${id}`, { method: "DELETE" })
    items.value = items.value.filter((x) => x.id !== id)
  }

  return { items, loading, error, fetchAll, create, update, remove }
})

if (import.meta.hot) {
  import.meta.hot.accept(acceptHMRUpdate(useUsersAdminStore, import.meta.hot))
}
