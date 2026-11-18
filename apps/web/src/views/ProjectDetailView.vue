<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, onUnmounted, ref, watch } from "vue"
import { storeToRefs } from "pinia"
import { useRoute, useRouter } from "vue-router"
import {
  NpBadge,
  NpButton,
  NpField,
  NpInput,
  NpModal,
  NpSelect,
  NpTextarea,
  toast,
} from "@ningpm/ui"
import {
  CYCLE_STATUS_LABELS,
  PRIORITY_LABELS,
  ROLE_LABELS,
  STATUS_LABELS,
  STATUS_ORDER,
  TYPE_LABELS,
  priorityTone,
  typeTone,
} from "../lib/labels"
import type { Cycle, Issue, Module, UserOption } from "../types"
import { useAuthStore } from "../stores/auth"
import { usePageHeaderStore } from "../stores/pageHeader"
import { useProjectStore } from "../stores/project"
import IssueDetailDrawer from "../components/IssueDetailDrawer.vue"
import ModuleTagPicker from "../components/ModuleTagPicker.vue"
import ProjectAttachmentsPanel from "../components/ProjectAttachmentsPanel.vue"
import UserAvatar from "../components/UserAvatar.vue"

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()
const pageHeader = usePageHeaderStore()
const projectStore = useProjectStore()
const {
  project,
  issues,
  members,
  cycles,
  modules,
  users: allUsers,
  loading,
  error,
  myRole,
  canManage,
  canWrite,
  memberUsers,
  activeCycle,
  canLeaveProject,
} = storeToRefs(projectStore)

/** board | list | cycles | members | attachments — 列表默认表格 */
const view = ref("board")
/** 工作区上次用的展示方式（从迭代/成员/附件返回时恢复） */
const workMode = ref<"board" | "list">("board")
const typeFilter = ref("all")
const cycleFilter = ref("active") // all | active | backlog | <cycleId>
const moduleFilter = ref("all")
const attachmentsPanel = ref<{ reload: () => Promise<void> } | null>(null)

const isWork = computed(() => view.value === "board" || view.value === "list")
const sideNav = computed(() => (isWork.value ? "work" : view.value))

const sideItems = [
  { value: "work", label: "看板", icon: "board" },
  { value: "cycles", label: "迭代", icon: "cycle" },
  { value: "attachments", label: "附件", icon: "attachments" },
  { value: "members", label: "成员", icon: "members" },
] as const

function goSide(value: (typeof sideItems)[number]["value"]) {
  if (value === "work") {
    view.value = workMode.value
    return
  }
  view.value = value
}
const open = ref(false)
const cycleOpen = ref(false)
const saving = ref(false)
const detailOpen = ref(false)
const activeIssueId = ref<string | null>(null)
const memberSaving = ref(false)
const cycleSaving = ref(false)
const inviteQuery = ref("")
const selectedUserId = ref("")
const inviteOpen = ref(false)
const addRole = ref("member")
const inviteHint = ref("")
const inviteRoot = ref<HTMLElement | null>(null)
const form = ref({
  type: "task",
  title: "",
  description: "",
  priority: "3",
  assigneeId: "",
  cycleId: "",
  moduleId: "",
})
const cycleForm = ref({
  name: "",
  goal: "",
  status: "planned",
  startDate: "",
  endDate: "",
})

const typeTabs = [
  { value: "all", label: "全部" },
  { value: "story", label: "需求" },
  { value: "task", label: "任务" },
  { value: "bug", label: "缺陷" },
]

const memberIdSet = computed(() => new Set(members.value.map((m) => m.userId)))

function userMatchesQuery(u: UserOption, q: string) {
  if (!q) return true
  const name = u.name.toLowerCase()
  const email = u.email.toLowerCase()
  const local = email.split("@")[0] || ""
  return name.includes(q) || local.includes(q) || email.startsWith(q)
}

const inviteCandidates = computed(() => {
  const q = inviteQuery.value.trim().toLowerCase()
  return allUsers.value
    .filter((u) => !memberIdSet.value.has(u.id) && userMatchesQuery(u, q))
    .slice(0, 8)
})

const inviteAlreadyMembers = computed(() => {
  const q = inviteQuery.value.trim().toLowerCase()
  if (!q) return [] as UserOption[]
  return allUsers.value
    .filter((u) => memberIdSet.value.has(u.id) && userMatchesQuery(u, q))
    .slice(0, 5)
})

const matchedInviteUser = computed(() => {
  if (selectedUserId.value) {
    return allUsers.value.find((u) => u.id === selectedUserId.value) || null
  }
  const q = inviteQuery.value.trim().toLowerCase()
  if (!q) return null
  return (
    allUsers.value.find(
      (u) => !memberIdSet.value.has(u.id) && (u.email.toLowerCase() === q || u.name.toLowerCase() === q),
    ) || null
  )
})

const canInviteNewEmail = computed(() => {
  const q = inviteQuery.value.trim().toLowerCase()
  if (!q.includes("@") || matchedInviteUser.value) return false
  const existing = allUsers.value.find((u) => u.email.toLowerCase() === q)
  if (existing && memberIdSet.value.has(existing.id)) return false
  return !existing
})

const inviteMenuVisible = computed(
  () =>
    inviteOpen.value &&
    (inviteCandidates.value.length > 0 ||
      inviteAlreadyMembers.value.length > 0 ||
      canInviteNewEmail.value ||
      Boolean(inviteQuery.value.trim())),
)

