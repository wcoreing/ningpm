<script setup lang="ts">
import { computed, ref, watch } from "vue"
import { NpBadge, NpButton, NpDrawer, NpField, NpInput, NpSelect, NpTextarea, toast } from "@ningpm/ui"
import { api } from "../lib/api"
import { fileKindLabel, previewKind } from "../lib/preview"
import {
  downloadResource,
  fetchResourceBlob,
  formatBytes,
  isImageContentType,
  uploadResource,
} from "../lib/upload"
import {
  PRIORITY_LABELS,
  STATUS_LABELS,
  STATUS_ORDER,
  TYPE_LABELS,
  priorityTone,
  typeTone,
} from "../lib/labels"
import type { Cycle, Issue, Module, UserOption } from "../types"
import AttachmentPreviewDialog from "./AttachmentPreviewDialog.vue"
import ModuleTagPicker from "./ModuleTagPicker.vue"

export type IssueComment = {
  id: string
  issueId: string
  authorId: string
  authorName: string
  body: string
  createdAt: string
}

export type IssueAttachment = {
  id: string
  issueId: string
  resourceId: string
  originalName: string
  contentType: string
  sizeBytes: number
  url: string
  createdBy: string
  creatorName?: string
  createdAt: string
}

const props = defineProps<{
  open: boolean
  issueId: string | null
  projectId?: string | null
  users: UserOption[]
  cycles?: Cycle[]
  modules?: Module[]
  canWrite?: boolean
}>()

const emit = defineEmits<{
  close: []
  updated: [issue: Issue]
  "open-issue": [issueId: string]
  "update:modules": [modules: Module[]]
}>()

const loading = ref(false)
const saving = ref(false)
const commenting = ref(false)
const childSaving = ref(false)
const uploading = ref(false)
const error = ref("")
const issue = ref<Issue | null>(null)
const comments = ref<IssueComment[]>([])
const attachments = ref<IssueAttachment[]>([])
const children = ref<Issue[]>([])
const draft = ref("")
const childTitle = ref("")
const childType = ref("task")
const fileInput = ref<HTMLInputElement | null>(null)
const previewDialog = ref<{ show: (o: { name: string; url: string; contentType?: string }) => Promise<void>; close: () => void } | null>(null)
const thumbUrls = ref<Record<string, string>>({})

function syncAttachmentCount() {
  if (!issue.value) return
  issue.value = { ...issue.value, attachmentCount: attachments.value.length }
  emit("updated", issue.value)
}

function revokeThumbs() {
  for (const url of Object.values(thumbUrls.value)) URL.revokeObjectURL(url)
  thumbUrls.value = {}
}

function kindOf(att: IssueAttachment) {
  return previewKind(att.originalName, att.contentType)
}

function canPreview(att: IssueAttachment) {
  return kindOf(att) !== "none"
}

async function loadThumbs(list: IssueAttachment[]) {
  revokeThumbs()
  const next: Record<string, string> = {}
  await Promise.all(
    list
      .filter((a) => isImageContentType(a.contentType))
      .map(async (a) => {
        try {
          const blob = await fetchResourceBlob(a.url || `/api/resources/${a.resourceId}`)
          next[a.id] = URL.createObjectURL(blob)
        } catch {
          /* ignore thumb errors */
        }
      }),
  )
  thumbUrls.value = next
}

const form = ref({
  title: "",
  description: "",
  type: "task",
  status: "todo",
  priority: "3",
  assigneeId: "",
  cycleId: "",
  moduleId: "",
})

const open = computed(() => props.open && Boolean(props.issueId))
const isRoot = computed(() => Boolean(issue.value) && !issue.value?.parentId)
const childProgress = computed(() => {
  const total = children.value.length
  const done = children.value.filter((c) => c.status === "done").length
  return { total, done }
})

