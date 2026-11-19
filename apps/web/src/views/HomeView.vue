<script setup lang="ts">
import { computed, onMounted, ref, watch } from "vue"
import { storeToRefs } from "pinia"
import { useRoute, useRouter } from "vue-router"
import { NpBadge, NpButton, NpPagination, NpSelect } from "@ningpm/ui"
import { PRIORITY_LABELS, STATUS_LABELS, TYPE_LABELS, priorityTone, typeTone } from "../lib/labels"
import { formatDueLabel, isOverdue } from "../lib/schedule"
import type { Issue, UserOption } from "../types"
import { useAuthStore } from "../stores/auth"
import { useProjectsStore } from "../stores/projects"
import { useWorkbenchStore } from "../stores/workbench"
import IssueDetailDrawer from "../components/IssueDetailDrawer.vue"

const auth = useAuthStore()
const projectsStore = useProjectsStore()
const workbench = useWorkbenchStore()
const { items: projects } = storeToRefs(projectsStore)
const {
  issues,
  loading,
  error,
  contextMembers,
  contextCycles,
  contextModules,
} = storeToRefs(workbench)

const route = useRoute()
const router = useRouter()

const detailOpen = ref(false)
const activeIssueId = ref<string | null>(null)

const projectFilter = ref("all")
const typeFilter = ref("all")
const statusFilter = ref("open")
const dueFilter = ref("all") // all | overdue | week | none
const scopeFilter = ref<"all" | "mine">("all")
const page = ref(1)
const pageSize = ref(20)

const memberUsers = computed<UserOption[]>(() =>
  contextMembers.value.map((m) => ({ id: m.userId, name: m.name, account: m.account, email: m.email })),
)

const visible = computed(() => {
  let list = issues.value
  if (statusFilter.value === "open") {
    list = list.filter((i) => i.status !== "done")
  }
  return list
})

const paged = computed(() => {
  const start = (page.value - 1) * pageSize.value
  return visible.value.slice(start, start + pageSize.value)
})

async function load() {
  await Promise.all([
    projectsStore.fetchAll().catch(() => {}),
    workbench
      .fetch({
        projectId: projectFilter.value,
        type: typeFilter.value,
        status: statusFilter.value,
        due: dueFilter.value,
        mine: scopeFilter.value === "mine",
      })
      .catch(() => {}),
  ])
}

async function openDetail(it: Issue) {
  activeIssueId.value = it.id
  detailOpen.value = true
  await workbench.loadIssueContext(it.projectId)
}

function onIssueUpdated(updated: Issue) {
  workbench.upsertIssue(updated)
}

function goProject(it: Issue) {
  void router.push({ name: "project-detail", params: { id: it.projectId } })
}

watch([projectFilter, typeFilter, statusFilter, dueFilter, scopeFilter], () => {
  page.value = 1
  void load()
})

watch(visible, (list) => {
  const maxPage = Math.max(1, Math.ceil(list.length / pageSize.value) || 1)
  if (page.value > maxPage) page.value = maxPage
})

onMounted(() => {
  if (route.query.scope === "mine") scopeFilter.value = "mine"
  void load()
})
</script>

<template>
  <div class="home">
    <div class="top">
      <div class="greet">
        <p class="lead">你好，{{ auth.user?.name || "同学" }}</p>
      </div>
      <div class="quick">
        <NpButton size="sm" variant="secondary" @click="$router.push('/projects')">去项目</NpButton>
      </div>
    </div>

    <div class="filters">
      <div class="filter-groups">
        <div class="filter-group">
          <span class="filter-label">项目</span>
          <NpSelect v-model="projectFilter" size="sm">
            <option value="all">全部项目</option>
            <option v-for="p in projects" :key="p.id" :value="p.id">{{ p.key }} · {{ p.name }}</option>
          </NpSelect>
        </div>
        <div class="filter-group">
          <span class="filter-label">范围</span>
          <NpSelect v-model="scopeFilter" size="sm">
            <option value="all">我可见的</option>
            <option value="mine">指派给我</option>
          </NpSelect>
        </div>
        <div class="filter-group">
          <span class="filter-label">类型</span>
          <NpSelect v-model="typeFilter" size="sm">
            <option value="all">全部</option>
            <option value="story">需求</option>
            <option value="task">任务</option>
            <option value="bug">缺陷</option>
            <option value="epic">史诗</option>
          </NpSelect>
        </div>
        <div class="filter-group">
          <span class="filter-label">状态</span>
          <NpSelect v-model="statusFilter" size="sm">
            <option value="open">未完成</option>
            <option value="all">全部状态</option>
            <option value="todo">待处理</option>
            <option value="doing">进行中</option>
            <option value="review">待验收</option>
            <option value="done">已完成</option>
          </NpSelect>
        </div>
        <div class="filter-group">
          <span class="filter-label">到期</span>
          <NpSelect v-model="dueFilter" size="sm">
            <option value="all">全部到期</option>
            <option value="overdue">已逾期</option>
            <option value="week">本周到期</option>
            <option value="none">未排期</option>
          </NpSelect>
        </div>
      </div>
      <NpButton size="sm" variant="ghost" :disabled="loading" @click="load">刷新</NpButton>
    </div>

    <p v-if="error" class="error">{{ error }}</p>
    <p v-if="loading" class="muted">加载中…</p>

    <template v-else>
      <div class="data-panel">
        <div class="table-wrap">
          <table class="table">
            <thead>
              <tr>
                <th>编号</th>
                <th>项目</th>
                <th>类型</th>
                <th>标题</th>
                <th>子项</th>
                <th>附件</th>
                <th>截止</th>
                <th>优先级</th>
                <th>状态</th>
                <th>指派给</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="it in paged" :key="it.id" class="click-row" @click="openDetail(it)">
                <td class="key">{{ it.key }}</td>
                <td>
                  <button type="button" class="proj" :title="it.projectName" @click.stop="goProject(it)">
                    {{ it.projectKey }}
                  </button>
                </td>
                <td><NpBadge :tone="typeTone(it.type)">{{ TYPE_LABELS[it.type] }}</NpBadge></td>
                <td class="title">
                  {{ it.title }}
                  <span v-if="it.parentKey" class="parent">← {{ it.parentKey }}</span>
                </td>
                <td class="nowrap">
                  <span v-if="it.childCount">{{ it.childDoneCount || 0 }}/{{ it.childCount }}</span>
                  <span v-else class="muted">—</span>
                </td>
                <td class="nowrap">
                  <span v-if="it.attachmentCount">{{ it.attachmentCount }}</span>
                  <span v-else class="muted">—</span>
                </td>
                <td class="nowrap">
                  <span
                    v-if="it.dueDate"
                    class="due-pill"
                    :class="{ 'is-overdue': isOverdue(it.dueDate, it.status) }"
                  >{{ formatDueLabel(it.dueDate) }}</span>
                  <span v-else class="muted">—</span>
                </td>
                <td><NpBadge :tone="priorityTone(it.priority)">{{ PRIORITY_LABELS[it.priority] }}</NpBadge></td>
                <td><NpBadge>{{ STATUS_LABELS[it.status] }}</NpBadge></td>
                <td>{{ it.assigneeName || (it.assigneeId === auth.user?.id ? "我" : "—") }}</td>
              </tr>
            </tbody>
          </table>
          <div v-if="!visible.length" class="empty">
            <p class="empty-title">没有匹配的事项</p>
            <p class="muted">换个筛选，或去项目里新建。</p>
          </div>
        </div>
        <NpPagination
          v-if="visible.length"
          v-model:page="page"
          v-model:page-size="pageSize"
          :total="visible.length"
          :page-sizes="[10, 20, 50]"
        />
      </div>
    </template>

    <IssueDetailDrawer
      :open="detailOpen"
      :issue-id="activeIssueId"
      :project-id="issues.find((i) => i.id === activeIssueId)?.projectId || null"
      :users="memberUsers"
      :cycles="contextCycles"
      :modules="contextModules"
      :can-write="true"
      @close="detailOpen = false"
      @updated="onIssueUpdated"
      @open-issue="(id) => { activeIssueId = id; detailOpen = true }"
      @update:modules="contextModules = $event"
    />
  </div>