const cycleFilterTabs = computed(() => {
  const tabs = [
    { value: "all", label: "全部" },
    { value: "active", label: activeCycle.value ? activeCycle.value.name : "当前迭代" },
    { value: "backlog", label: "待规划" },
  ]
  for (const cy of cycles.value) {
    if (cy.status !== "active") {
      tabs.push({ value: cy.id, label: cy.name })
    }
  }
  return tabs
})

const filtered = computed(() => {
  let list = issues.value
  if (typeFilter.value !== "all") {
    list = list.filter((i) => i.type === typeFilter.value)
  }
  if (cycleFilter.value === "active") {
    const id = activeCycle.value?.id
    list = id ? list.filter((i) => i.cycleId === id) : list.filter((i) => !i.cycleId)
  } else if (cycleFilter.value === "backlog") {
    list = list.filter((i) => !i.cycleId)
  } else if (cycleFilter.value !== "all") {
    list = list.filter((i) => i.cycleId === cycleFilter.value)
  }
  if (moduleFilter.value !== "all") {
    list = list.filter((i) => i.moduleId === moduleFilter.value)
  }
  return list
})

const columns = computed(() =>
  STATUS_ORDER.map((status) => ({
    status,
    label: STATUS_LABELS[status],
    items: filtered.value.filter((i) => i.status === status),
  })),
)

function defaultWeekRange() {
  const now = new Date()
  const day = now.getDay() || 7
  const start = new Date(now)
  start.setDate(now.getDate() - day + 1)
  const end = new Date(start)
  end.setDate(start.getDate() + 6)
  const fmt = (d: Date) => d.toISOString().slice(0, 10)
  return { startDate: fmt(start), endDate: fmt(end) }
}

async function load() {
  const id = String(route.params.id)
  try {
    await projectStore.load(id)
    const p = project.value
    if (p) {
      pageHeader.set({
        title: p.name,
        badge: p.key,
        subtitle: p.description || "",
      })
    }
    if (!form.value.assigneeId && auth.user?.id) {
      form.value.assigneeId = auth.user.id
    }
    if (!form.value.cycleId && activeCycle.value) {
      form.value.cycleId = activeCycle.value.id
    }
  } catch {
    /* error already on store */
  }
}

function pickInviteUser(u: UserOption) {
  selectedUserId.value = u.id
  inviteQuery.value = `${u.name} <${u.email}>`
  inviteOpen.value = false
}

function onInviteInput() {
  selectedUserId.value = ""
  inviteOpen.value = true
}

watch(inviteQuery, () => {
  if (selectedUserId.value) {
    const picked = allUsers.value.find((u) => u.id === selectedUserId.value)
    const label = picked ? `${picked.name} <${picked.email}>` : ""
    if (inviteQuery.value === label) return
    selectedUserId.value = ""
  }
  inviteOpen.value = true
})

function parseInviteEmail(raw: string) {
  const angled = raw.match(/<([^>]+@[^>]+)>/)
  if (angled?.[1]) return angled[1].trim()
  return raw.trim()
}

async function addMember() {
  const matched = matchedInviteUser.value
  const email = matched?.email || parseInviteEmail(inviteQuery.value)
  if (!email) {
    toast.warning("请输入邮箱或选择系统账号")
    return
  }
  if (!matched && !email.includes("@")) {
    toast.warning("未匹配到账号时，请输入完整邮箱")
    return
  }
  memberSaving.value = true
  inviteHint.value = ""
  try {
    const body = matched
      ? { userId: matched.id, role: addRole.value }
      : { email, role: addRole.value }
    const res = await projectStore.addMember(body)
    if (res.createdUser && res.temporaryPassword) {
      inviteHint.value = `已创建账号 ${res.member.email}，临时密码：${res.temporaryPassword}（请自行告知对方并尽快修改）`
      toast.success(`已邀请 ${res.member.name}，请复制临时密码`)
    } else {
      toast.success(`已添加 ${res.member.name}`)
    }
    inviteQuery.value = ""
    selectedUserId.value = ""
    inviteOpen.value = false
    addRole.value = "member"
  } catch (e) {
    toast.error(e instanceof Error ? e.message : "添加失败")
  } finally {
    memberSaving.value = false
  }
}

function onInviteDocPointer(ev: PointerEvent) {
  if (!inviteRoot.value?.contains(ev.target as Node)) inviteOpen.value = false
}

async function changeRole(userId: string, role: string) {
  try {
    await projectStore.changeRole(userId, role)
  } catch (e) {
    toast.error(e instanceof Error ? e.message : "更新角色失败")
    await load()
  }
}

async function removeMember(userId: string) {
  const self = userId === auth.user?.id
  if (!confirm(self ? "确定退出该项目？" : "确定移除该成员？")) return
  try {
    await projectStore.removeMember(userId)
    if (self) {
      toast.success("已退出项目")
      await router.push({ name: "projects" })
    }
  } catch (e) {
    toast.error(e instanceof Error ? e.message : "移除失败")
  }
}

async function createIssue() {
  saving.value = true
  try {
    await projectStore.createIssue({
      type: form.value.type,
      title: form.value.title,
      description: form.value.description,
      priority: Number(form.value.priority),
      assigneeId: form.value.assigneeId || undefined,
      cycleId: form.value.cycleId || undefined,
      moduleId: form.value.moduleId || undefined,
      status: "todo",
    })
    open.value = false
    form.value.title = ""
    form.value.description = ""
  } catch (e) {
    toast.error(e instanceof Error ? e.message : "创建失败")
  } finally {
    saving.value = false
  }
}