async function load() {
  if (!props.issueId) return
  loading.value = true
  error.value = ""
  try {
    const [it, cms, atts] = await Promise.all([
      api<Issue>(`/api/issues/${props.issueId}`),
      api<IssueComment[]>(`/api/issues/${props.issueId}/comments`),
      api<IssueAttachment[]>(`/api/issues/${props.issueId}/attachments`),
    ])
    issue.value = it
    comments.value = cms
    attachments.value = atts
    void loadThumbs(atts)
    form.value = {
      title: it.title,
      description: it.description || "",
      type: it.type,
      status: it.status,
      priority: String(it.priority),
      assigneeId: it.assigneeId || "",
      cycleId: it.cycleId || "",
      moduleId: it.moduleId || "",
    }
    if (!it.parentId) {
      children.value = await api<Issue[]>(`/api/issues?parentId=${encodeURIComponent(it.id)}`)
    } else {
      children.value = []
    }
  } catch (e) {
    error.value = e instanceof Error ? e.message : "加载失败"
  } finally {
    loading.value = false
  }
}

async function createChild() {
  if (!issue.value || !childTitle.value.trim() || props.canWrite === false) return
  childSaving.value = true
  error.value = ""
  try {
    const created = await api<Issue>("/api/issues", {
      method: "POST",
      body: JSON.stringify({
        projectId: issue.value.projectId,
        parentId: issue.value.id,
        type: childType.value,
        title: childTitle.value.trim(),
        priority: Number(form.value.priority) || 3,
        assigneeId: form.value.assigneeId || "",
        cycleId: form.value.cycleId || "",
        moduleId: form.value.moduleId || "",
        status: "todo",
      }),
    })
    children.value = [...children.value, created]
    childTitle.value = ""
    issue.value = {
      ...issue.value,
      childCount: (issue.value.childCount || 0) + 1,
      childDoneCount: issue.value.childDoneCount || 0,
    }
    emit("updated", issue.value)
    toast.success("已添加子事项")
  } catch (e) {
    const msg = e instanceof Error ? e.message : "创建子事项失败"
    error.value = msg
    toast.error(msg)
  } finally {
    childSaving.value = false
  }
}

async function toggleChildDone(child: Issue) {
  if (props.canWrite === false) return
  const next = child.status === "done" ? "todo" : "done"
  try {
    const updated = await api<Issue>(`/api/issues/${child.id}/status`, {
      method: "PATCH",
      body: JSON.stringify({ status: next }),
    })
    children.value = children.value.map((c) => (c.id === child.id ? updated : c))
    if (issue.value) {
      const done = children.value.filter((c) => c.status === "done").length
      issue.value = { ...issue.value, childCount: children.value.length, childDoneCount: done }
      emit("updated", issue.value)
    }
  } catch (e) {
    toast.error(e instanceof Error ? e.message : "更新子事项失败")
  }
}

async function save() {
  if (!props.issueId) return
  saving.value = true
  error.value = ""
  try {
    const updated = await api<Issue>(`/api/issues/${props.issueId}`, {
      method: "PATCH",
      body: JSON.stringify({
        title: form.value.title,
        description: form.value.description,
        type: form.value.type,
        status: form.value.status,
        priority: Number(form.value.priority),
        assigneeId: form.value.assigneeId,
        cycleId: form.value.cycleId,
        moduleId: form.value.moduleId,
      }),
    })
    issue.value = updated
    emit("updated", updated)
    toast.success("已保存")
    emit("close")
  } catch (e) {
    const msg = e instanceof Error ? e.message : "保存失败"
    error.value = msg
    toast.error(msg)
  } finally {
    saving.value = false
  }
}

async function sendComment() {
  if (!props.issueId || !draft.value.trim()) return
  commenting.value = true
  error.value = ""
  try {
    const cm = await api<IssueComment>(`/api/issues/${props.issueId}/comments`, {
      method: "POST",
      body: JSON.stringify({ body: draft.value.trim() }),
    })
    comments.value = [...comments.value, cm]
    draft.value = ""
    toast.success("评论已发表")
  } catch (e) {
    const msg = e instanceof Error ? e.message : "评论失败"
    error.value = msg
    toast.error(msg)
  } finally {
    commenting.value = false
  }
}

