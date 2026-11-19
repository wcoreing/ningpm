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
import {
  addDaysISO,
  dayOffset,
  enumerateDays,
  formatDueLabel,
  isDueThisWeek,
  isOverdue,
  todayISO,
} from "../lib/schedule"
import type { Cycle, Issue, Module, UserOption } from "../types"
import { exportProjectReport } from "../lib/reportExport"
import { useAuthStore } from "../stores/auth"
import { usePageHeaderStore } from "../stores/pageHeader"
import { useProjectStore } from "../stores/project"
import { useProjectsStore } from "../stores/projects"
import IssueDetailDrawer from "../components/IssueDetailDrawer.vue"
import ModuleTagPicker from "../components/ModuleTagPicker.vue"
import ProjectAttachmentsPanel from "../components/ProjectAttachmentsPanel.vue"
import UserAvatar from "../components/UserAvatar.vue"

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()
const pageHeader = usePageHeaderStore()
const projectStore = useProjectStore()
const projectsStore = useProjectsStore()
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

/** board | list | tree | schedule | cycles | reports | members | files */
const view = ref("board")
/** 工作区上次用的展示方式（从迭代/成员/文件/报表返回时恢复） */
const workMode = ref<"board" | "list" | "tree" | "schedule">("board")
const typeFilter = ref("all")
const statusFilter = ref("all") // all | todo | doing | review | done
const dueFilter = ref("all") // all | overdue | week | none
const cycleFilter = ref("all") // all | active | backlog | <cycleId>
const moduleFilter = ref("all")
const scheduleSpan = ref<"week" | "month">("week")
const scheduleAnchor = ref(todayISO())
const attachmentsPanel = ref<{ reload: () => Promise<void> } | null>(null)
/** 树形视图：折叠的根事项 id */
const treeCollapsed = ref<Set<string>>(new Set())

const isWork = computed(
  () =>
    view.value === "board" ||
    view.value === "list" ||
    view.value === "tree" ||
    view.value === "schedule",
)
const sideNav = computed(() => (isWork.value ? "work" : view.value))

const sideItems = [
  { value: "work", label: "看板", icon: "board" },
  { value: "cycles", label: "迭代", icon: "cycle" },
  { value: "reports", label: "报表", icon: "reports" },
  { value: "files", label: "文件", icon: "attachments" },
  { value: "members", label: "成员", icon: "members" },
] as const

function goSide(value: (typeof sideItems)[number]["value"]) {
  if (value === "work") {
    view.value = workMode.value
    return
  }
  view.value = value
}
const cycleOpen = ref(false)
const editOpen = ref(false)
const editSaving = ref(false)
const editForm = ref({ name: "", description: "" })
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
const cycleForm = ref({
  name: "",
  goal: "",
  status: "planned",
  startDate: "",
  endDate: "",
})

const typeTabs = [
  { value: "all", label: "全部" },
  { value: "epic", label: "史诗" },
  { value: "story", label: "需求" },
  { value: "task", label: "任务" },
  { value: "bug", label: "缺陷" },
]

const statusTabs = [
  { value: "all", label: "全部状态" },
  ...STATUS_ORDER.map((status) => ({ value: status, label: STATUS_LABELS[status] })),
]

const dueTabs = [
  { value: "all", label: "全部到期" },
  { value: "overdue", label: "已逾期" },
  { value: "week", label: "本周到期" },
  { value: "none", label: "未排期" },
]

const memberIdSet = computed(() => new Set(members.value.map((m) => m.userId)))

function userMatchesQuery(u: UserOption, q: string) {
  if (!q) return true
  const name = u.name.toLowerCase()
  const account = (u.account || u.email || "").toLowerCase()
  const local = account.split("@")[0] || ""
  return name.includes(q) || local.includes(q) || account.startsWith(q) || account.includes(q)
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
      (u) =>
        !memberIdSet.value.has(u.id) &&
        ((u.account || "").toLowerCase() === q ||
          (u.email || "").toLowerCase() === q ||
          u.name.toLowerCase() === q),
    ) || null
  )
})

const canInviteNewAccount = computed(() => {
  const q = inviteQuery.value.trim().toLowerCase()
  if (!q || matchedInviteUser.value) return false
  const existing = allUsers.value.find(
    (u) => (u.account || "").toLowerCase() === q || (u.email || "").toLowerCase() === q,
  )
  if (existing && memberIdSet.value.has(existing.id)) return false
  return !existing
})

