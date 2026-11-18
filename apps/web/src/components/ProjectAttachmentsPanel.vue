<script setup lang="ts">
import { computed, onUnmounted, ref, watch } from "vue"
import { NpButton, NpInput, toast } from "@ningpm/ui"
import { api } from "../lib/api"
import { fileKindLabel, previewKind } from "../lib/preview"
import {
  downloadResource,
  fetchResourceBlob,
  formatBytes,
  isImageContentType,
  uploadResource,
} from "../lib/upload"
import AttachmentPreviewDialog from "./AttachmentPreviewDialog.vue"

export type ProjectAttachment = {
  id: string
  issueId: string
  issueKey?: string
  issueTitle?: string
  resourceId: string
  originalName: string
  contentType: string
  sizeBytes: number
  url: string
  createdBy: string
  creatorName?: string
  createdAt: string
}

export type ProjectFile = {
  id: string
  projectId: string
  resourceId: string
  originalName: string
  contentType: string
  sizeBytes: number
  url: string
  createdBy: string
  creatorName?: string
  createdAt: string
}

type FileRow = {
  id: string
  resourceId: string
  originalName: string
  contentType: string
  sizeBytes: number
  url: string
  creatorName?: string
  createdAt: string
  issueId?: string
  issueKey?: string
  issueTitle?: string
}

const props = defineProps<{
  projectId: string
  canWrite?: boolean
  active?: boolean
}>()

const emit = defineEmits<{
  "open-issue": [issueId: string]
  changed: []
}>()

type Tab = "materials" | "attachments"
const tab = ref<Tab>("materials")

const loading = ref(false)
const uploading = ref(false)
const error = ref("")
const materials = ref<ProjectFile[]>([])
const attachments = ref<ProjectAttachment[]>([])
const query = ref("")
const fileInput = ref<HTMLInputElement | null>(null)
const thumbUrls = ref<Record<string, string>>({})
const previewDialog = ref<{
  show: (o: { name: string; url: string; contentType?: string }) => Promise<void>
  close: () => void
} | null>(null)

const items = computed<FileRow[]>(() => {
  if (tab.value === "materials") {
    return materials.value.map((f) => ({
      id: f.id,
      resourceId: f.resourceId,
      originalName: f.originalName,
      contentType: f.contentType,
      sizeBytes: f.sizeBytes,
      url: f.url,
      creatorName: f.creatorName,
      createdAt: f.createdAt,
    }))
  }
  return attachments.value.map((a) => ({
    id: a.id,
    resourceId: a.resourceId,
    originalName: a.originalName,
    contentType: a.contentType,
    sizeBytes: a.sizeBytes,
    url: a.url,
    creatorName: a.creatorName,
    createdAt: a.createdAt,
    issueId: a.issueId,
    issueKey: a.issueKey,
    issueTitle: a.issueTitle,
  }))
})

const filtered = computed(() => {
  const q = query.value.trim().toLowerCase()
  if (!q) return items.value
  return items.value.filter((a) => {
    const hay = `${a.originalName} ${a.issueKey || ""} ${a.issueTitle || ""} ${a.creatorName || ""}`.toLowerCase()
    return hay.includes(q)
  })
})

const totalBytes = computed(() => items.value.reduce((sum, a) => sum + (a.sizeBytes || 0), 0))
const materialCount = computed(() => materials.value.length)
const attachmentCount = computed(() => attachments.value.length)

function kindOf(att: FileRow) {
  return previewKind(att.originalName, att.contentType)
}

function canPreview(att: FileRow) {
  return kindOf(att) !== "none"
}

function revokeThumbs() {
  for (const url of Object.values(thumbUrls.value)) URL.revokeObjectURL(url)
  thumbUrls.value = {}
}

async function loadThumbs(list: FileRow[]) {
  revokeThumbs()
  const next: Record<string, string> = {}
  await Promise.all(
    list
      .filter((a) => isImageContentType(a.contentType) || kindOf(a) === "image")
      .map(async (a) => {
        try {
          const blob = await fetchResourceBlob(a.url || `/api/resources/${a.resourceId}`)
          next[a.id] = URL.createObjectURL(blob)
        } catch {
          /* ignore */
        }
      }),
  )
  thumbUrls.value = next
}