async function onPickAttachment(ev: Event) {
  const input = ev.target as HTMLInputElement
  const file = input.files?.[0]
  input.value = ""
  if (!file || !props.issueId || props.canWrite === false) return
  uploading.value = true
  error.value = ""
  try {
    const res = await uploadResource(file)
    const att = await api<IssueAttachment>(`/api/issues/${props.issueId}/attachments`, {
      method: "POST",
      body: JSON.stringify({ resourceId: res.id }),
    })
    attachments.value = [...attachments.value, att]
    syncAttachmentCount()
    if (isImageContentType(att.contentType)) {
      try {
        const blob = await fetchResourceBlob(att.url || `/api/resources/${att.resourceId}`)
        thumbUrls.value = { ...thumbUrls.value, [att.id]: URL.createObjectURL(blob) }
      } catch {
        /* ignore */
      }
    }
    toast.success("附件已上传")
  } catch (e) {
    const msg = e instanceof Error ? e.message : "上传附件失败"
    error.value = msg
    toast.error(msg)
  } finally {
    uploading.value = false
  }
}

async function removeAttachment(att: IssueAttachment) {
  if (!props.issueId || props.canWrite === false) return
  try {
    await api(`/api/issues/${props.issueId}/attachments/${att.id}`, { method: "DELETE" })
    if (thumbUrls.value[att.id]) {
      URL.revokeObjectURL(thumbUrls.value[att.id])
      const { [att.id]: _, ...rest } = thumbUrls.value
      thumbUrls.value = rest
    }
    attachments.value = attachments.value.filter((a) => a.id !== att.id)
    syncAttachmentCount()
    toast.success("已移除附件")
  } catch (e) {
    toast.error(e instanceof Error ? e.message : "删除附件失败")
  }
}

async function onOpenAttachment(att: IssueAttachment) {
  await previewDialog.value?.show({
    name: att.originalName,
    url: att.url || `/api/resources/${att.resourceId}`,
    contentType: att.contentType,
  })
}

async function onDownloadAttachment(att: IssueAttachment) {
  try {
    await downloadResource(att.url || `/api/resources/${att.resourceId}`, att.originalName)
  } catch (e) {
    toast.error(e instanceof Error ? e.message : "下载失败")
  }
}

watch(
  () => props.open,
  (isOpen) => {
    if (!isOpen) {
      previewDialog.value?.close()
      revokeThumbs()
    }
  },
)

function formatTime(iso: string) {
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return iso
  return d.toLocaleString("zh-CN", {
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
  })
}

watch(
  () => [props.open, props.issueId] as const,
  ([isOpen, id]) => {
    if (isOpen && id) void load()
  },
)
</script>