function openCreateCycle() {
  const range = defaultWeekRange()
  cycleForm.value = {
    name: "本周迭代",
    goal: "",
    status: "planned",
    startDate: range.startDate,
    endDate: range.endDate,
  }
  cycleOpen.value = true
}

async function createCycle() {
  cycleSaving.value = true
  try {
    await projectStore.createCycle(cycleForm.value)
    cycleOpen.value = false
  } catch (e) {
    toast.error(e instanceof Error ? e.message : "创建迭代失败")
  } finally {
    cycleSaving.value = false
  }
}

async function setCycleStatus(cy: Cycle, status: Cycle["status"]) {
  try {
    await projectStore.setCycleStatus(cy.id, status)
  } catch (e) {
    toast.error(e instanceof Error ? e.message : "更新迭代失败")
  }
}

function onModulesUpdated(next: Module[]) {
  projectStore.setModules(next)
}

async function moveIssue(issueId: string, status: string) {
  try {
    await projectStore.moveIssue(issueId, status)
  } catch {
    /* store already rolled back */
  }
}

function onDrop(status: string, event: DragEvent) {
  event.preventDefault()
  const id = event.dataTransfer?.getData("text/issue-id")
  if (!id) return
  const current = issues.value.find((i) => i.id === id)
  if (!current || current.status === status) return
  void moveIssue(id, status)
}

function onDragStart(issueId: string, event: DragEvent) {
  event.dataTransfer?.setData("text/issue-id", issueId)
  event.dataTransfer!.effectAllowed = "move"
}

function memberById(userId?: string) {
  if (!userId) return undefined
  return members.value.find((m) => m.userId === userId)
}

async function onProjectAttachmentsChanged() {
  const id = String(route.params.id || "")
  if (id) await projectStore.load(id)
}

function openDetail(issueId: string) {
  activeIssueId.value = issueId
  detailOpen.value = true
}

function onIssueUpdated(updated: Issue) {
  projectStore.upsertIssue(updated)
}

watch(view, (v) => {
  if (v === "modules") view.value = "board"
  if (v === "board" || v === "list") workMode.value = v
})

function openIssueFromQuery() {
  const issueId = String(route.query.issueId || "")
  if (!issueId) return
  activeIssueId.value = issueId
  detailOpen.value = true
}

watch(() => route.params.id, load)
watch(
  () => [route.query.issueId, loading.value] as const,
  () => {
    if (!loading.value) openIssueFromQuery()
  },
)
onMounted(async () => {
  document.addEventListener("pointerdown", onInviteDocPointer)
  await load()
  openIssueFromQuery()
})
onUnmounted(() => {
  document.removeEventListener("pointerdown", onInviteDocPointer)
  pageHeader.clear()
  projectStore.clear()
})
onBeforeUnmount(() => document.removeEventListener("pointerdown", onInviteDocPointer))
</script>

