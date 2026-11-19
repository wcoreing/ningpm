import { defineStore } from "pinia"
import { computed, ref } from "vue"
import { api } from "../lib/api"
import { useAuthStore } from "./auth"
import type { Cycle, Issue, Module, Project, ProjectMember, UserOption } from "../types"

export type CreateIssueBody = {
  type: string
  title: string
  description?: string
  priority: number
  assigneeId?: string
  cycleId?: string
  moduleId?: string
  status?: string
}

export type AddMemberBody =
  | { userId: string; role: string }
  | { account: string; name?: string; role: string }

export const useProjectStore = defineStore("project", () => {
  const auth = useAuthStore()

  const project = ref<Project | null>(null)
  const issues = ref<Issue[]>([])
  const members = ref<ProjectMember[]>([])
  const cycles = ref<Cycle[]>([])
  const modules = ref<Module[]>([])
  const users = ref<UserOption[]>([])
  const loading = ref(false)
  const error = ref("")

  const myRole = computed(() => members.value.find((m) => m.userId === auth.user?.id)?.role)
  const canManage = computed(() => myRole.value === "owner" || myRole.value === "admin")
  const canWrite = computed(() => myRole.value !== "viewer")
  const memberUsers = computed<UserOption[]>(() =>
    members.value.map((m) => ({
      id: m.userId,
      name: m.name,
      account: m.account,
      email: m.email,
      avatarUrl: m.avatarUrl,
    })),
  )
  const activeCycle = computed(() => cycles.value.find((c) => c.status === "active") || null)
  const canLeaveProject = computed(() => {
    if (!auth.user?.id || !myRole.value) return false
    if (myRole.value !== "owner") return true
    return members.value.filter((m) => m.role === "owner").length > 1
  })

  async function load(id: string) {
    loading.value = true
    error.value = ""
    try {
      const [p, list, ms, cs, mds, us] = await Promise.all([
        api<Project>(`/api/projects/${id}`),
        api<Issue[]>(`/api/issues?projectId=${id}&includeChildren=1`),
        api<ProjectMember[]>(`/api/projects/${id}/members`),
        api<Cycle[]>(`/api/projects/${id}/cycles`),
        api<Module[]>(`/api/projects/${id}/modules`),
        api<UserOption[]>("/api/users"),
      ])
      project.value = p
      issues.value = list
      members.value = ms
      cycles.value = cs
      modules.value = mds
      users.value = us
    } catch (e) {
      error.value = e instanceof Error ? e.message : "加载失败"
      throw e
    } finally {
      loading.value = false
    }
  }

  /** 仅拉成员/迭代/模块（工作台抽屉用） */
  async function loadMeta(projectId: string) {
    const [ms, cs, mds] = await Promise.all([
      api<ProjectMember[]>(`/api/projects/${projectId}/members`),
      api<Cycle[]>(`/api/projects/${projectId}/cycles`),
      api<Module[]>(`/api/projects/${projectId}/modules`),
    ])
    if (project.value?.id === projectId) {
      members.value = ms
      cycles.value = cs
      modules.value = mds
    }
    return { members: ms, cycles: cs, modules: mds }
  }

  function upsertIssue(updated: Issue) {
    const i = issues.value.findIndex((x) => x.id === updated.id)
    if (i >= 0) issues.value[i] = updated
    else issues.value = [updated, ...issues.value]
  }

  async function createIssue(body: CreateIssueBody) {
    if (!project.value) throw new Error("项目未加载")
    const created = await api<Issue>("/api/issues", {
      method: "POST",
      body: JSON.stringify({ projectId: project.value.id, ...body }),
    })
    await load(project.value.id)
    return created
  }

  async function moveIssue(issueId: string, status: string) {
    return reorderBoard(issueId, { status })
  }

  /** 看板排序：beforeId 上方、afterId 下方；都空则置顶（按最后修改） */
  async function reorderBoard(
    issueId: string,
    body: { status: string; beforeId?: string; afterId?: string },
  ) {
    const prev = issues.value
    const now = Date.now()
    const moving = issues.value.find((i) => i.id === issueId)
    let optimisticOrder = now
    if (body.beforeId || body.afterId) {
      const before = body.beforeId ? issues.value.find((i) => i.id === body.beforeId) : undefined
      const after = body.afterId ? issues.value.find((i) => i.id === body.afterId) : undefined
      if (before && after) optimisticOrder = ((before.boardOrder || 0) + (after.boardOrder || 0)) / 2
      else if (before) optimisticOrder = (before.boardOrder || now) - 1000
      else if (after) optimisticOrder = (after.boardOrder || now) + 1000
    }
    issues.value = issues.value.map((i) =>
      i.id === issueId
        ? {
            ...i,
            status: body.status as Issue["status"],
            boardOrder: optimisticOrder,
            updatedAt: moving?.updatedAt || i.updatedAt,
          }
        : i,
    )
    try {
      const updated = await api<Issue>(`/api/issues/${issueId}/board`, {
        method: "PATCH",
        body: JSON.stringify(body),
      })
      upsertIssue(updated)
      if (updated.parentId) {
        try {
          const parent = await api<Issue>(`/api/issues/${updated.parentId}`)
          upsertIssue(parent)
        } catch {
          /* parent refresh best-effort */
        }
      }
      return updated
    } catch (e) {
      issues.value = prev
      error.value = e instanceof Error ? e.message : "更新看板失败"
      throw e
    }
  }

  async function addMember(body: AddMemberBody) {
    if (!project.value) throw new Error("项目未加载")
    const res = await api<{
      member: ProjectMember
      createdUser: boolean
      temporaryPassword?: string
    }>(`/api/projects/${project.value.id}/members`, {
      method: "POST",
      body: JSON.stringify(body),
    })
    members.value = [...members.value.filter((m) => m.userId !== res.member.userId), res.member]
    if (!users.value.some((u) => u.id === res.member.userId)) {
      users.value = [
        ...users.value,
        {
          id: res.member.userId,
          name: res.member.name,
          account: res.member.account,
          email: res.member.email,
          avatarUrl: res.member.avatarUrl,
        },
      ]
    }
    return res
  }

  async function changeRole(userId: string, role: string) {
    if (!project.value) throw new Error("项目未加载")
    const updated = await api<ProjectMember>(
      `/api/projects/${project.value.id}/members/${userId}`,
      { method: "PATCH", body: JSON.stringify({ role }) },
    )
    members.value = members.value.map((m) => (m.userId === updated.userId ? updated : m))
    return updated
  }

  async function removeMember(userId: string) {
    if (!project.value) throw new Error("项目未加载")
    await api(`/api/projects/${project.value.id}/members/${userId}`, { method: "DELETE" })
    members.value = members.value.filter((m) => m.userId !== userId)
  }

  async function updateProject(body: { name: string; description?: string }) {
    if (!project.value) throw new Error("项目未加载")
    const updated = await api<Project>(`/api/projects/${project.value.id}`, {
      method: "PATCH",
      body: JSON.stringify(body),
    })
    project.value = { ...project.value, ...updated, myRole: project.value.myRole }
    return project.value
  }

  async function createCycle(body: {
    name: string
    goal?: string
    status: string
    startDate?: string
    endDate?: string
  }) {
    if (!project.value) throw new Error("项目未加载")
    await api<Cycle>(`/api/projects/${project.value.id}/cycles`, {
      method: "POST",
      body: JSON.stringify(body),
    })
    await load(project.value.id)
  }

  async function setCycleStatus(cycleId: string, status: Cycle["status"]) {
    if (!project.value) throw new Error("项目未加载")
    const updated = await api<Cycle>(`/api/projects/${project.value.id}/cycles/${cycleId}`, {
      method: "PATCH",
      body: JSON.stringify({ status }),
    })
    cycles.value = cycles.value.map((c) => (c.id === updated.id ? updated : c))
    return updated
  }

  function setModules(next: Module[]) {
    const ids = new Set(next.map((m) => m.id))
    const removed = modules.value.filter((m) => !ids.has(m.id)).map((m) => m.id)
    modules.value = next
    if (removed.length) {
      issues.value = issues.value.map((it) =>
        removed.includes(it.moduleId || "")
          ? { ...it, moduleId: undefined, moduleName: undefined }
          : it,
      )
    }
  }

  function clear() {
    project.value = null
    issues.value = []
    members.value = []
    cycles.value = []
    modules.value = []
    users.value = []
    error.value = ""
  }

  return {
    project,
    issues,
    members,
    cycles,
    modules,
    users,
    loading,
    error,
    myRole,
    canManage,
    canWrite,
    memberUsers,
    activeCycle,
    canLeaveProject,
    load,
    loadMeta,
    upsertIssue,
    createIssue,
    moveIssue,
    reorderBoard,
    addMember,
    changeRole,
    removeMember,
    updateProject,
    createCycle,
    setCycleStatus,
    setModules,
    clear,
  }
})