<template>
  <NpDrawer :open="open" :title="form.title || '事项详情'" width="520px" @close="emit('close')">
    <template #eyebrow>
      <span v-if="issue">{{ issue.key }} · {{ TYPE_LABELS[issue.type] }}</span>
    </template>

    <p v-if="error" class="error">{{ error }}</p>
    <p v-if="loading" class="muted">加载中…</p>

    <template v-else-if="issue">
      <p v-if="issue.parentId" class="parent-link">
        父事项
        <button type="button" class="link" @click="emit('open-issue', issue.parentId!)">
          {{ issue.parentKey || "查看" }}
        </button>
      </p>

      <div class="meta">
        <NpBadge :tone="typeTone(form.type)">{{ TYPE_LABELS[form.type] }}</NpBadge>
        <NpBadge :tone="priorityTone(Number(form.priority))">
          {{ PRIORITY_LABELS[Number(form.priority)] }}
        </NpBadge>
        <NpBadge>{{ STATUS_LABELS[form.status] }}</NpBadge>
        <NpBadge v-if="isRoot && childProgress.total" tone="accent">
          子项 {{ childProgress.done }}/{{ childProgress.total }}
        </NpBadge>
      </div>

      <div class="fields">
        <NpField label="标题">
          <NpInput v-model="form.title" />
        </NpField>
        <div class="row">
          <NpField label="类型">
            <NpSelect v-model="form.type">
              <option value="story">需求</option>
              <option value="task">任务</option>
              <option value="bug">缺陷</option>
              <option value="epic">史诗</option>
            </NpSelect>
          </NpField>
          <NpField label="状态">
            <NpSelect v-model="form.status">
              <option v-for="s in STATUS_ORDER" :key="s" :value="s">{{ STATUS_LABELS[s] }}</option>
            </NpSelect>
          </NpField>
        </div>
        <div class="row">
          <NpField label="优先级">
            <NpSelect v-model="form.priority">
              <option value="1">紧急</option>
              <option value="2">高</option>
              <option value="3">中</option>
              <option value="4">低</option>
            </NpSelect>
          </NpField>
          <NpField label="指派给">
            <NpSelect v-model="form.assigneeId">
              <option value="">未指派</option>
              <option v-for="u in users" :key="u.id" :value="u.id">{{ u.name }}</option>
            </NpSelect>
          </NpField>
        </div>
        <NpField label="所属迭代">
          <NpSelect v-model="form.cycleId">
            <option value="">待规划</option>
            <option v-for="cy in cycles || []" :key="cy.id" :value="cy.id">
              {{ cy.name }}（{{ cy.status === "active" ? "进行中" : cy.status === "planned" ? "未开始" : "已完成" }}）
            </option>
          </NpSelect>
        </NpField>
        <NpField label="模块标签">
          <ModuleTagPicker
            v-if="projectId || issue?.projectId"
            :project-id="(projectId || issue?.projectId)!"
            :modules="modules || []"
            v-model="form.moduleId"
            :can-write="canWrite !== false"
            @update:modules="emit('update:modules', $event)"
          />
        </NpField>
        <NpField label="描述">
          <NpTextarea v-model="form.description" :rows="5" />
        </NpField>
      </div>

      <section v-if="isRoot" class="children">
        <h4>子事项 · {{ childProgress.done }}/{{ childProgress.total || 0 }}</h4>
        <ul v-if="children.length" class="child-list">
          <li v-for="ch in children" :key="ch.id" class="child-row">
            <label class="check">
              <input
                type="checkbox"
                :checked="ch.status === 'done'"
                :disabled="canWrite === false"
                @change="toggleChildDone(ch)"
              />
            </label>
            <button type="button" class="child-main" @click="emit('open-issue', ch.id)">
              <span class="key">{{ ch.key }}</span>
              <span class="child-title" :class="{ done: ch.status === 'done' }">{{ ch.title }}</span>
              <NpBadge :tone="typeTone(ch.type)">{{ TYPE_LABELS[ch.type] }}</NpBadge>
            </button>
          </li>
        </ul>
        <p v-else class="muted">还没有子事项</p>
        <div v-if="canWrite !== false" class="child-create">
          <NpSelect v-model="childType" size="sm">
            <option value="task">任务</option>
            <option value="bug">缺陷</option>
            <option value="story">需求</option>
          </NpSelect>
          <NpInput v-model="childTitle" placeholder="子事项标题，回车添加" @keydown.enter.prevent="createChild" />
          <NpButton size="sm" variant="secondary" :disabled="childSaving || !childTitle.trim()" @click="createChild">
            {{ childSaving ? "添加中…" : "添加" }}
          </NpButton>
        </div>
      </section>

      <section class="attachments">
        <div class="attachments-head">
          <h4>附件 · {{ attachments.length }}</h4>
          <template v-if="canWrite !== false">
            <input
              ref="fileInput"
              type="file"
              class="file-input"
              @change="onPickAttachment"
            />
            <NpButton
              size="sm"
              variant="secondary"
              :disabled="uploading"
              @click="fileInput?.click()"
            >
              {{ uploading ? "上传中…" : "上传附件" }}
            </NpButton>
          </template>
        </div>
        <p v-if="!attachments.length" class="muted">还没有附件</p>
        <ul v-else class="att-list">
          <li v-for="att in attachments" :key="att.id" class="att-row">
            <button type="button" class="att-main" @click="onOpenAttachment(att)">
              <span class="att-thumb-wrap">
                <img
                  v-if="thumbUrls[att.id]"
                  class="att-thumb"
                  :src="thumbUrls[att.id]"
                  alt=""
                />
                <span v-else class="att-icon" :data-kind="kindOf(att)">{{ fileKindLabel(kindOf(att)) }}</span>
              </span>
              <span class="att-text">
                <span class="att-name">{{ att.originalName }}</span>
                <span class="att-meta">
                  {{ formatBytes(att.sizeBytes) }} · {{ att.creatorName || "用户" }}
                  <template v-if="canPreview(att)"> · 点击预览</template>
                </span>
              </span>
            </button>
            <div class="att-ops">
              <button v-if="canPreview(att)" type="button" class="att-op" @click="onOpenAttachment(att)">预览</button>
              <button type="button" class="att-op" @click="onDownloadAttachment(att)">下载</button>
              <button
                v-if="canWrite !== false"
                type="button"
                class="att-op danger"
                @click="removeAttachment(att)"
              >
                删除
              </button>
            </div>
          </li>
        </ul>
      </section>

      <AttachmentPreviewDialog ref="previewDialog" />

      <section class="comments">
        <h4>评论 · {{ comments.length }}</h4>
        <div v-if="!comments.length" class="muted">还没有评论</div>
        <article v-for="cm in comments" :key="cm.id" class="comment">
          <header>
            <strong>{{ cm.authorName || "用户" }}</strong>
            <span>{{ formatTime(cm.createdAt) }}</span>
          </header>
          <p>{{ cm.body }}</p>
        </article>

        <div class="composer">
          <NpTextarea v-model="draft" placeholder="写下进展、疑问或验收意见…" :rows="3" />
          <NpButton
            variant="secondary"
            :disabled="commenting || !draft.trim() || canWrite === false"
            @click="sendComment"
          >
            {{ commenting ? "发送中…" : "发表评论" }}
          </NpButton>
        </div>
      </section>
    </template>

    <template #footer>
      <NpButton variant="ghost" @click="emit('close')">关闭</NpButton>
      <NpButton variant="primary" :disabled="saving || loading || canWrite === false" @click="save">
        {{ saving ? "保存中…" : "保存修改" }}
      </NpButton>
    </template>
  </NpDrawer>