const inviteMenuVisible = computed(
  () =>
    inviteOpen.value &&
    (inviteCandidates.value.length > 0 ||
      inviteAlreadyMembers.value.length > 0 ||
      canInviteNewAccount.value ||
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

/** 看板/列表只展示根事项；子任务在详情抽屉内推进 */
const rootIssues = computed(() => issues.value.filter((i) => !i.parentId))

const filtered = computed(() => {
  let list = rootIssues.value
  if (typeFilter.value !== "all") {
    list = list.filter((i) => i.type === typeFilter.value)
  }
  if (statusFilter.value !== "all") {
    list = list.filter((i) => i.status === statusFilter.value)
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
  if (dueFilter.value === "overdue") {
    list = list.filter((i) => isOverdue(i.dueDate, i.status))
  } else if (dueFilter.value === "week") {
    list = list.filter((i) => isDueThisWeek(i.dueDate))
  } else if (dueFilter.value === "none") {
    list = list.filter((i) => !i.dueDate)
  }
  return list
})

type StatRow = { key: string; label: string; total: number; done: number; doing: number; todo: number; review: number }

function buildRows(
  source: typeof issues.value,
  keyOf: (i: (typeof issues.value)[number]) => string,
  labelOf: (key: string, sample?: (typeof issues.value)[number]) => string,
): StatRow[] {
  const map = new Map<string, StatRow>()
  for (const it of source) {
    const key = keyOf(it) || "_"
    let row = map.get(key)
    if (!row) {
      row = { key, label: labelOf(key, it), total: 0, done: 0, doing: 0, todo: 0, review: 0 }
      map.set(key, row)
    }
    row.total += 1
    if (it.status === "done") row.done += 1
    else if (it.status === "doing") row.doing += 1
    else if (it.status === "review") row.review += 1
    else row.todo += 1
  }
  return [...map.values()].sort((a, b) => b.total - a.total || a.label.localeCompare(b.label, "zh"))
}

/** 报表：优先用已加载的子事项；若仅有根事项则用 childCount/childDoneCount 汇总 */
const reportScope = computed(() => issues.value)
const loadedChildren = computed(() => issues.value.filter((i) => Boolean(i.parentId)))

const reportSummary = computed(() => {
  const roots = rootIssues.value
  const childrenLoaded = loadedChildren.value
  const childTotal =
    childrenLoaded.length > 0
      ? childrenLoaded.length
      : roots.reduce((s, i) => s + (i.childCount || 0), 0)
  const childDone =
    childrenLoaded.length > 0
      ? childrenLoaded.filter((i) => i.status === "done").length
      : roots.reduce((s, i) => s + (i.childDoneCount || 0), 0)
  const rootDone = roots.filter((i) => i.status === "done").length
  const all = childrenLoaded.length > 0 ? reportScope.value : null
  const done = all
    ? all.filter((i) => i.status === "done").length
    : rootDone + childDone
  const doing = all
    ? all.filter((i) => i.status === "doing").length
    : roots.filter((i) => i.status === "doing").length
  const review = all
    ? all.filter((i) => i.status === "review").length
    : roots.filter((i) => i.status === "review").length
  const todo = all
    ? all.filter((i) => i.status === "todo").length
    : roots.filter((i) => i.status === "todo").length
  const total = all ? all.length : roots.length + childTotal
  const rate = total ? Math.round((done / total) * 100) : 0
  return {
    total,
    done,
    doing,
    review,
    todo,
    rate,
    roots: roots.length,
    rootDone,
    children: childTotal,
    childDone,
    childRate: childTotal ? Math.round((childDone / childTotal) * 100) : 0,
  }
})

const reportByStatus = computed(() => {
  if (loadedChildren.value.length > 0) {
    return STATUS_ORDER.map((status) => ({
      status,
      label: STATUS_LABELS[status],
      count: reportScope.value.filter((i) => i.status === status).length,
    }))
  }
  // 仅有根事项时：状态条按「根+子完成数」近似——子项未加载时把子完成计入 done，其余挂在根状态上
  return STATUS_ORDER.map((status) => {
    const rootCount = rootIssues.value.filter((i) => i.status === status).length
    const extraDone =
      status === "done"
        ? rootIssues.value.reduce((s, i) => s + (i.childDoneCount || 0), 0)
        : 0
    return { status, label: STATUS_LABELS[status], count: rootCount + extraDone }
  })
})

const reportByType = computed(() =>
  buildRows(
    loadedChildren.value.length > 0 ? reportScope.value : rootIssues.value,
    (i) => i.type,
    (key) => TYPE_LABELS[key] || key,
  ),
)

const reportByModule = computed(() =>
  buildRows(
    loadedChildren.value.length > 0 ? reportScope.value : rootIssues.value,
    (i) => i.moduleId || "",
    (key, sample) => (key ? sample?.moduleName || modules.value.find((m) => m.id === key)?.name || "未命名模块" : "未归模块"),
  ),
)

const reportByCycle = computed(() =>
  buildRows(
    loadedChildren.value.length > 0 ? reportScope.value : rootIssues.value,
    (i) => i.cycleId || "",
    (key, sample) => (key ? sample?.cycleName || cycles.value.find((c) => c.id === key)?.name || "未命名迭代" : "待办池"),
  ),
)

function pct(part: number, total: number) {
  if (!total) return 0
  return Math.round((part / total) * 100)
}

async function exportReport() {
  const p = project.value
  if (!p) return
  try {
    await exportProjectReport({
      projectKey: p.key,
      projectName: p.name,
      summary: reportSummary.value,
      byStatus: reportByStatus.value.map((row) => ({
        status: row.status,
        label: row.label,
        count: row.count,
      })),
      byType: reportByType.value,
      byCycle: reportByCycle.value,
      byModule: reportByModule.value,
    })
    toast.success("报表已导出")
  } catch (e) {
    toast.error(e instanceof Error ? e.message : "导出失败")
  }
}

function boardSort(a: Issue, b: Issue) {
  const ao = a.boardOrder ?? 0
  const bo = b.boardOrder ?? 0
  if (ao !== bo) return bo - ao
  return (b.updatedAt || "").localeCompare(a.updatedAt || "")
}

const columns = computed(() => {
  const statuses =
    statusFilter.value === "all"
      ? [...STATUS_ORDER]
      : STATUS_ORDER.filter((s) => s === statusFilter.value)
  return statuses.map((status) => ({
    status,
    label: STATUS_LABELS[status],
    items: filtered.value.filter((i) => i.status === status).sort(boardSort),
  }))
})

type TreeRow = {
  issue: Issue
  depth: number
  hasChildren: boolean
  expanded: boolean
  childTotal: number
  childDone: number
}

/** 树形：筛选后的根事项 + 其下全部子事项（便于看完成度） */
const treeRows = computed((): TreeRow[] => {
  const byParent = new Map<string, Issue[]>()
  for (const it of issues.value) {
    if (!it.parentId) continue
    const list = byParent.get(it.parentId)
    if (list) list.push(it)
    else byParent.set(it.parentId, [it])
  }
  const rows: TreeRow[] = []
  for (const root of filtered.value) {
    const kids = byParent.get(root.id) || []
    kids.sort((a, b) => a.number - b.number)
    const childTotal = kids.length || root.childCount || 0
    const childDone =
      kids.length > 0 ? kids.filter((k) => k.status === "done").length : root.childDoneCount || 0
    const hasChildren = childTotal > 0
    const expanded = hasChildren && !treeCollapsed.value.has(root.id)
    rows.push({
      issue: root,
      depth: 0,
      hasChildren,
      expanded,
      childTotal,
      childDone,
    })
    if (expanded) {
      for (const child of kids) {
        rows.push({
          issue: child,
          depth: 1,
          hasChildren: false,
          expanded: false,
          childTotal: 0,
          childDone: 0,
        })
      }
    }
  }
  return rows
})

function toggleTreeRoot(id: string) {
  const next = new Set(treeCollapsed.value)
  if (next.has(id)) next.delete(id)
  else next.add(id)
  treeCollapsed.value = next
}

function childProgressPct(done: number, total: number) {
  if (!total) return 0
  return Math.round((done / total) * 100)
}

/** 看板卡片上的紧凑「最后更新」文案 */
function formatUpdatedLabel(iso?: string) {
  if (!iso) return ""
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return ""
  const now = new Date()
  const diffMs = now.getTime() - d.getTime()
  const abs = Math.abs(diffMs)
  if (abs < 60_000) return "刚刚"
  if (abs < 3600_000) return `${Math.floor(abs / 60_000)} 分钟前`
  const sameDay =
    d.getFullYear() === now.getFullYear() &&
    d.getMonth() === now.getMonth() &&
    d.getDate() === now.getDate()
  const hm = `${String(d.getHours()).padStart(2, "0")}:${String(d.getMinutes()).padStart(2, "0")}`
  if (sameDay) return `今天 ${hm}`
  const yest = new Date(now)
  yest.setDate(now.getDate() - 1)
  if (
    d.getFullYear() === yest.getFullYear() &&
    d.getMonth() === yest.getMonth() &&
    d.getDate() === yest.getDate()
  ) {
    return `昨天 ${hm}`
  }
  if (d.getFullYear() === now.getFullYear()) {
    return `${d.getMonth() + 1}/${d.getDate()} ${hm}`
  }
  return `${d.getFullYear()}/${d.getMonth() + 1}/${d.getDate()}`
}

function formatUpdatedFull(iso?: string) {
  if (!iso) return ""
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return iso
  return d.toLocaleString("zh-CN", { hour12: false })
}

function dueClass(it: Issue) {
  if (!it.dueDate) return ""
  if (isOverdue(it.dueDate, it.status)) return "is-overdue"
  if (isDueThisWeek(it.dueDate)) return "is-due-soon"
  return ""
}

const scheduleDayCount = computed(() => (scheduleSpan.value === "week" ? 14 : 42))

const scheduleRangeStart = computed(() => {
  const anchor = scheduleAnchor.value || todayISO()
  if (scheduleSpan.value === "week") {
    const d = new Date(`${anchor}T00:00:00`)
    const day = d.getDay() || 7
    d.setDate(d.getDate() - day + 1)
    return todayISO(d)
  }
  const d = new Date(`${anchor}T00:00:00`)
  d.setDate(1)
  return todayISO(d)
})

const scheduleDays = computed(() => enumerateDays(scheduleRangeStart.value, scheduleDayCount.value))

type ScheduleRow = {
  issue: Issue
  depth: number
  barStart: number
  barSpan: number
  hasBar: boolean
  overdue: boolean
}

/** 排期：根事项 + 子事项；无日期的仍占行便于发现未排期 */
const scheduleRows = computed((): ScheduleRow[] => {
  const byParent = new Map<string, Issue[]>()
  for (const it of issues.value) {
    if (!it.parentId) continue
    const list = byParent.get(it.parentId)
    if (list) list.push(it)
    else byParent.set(it.parentId, [it])
  }
  const start = scheduleRangeStart.value
  const days = scheduleDayCount.value
  const rows: ScheduleRow[] = []

  function barFor(it: Issue) {
    const due = it.dueDate?.slice(0, 10) || ""
    const begin = (it.startDate || due).slice(0, 10)
    if (!due && !it.startDate) {
      return { barStart: 0, barSpan: 0, hasBar: false, overdue: false }
    }
    const end = due || begin
    let left = dayOffset(start, begin)
    let right = dayOffset(start, end)
    if (right < left) right = left
    const clippedLeft = Math.max(0, left)
    const clippedRight = Math.min(days - 1, right)
    if (clippedRight < 0 || clippedLeft >= days) {
      return { barStart: 0, barSpan: 0, hasBar: false, overdue: isOverdue(it.dueDate, it.status) }
    }
    return {
      barStart: clippedLeft,
      barSpan: Math.max(1, clippedRight - clippedLeft + 1),
      hasBar: true,
      overdue: isOverdue(it.dueDate, it.status),
    }
  }

  for (const root of filtered.value) {
    const kids = (byParent.get(root.id) || []).slice().sort((a, b) => a.number - b.number)
    rows.push({ issue: root, depth: 0, ...barFor(root) })
    for (const child of kids) {
      rows.push({ issue: child, depth: 1, ...barFor(child) })
    }
  }
  return rows
})

function shiftSchedule(delta: number) {
  scheduleAnchor.value = addDaysISO(scheduleAnchor.value || todayISO(), delta)
}

function resetScheduleToday() {
  scheduleAnchor.value = todayISO()
}

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

function syncPageHeader() {
  const p = project.value
  if (!p) return
  // 工作区（看板/列表/树形/排期）标题进面板顶栏，壳层只保留导航默认标题
  if (isWork.value) {
    pageHeader.clear()
    return
  }
  pageHeader.set({
    title: p.name,
    badge: p.key,
  })
}

function openEditProject() {
  const p = project.value
  if (!p) return
  editForm.value = { name: p.name, description: p.description || "" }
  editOpen.value = true
}

async function saveProject() {
  const name = editForm.value.name.trim()
  if (!name) {
    toast.warning("项目名称不能为空")
    return
  }
  editSaving.value = true
  try {
    const updated = await projectStore.updateProject({
      name,
      description: editForm.value.description.trim(),
    })
    projectsStore.upsert(updated)
    syncPageHeader()
    editOpen.value = false
    toast.success("项目已更新")
  } catch (e) {
    toast.error(e instanceof Error ? e.message : "更新失败")
  } finally {
    editSaving.value = false
  }
}

async function load() {
  const id = String(route.params.id)
  try {
    await projectStore.load(id)
    syncPageHeader()
  } catch {
    /* error already on store */
  }
}

function pickInviteUser(u: UserOption) {
  selectedUserId.value = u.id
  inviteQuery.value = `${u.name} <${u.account}>`
  inviteOpen.value = false
}

function onInviteInput() {
  selectedUserId.value = ""
  inviteOpen.value = true
}

watch(inviteQuery, () => {
  if (selectedUserId.value) {
    const picked = allUsers.value.find((u) => u.id === selectedUserId.value)
    const label = picked ? `${picked.name} <${picked.account}>` : ""
    if (inviteQuery.value === label) return
    selectedUserId.value = ""
  }
  inviteOpen.value = true
})

function parseInviteAccount(raw: string) {
  const angled = raw.match(/<([^>]+)>/)
  if (angled?.[1]) return angled[1].trim()
  return raw.trim()
}

async function addMember() {
  const matched = matchedInviteUser.value
  const account = matched?.account || parseInviteAccount(inviteQuery.value)
  if (!account) {
    toast.warning("请输入账号或选择系统用户")
    return
  }
  memberSaving.value = true
  inviteHint.value = ""
  try {
    const body = matched
      ? { userId: matched.id, role: addRole.value }
      : { account, role: addRole.value }
    const res = await projectStore.addMember(body)
    if (res.createdUser && res.temporaryPassword) {
      inviteHint.value = `已创建账号 ${res.member.account}，临时密码：${res.temporaryPassword}（请自行告知对方并尽快修改）`
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

function openCreate() {
  activeIssueId.value = null
  detailOpen.value = true
}

function onIssueCreated(issue: Issue) {
  projectStore.upsertIssue(issue)
  activeIssueId.value = issue.id
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

const dragIssueId = ref("")
const dropOverId = ref("") // 悬停目标卡片（插入其上方）
const dropOverStatus = ref("")
const dragMoved = ref(false)

function onDragStart(issueId: string, event: DragEvent) {
  dragIssueId.value = issueId
  dropOverId.value = ""
  dropOverStatus.value = ""
  dragMoved.value = false
  event.dataTransfer?.setData("text/issue-id", issueId)
  event.dataTransfer!.effectAllowed = "move"
}

function onDragEnd() {
  dragIssueId.value = ""
  dropOverId.value = ""
  dropOverStatus.value = ""
}

function onCardDragOver(status: string, targetId: string, event: DragEvent) {
  event.preventDefault()
  event.stopPropagation()
  if (!dragIssueId.value || dragIssueId.value === targetId) return
  dragMoved.value = true
  dropOverStatus.value = status
  dropOverId.value = targetId
}

function onColumnDragOver(status: string, event: DragEvent) {
  event.preventDefault()
  if (!dragIssueId.value) return
  dragMoved.value = true
  dropOverStatus.value = status
  // 悬停在泳道空白处：清掉卡片锚点，松手后置顶或落到末尾由落点逻辑处理
  dropOverId.value = ""
}

async function applyBoardDrop(status: string, afterId?: string) {
  const id = dragIssueId.value
  if (!id) return
  const colItems = filtered.value.filter((i) => i.status === status && i.id !== id).sort(boardSort)
  const resolvedAfter = afterId && afterId !== id ? afterId : undefined
  let beforeId: string | undefined
  if (resolvedAfter) {
    const idx = colItems.findIndex((i) => i.id === resolvedAfter)
    if (idx > 0) beforeId = colItems[idx - 1]?.id
  }
  // 泳道空白：置顶（与「默认按最后修改」一致）；拖到卡片上则插到该卡上方

  const full = filtered.value.filter((i) => i.status === status).sort(boardSort)
  const curIdx = full.findIndex((i) => i.id === id)
  if (curIdx >= 0) {
    if (resolvedAfter && curIdx === full.findIndex((i) => i.id === resolvedAfter) - 1) {
      onDragEnd()
      return
    }
    if (!resolvedAfter && curIdx === 0) {
      onDragEnd()
      return
    }
  }

  try {
    await projectStore.reorderBoard(id, {
      status,
      beforeId,
      afterId: resolvedAfter,
    })
  } catch {
    /* store rolled back */
  } finally {
    onDragEnd()
  }
}

function onCardDrop(status: string, targetId: string, event: DragEvent) {
  event.preventDefault()
  event.stopPropagation()
  const id = event.dataTransfer?.getData("text/issue-id") || dragIssueId.value
  if (!id || id === targetId) {
    onDragEnd()
    return
  }
  dragIssueId.value = id
  void applyBoardDrop(status, targetId)
}

function onColumnDrop(status: string, event: DragEvent) {
  event.preventDefault()
  const id = event.dataTransfer?.getData("text/issue-id") || dragIssueId.value
  if (!id) {
    onDragEnd()
    return
  }
  dragIssueId.value = id
  // 若松手前悬停在某卡片上，按卡片落点；否则落到泳道末尾
  const after = dropOverId.value && dropOverStatus.value === status ? dropOverId.value : undefined
  void applyBoardDrop(status, after)
}

function onCardClick(issueId: string) {
  if (dragMoved.value) {
    dragMoved.value = false
    return
  }
  openDetail(issueId)
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
  if (v === "board" || v === "list" || v === "tree" || v === "schedule") workMode.value = v
  syncPageHeader()
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
  <div class="page" :class="{ 'page--board': view === 'board' }">
    <aside class="side" aria-label="项目菜单">
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
          <svg v-else-if="item.icon === 'reports'" viewBox="0 0 16 16" width="16" height="16" aria-hidden="true">
            <path
              fill="currentColor"
              d="M2 13.5h12v1.2H2v-1.2zm1.2-1.8V7.2h1.6v4.5H3.2zm3.3 0V4.2h1.6v7.5H6.5zm3.3 0V5.8h1.6v5.9H9.8zm3.3 0V3h1.6v8.7h-1.6z"
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
      <div v-if="canManage" class="side-foot">
        <NpButton size="sm" variant="ghost" block @click="openEditProject">编辑项目</NpButton>
      </div>
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

      <p v-if="error" class="error">{{ error }}</p>
      <p v-if="loading" class="muted">加载中…</p>

      <template v-else>
      <div v-if="isWork" class="data-panel">
        <div class="data-head">
          <div class="panel-lead">
            <div v-if="project" class="panel-identity" :title="project.description || project.name">
              <NpBadge tone="accent">{{ project.key }}</NpBadge>
              <h2 class="panel-identity__title">{{ project.name }}</h2>
            </div>
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
              <button
                type="button"
                class="view-btn"
                :class="{ 'is-on': view === 'tree' }"
                role="tab"
                :aria-selected="view === 'tree'"
                title="树形"
                @click="view = 'tree'"
              >
                <span>树形</span>
              </button>
              <button
                type="button"
                class="view-btn"
                :class="{ 'is-on': view === 'schedule' }"
                role="tab"
                :aria-selected="view === 'schedule'"
                title="排期"
                @click="view = 'schedule'"
              >
                <span>排期</span>
              </button>
            </div>
          </div>
          <div class="panel-tools">
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
              <div class="filter-group">
                <span class="filter-label">状态</span>
                <NpSelect v-model="statusFilter" size="sm">
                  <option v-for="t in statusTabs" :key="t.value" :value="t.value">{{ t.label }}</option>
                </NpSelect>
              </div>
              <div class="filter-group">
                <span class="filter-label">到期</span>
                <NpSelect v-model="dueFilter" size="sm">
                  <option v-for="t in dueTabs" :key="t.value" :value="t.value">{{ t.label }}</option>
                </NpSelect>
              </div>
            </div>
            <NpButton size="sm" variant="primary" :disabled="!canWrite" @click="openCreate">
              新建事项
            </NpButton>
          </div>
        </div>

        <div
          v-if="view === 'board'"
          class="board"
          :style="{ '--board-cols': String(columns.length) }"
        >
          <section
            v-for="col in columns"
            :key="col.status"
            class="column"
            @dragover="onColumnDragOver(col.status, $event)"
            @drop="onColumnDrop(col.status, $event)"
          >
            <header>
              <strong>{{ col.label }}</strong>
              <span>{{ col.items.length }}</span>
            </header>
            <div class="column-cards">
            <article
              v-for="it in col.items"
              :key="it.id"
              class="card"
              :class="{
                'is-dragging': dragIssueId === it.id,
                'is-drop-target': dropOverId === it.id && dragIssueId !== it.id,
              }"
              draggable="true"
              @dragstart="onDragStart(it.id, $event)"
              @dragend="onDragEnd"
              @dragover="onCardDragOver(col.status, it.id, $event)"
              @drop="onCardDrop(col.status, it.id, $event)"
              @click="onCardClick(it.id)"
            >
              <div class="card-row card-row--main">
                <span class="key">{{ it.key }}</span>
                <h4 :title="it.title">{{ it.title }}</h4>
                <NpBadge :tone="typeTone(it.type)">{{ TYPE_LABELS[it.type] }}</NpBadge>
                <NpBadge :tone="priorityTone(it.priority)">{{ PRIORITY_LABELS[it.priority] }}</NpBadge>
                <span
                  v-if="it.dueDate"
                  class="due-pill"
                  :class="dueClass(it)"
                  :title="`截止 ${formatDueLabel(it.dueDate)}`"
                >{{ formatDueLabel(it.dueDate) }}</span>
                <span v-if="it.attachmentCount" class="att-pill" title="附件数">附 {{ it.attachmentCount }}</span>
                <UserAvatar
                  v-if="it.assigneeId"
                  :name="memberById(it.assigneeId)?.name || it.assigneeName"
                  :account="memberById(it.assigneeId)?.account"
                  :avatar-url="memberById(it.assigneeId)?.avatarUrl"
                  size="sm"
                />
              </div>
              <div
                v-if="it.childCount || it.updatedAt"
                class="card-row card-row--foot"
              >
                <div v-if="it.childCount" class="card-progress">
                  <div
                    class="card-progress__bar"
                    role="progressbar"
                    :aria-valuenow="childProgressPct(it.childDoneCount || 0, it.childCount)"
                    aria-valuemin="0"
                    aria-valuemax="100"
                    :title="`${it.childDoneCount || 0}/${it.childCount}`"
                  >
                    <span
                      class="card-progress__fill"
                      :style="{ width: `${childProgressPct(it.childDoneCount || 0, it.childCount)}%` }"
                    />
                  </div>
                  <span class="card-progress__label">{{ it.childDoneCount || 0 }}/{{ it.childCount }}</span>
                </div>
                <div
                  v-if="it.updatedAt"
                  class="card-updated"
                  :title="`最后更新 ${formatUpdatedFull(it.updatedAt)}`"
                >
                  更新 {{ formatUpdatedLabel(it.updatedAt) }}
                </div>
              </div>
            </article>
            </div>
          </section>
        </div>

        <div v-else-if="view === 'tree'" class="tree-wrap">
          <div class="tree-head" aria-hidden="true">
            <span class="tree-col tree-col--title">事项</span>
            <span class="tree-col tree-col--progress">子项进度</span>
            <span class="tree-col tree-col--due">截止</span>
            <span class="tree-col tree-col--status">状态</span>
            <span class="tree-col tree-col--assignee">指派</span>
          </div>
          <ul class="tree-list" role="tree">
            <li
              v-for="row in treeRows"
              :key="row.issue.id"
              class="tree-row"
              :class="[
                `is-status-${row.issue.status}`,
                {
                  'is-root': row.depth === 0,
                  'is-child': row.depth > 0,
                  'is-done': row.issue.status === 'done',
                },
              ]"
              role="treeitem"
              :aria-expanded="row.hasChildren ? row.expanded : undefined"
              :aria-level="row.depth + 1"
              @click="openDetail(row.issue.id)"
            >
              <div class="tree-col tree-col--title" :style="{ paddingLeft: `${10 + row.depth * 22}px` }">
                <button
                  v-if="row.hasChildren"
                  type="button"
                  class="tree-toggle"
                  :aria-label="row.expanded ? '折叠子事项' : '展开子事项'"
                  @click.stop="toggleTreeRoot(row.issue.id)"
                >
                  <svg viewBox="0 0 16 16" width="12" height="12" aria-hidden="true">
                    <path
                      fill="currentColor"
                      :d="row.expanded ? 'M3.5 6.2 8 10.7l4.5-4.5z' : 'M6.2 3.5 10.7 8 6.2 12.5z'"
                    />
                  </svg>
                </button>
                <span v-else class="tree-toggle-spacer" aria-hidden="true" />
                <span class="key">{{ row.issue.key }}</span>
                <NpBadge :tone="typeTone(row.issue.type)">{{ TYPE_LABELS[row.issue.type] }}</NpBadge>
                <span class="tree-title">{{ row.issue.title }}</span>
              </div>
              <div class="tree-col tree-col--progress">
                <template v-if="row.depth === 0 && row.childTotal">
                  <div
                    class="tree-bar"
                    :title="`${row.childDone}/${row.childTotal}`"
                    role="progressbar"
                    :aria-valuenow="childProgressPct(row.childDone, row.childTotal)"
                    aria-valuemin="0"
                    aria-valuemax="100"
                  >
                    <span
                      class="tree-bar__fill"
                      :style="{ width: `${childProgressPct(row.childDone, row.childTotal)}%` }"
                    />
                  </div>
                  <span class="tree-bar__label">{{ row.childDone }}/{{ row.childTotal }}</span>
                </template>
                <span v-else class="muted">—</span>
              </div>
              <div class="tree-col tree-col--due">
                <span
                  v-if="row.issue.dueDate"
                  class="due-pill"
                  :class="dueClass(row.issue)"
                >{{ formatDueLabel(row.issue.dueDate) }}</span>
                <span v-else class="muted">—</span>
              </div>
              <div
                class="tree-col tree-col--status"
                :class="`status-select--${row.issue.status}`"
                @click.stop
              >
                <NpSelect
                  :model-value="row.issue.status"
                  size="sm"
                  :disabled="!canWrite"
                  @update:model-value="moveIssue(row.issue.id, $event)"
                >
                  <option v-for="s in STATUS_ORDER" :key="s" :value="s">{{ STATUS_LABELS[s] }}</option>
                </NpSelect>
              </div>
              <div class="tree-col tree-col--assignee">
                <span class="tree-assignee">
                  <UserAvatar
                    v-if="row.issue.assigneeId"
                    :name="memberById(row.issue.assigneeId)?.name || row.issue.assigneeName"
                    :account="memberById(row.issue.assigneeId)?.account"
                    :avatar-url="memberById(row.issue.assigneeId)?.avatarUrl"
                    size="sm"
                  />
                  <span class="tree-assignee-name">{{ row.issue.assigneeName || "—" }}</span>
                </span>
              </div>
            </li>
          </ul>
          <p v-if="!treeRows.length" class="muted empty">暂无事项</p>
        </div>

        <div v-else-if="view === 'schedule'" class="schedule-wrap">
          <div class="schedule-toolbar">
            <div class="schedule-nav">
              <NpButton size="sm" variant="ghost" @click="shiftSchedule(scheduleSpan === 'week' ? -14 : -30)">
                上一{{ scheduleSpan === "week" ? "段" : "月" }}
              </NpButton>
              <NpButton size="sm" variant="ghost" @click="resetScheduleToday">今天</NpButton>
              <NpButton size="sm" variant="ghost" @click="shiftSchedule(scheduleSpan === 'week' ? 14 : 30)">
                下一{{ scheduleSpan === "week" ? "段" : "月" }}
              </NpButton>
              <span class="schedule-range muted">
                {{ scheduleRangeStart }} ~ {{ scheduleDays[scheduleDays.length - 1] }}
              </span>
            </div>
            <div class="view-switch" role="tablist" aria-label="排期跨度">
              <button
                type="button"
                class="view-btn"
                :class="{ 'is-on': scheduleSpan === 'week' }"
                @click="scheduleSpan = 'week'"
              >
                两周
              </button>
              <button
                type="button"
                class="view-btn"
                :class="{ 'is-on': scheduleSpan === 'month' }"
                @click="scheduleSpan = 'month'"
              >
                月
              </button>
            </div>
          </div>
          <div class="schedule-scroll">
            <div
              class="schedule-grid"
              :style="{ '--sched-days': String(scheduleDayCount) }"
            >
              <div class="schedule-corner">事项</div>
              <div
                v-for="d in scheduleDays"
                :key="d"
                class="schedule-dayhead"
                :class="{ 'is-today': d === todayISO() }"
              >
                <span class="schedule-dayhead__md">{{ d.slice(5) }}</span>
              </div>
              <template v-for="row in scheduleRows" :key="row.issue.id">
                <button
                  type="button"
                  class="schedule-label"
                  :class="{ 'is-child': row.depth > 0 }"
                  :style="{ paddingLeft: `${10 + row.depth * 14}px` }"
                  @click="openDetail(row.issue.id)"
                >
                  <span class="key">{{ row.issue.key }}</span>
                  <span class="schedule-label__title">{{ row.issue.title }}</span>
                  <span
                    v-if="row.issue.dueDate"
                    class="due-pill"
                    :class="dueClass(row.issue)"
                  >{{ formatDueLabel(row.issue.dueDate) }}</span>
                </button>
                <div class="schedule-track" @click="openDetail(row.issue.id)">
                  <span
                    v-for="d in scheduleDays"
                    :key="`${row.issue.id}-${d}`"
                    class="schedule-cell"
                    :class="{ 'is-today': d === todayISO() }"
                  />
                  <span
                    v-if="row.hasBar"
                    class="schedule-bar"
                    :class="{
                      'is-overdue': row.overdue,
                      'is-done': row.issue.status === 'done',
                      [`is-status-${row.issue.status}`]: true,
                    }"
                    :style="{
                      left: `${(row.barStart / scheduleDayCount) * 100}%`,
                      width: `${(row.barSpan / scheduleDayCount) * 100}%`,
                    }"
                    :title="`${row.issue.startDate || row.issue.dueDate || ''} → ${row.issue.dueDate || ''}`"
                  />
                </div>
              </template>
            </div>
            <p v-if="!scheduleRows.length" class="muted empty">暂无事项</p>
          </div>
        </div>

        <div v-else class="table-wrap">
          <table class="table">
            <thead>
              <tr>
                <th>编号</th>
                <th>类型</th>
                <th>标题</th>
                <th>截止</th>
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
                  <span
                    v-if="it.dueDate"
                    class="due-pill"
                    :class="dueClass(it)"
                  >{{ formatDueLabel(it.dueDate) }}</span>
                  <span v-else class="muted">—</span>
                </td>
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
                <td class="status-cell" :class="`status-select--${it.status}`" @click.stop>
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
              <td v-if="canWrite" @click.stop>
                <div class="actions-cell">
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
                </div>
              </td>
            </tr>
          </tbody>
        </table>
        <p v-if="!cycles.length" class="muted empty">还没有迭代。创建「本周迭代」后，把事项收进去推进。</p>
      </div>

      <div v-else-if="view === 'reports'" class="reports">
        <div class="toolbar">
          <div class="title-row" />
          <div class="toolbar-actions">
            <NpButton
              size="sm"
              variant="secondary"
              :disabled="!reportSummary.total"
              @click="exportReport"
            >
              导出 Excel
            </NpButton>
          </div>
        </div>
        <div class="report-cards">
          <div class="report-card">
            <span class="report-card__label">全部事项</span>
            <strong class="report-card__value">{{ reportSummary.total }}</strong>
            <span class="report-card__hint">史诗 {{ reportSummary.roots }} · 子事项 {{ reportSummary.children }}</span>
          </div>
          <div class="report-card">
            <span class="report-card__label">整体完成率</span>
            <strong class="report-card__value">{{ reportSummary.rate }}%</strong>
            <span class="report-card__hint">已完成 {{ reportSummary.done }} / {{ reportSummary.total }}</span>
          </div>
          <div class="report-card">
            <span class="report-card__label">子事项完成</span>
            <strong class="report-card__value">{{ reportSummary.childRate }}%</strong>
            <span class="report-card__hint">{{ reportSummary.childDone }} / {{ reportSummary.children }}</span>
          </div>
          <div class="report-card">
            <span class="report-card__label">进行中 / 待验收</span>
            <strong class="report-card__value">{{ reportSummary.doing }} / {{ reportSummary.review }}</strong>
            <span class="report-card__hint">待处理 {{ reportSummary.todo }} · 史诗完成 {{ reportSummary.rootDone }}</span>
          </div>
        </div>

        <div class="report-panel">
          <h3 class="report-title">状态分布</h3>
          <div class="status-bars">
            <div v-for="row in reportByStatus" :key="row.status" class="status-bar-row">
              <span class="status-bar-label">{{ row.label }}</span>
              <div class="status-bar-track" aria-hidden="true">
                <div
                  class="status-bar-fill"
                  :class="`is-${row.status}`"
                  :style="{ width: `${pct(row.count, reportSummary.total)}%` }"
                />
              </div>
              <span class="status-bar-count">{{ row.count }} · {{ pct(row.count, reportSummary.total) }}%</span>
            </div>
          </div>
        </div>

        <div class="report-grid">
          <div class="report-panel">
            <h3 class="report-title">按类型</h3>
            <div class="table-wrap">
              <table class="table">
                <thead>
                  <tr>
                    <th>类型</th>
                    <th>总数</th>
                    <th>已完成</th>
                    <th>完成率</th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-for="row in reportByType" :key="row.key">
                    <td>{{ row.label }}</td>
                    <td>{{ row.total }}</td>
                    <td>{{ row.done }}</td>
                    <td>{{ pct(row.done, row.total) }}%</td>
                  </tr>
                </tbody>
              </table>
              <p v-if="!reportByType.length" class="muted empty">暂无事项</p>
            </div>
          </div>

          <div class="report-panel">
            <h3 class="report-title">按迭代</h3>
            <div class="table-wrap">
              <table class="table">
                <thead>
                  <tr>
                    <th>迭代</th>
                    <th>总数</th>
                    <th>已完成</th>
                    <th>完成率</th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-for="row in reportByCycle" :key="row.key || 'backlog'">
                    <td>{{ row.label }}</td>
                    <td>{{ row.total }}</td>
                    <td>{{ row.done }}</td>
                    <td>{{ pct(row.done, row.total) }}%</td>
                  </tr>
                </tbody>
              </table>
              <p v-if="!reportByCycle.length" class="muted empty">暂无事项</p>
            </div>
          </div>
        </div>

        <div class="report-panel">
          <h3 class="report-title">按模块</h3>
          <div class="table-wrap">
            <table class="table">
              <thead>
                <tr>
                  <th>模块</th>
                  <th>总数</th>
                  <th>待处理</th>
                  <th>进行中</th>
                  <th>待验收</th>
                  <th>已完成</th>
                  <th>完成率</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="row in reportByModule" :key="row.key || 'none'">
                  <td>{{ row.label }}</td>
                  <td>{{ row.total }}</td>
                  <td>{{ row.todo }}</td>
                  <td>{{ row.doing }}</td>
                  <td>{{ row.review }}</td>
                  <td>{{ row.done }}</td>
                  <td>{{ pct(row.done, row.total) }}%</td>
                </tr>
              </tbody>
            </table>
            <p v-if="!reportByModule.length" class="muted empty">暂无事项</p>
          </div>
        </div>
      </div>

      <div v-else-if="view === 'files'" class="attachments-view">
        <ProjectAttachmentsPanel
          v-if="project"
          ref="attachmentsPanel"
          :project-id="project.id"
          :can-write="canWrite"
          :active="view === 'files'"
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
                  <UserAvatar :name="u.name" :account="u.account" :avatar-url="u.avatarUrl" size="sm" />
                  <span class="invite-option__text">
                    <strong>{{ u.name }}</strong>
                    <span>{{ u.account }}</span>
                  </span>
                </button>
                <div
                  v-for="u in inviteAlreadyMembers"
                  :key="`m-${u.id}`"
                  class="invite-option invite-option--disabled"
                >
                  <UserAvatar :name="u.name" :account="u.account" :avatar-url="u.avatarUrl" size="sm" />
                  <span class="invite-option__text">
                    <strong>{{ u.name }}</strong>
                    <span>{{ u.account }} · 已是成员</span>
                  </span>
                </div>
                <button
                  v-if="canInviteNewAccount"
                  type="button"
                  class="invite-option invite-option--new"
                  @mousedown.prevent="
                    selectedUserId = '';
                    inviteOpen = false;
                    void nextTick(() => addMember())
                  "
                >
                  <strong>邀请新账号</strong>
                  <span>{{ inviteQuery.trim() }}</span>
                </button>
                <p
                  v-if="!inviteCandidates.length && !inviteAlreadyMembers.length && !canInviteNewAccount"
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
                <th>用户名</th>
                <th>账号</th>
                <th>角色</th>
                <th>操作</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="m in members" :key="m.userId">
                <td>
                  <div class="member-name">
                    <UserAvatar :name="m.name" :account="m.account" :avatar-url="m.avatarUrl" size="sm" />
                    <span>{{ m.name }}</span>
                  </div>
                </td>
                <td>{{ m.account }}</td>
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

    <NpModal :open="editOpen" title="编辑项目" @close="editOpen = false">
      <div class="form">
        <NpField label="项目标识">
          <NpInput :model-value="project?.key || ''" disabled />
        </NpField>
        <NpField label="项目名称">
          <NpInput v-model="editForm.name" placeholder="项目名称" />
        </NpField>
        <NpField label="描述">
          <NpTextarea v-model="editForm.description" placeholder="项目目标与范围" :rows="3" />
        </NpField>
      </div>
      <template #footer>
        <NpButton variant="ghost" @click="editOpen = false">取消</NpButton>
        <NpButton variant="primary" :disabled="editSaving" @click="saveProject">
          {{ editSaving ? "保存中…" : "保存" }}
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
      :default-assignee-id="auth.user?.id || ''"
      :default-cycle-id="activeCycle?.id || ''"
      :default-module-id="moduleFilter !== 'all' ? moduleFilter : ''"
      @close="detailOpen = false"
      @updated="onIssueUpdated"
      @created="onIssueCreated"
      @open-issue="openDetail"
      @update:modules="onModulesUpdated"
    />
  </div>
</template>

<style scoped>
.page {
  display: grid;
  grid-template-columns: 168px minmax(0, 1fr);
  gap: 0;
  align-items: stretch;
  /* 抵消壳层 content 内边距，左右贴边 */
  margin: -16px -20px;
  min-height: calc(100vh - 44px);
  width: calc(100% + 40px);
}

.page--board {
  height: calc(100vh - 44px);
  max-height: calc(100vh - 44px);
  overflow: hidden;
  grid-template-rows: minmax(0, 1fr);
}

.page--board .main {
  min-height: 0;
  height: 100%;
  overflow: hidden;
}

.page--board .data-panel {
  flex: 1 1 auto;
  min-height: 0;
  overflow: hidden;
}

.page--board .data-head {
  flex-shrink: 0;
}

.side {
  display: flex;
  flex-direction: column;
  gap: 10px;
  align-self: stretch;
  position: sticky;
  top: 0;
  min-height: calc(100vh - 44px);
  padding: 14px 10px;
  border: none;
  border-right: 1px solid var(--np-border);
  border-radius: 0;
  background:
    linear-gradient(180deg, color-mix(in srgb, #fff 88%, var(--np-bg)) 0%, var(--np-bg-elevated) 100%);
  box-shadow: none;
}

.side-nav {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.side-foot {
  margin-top: 8px;
  padding-top: 10px;
  border-top: 1px solid var(--np-border);
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
  gap: 8px;
  min-width: 0;
  min-height: 0;
  padding: 8px 12px 12px;
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
  gap: 8px;
  min-width: 0;
  padding: 0;
  border: none;
  border-radius: 0;
  background: transparent;
  box-shadow: none;
}

.data-head {
  display: flex;
  flex-wrap: nowrap;
  justify-content: space-between;
  align-items: center;
  gap: 10px 14px;
  min-width: 0;
}

.panel-lead {
  display: flex;
  flex-wrap: nowrap;
  align-items: center;
  gap: 10px;
  min-width: 0;
  flex: 1 1 auto;
}

.panel-identity {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
  max-width: min(360px, 42vw);
}

.panel-identity__title {
  margin: 0;
  min-width: 0;
  font-family: var(--np-font-display);
  font-size: var(--np-text-md);
  font-weight: 700;
  letter-spacing: -0.03em;
  line-height: 1.25;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.panel-tools {
  display: flex;
  flex-wrap: nowrap;
  align-items: center;
  justify-content: flex-end;
  gap: 6px 10px;
  min-width: 0;
  flex: 0 1 auto;
}

.panel-tools .filter-groups {
  flex-wrap: nowrap;
  overflow-x: auto;
  max-width: min(100%, 52vw);
  padding-bottom: 1px;
}

.view-switch {
  display: inline-flex;
  gap: 2px;
  padding: 0;
  border-radius: 0;
  background: transparent;
  border: none;
}

.view-btn {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  height: 26px;
  padding: 0 8px;
  border: none;
  border-radius: 4px;
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
  background: color-mix(in srgb, var(--np-accent-muted) 45%, transparent);
}

.view-btn.is-on {
  background: var(--np-accent-muted);
  color: var(--np-accent-ink);
  box-shadow: none;
}

@media (max-width: 860px) {
  .page {
    grid-template-columns: 1fr;
    margin: 0;
    width: 100%;
    min-height: auto;
  }

  .side {
    position: static;
    min-height: auto;
    padding: 10px;
    border: none;
    border-bottom: 1px solid var(--np-border);
    border-radius: 0;
  }

  .main {
    padding: 8px 0 0;
  }

  .board {
    margin: 0;
    padding: 0;
  }

  .page--board {
    height: auto;
    max-height: none;
    overflow: visible;
  }

  .column {
    min-height: 220px;
    height: auto;
    max-height: 52vh;
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

.filter-groups {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px 12px;
  min-width: 0;
}

.filter-group {
  display: flex;
  flex-wrap: nowrap;
  align-items: center;
  gap: 6px;
}

.filter-group :deep(.np-select) {
  width: auto;
  min-width: 7.5rem;
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
  grid-template-columns: repeat(var(--board-cols, 4), minmax(180px, 1fr));
  gap: 10px;
  align-items: stretch;
  flex: 1;
  min-width: 0;
  min-height: 0;
  margin: 0 -8px;
  padding: 8px 4px 4px;
  border-top: 1px solid var(--np-border);
  overflow: hidden;
}

.column {
  display: flex;
  flex-direction: column;
  min-height: 0;
  height: 100%;
  padding: 10px 8px 0;
  border-radius: 10px;
  background: color-mix(in srgb, var(--np-bg-sunken) 78%, #fff);
  border: none;
  overflow: hidden;
}

.column header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  flex-shrink: 0;
  margin-bottom: 8px;
  padding: 0 2px 8px;
  border-bottom: none;
  color: var(--np-text);
  font-family: var(--np-font-display);
  font-size: var(--np-text-sm);
  font-weight: 700;
  letter-spacing: -0.02em;
}

.column-cards {
  flex: 1;
  min-height: 0;
  overflow-x: hidden;
  overflow-y: auto;
  padding: 0 0 10px;
  overscroll-behavior: contain;
}

.column header span {
  min-width: 18px;
  height: 18px;
  padding: 0 5px;
  border-radius: 4px;
  background: color-mix(in srgb, var(--np-bg-sunken) 70%, #fff);
  color: var(--np-text-muted);
  font-family: var(--np-font-mono);
  font-size: 11px;
  font-variant-numeric: tabular-nums;
  line-height: 18px;
  text-align: center;
  border: none;
}

.card {
  display: flex;
  flex-direction: column;
  gap: 6px;
  min-width: 0;
  padding: 8px 9px;
  margin-bottom: 6px;
  border-radius: 6px;
  background: color-mix(in srgb, #fff 88%, var(--np-bg));
  border: 1px solid color-mix(in srgb, var(--np-border) 80%, transparent);
  box-shadow: none;
  cursor: grab;
  transition:
    border-color var(--np-duration-fast) var(--np-ease),
    background var(--np-duration-fast) var(--np-ease);
}

.card:hover {
  border-color: color-mix(in srgb, var(--np-accent) 45%, var(--np-border));
  background: #fff;
}

.card.is-dragging {
  opacity: 0.45;
}

.card.is-drop-target {
  box-shadow: inset 0 2px 0 0 var(--np-accent);
  border-color: color-mix(in srgb, var(--np-accent) 55%, var(--np-border));
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

.card-row--main {
  flex-wrap: nowrap;
}

.card-row--foot {
  justify-content: space-between;
  gap: 8px;
}

.card-row--foot .card-progress {
  flex: 1;
  min-width: 0;
  margin-top: 0;
}

.card-row--foot .card-updated {
  flex-shrink: 0;
  margin-left: auto;
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

.att-pill,
.due-pill {
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
  font-family: var(--np-font-mono);
}

.card-updated {
  color: var(--np-text-muted);
  font-size: 10px;
  font-variant-numeric: tabular-nums;
  font-family: var(--np-font-mono);
  line-height: 1.3;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.card-progress {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-top: 2px;
}

.card-progress__bar {
  flex: 1;
  min-width: 0;
  height: 4px;
  border-radius: 999px;
  background: color-mix(in srgb, var(--np-bg-sunken) 70%, #fff);
  overflow: hidden;
  border: 1px solid var(--np-border);
}

.card-progress__fill {
  display: block;
  height: 100%;
  border-radius: inherit;
  background: linear-gradient(90deg, var(--np-accent), color-mix(in srgb, var(--np-accent) 70%, #3d9a6a));
  transition: width var(--np-duration-fast) var(--np-ease);
}

.card-progress__label {
  flex-shrink: 0;
  font-size: 10px;
  font-weight: 700;
  font-variant-numeric: tabular-nums;
  font-family: var(--np-font-mono);
  color: var(--np-text-muted);
  line-height: 1;
}

.att-pill {
  background: color-mix(in srgb, var(--np-accent-muted) 80%, #fff);
  color: var(--np-accent-ink);
}

.due-pill {
  background: color-mix(in srgb, var(--np-bg-sunken) 65%, #fff);
  color: var(--np-text-muted);
  border: 1px solid var(--np-border);
}

.due-pill.is-due-soon {
  background: color-mix(in srgb, var(--np-warning-muted) 80%, #fff);
  color: color-mix(in srgb, var(--np-warning) 80%, #000);
  border-color: color-mix(in srgb, var(--np-warning) 40%, var(--np-border));
}

.due-pill.is-overdue {
  background: color-mix(in srgb, var(--np-danger-muted) 80%, #fff);
  color: var(--np-danger);
  border-color: color-mix(in srgb, var(--np-danger) 40%, var(--np-border));
}

.tree-wrap {
  display: flex;
  flex-direction: column;
  gap: 0;
  min-width: 0;
  overflow-x: auto;
  border: none;
  border-top: 1px solid var(--np-border);
  border-radius: 0;
  background: transparent;
  padding: 8px 0 4px;
}

.tree-head,
.tree-row {
  display: grid;
  grid-template-columns: minmax(200px, 1.5fr) minmax(100px, 0.65fr) 5.5rem 7.5rem minmax(80px, 0.5fr);
  gap: 8px;
  align-items: center;
  min-width: 560px;
}

.tree-head {
  padding: 0 8px 8px;
  border-bottom: 1px solid var(--np-border);
  color: var(--np-text-muted);
  font-size: var(--np-text-xs);
  font-weight: 600;
}

.tree-list {
  list-style: none;
  margin: 0;
  padding: 0;
}

.tree-row {
  padding: 8px;
  border-bottom: 1px solid var(--np-border);
  box-shadow: inset 3px 0 0 transparent;
  cursor: pointer;
  transition:
    background var(--np-duration-fast) var(--np-ease),
    box-shadow var(--np-duration-fast) var(--np-ease);
}

.tree-row:hover {
  background: color-mix(in srgb, var(--np-accent-muted) 45%, transparent);
}

.tree-row.is-root {
  font-weight: 600;
}

.tree-row.is-child {
  background: color-mix(in srgb, var(--np-bg-sunken) 35%, transparent);
}

.tree-row.is-child:hover {
  background: color-mix(in srgb, var(--np-accent-muted) 40%, transparent);
}

.tree-row.is-done .tree-title {
  color: var(--np-text-muted);
  text-decoration: line-through;
  text-decoration-thickness: 1px;
}

.tree-col {
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 6px;
}

.tree-col--title {
  overflow: hidden;
}

.tree-col--progress {
  gap: 8px;
}

.tree-col--due {
  justify-content: flex-start;
}

.tree-col--status :deep(.np-select) {
  width: 100%;
  min-width: 0;
  font-weight: 600;
}

.tree-row.is-status-todo {
  box-shadow: inset 3px 0 0 color-mix(in srgb, var(--np-text-muted) 55%, var(--np-border));
}

.tree-row.is-status-doing {
  box-shadow: inset 3px 0 0 var(--np-accent);
}

.tree-row.is-status-review {
  box-shadow: inset 3px 0 0 var(--np-warning);
}

.tree-row.is-status-done {
  box-shadow: inset 3px 0 0 var(--np-success);
}

.status-select--todo :deep(.np-select) {
  color: var(--np-text-muted);
  border-color: color-mix(in srgb, var(--np-text-muted) 35%, var(--np-border));
  background-color: color-mix(in srgb, var(--np-bg-sunken) 70%, #fff);
}

.status-select--doing :deep(.np-select) {
  color: var(--np-accent-ink);
  border-color: color-mix(in srgb, var(--np-accent) 45%, var(--np-border));
  background-color: color-mix(in srgb, var(--np-accent-muted) 75%, #fff);
}

.status-select--review :deep(.np-select) {
  color: color-mix(in srgb, var(--np-warning) 75%, #000);
  border-color: color-mix(in srgb, var(--np-warning) 45%, var(--np-border));
  background-color: color-mix(in srgb, var(--np-warning-muted) 75%, #fff);
}

.status-select--done :deep(.np-select) {
  color: color-mix(in srgb, var(--np-success) 70%, #000);
  border-color: color-mix(in srgb, var(--np-success) 45%, var(--np-border));
  background-color: color-mix(in srgb, var(--np-success-muted) 75%, #fff);
}

.status-cell :deep(.np-select) {
  font-weight: 600;
  min-width: 6.5rem;
}

.tree-toggle,
.tree-toggle-spacer {
  flex-shrink: 0;
  width: 20px;
  height: 20px;
}

.tree-toggle {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 0;
  border: 1px solid var(--np-border);
  border-radius: 4px;
  background: var(--np-bg-elevated);
  color: var(--np-text-muted);
  cursor: pointer;
}

.tree-toggle:hover {
  color: var(--np-accent);
  border-color: color-mix(in srgb, var(--np-accent) 45%, var(--np-border));
}

.tree-title {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: var(--np-text-sm);
  font-weight: inherit;
}

.tree-bar {
  flex: 1;
  min-width: 48px;
  height: 6px;
  border-radius: 999px;
  background: color-mix(in srgb, var(--np-bg-sunken) 70%, #fff);
  overflow: hidden;
  border: 1px solid var(--np-border);
}

.tree-bar__fill {
  display: block;
  height: 100%;
  border-radius: inherit;
  background: linear-gradient(90deg, var(--np-accent), color-mix(in srgb, var(--np-accent) 70%, #3d9a6a));
  transition: width var(--np-duration-fast) var(--np-ease);
}

.tree-bar__label {
  flex-shrink: 0;
  font-family: var(--np-font-mono);
  font-size: 11px;
  font-variant-numeric: tabular-nums;
  color: var(--np-accent-ink);
}

.tree-assignee {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  min-width: 0;
}

.tree-assignee-name {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: var(--np-text-xs);
  font-weight: 500;
  color: var(--np-text-muted);
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

.reports {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.report-cards {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 12px;
}

.report-card {
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 14px 16px;
  border: 1px solid var(--np-border);
  border-radius: var(--np-radius-lg);
  background: var(--np-bg-elevated);
  box-shadow: var(--np-shadow-sm);
}

.report-card__label {
  color: var(--np-text-muted);
  font-size: var(--np-text-xs);
  font-weight: 600;
}

.report-card__value {
  font-size: 1.6rem;
  font-weight: 700;
  letter-spacing: -0.02em;
  color: var(--np-text);
}

.report-card__hint {
  color: var(--np-text-secondary);
  font-size: var(--np-text-xs);
}

.report-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
}

.report-panel {
  padding: 14px 16px 16px;
  border: 1px solid var(--np-border);
  border-radius: var(--np-radius-lg);
  background: var(--np-bg-elevated);
  box-shadow: var(--np-shadow-sm);
}

.report-title {
  margin: 0 0 12px;
  font-size: var(--np-text-sm);
  font-weight: 700;
  color: var(--np-text);
}

.status-bars {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.status-bar-row {
  display: grid;
  grid-template-columns: 4.5rem minmax(0, 1fr) 5.5rem;
  gap: 10px;
  align-items: center;
}

.status-bar-label {
  font-size: var(--np-text-sm);
  color: var(--np-text-secondary);
}

.status-bar-track {
  height: 8px;
  border-radius: 999px;
  background: color-mix(in srgb, var(--np-border) 70%, #fff);
  overflow: hidden;
}

.status-bar-fill {
  height: 100%;
  border-radius: inherit;
  background: var(--np-accent);
  min-width: 0;
  transition: width var(--np-duration-fast) var(--np-ease);
}

.status-bar-fill.is-todo {
  background: color-mix(in srgb, var(--np-text-muted) 55%, #fff);
}
.status-bar-fill.is-doing {
  background: var(--np-accent);
}
.status-bar-fill.is-review {
  background: var(--np-warning);
}
.status-bar-fill.is-done {
  background: var(--np-success);
}

.status-bar-count {
  font-size: var(--np-text-xs);
  color: var(--np-text-muted);
  text-align: right;
  white-space: nowrap;
}

@media (max-width: 960px) {
  .report-cards,
  .report-grid {
    grid-template-columns: 1fr 1fr;
  }
}

@media (max-width: 640px) {
  .report-cards,
  .report-grid {
    grid-template-columns: 1fr;
  }

  .status-bar-row {
    grid-template-columns: 3.5rem minmax(0, 1fr) 4.8rem;
  }
}

.table-wrap {
  overflow: auto;
  border: none;
  border-top: 1px solid var(--np-border);
  border-radius: 0;
  box-shadow: none;
  background: transparent;
}

.data-panel > .table-wrap,
.data-panel > .tree-wrap,
.data-panel > .schedule-wrap {
  margin-top: 0;
}

.schedule-wrap {
  border: none;
  border-top: 1px solid var(--np-border);
  border-radius: 0;
  background: transparent;
  padding: 8px 0 0;
  display: flex;
  flex-direction: column;
  gap: 10px;
  min-width: 0;
}

.schedule-toolbar {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
}

.schedule-nav {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 6px;
}

.schedule-range {
  font-family: var(--np-font-mono);
  font-size: var(--np-text-xs);
}

.schedule-scroll {
  overflow: auto;
  max-height: min(70vh, 720px);
}

.schedule-grid {
  display: grid;
  grid-template-columns: minmax(200px, 240px) repeat(var(--sched-days, 14), minmax(28px, 1fr));
  min-width: max(100%, calc(200px + var(--sched-days, 14) * 28px));
  align-items: stretch;
}

.schedule-corner,
.schedule-dayhead {
  position: sticky;
  top: 0;
  z-index: 2;
  background: color-mix(in srgb, var(--np-bg-sunken) 55%, #fff);
  border-bottom: 1px solid var(--np-border);
  font-size: 10px;
  font-weight: 600;
  color: var(--np-text-muted);
  padding: 6px 4px;
}

.schedule-corner {
  left: 0;
  z-index: 3;
}

.schedule-dayhead {
  text-align: center;
  font-family: var(--np-font-mono);
}

.schedule-dayhead.is-today,
.schedule-cell.is-today {
  background: color-mix(in srgb, var(--np-accent-muted) 55%, transparent);
}

.schedule-label {
  grid-column: 1;
  position: sticky;
  left: 0;
  z-index: 1;
  display: flex;
  align-items: center;
  gap: 6px;
  min-width: 0;
  height: 36px;
  padding: 0 8px;
  border: none;
  border-bottom: 1px solid var(--np-border);
  border-right: 1px solid var(--np-border);
  background: #fff;
  text-align: left;
  cursor: pointer;
  font: inherit;
}

.schedule-label:hover {
  background: color-mix(in srgb, var(--np-accent-muted) 40%, #fff);
}

.schedule-label.is-child {
  background: color-mix(in srgb, var(--np-bg-sunken) 40%, #fff);
  font-weight: 500;
}

.schedule-label__title {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: var(--np-text-xs);
}

.schedule-track {
  grid-column: 2 / -1;
  position: relative;
  display: grid;
  grid-template-columns: repeat(var(--sched-days, 14), minmax(0, 1fr));
  height: 36px;
  border-bottom: 1px solid var(--np-border);
  cursor: pointer;
  min-width: 0;
}

.schedule-cell {
  border-right: 1px solid color-mix(in srgb, var(--np-border) 70%, transparent);
}

.schedule-bar {
  position: absolute;
  top: 10px;
  height: 16px;
  border-radius: 4px;
  background: var(--np-accent);
  box-shadow: var(--np-shadow-sm);
  pointer-events: none;
  min-width: 8px;
}

.schedule-bar.is-status-todo {
  background: color-mix(in srgb, var(--np-text-muted) 55%, #fff);
}

.schedule-bar.is-status-doing {
  background: var(--np-accent);
}

.schedule-bar.is-status-review {
  background: var(--np-warning);
}

.schedule-bar.is-status-done {
  background: var(--np-success);
  opacity: 0.75;
}

.schedule-bar.is-overdue:not(.is-done) {
  background: var(--np-danger);
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