<template>
  <div class="page">
    <aside class="side" aria-label="项目菜单">
      <p class="side-label">项目</p>
      <nav class="side-nav">
        <button
          v-for="item in sideItems"
          :key="item.value"
          type="button"
          class="side-item"
          :class="{ 'is-on': sideNav === item.value }"
          @click="goSide(item.value)"
        >
          <svg v-if="item.icon === 'board'" viewBox="0 0 16 16" width="16" height="16" aria-hidden="true">
            <rect x="1.5" y="1.5" width="5" height="13" rx="1" fill="currentColor" />
            <rect x="9.5" y="1.5" width="5" height="8" rx="1" fill="currentColor" />
          </svg>
          <svg v-else-if="item.icon === 'cycle'" viewBox="0 0 16 16" width="16" height="16" aria-hidden="true">
            <path
              fill="currentColor"
              d="M8 1.5a6.5 6.5 0 1 0 6.2 8.3h-1.55A5 5 0 1 1 8 3v2.2L11.2 2.8 8 1.5z"
            />
          </svg>
          <svg v-else-if="item.icon === 'attachments'" viewBox="0 0 16 16" width="16" height="16" aria-hidden="true">
            <path
              fill="currentColor"
              d="M4.2 7.4l4.6-4.6a2.6 2.6 0 1 1 3.7 3.7L7 12a1.6 1.6 0 1 1-2.3-2.3l5-5-.9-.9-5 5A2.9 2.9 0 1 0 8 13.9l5.4-5.4a3.9 3.9 0 0 0-5.5-5.5L3.3 6.5l.9.9z"
            />
          </svg>
          <svg v-else viewBox="0 0 16 16" width="16" height="16" aria-hidden="true">
            <path
              fill="currentColor"
              d="M8 8a2.5 2.5 0 1 0 0-5 2.5 2.5 0 0 0 0 5zm-4.8 5.2c0-2.1 2.1-3.4 4.8-3.4s4.8 1.3 4.8 3.4V14H3.2v-.8z"
            />
          </svg>
          <span>{{ item.label }}</span>
        </button>
      </nav>
    </aside>

    <section class="main">
      <div v-if="view === 'cycles'" class="toolbar">
        <div class="title-row" />
        <div class="toolbar-actions">
          <NpButton v-if="canWrite" size="sm" variant="secondary" @click="openCreateCycle">
            新建迭代
          </NpButton>
        </div>
      </div>

      <div v-if="isWork" class="filter-bar">
        <div class="filter-groups">
          <div class="filter-group">
            <span class="filter-label">迭代</span>
            <NpSelect v-model="cycleFilter" size="sm">
              <option v-for="t in cycleFilterTabs" :key="t.value" :value="t.value">{{ t.label }}</option>
            </NpSelect>
          </div>
          <div class="filter-group">
            <span class="filter-label">模块</span>
            <NpSelect v-model="moduleFilter" size="sm">
              <option value="all">全部模块</option>
              <option v-for="md in modules" :key="md.id" :value="md.id">{{ md.name }}</option>
            </NpSelect>
          </div>
          <div class="filter-group">
            <span class="filter-label">类型</span>
            <NpSelect v-model="typeFilter" size="sm">
              <option v-for="t in typeTabs" :key="t.value" :value="t.value">{{ t.label }}</option>
            </NpSelect>
          </div>
        </div>
        <NpButton size="sm" variant="primary" :disabled="!canWrite" @click="open = true">
          新建事项
        </NpButton>
      </div>

      <p v-if="error" class="error">{{ error }}</p>
      <p v-if="loading" class="muted">加载中…</p>

      <template v-else>
      <div v-if="isWork" class="data-panel">
        <div class="data-head">
          <div class="view-switch" role="tablist" aria-label="展示方式">
            <button
              type="button"
              class="view-btn"
              :class="{ 'is-on': view === 'board' }"
              role="tab"
              :aria-selected="view === 'board'"
              title="看板"
              @click="view = 'board'"
            >
              <span>看板</span>
            </button>
            <button
              type="button"
              class="view-btn"
              :class="{ 'is-on': view === 'list' }"
              role="tab"
              :aria-selected="view === 'list'"
              title="列表"
              @click="view = 'list'"
            >
              <span>列表</span>
            </button>
          </div>
        </div>

        <div v-if="view === 'board'" class="board">
          <section
            v-for="col in columns"
            :key="col.status"
            class="column"
            @dragover.prevent
            @drop="onDrop(col.status, $event)"
          >
            <header>
              <strong>{{ col.label }}</strong>
              <span>{{ col.items.length }}</span>
            </header>
            <article
              v-for="it in col.items"
              :key="it.id"
              class="card"
              draggable="true"
              @dragstart="onDragStart(it.id, $event)"
              @click="openDetail(it.id)"
            >
              <div class="card-row card-row--title">
                <h4>{{ it.title }}</h4>
                <UserAvatar
                  v-if="it.assigneeId"
                  :name="memberById(it.assigneeId)?.name || it.assigneeName"
                  :email="memberById(it.assigneeId)?.email"
                  :avatar-url="memberById(it.assigneeId)?.avatarUrl"
                  size="sm"
                />
              </div>
              <div class="card-row card-row--meta">
                <span class="key">{{ it.key }}</span>
                <NpBadge :tone="typeTone(it.type)">{{ TYPE_LABELS[it.type] }}</NpBadge>
                <NpBadge :tone="priorityTone(it.priority)">{{ PRIORITY_LABELS[it.priority] }}</NpBadge>
                <span v-if="it.childCount" class="child-pill">{{ it.childDoneCount || 0 }}/{{ it.childCount }}</span>
                <span v-if="it.attachmentCount" class="att-pill" title="附件数">附 {{ it.attachmentCount }}</span>
              </div>
            </article>
          </section>
        </div>

        <div v-else class="table-wrap">
          <table class="table">
            <thead>
              <tr>
                <th>编号</th>
                <th>类型</th>
                <th>标题</th>
                <th>子项</th>
                <th>附件</th>
                <th>模块</th>
                <th>优先级</th>
                <th>状态</th>
                <th>指派给</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="it in filtered" :key="it.id" class="click-row" @click="openDetail(it.id)">
                <td class="key">{{ it.key }}</td>
                <td><NpBadge :tone="typeTone(it.type)">{{ TYPE_LABELS[it.type] }}</NpBadge></td>
                <td>{{ it.title }}</td>
                <td class="nowrap">
                  <span v-if="it.childCount">{{ it.childDoneCount || 0 }}/{{ it.childCount }}</span>
                  <span v-else class="muted">—</span>
                </td>
                <td class="nowrap">
                  <span v-if="it.attachmentCount">{{ it.attachmentCount }}</span>
                  <span v-else class="muted">—</span>
                </td>
                <td>{{ it.moduleName || "—" }}</td>
                <td><NpBadge :tone="priorityTone(it.priority)">{{ PRIORITY_LABELS[it.priority] }}</NpBadge></td>
                <td @click.stop>
                  <NpSelect
                    :model-value="it.status"
                    size="sm"
                    @update:model-value="moveIssue(it.id, $event)"
                  >
                    <option v-for="s in STATUS_ORDER" :key="s" :value="s">{{ STATUS_LABELS[s] }}</option>
                  </NpSelect>
                </td>
                <td>{{ it.assigneeName || "—" }}</td>
              </tr>
            </tbody>
          </table>
          <p v-if="!filtered.length" class="muted empty">暂无事项</p>
        </div>
      </div>

      <div v-else-if="view === 'cycles'" class="table-wrap">
        <table class="table">
          <thead>
            <tr>
              <th>名称</th>
              <th>状态</th>
              <th>周期</th>
              <th>目标</th>
              <th>进度</th>
              <th v-if="canWrite">操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="cy in cycles" :key="cy.id">
              <td>{{ cy.name }}</td>
              <td>
                <NpBadge :tone="cy.status === 'active' ? 'accent' : cy.status === 'closed' ? 'success' : 'neutral'">
                  {{ CYCLE_STATUS_LABELS[cy.status] }}
                </NpBadge>
              </td>
              <td class="nowrap">{{ cy.startDate }} ~ {{ cy.endDate }}</td>
              <td class="goal-cell">{{ cy.goal || "—" }}</td>
              <td class="nowrap">{{ cy.doneCount }}/{{ cy.issueCount }}</td>
              <td v-if="canWrite" class="actions-cell" @click.stop>
                <NpButton size="sm" variant="ghost" @click="cycleFilter = cy.id; view = 'list'">查看事项</NpButton>
                <NpButton
                  v-if="cy.status !== 'active'"
                  size="sm"
                  variant="secondary"
                  @click="setCycleStatus(cy, 'active')"
                >
                  设为进行中
                </NpButton>
                <NpButton
                  v-if="cy.status !== 'closed'"
                  size="sm"
                  variant="ghost"
                  @click="setCycleStatus(cy, 'closed')"
                >
                  结束
                </NpButton>
              </td>
            </tr>
          </tbody>
        </table>
        <p v-if="!cycles.length" class="muted empty">还没有迭代。创建「本周迭代」后，把事项收进去推进。</p>
      </div>

      <div v-else-if="view === 'attachments'" class="attachments-view">
        <ProjectAttachmentsPanel
          v-if="project"
          ref="attachmentsPanel"
          :project-id="project.id"
          :can-write="canWrite"
          :active="view === 'attachments'"
          @open-issue="openDetail"
          @changed="onProjectAttachmentsChanged"
        />
      </div>

      <div v-else-if="view === 'members'" class="members">
        <div v-if="canManage" class="invite-box">
          <div class="invite-bar">
            <div ref="inviteRoot" class="invite-combo">
              <NpInput
                v-model="inviteQuery"
                placeholder="输入邮箱或姓名，自动匹配系统账号"
                @focus="inviteOpen = true"
                @input="onInviteInput"
                @keydown.enter.prevent="addMember"
                @keydown.down.prevent="inviteOpen = true"
                @keydown.esc.prevent="inviteOpen = false"
              />
              <div v-if="inviteMenuVisible" class="invite-menu">
                <button
                  v-for="u in inviteCandidates"
                  :key="u.id"
                  type="button"
                  class="invite-option"
                  @mousedown.prevent="pickInviteUser(u)"
                >
                  <UserAvatar :name="u.name" :email="u.email" :avatar-url="u.avatarUrl" size="sm" />
                  <span class="invite-option__text">
                    <strong>{{ u.name }}</strong>
                    <span>{{ u.email }}</span>
                  </span>
                </button>
                <div
                  v-for="u in inviteAlreadyMembers"
                  :key="`m-${u.id}`"
                  class="invite-option invite-option--disabled"
                >
                  <UserAvatar :name="u.name" :email="u.email" :avatar-url="u.avatarUrl" size="sm" />
                  <span class="invite-option__text">
                    <strong>{{ u.name }}</strong>
                    <span>{{ u.email }} · 已是成员</span>
                  </span>
                </div>
                <button
                  v-if="canInviteNewEmail"
                  type="button"
                  class="invite-option invite-option--new"
                  @mousedown.prevent="
                    selectedUserId = '';
                    inviteOpen = false;
                    void nextTick(() => addMember())
                  "
                >
                  <strong>邀请新邮箱</strong>
                  <span>{{ inviteQuery.trim() }}</span>
                </button>
                <p
                  v-if="!inviteCandidates.length && !inviteAlreadyMembers.length && !canInviteNewEmail"
                  class="invite-empty"
                >
                  无匹配账号
                </p>
              </div>
            </div>
            <NpSelect v-model="addRole" class="invite-role">
              <option value="admin">管理员</option>
              <option value="member">成员</option>
              <option value="viewer">只读</option>
            </NpSelect>
            <NpButton
              variant="primary"
              :disabled="memberSaving || !inviteQuery.trim()"
              @click="addMember"
            >
              {{ memberSaving ? "邀请中…" : "邀请" }}
            </NpButton>
          </div>
          <p v-if="inviteHint" class="invite-hint">{{ inviteHint }}</p>
        </div>

        <div class="table-wrap">
          <table class="table">
            <thead>
              <tr>
                <th>姓名</th>
                <th>邮箱</th>
                <th>角色</th>
                <th>操作</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="m in members" :key="m.userId">
                <td>
                  <div class="member-name">
                    <UserAvatar :name="m.name" :email="m.email" :avatar-url="m.avatarUrl" size="sm" />
                    <span>{{ m.name }}</span>
                  </div>
                </td>
                <td>{{ m.email }}</td>
                <td>
                  <NpSelect
                    v-if="canManage && !(m.role === 'owner' && myRole !== 'owner')"
                    :model-value="m.role"
                    size="sm"
                    @update:model-value="changeRole(m.userId, $event)"
                  >
                    <option v-if="myRole === 'owner'" value="owner">负责人</option>
                    <option value="admin">管理员</option>
                    <option value="member">成员</option>
                    <option value="viewer">只读</option>
                  </NpSelect>
                  <NpBadge v-else tone="accent">{{ ROLE_LABELS[m.role] }}</NpBadge>
                </td>
                <td>
                  <NpButton
                    v-if="m.userId === auth.user?.id && canLeaveProject"
                    size="sm"
                    variant="ghost"
                    @click="removeMember(m.userId)"
                  >
                    退出项目
                  </NpButton>
                  <NpButton
                    v-else-if="canManage && m.userId !== auth.user?.id && (m.role !== 'owner' || myRole === 'owner')"
                    size="sm"
                    variant="ghost"
                    @click="removeMember(m.userId)"
                  >
                    移除
                  </NpButton>
                  <span v-else class="muted">—</span>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
      </template>
    </section>

    <NpModal :open="open" title="新建事项" @close="open = false">
      <div class="form">
        <div class="form-row">
          <NpField label="类型">
            <NpSelect v-model="form.type">
              <option value="story">需求</option>
              <option value="task">任务</option>
              <option value="bug">缺陷</option>
              <option value="epic">史诗</option>
            </NpSelect>
          </NpField>
          <NpField label="优先级">
            <NpSelect v-model="form.priority">
              <option value="1">紧急</option>
              <option value="2">高</option>
              <option value="3">中</option>
              <option value="4">低</option>
            </NpSelect>
          </NpField>
        </div>
        <NpField label="标题">
          <NpInput v-model="form.title" placeholder="一句话说清楚要做什么" />
        </NpField>
        <NpField label="描述">
          <NpTextarea v-model="form.description" placeholder="背景、验收标准、复现步骤…" />
        </NpField>
        <NpField label="指派给">
          <NpSelect v-model="form.assigneeId">
            <option value="">未指派</option>
            <option v-for="u in memberUsers" :key="u.id" :value="u.id">{{ u.name }}</option>
          </NpSelect>
        </NpField>
        <NpField label="所属迭代">
          <NpSelect v-model="form.cycleId">
            <option value="">待规划</option>
            <option v-for="cy in cycles" :key="cy.id" :value="cy.id">
              {{ cy.name }}（{{ CYCLE_STATUS_LABELS[cy.status] }}）
            </option>
          </NpSelect>
        </NpField>
        <NpField label="模块标签">
          <ModuleTagPicker
            v-if="project"
            :project-id="project.id"
            :modules="modules"
            v-model="form.moduleId"
            :can-write="canWrite"
            @update:modules="onModulesUpdated"
          />
        </NpField>
      </div>
      <template #footer>
        <NpButton variant="ghost" @click="open = false">取消</NpButton>
        <NpButton variant="primary" :disabled="saving" @click="createIssue">
          {{ saving ? "提交中…" : "创建" }}
        </NpButton>
      </template>
    </NpModal>

    <NpModal :open="cycleOpen" title="新建迭代" @close="cycleOpen = false">
      <div class="form">
        <NpField label="名称">
          <NpInput v-model="cycleForm.name" placeholder="例如：本周迭代 / Sprint 12" />
        </NpField>
        <NpField label="目标">
          <NpTextarea v-model="cycleForm.goal" placeholder="本迭代要交付什么" :rows="3" />
        </NpField>
        <div class="form-row">
          <NpField label="开始">
            <NpInput v-model="cycleForm.startDate" type="date" />
          </NpField>
          <NpField label="结束">
            <NpInput v-model="cycleForm.endDate" type="date" />
          </NpField>
        </div>
        <NpField label="状态">
          <NpSelect v-model="cycleForm.status">
            <option value="planned">未开始</option>
            <option value="active">进行中</option>
            <option value="closed">已完成</option>
          </NpSelect>
        </NpField>
      </div>
      <template #footer>
        <NpButton variant="ghost" @click="cycleOpen = false">取消</NpButton>
        <NpButton variant="primary" :disabled="cycleSaving" @click="createCycle">
          {{ cycleSaving ? "创建中…" : "创建" }}
        </NpButton>
      </template>
    </NpModal>

    <IssueDetailDrawer
      :open="detailOpen"
      :issue-id="activeIssueId"
      :project-id="project?.id || null"
      :users="memberUsers"
      :cycles="cycles"
      :modules="modules"
      :can-write="canWrite"
      @close="detailOpen = false"
      @updated="onIssueUpdated"
      @open-issue="openDetail"
      @update:modules="onModulesUpdated"
    />
  </div>