</template>

<style scoped>
.meta {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-bottom: 16px;
}

.fields {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.row {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 12px;
}

.parent-link {
  margin: 0 0 10px;
  font-size: var(--np-text-xs);
  color: var(--np-text-muted);
}

.link {
  border: none;
  background: transparent;
  color: var(--np-accent);
  font: inherit;
  font-weight: 600;
  cursor: pointer;
  padding: 0 2px;
}

.children {
  margin-top: 22px;
  padding-top: 16px;
  border-top: 1px solid var(--np-border);
}

.children h4 {
  margin: 0 0 10px;
  font-size: var(--np-text-md);
}

.child-list {
  list-style: none;
  margin: 0 0 12px;
  padding: 0;
}

.child-row {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 0;
  border-bottom: 1px solid var(--np-border);
}

.check {
  display: inline-flex;
  flex-shrink: 0;
}

.child-main {
  display: flex;
  flex: 1;
  min-width: 0;
  align-items: center;
  gap: 8px;
  border: none;
  background: transparent;
  padding: 0;
  text-align: left;
  cursor: pointer;
  color: inherit;
}

.child-title {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: var(--np-text-sm);
  font-weight: 600;
}

.child-title.done {
  color: var(--np-text-muted);
  text-decoration: line-through;
}

.child-create {
  display: grid;
  grid-template-columns: 88px minmax(0, 1fr) auto;
  gap: 8px;
  align-items: center;
}

.attachments {
  margin-top: 22px;
  padding-top: 16px;
  border-top: 1px solid var(--np-border);
}

.attachments-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 10px;
}