async function load() {
  if (!props.projectId) return
  loading.value = true
  error.value = ""
  try {
    const [mats, atts] = await Promise.all([
      api<ProjectFile[]>(`/api/projects/${props.projectId}/files`),
      api<ProjectAttachment[]>(`/api/projects/${props.projectId}/attachments`),
    ])
    materials.value = mats
    attachments.value = atts
    const rows: FileRow[] =
      tab.value === "materials"
        ? mats.map((f) => ({
            id: f.id,
            resourceId: f.resourceId,
            originalName: f.originalName,
            contentType: f.contentType,
            sizeBytes: f.sizeBytes,
            url: f.url,
            creatorName: f.creatorName,
            createdAt: f.createdAt,
          }))
        : atts.map((a) => ({
            id: a.id,
            resourceId: a.resourceId,
            originalName: a.originalName,
            contentType: a.contentType,
            sizeBytes: a.sizeBytes,
            url: a.url,
            creatorName: a.creatorName,
            createdAt: a.createdAt,
            issueId: a.issueId,
            issueKey: a.issueKey,
            issueTitle: a.issueTitle,
          }))
    void loadThumbs(rows)
  } catch (e) {
    error.value = e instanceof Error ? e.message : "加载文件失败"
  } finally {
    loading.value = false
  }
}

async function openPreview(att: FileRow) {
  await previewDialog.value?.show({
    name: att.originalName,
    url: att.url || `/api/resources/${att.resourceId}`,
    contentType: att.contentType,
  })
}

async function onDownload(att: FileRow) {
  try {
    await downloadResource(att.url || `/api/resources/${att.resourceId}`, att.originalName)
  } catch (e) {
    toast.error(e instanceof Error ? e.message : "下载失败")
  }
}

async function onRemove(att: FileRow) {
  if (props.canWrite === false) return
  try {
    if (tab.value === "materials") {
      await api(`/api/projects/${props.projectId}/files/${att.id}`, { method: "DELETE" })
      materials.value = materials.value.filter((f) => f.id !== att.id)
      toast.success("已删除资料")
    } else {
      if (!att.issueId) return
      await api(`/api/issues/${att.issueId}/attachments/${att.id}`, { method: "DELETE" })
      attachments.value = attachments.value.filter((a) => a.id !== att.id)
      toast.success("已删除附件")
    }
    if (thumbUrls.value[att.id]) {
      URL.revokeObjectURL(thumbUrls.value[att.id])
      const { [att.id]: _, ...rest } = thumbUrls.value
      thumbUrls.value = rest
    }
    emit("changed")
  } catch (e) {
    toast.error(e instanceof Error ? e.message : "删除失败")
  }
}

async function onPickFile(ev: Event) {
  const input = ev.target as HTMLInputElement
  const file = input.files?.[0]
  input.value = ""
  if (!file || !props.projectId || props.canWrite === false) return
  uploading.value = true
  error.value = ""
  try {
    const res = await uploadResource(file)
    const created = await api<ProjectFile>(`/api/projects/${props.projectId}/files`, {
      method: "POST",
      body: JSON.stringify({ resourceId: res.id }),
    })
    materials.value = [created, ...materials.value]
    tab.value = "materials"
    if (isImageContentType(created.contentType)) {
      try {
        const blob = await fetchResourceBlob(created.url || `/api/resources/${created.resourceId}`)
        thumbUrls.value = { ...thumbUrls.value, [created.id]: URL.createObjectURL(blob) }
      } catch {
        /* ignore */
      }
    }
    emit("changed")
    toast.success("资料已上传")
  } catch (e) {
    const msg = e instanceof Error ? e.message : "上传资料失败"
    error.value = msg
    toast.error(msg)
  } finally {
    uploading.value = false
  }
}

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

function setTab(next: Tab) {
  if (tab.value === next) return
  tab.value = next
  void loadThumbs(items.value)
}

watch(
  () => [props.projectId, props.active] as const,
  ([id, active]) => {
    if (id && active) void load()
    if (!active) previewDialog.value?.close()
  },
  { immediate: true },
)

onUnmounted(() => {
  previewDialog.value?.close()
  revokeThumbs()
})

defineExpose({ reload: load })
</script>