</template>

<style scoped>
.page {
  display: grid;
  grid-template-columns: 176px minmax(0, 1fr);
  gap: 16px;
  align-items: stretch;
  min-height: calc(100vh - 120px);
  margin: -4px 0 0;
}

.side {
  display: flex;
  flex-direction: column;
  gap: 10px;
  align-self: start;
  position: sticky;
  top: 0;
  padding: 14px 10px;
  border: 1px solid var(--np-border);
  border-radius: var(--np-radius-lg);
  background:
    linear-gradient(180deg, color-mix(in srgb, #fff 88%, var(--np-bg)) 0%, var(--np-bg-elevated) 100%);
  box-shadow: var(--np-shadow-sm);
}

.side-label {
  margin: 0 8px 2px;
  color: var(--np-text-muted);
  font-size: 11px;
  font-weight: 700;
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

.side-nav {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.side-item {
  display: flex;
  align-items: center;
  gap: 10px;
  width: 100%;
  height: 38px;
  padding: 0 10px 0 12px;
  border: none;
  border-radius: var(--np-radius-md);
  background: transparent;
  color: var(--np-text-secondary);
  font-size: var(--np-text-sm);
  font-weight: 600;
  text-align: left;
  cursor: pointer;
  box-shadow: inset 3px 0 0 transparent;
  transition:
    background var(--np-duration-fast) var(--np-ease),
    color var(--np-duration-fast) var(--np-ease),
    box-shadow var(--np-duration-fast) var(--np-ease);
}

.side-item:hover {
  color: var(--np-text);
  background: color-mix(in srgb, var(--np-accent-muted) 55%, transparent);
}

.side-item.is-on {
  color: var(--np-accent-ink);
  background: color-mix(in srgb, var(--np-accent) 16%, #fff);
  box-shadow: inset 3px 0 0 var(--np-accent);
}

.side-item svg {
  flex-shrink: 0;
  opacity: 0.88;
}

.side-item.is-on svg {
  opacity: 1;
  color: var(--np-accent);
}

.main {
  display: flex;
  flex-direction: column;
  gap: 14px;
  min-width: 0;
  padding: 0;
}

.toolbar {
  display: flex;
  justify-content: space-between;
  gap: 12px;
  align-items: center;
}

.toolbar-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  flex-shrink: 0;
  align-items: center;
}

.title-row {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 10px;
  min-width: 0;
}

.data-panel {
  display: flex;
  flex-direction: column;
  gap: 12px;
  min-width: 0;
  padding: 12px;
  border: 1px solid var(--np-border);
  border-radius: var(--np-radius-lg);
  background:
    linear-gradient(180deg, color-mix(in srgb, #fff 72%, var(--np-bg)) 0%, var(--np-bg-elevated) 100%);
  box-shadow: var(--np-shadow-sm);
}

.data-head {
  display: flex;
  justify-content: flex-end;
  align-items: center;
  gap: 10px;
}

.view-switch {
  display: inline-flex;
  gap: 2px;
  padding: 3px;
  border-radius: var(--np-radius-md);
  background: color-mix(in srgb, var(--np-bg-sunken) 55%, #fff);
  border: 1px solid var(--np-border);
}

.view-btn {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  height: 26px;
  padding: 0 8px;
  border: none;
  border-radius: 3px;
  background: transparent;
  color: var(--np-text-muted);
  font-size: var(--np-text-xs);
  font-weight: 600;
  cursor: pointer;
  transition:
    background var(--np-duration-fast) var(--np-ease),
    color var(--np-duration-fast) var(--np-ease);
}

.view-btn:hover {
  color: var(--np-text);
}

.view-btn.is-on {
  background: var(--np-bg-elevated);
  color: var(--np-accent);
  box-shadow: var(--np-shadow-sm);
}

@media (max-width: 860px) {
  .page {
    grid-template-columns: 1fr;
    margin: 0;
  }

  .side {
    position: static;
    padding: 10px;
  }

  .side-label {
    margin-bottom: 4px;
  }

  .side-nav {
    flex-direction: row;
    flex-wrap: wrap;
    gap: 4px;
  }

  .side-item {
    width: auto;
    flex: 1 1 auto;
    justify-content: center;
    box-shadow: none;
  }

  .side-item.is-on {
    box-shadow: inset 0 -3px 0 var(--np-accent);
  }
}

.filter-bar {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: 12px 18px;
  padding: 10px 12px;
  border: 1px solid var(--np-border);
  border-radius: var(--np-radius-lg);
  background: var(--np-bg-elevated);
  box-shadow: var(--np-shadow-sm);
}

.filter-groups {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 14px 18px;
  min-width: 0;
}

.filter-group {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
}

.filter-group :deep(.np-select) {
  width: auto;
  min-width: 9.5rem;
}

.filter-label {
  flex-shrink: 0;
  color: var(--np-text-muted);
  font-size: var(--np-text-xs);
  font-weight: 700;
  letter-spacing: 0.04em;
}

.empty-cta {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 12px;
}

.board {
  display: grid;
  grid-template-columns: repeat(4, minmax(180px, 1fr));
  gap: 10px;
  align-items: start;
}

.column {
  min-height: 260px;
  padding: 10px;
  border-radius: var(--np-radius-lg);
  background:
    linear-gradient(180deg, color-mix(in srgb, var(--np-bg-sunken) 70%, #fff) 0%, color-mix(in srgb, var(--np-accent-muted) 40%, #fff) 100%);
  border: 1px solid var(--np-border);
}

.column header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 10px;
  padding: 2px 4px;
  color: var(--np-text);
  font-family: var(--np-font-display);
  font-size: var(--np-text-sm);
  font-weight: 700;
  letter-spacing: -0.02em;
}

.column header span {
  min-width: 20px;
  height: 20px;
  padding: 0 6px;
  border-radius: 999px;
  background: var(--np-bg-elevated);
  color: var(--np-accent-ink);
  font-family: var(--np-font-mono);
  font-size: 11px;
  font-variant-numeric: tabular-nums;
  line-height: 20px;
  text-align: center;
  border: 1px solid var(--np-border);
}

.card {
  display: flex;
  flex-direction: column;
  gap: 6px;
  min-width: 0;
  padding: 10px 11px;
  margin-bottom: 8px;
  border-radius: var(--np-radius-md);
  background: #fff;
  border: 1px solid var(--np-border);
  box-shadow: var(--np-shadow-sm);
  cursor: grab;
  transition:
    border-color var(--np-duration-fast) var(--np-ease),
    box-shadow var(--np-duration-fast) var(--np-ease),
    transform var(--np-duration-fast) var(--np-ease);
}

.card:hover {
  border-color: color-mix(in srgb, var(--np-accent) 55%, var(--np-border));
  box-shadow: var(--np-shadow-md);
  transform: translateY(-1px);
}

.card:active {
  cursor: grabbing;
}

.card-row {
  display: flex;
  align-items: center;
  gap: 6px;
  min-width: 0;
}

.card-row--title {
  justify-content: space-between;
}

.card-row--meta {
  flex-wrap: wrap;
}

.card :deep(.np-badge) {
  flex-shrink: 0;
  padding: 0 4px;
  height: 16px;
  font-size: 10px;
  line-height: 16px;
}

.card h4 {
  flex: 1;
  min-width: 0;
  margin: 0;
  font-size: var(--np-text-sm);
  font-weight: 600;
  line-height: 1.35;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.card .key {
  flex-shrink: 0;
  font-size: 10px;
  color: var(--np-text-muted);
  font-family: var(--np-font-mono);
}

.child-pill,
.att-pill {
  flex-shrink: 0;
  padding: 0 5px;
  height: 16px;
  border-radius: 3px;
  background: var(--np-accent-muted);
  color: var(--np-accent-ink);
  font-size: 10px;
  font-weight: 700;
  line-height: 16px;
  font-variant-numeric: tabular-nums;
}

.att-pill {
  background: color-mix(in srgb, var(--np-accent-muted) 80%, #fff);
  color: var(--np-accent-ink);
}

.module-tag {
  max-width: 7em;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: var(--np-accent);
}

.key {
  font-family: var(--np-font-mono);
  font-size: var(--np-text-xs);
  color: var(--np-text-muted);
}

.table {
  width: 100%;
  border-collapse: collapse;
}

.table th,
.table td {
  padding: 10px 8px;
  border-bottom: 1px solid var(--np-border);
  text-align: left;
  font-size: var(--np-text-sm);
  vertical-align: middle;
}

.table th {
  color: var(--np-text-muted);
  font-weight: 600;
}

.click-row {
  cursor: pointer;
}

.click-row:hover td {
  background: color-mix(in srgb, var(--np-accent-muted) 45%, transparent);
}

.table-wrap {
  overflow: auto;
  border: 1px solid var(--np-border);
  border-radius: var(--np-radius-md);
  box-shadow: none;
  background: #fff;
}

.data-panel > .table-wrap {
  margin-top: 2px;
}

.nowrap {
  white-space: nowrap;
}

.goal-cell {
  max-width: 20rem;
  color: var(--np-text-secondary);
}

.actions-cell {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.invite-box {
  margin-bottom: 16px;
  padding: 12px;
  border: 1px solid var(--np-border);
  border-radius: var(--np-radius-lg);
  background: var(--np-bg-elevated);
  box-shadow: var(--np-shadow-sm);
}

.invite-bar {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 7.5rem auto;
  gap: 10px;
  align-items: start;
}

.invite-combo {
  position: relative;
  min-width: 0;
}

.invite-combo :deep(.np-input) {
  width: 100%;
}

.invite-menu {
  position: absolute;
  z-index: 20;
  top: calc(100% + 4px);
  left: 0;
  right: 0;
  max-height: 220px;
  overflow: auto;
  padding: 6px;
  border: 1px solid var(--np-border);
  border-radius: var(--np-radius-md);
  background: var(--np-bg-elevated);
  box-shadow: var(--np-shadow-md);
}

.member-name {
  display: inline-flex;
  align-items: center;
  gap: 8px;
}

.invite-option {
  display: flex;
  flex-direction: row;
  align-items: center;
  gap: 10px;
  width: 100%;
  padding: 8px 10px;
  border: none;
  border-radius: var(--np-radius-sm);
  background: transparent;
  text-align: left;
  cursor: pointer;
  color: inherit;
}

.invite-option__text {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}

.invite-option:hover:not(.invite-option--disabled),
.invite-option--new {
  background: var(--np-accent-muted);
}

.invite-option--disabled {
  cursor: default;
  opacity: 0.72;
}

.invite-option strong {
  font-size: var(--np-text-sm);
}

.invite-option span,
.invite-empty {
  font-size: var(--np-text-xs);
  color: var(--np-text-muted);
}

.invite-empty {
  margin: 0;
  padding: 10px;
  text-align: center;
}

.invite-role {
  width: 7.5rem;
}

.invite-hint {
  margin: 10px 0 0;
  padding: 10px 12px;
  border-radius: var(--np-radius-sm);
  background: color-mix(in srgb, var(--np-warning-muted) 70%, var(--np-bg-elevated));
  border: 1px solid color-mix(in srgb, var(--np-warning) 35%, var(--np-border));
  color: var(--np-text);
  font-size: var(--np-text-sm);
  line-height: 1.5;
  word-break: break-all;
}

@media (max-width: 860px) {
  .invite-bar {
    grid-template-columns: 1fr;
  }

  .invite-role {
    width: 100%;
  }
}

.form {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.check {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: var(--np-text-sm);
  color: var(--np-text-secondary);
  cursor: pointer;
}

.form-row {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 12px;
}

.muted {
  margin: 6px 0 0;
  color: var(--np-text-muted);
  font-size: var(--np-text-sm);
}

.error {
  color: var(--np-danger);
}

.empty {
  padding: 12px 0 0;
}

@media (max-width: 1100px) {
  .board {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 700px) {
  .board {
    grid-template-columns: 1fr;
  }

  .form-row,
  .add-row {
    grid-template-columns: 1fr;
  }
}
</style>