.attachments-head h4 {
  margin: 0;
  font-size: var(--np-text-md);
}

.file-input {
  display: none;
}

.att-list {
  list-style: none;
  margin: 0;
  padding: 0;
}

.att-row {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 0;
  border-bottom: 1px solid var(--np-border);
}

.att-main {
  flex: 1;
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 10px;
  border: none;
  background: transparent;
  padding: 0;
  text-align: left;
  cursor: pointer;
  color: inherit;
}

.att-thumb-wrap {
  width: 40px;
  height: 40px;
  flex: 0 0 40px;
  border-radius: 8px;
  overflow: hidden;
  border: 1px solid var(--np-border);
  background: #f8fafc;
}

.att-thumb,
.att-icon {
  width: 100%;
  height: 100%;
  display: block;
  object-fit: cover;
}

.att-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  font-size: 10px;
  font-weight: 700;
  color: #3730a3;
  background: #eef2ff;
}

.att-icon[data-kind="pdf"] {
  color: #b91c1c;
  background: #fee2e2;
}

.att-icon[data-kind="docx"] {
  color: #1d4ed8;
  background: #dbeafe;
}

.att-icon[data-kind="excel"] {
  color: #047857;
  background: #d1fae5;
}

.att-text {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}

.att-name {
  font-size: var(--np-text-sm);
  font-weight: 600;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  max-width: 100%;
}

.att-meta {
  font-size: var(--np-text-xs);
  color: var(--np-text-muted);
}

.att-ops {
  display: flex;
  align-items: center;
  gap: 2px;
  flex-shrink: 0;
}

.att-op {
  border: none;
  background: transparent;
  padding: 4px 6px;
  border-radius: 6px;
  color: var(--np-text-muted);
  font: inherit;
  font-size: var(--np-text-xs);
  font-weight: 600;
  cursor: pointer;
}

.att-op:hover {
  color: var(--np-accent);
  background: #eff6ff;
}

.att-op.danger:hover {
  color: var(--np-danger);
  background: #fef2f2;
}

.comments {
  margin-top: 28px;
  padding-top: 18px;
  border-top: 1px solid var(--np-border);
}

.comments h4 {
  margin: 0 0 12px;
  font-size: var(--np-text-md);
}

.comment {
  padding: 10px 0;
  border-bottom: 1px solid var(--np-border);
}

.comment header {
  display: flex;
  justify-content: space-between;
  gap: 8px;
  margin-bottom: 6px;
  font-size: var(--np-text-xs);
  color: var(--np-text-muted);
}

.comment header strong {
  color: var(--np-text);
  font-size: var(--np-text-sm);
}

.comment p {
  margin: 0;
  white-space: pre-wrap;
  line-height: 1.55;
  font-size: var(--np-text-sm);
}

.composer {
  display: flex;
  flex-direction: column;
  gap: 10px;
  margin-top: 14px;
  align-items: flex-end;
}

.composer :deep(.np-textarea) {
  width: 100%;
}

.muted {
  color: var(--np-text-muted);
  font-size: var(--np-text-sm);
}

.error {
  color: var(--np-danger);
  font-size: var(--np-text-sm);
}

@media (max-width: 640px) {
  .row {
    grid-template-columns: 1fr;
  }
}
</style>
