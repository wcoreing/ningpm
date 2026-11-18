import { acceptHMRUpdate, defineStore } from "pinia"
import { computed, ref } from "vue"
import { api, ApiError, clearToken, getToken, setToken } from "../lib/api"
import { compressImage } from "../lib/image"

export type AuthUser = {
  id: string
  name: string
  email: string
  avatarUrl?: string
}

export const useAuthStore = defineStore("auth", () => {
  const token = ref(getToken())
  const user = ref<AuthUser | null>(null)
  const booted = ref(false)

  const isAuthed = computed(() => Boolean(token.value))

  async function login(email: string, password: string) {
    const data = await api<{ token: string; user: AuthUser }>("/api/auth/login", {
      method: "POST",
      body: JSON.stringify({ email, password }),
    })
    setToken(data.token)
    token.value = data.token
    user.value = data.user
  }

  async function fetchMe() {
    if (!token.value) {
      user.value = null
      booted.value = true
      return
    }
    try {
      user.value = await api<AuthUser>("/api/auth/me")
    } catch {
      clearToken()
      token.value = ""
      user.value = null
    } finally {
      booted.value = true
    }
  }

  async function updateProfile(name: string) {
    const updated = await api<AuthUser>("/api/auth/me", {
      method: "PATCH",
      body: JSON.stringify({ name }),
    })
    user.value = updated
  }

  async function changePassword(currentPassword: string, newPassword: string) {
    await api<{ changed: boolean }>("/api/auth/password", {
      method: "POST",
      body: JSON.stringify({ currentPassword, newPassword }),
    })
  }

  async function uploadAvatar(file: File) {
    const compressed = await compressImage(file, { maxEdge: 512, maxBytes: 500 * 1024 })
    const headers = new Headers()
    const t = getToken()
    if (t) headers.set("Authorization", `Bearer ${t}`)
    const body = new FormData()
    body.append("file", compressed)
    const res = await fetch("/api/auth/avatar", { method: "POST", headers, body })
    const json = (await res.json().catch(() => null)) as {
      ok?: boolean
      data?: AuthUser
      error?: string
    } | null
    if (!res.ok || !json?.ok || !json.data) {
      if (res.status === 401) {
        clearToken()
        token.value = ""
        user.value = null
      }
      throw new ApiError(res.status, json?.error || "上传失败")
    }
    user.value = {
      ...json.data,
      avatarUrl: json.data.avatarUrl
        ? `${json.data.avatarUrl}?t=${Date.now()}`
        : undefined,
    }
    return user.value
  }

  async function clearAvatar() {
    const updated = await api<AuthUser>("/api/auth/avatar", { method: "DELETE" })
    user.value = updated
  }

  function logout() {
    clearToken()
    token.value = ""
    user.value = null
  }

  return {
    token,
    user,
    booted,
    isAuthed,
    login,
    fetchMe,
    updateProfile,
    changePassword,
    uploadAvatar,
    clearAvatar,
    logout,
  }
})

if (import.meta.hot) {
  import.meta.hot.accept(acceptHMRUpdate(useAuthStore, import.meta.hot))
}