<template>
  <div class="panel">
    <div class="tabs">
      <button
        type="button"
        class="tab"
        :class="{ active: tab === 'materials' }"
        @click="setTab('materials')"
      >
        资料
        <span class="count">{{ materialCount }}</span>
      </button>
      <button
        type="button"
        class="tab"
        :class="{ active: tab === 'attachments' }"
        @click="setTab('attachments')"
      >
        附件
        <span class="count">{{ attachmentCount }}</span>
      </button>
    </div>

    <div class="toolbar">
      <div class="stats">
        <strong>{{ items.length }}</strong>
        <span>{{ tab === "materials" ? "个资料" : "个附件" }}</span>
        <span class="dot">·</span>
        <span>{{ formatBytes(totalBytes) }}</span>
      </div>
      <div class="actions">
        <NpInput
          v-model="query"
          :placeholder="tab === 'materials' ? '搜文件名 / 上传人' : '搜文件名 / 事项 / 上传人'"
        />
        <NpButton size="sm" variant="secondary" :disabled="loading" @click="load">
          {{ loading ? "刷新中…" : "刷新" }}
        </NpButton>
        <template v-if="tab === 'materials' && canWrite !== false">
          <input
            ref="fileInput"
            type="file"
            class="file-input"
            @change="onPickFile"
          />
          <NpButton
            size="sm"
            :disabled="uploading"
            @click="fileInput?.click()"
          >
            {{ uploading ? "上传中…" : "上传资料" }}
          </NpButton>
        </template>
      </div>
    </div>

    <p v-if="error" class="error">{{ error }}</p>
    <p v-else-if="loading && !items.length" class="muted">加载中…</p>
    <p v-else-if="!filtered.length" class="muted empty">
      <template v-if="items.length">没有匹配的{{ tab === "materials" ? "资料" : "附件" }}</template>
      <template v-else-if="tab === 'materials'">
        还没有项目资料，可直接上传需求文档、合同、设计稿等
      </template>
      <template v-else>还没有事项附件</template>
    </p>

    <div v-else class="table-wrap">
      <table class="table" :class="{ 'no-issue': tab === 'materials' }">
        <colgroup>
          <col class="col-file" />
          <col v-if="tab === 'attachments'" class="col-issue" />
          <col class="col-size" />
          <col class="col-user" />
          <col class="col-time" />
          <col class="col-ops" />
        </colgroup>
        <thead>
          <tr>
            <th>文件</th>
            <th v-if="tab === 'attachments'">事项</th>
            <th>大小</th>
            <th>上传人</th>
            <th>时间</th>
            <th>操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="att in filtered" :key="att.id">
            <td>
              <div class="file-cell">
                <button type="button" class="thumb" :title="att.originalName" @click="openPreview(att)">
                  <img v-if="thumbUrls[att.id]" :src="thumbUrls[att.id]" alt="" />
                  <span v-else class="kind" :data-kind="kindOf(att)">{{ fileKindLabel(kindOf(att)) }}</span>
                </button>
                <button type="button" class="file-name" @click="openPreview(att)">
                  {{ att.originalName }}
                </button>
              </div>
            </td>
            <td v-if="tab === 'attachments'">
              <button
                v-if="att.issueId"
                type="button"
                class="issue-link"
                @click="emit('open-issue', att.issueId)"
              >
                <span class="key">{{ att.issueKey || "事项" }}</span>
                <span class="title">{{ att.issueTitle }}</span>
              </button>
              <span v-else class="muted">—</span>
            </td>
            <td class="cell-muted nowrap">{{ formatBytes(att.sizeBytes) }}</td>
            <td class="cell-muted">{{ att.creatorName || "—" }}</td>
            <td class="cell-muted nowrap">{{ formatTime(att.createdAt) }}</td>
            <td>
              <div class="ops">
                <button
                  v-if="canPreview(att)"
                  type="button"
                  class="op"
                  @click="openPreview(att)"
                >
                  预览
                </button>
                <button type="button" class="op" @click="onDownload(att)">下载</button>
                <button
                  v-if="canWrite !== false"
                  type="button"
                  class="op danger"
                  @click="onRemove(att)"
                >
                  删除
                </button>
              </div>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <AttachmentPreviewDialog ref="previewDialog" />
  </div>
</template>

