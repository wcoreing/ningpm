import { defineStore } from "pinia"
import { ref } from "vue"
import { api } from "../lib/api"
import type { Project } from "../types"

export const useProjectsStore = defineStore("projects", () => {
  const items = ref<Project[]>([])
  const loading = ref(false)
  const error = ref("")

  async function fetchAll() {
    loading.value = true
    error.value = ""
    try {
      items.value = await api<Project[]>("/api/projects")
    } catch (e) {
      error.value = e instanceof Error ? e.message : "加载项目失败"
      throw e
    } finally {
      loading.value = false
    }
  }

  async function create(body: { key: string; name: string; description?: string }) {
    const p = await api<Project>("/api/projects", {
      method: "POST",
      body: JSON.stringify(body),
    })
    items.value = [p, ...items.value.filter((x) => x.id !== p.id)]
    return p
  }

  async function update(id: string, body: { name: string; description?: string }) {
    const p = await api<Project>(`/api/projects/${id}`, {
      method: "PATCH",
      body: JSON.stringify(body),
    })
    const prev = items.value.find((x) => x.id === id)
    const next = { ...p, myRole: prev?.myRole || p.myRole, issueCount: prev?.issueCount ?? p.issueCount }
    upsert(next)
    return next
  }

  function upsert(p: Project) {
    const i = items.value.findIndex((x) => x.id === p.id)
    if (i >= 0) items.value[i] = p
    else items.value = [p, ...items.value]
  }

  return { items, loading, error, fetchAll, create, update, upsert }
})