</template>

<style scoped>
.home {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.top {
  display: flex;
  flex-wrap: wrap;
  align-items: flex-end;
  justify-content: space-between;
  gap: 10px 16px;
}

.greet {
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-width: 0;
}

.lead {
  margin: 0;
  font-family: var(--np-font-display);
  font-size: var(--np-text-lg);
  font-weight: 700;
  letter-spacing: -0.03em;
  color: var(--np-text);
}

.quick {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.filters {
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

.table-wrap {
  overflow: auto;
  border: 1px solid var(--np-border);
  border-radius: var(--np-radius-md);
  background: #fff;
  box-shadow: none;
}

.table {
  width: 100%;
  border-collapse: collapse;
}

.table th,
.table td {
  padding: 11px 12px;
  border-bottom: 1px solid var(--np-border);
  text-align: left;
  font-size: var(--np-text-sm);
  vertical-align: middle;
}

.table th {
  color: var(--np-accent-ink);
  font-weight: 600;
  background: color-mix(in srgb, var(--np-accent-muted) 48%, #fff);
}

.table tbody tr:last-child td {
  border-bottom: none;
}

.title {
  font-weight: 600;
  color: var(--np-text);
}

.click-row {
  cursor: pointer;
  transition: background var(--np-duration-fast) var(--np-ease);
}

.click-row:hover td {
  background: color-mix(in srgb, var(--np-accent-muted) 35%, transparent);
}

.key {
  font-family: var(--np-font-mono);
  font-size: var(--np-text-xs);
  color: var(--np-text-muted);
}

.proj {
  border: none;
  background: transparent;
  color: var(--np-accent);
  font: inherit;
  font-weight: 600;
  font-family: var(--np-font-mono);
  font-size: var(--np-text-xs);
  cursor: pointer;
  padding: 0;
}

.parent {
  margin-left: 6px;
  color: var(--np-text-muted);
  font-size: var(--np-text-xs);
  font-weight: 400;
}

.nowrap {
  white-space: nowrap;
  font-variant-numeric: tabular-nums;
}

.due-pill {
  display: inline-block;
  padding: 0 5px;
  height: 18px;
  border-radius: 3px;
  background: color-mix(in srgb, var(--np-bg-sunken) 65%, #fff);
  color: var(--np-text-muted);
  border: 1px solid var(--np-border);
  font-size: 11px;
  font-weight: 700;
  line-height: 18px;
  font-family: var(--np-font-mono);
  font-variant-numeric: tabular-nums;
}

.due-pill.is-overdue {
  background: color-mix(in srgb, var(--np-danger-muted) 80%, #fff);
  color: var(--np-danger);
  border-color: color-mix(in srgb, var(--np-danger) 40%, var(--np-border));
}

.muted {
  margin: 0;
  color: var(--np-text-muted);
  font-size: var(--np-text-sm);
}

.empty {
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 28px 16px;
}

.empty-title {
  margin: 0;
  font-family: var(--np-font-display);
  font-size: var(--np-text-lg);
  font-weight: 700;
  letter-spacing: -0.03em;
}

.error {
  color: var(--np-danger);
  font-size: var(--np-text-sm);
}
</style>