<style scoped>
.panel {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.tabs {
  display: flex;
  gap: 4px;
  width: fit-content;
  padding: 3px;
  border-radius: 10px;
  background: #f1f5f9;
}

.tab {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  border: none;
  background: transparent;
  padding: 7px 14px;
  border-radius: 8px;
  color: var(--np-text-muted);
  font: inherit;
  font-size: var(--np-text-sm);
  font-weight: 600;
  cursor: pointer;
}

.tab.active {
  background: #fff;
  color: var(--np-text);
  box-shadow: 0 1px 2px rgba(15, 23, 42, 0.08);
}

.count {
  min-width: 18px;
  height: 18px;
  padding: 0 5px;
  border-radius: 999px;
  background: #e2e8f0;
  color: var(--np-text-muted);
  font-size: 11px;
  font-weight: 700;
  line-height: 18px;
  text-align: center;
}

.tab.active .count {
  background: #e0e7ff;
  color: #3730a3;
}

.toolbar {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.stats {
  display: flex;
  align-items: baseline;
  gap: 6px;
  color: var(--np-text-muted);
  font-size: var(--np-text-sm);
}

.stats strong {
  color: var(--np-text);
  font-size: var(--np-text-lg);
}

.dot {
  opacity: 0.5;
}

.actions {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: min(520px, 100%);
}

.actions :deep(.np-input) {
  flex: 1;
  min-width: 180px;
}

.file-input {
  display: none;
}

.table-wrap {
  overflow: auto;
  border: 1px solid var(--np-border);
  border-radius: 12px;
  background: #fff;
}

.table {
  width: 100%;
  border-collapse: collapse;
  table-layout: fixed;
  font-size: var(--np-text-sm);
}

.col-file {
  width: 34%;
}
.col-issue {
  width: 22%;
}
.col-size {
  width: 9%;
}
.col-user {
  width: 12%;
}
.col-time {
  width: 12%;
}
.col-ops {
  width: 11%;
}

.table.no-issue .col-file {
  width: 48%;
}
.table.no-issue .col-size {
  width: 12%;
}
.table.no-issue .col-user {
  width: 16%;
}
.table.no-issue .col-time {
  width: 14%;
}
.table.no-issue .col-ops {
  width: 10%;
}

.table th,
.table td {
  padding: 0 12px;
  height: 56px;
  border-bottom: 1px solid var(--np-border);
  text-align: left;
  vertical-align: middle;
  box-sizing: border-box;
}

.table tbody tr:last-child td {
  border-bottom: none;
}

.table th {
  height: 40px;
  font-size: var(--np-text-xs);
  color: var(--np-text-muted);
  font-weight: 600;
  background: #fafbfc;
}

.file-cell {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
}

.thumb {
  width: 36px;
  height: 36px;
  flex: 0 0 36px;
  padding: 0;
  border: 1px solid var(--np-border);
  border-radius: 8px;
  overflow: hidden;
  background: #f8fafc;
  cursor: pointer;
}

.thumb img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
}

.kind {
  width: 100%;
  height: 100%;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  font-size: 10px;
  font-weight: 700;
  color: #3730a3;
  background: #eef2ff;
}

.kind[data-kind="pdf"] {
  color: #b91c1c;
  background: #fee2e2;
}

.kind[data-kind="docx"] {
  color: #1d4ed8;
  background: #dbeafe;
}

.kind[data-kind="excel"] {
  color: #047857;
  background: #d1fae5;
}

.file-name {
  min-width: 0;
  flex: 1;
  border: none;
  background: transparent;
  padding: 0;
  text-align: left;
  cursor: pointer;
  color: inherit;
  font: inherit;
  font-weight: 600;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.issue-link {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 2px;
  max-width: 100%;
  border: none;
  background: transparent;
  padding: 0;
  text-align: left;
  cursor: pointer;
  color: inherit;
  font: inherit;
}

.key {
  color: var(--np-accent);
  font-weight: 700;
  font-family: var(--np-font-mono);
  font-size: var(--np-text-xs);
}

.title {
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: var(--np-text-muted);
  font-size: var(--np-text-xs);
}

.cell-muted {
  color: var(--np-text-muted);
}

.nowrap {
  white-space: nowrap;
}

.ops {
  display: flex;
  align-items: center;
  gap: 2px;
  white-space: nowrap;
}

.op {
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

.op:hover {
  color: var(--np-accent);
  background: #eff6ff;
}

.op.danger:hover {
  color: var(--np-danger);
  background: #fef2f2;
}

.muted {
  color: var(--np-text-muted);
  font-size: var(--np-text-sm);
}

.empty {
  margin: 24px 0;
}

.error {
  color: var(--np-danger);
  font-size: var(--np-text-sm);
}

@media (max-width: 720px) {
  .actions {
    width: 100%;
    min-width: 0;
  }

  .col-ops {
    width: 18%;
  }
}
</style>
