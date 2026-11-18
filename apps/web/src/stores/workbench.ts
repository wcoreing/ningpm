import { defineStore } from "pinia"
import { ref } from "vue"
import { api } from "../lib/api"
import type { Cycle, Issue, Module, ProjectMember } from "../types"

export type WorkbenchQuery = {
  projectId?: string
  type?: string
  status?: string
  mine?: boolean
}

export const useWorkbenchStore = defineStore("workbench", () => {
  const issues = ref<Issue[]>([])
  const loading = ref(false)
  const error = ref("")

  /** 打开跨项目事项抽屉时的上下文 */
  const contextMembers = ref<ProjectMember[]>([])
  const contextCycles = ref<Cycle[]>([])
  const contextModules = ref<Module[]>([])

  async function fetch(query: WorkbenchQuery = {}) {
    loading.value = true
    error.value = ""
    try {
      const q = new URLSearchParams()
      q.set("rootsOnly", "1")
      if (query.projectId && query.projectId !== "all") q.set("projectId", query.projectId)
      if (query.type && query.type !== "all") q.set("type", query.type)
      if (query.mine) q.set("mine", "1")
      if (query.status && query.status !== "all" && query.status !== "open") {
        q.set("status", query.status)
      }
      issues.value = await api<Issue[]>(`/api/issues?${q.toString()}`)
    } catch (e) {
      error.value = e instanceof Error ? e.message : "加载失败"
      throw e
    } finally {
      loading.value = false
    }
  }

  async function loadIssueContext(projectId: string) {
    try {
      const [ms, cs, mds] = await Promise.all([
        api<ProjectMember[]>(`/api/projects/${projectId}/members`),
        api<Cycle[]>(`/api/projects/${projectId}/cycles`),
        api<Module[]>(`/api/projects/${projectId}/modules`),
      ])
      contextMembers.value = ms
      contextCycles.value = cs
      contextModules.value = mds
    } catch {
      contextMembers.value = []
      contextCycles.value = []
      contextModules.value = []
    }
  }

  function upsertIssue(updated: Issue) {
    const i = issues.value.findIndex((x) => x.id === updated.id)
    if (i >= 0) issues.value[i] = updated
    else issues.value = [updated, ...issues.value]
  }

  return {
    issues,
    loading,
    error,
    contextMembers,
    contextCycles,
    contextModules,
    fetch,
    loadIssueContext,
    upsertIssue,
  }
})
