<script setup lang="ts">
import { onMounted, ref } from "vue"
import { storeToRefs } from "pinia"
import { useRouter } from "vue-router"
import { NpBadge, NpButton, NpField, NpInput, NpModal, NpTextarea, toast } from "@ningpm/ui"
import { ROLE_LABELS } from "../lib/labels"
import type { Project } from "../types"
import { useProjectsStore } from "../stores/projects"

const router = useRouter()
const projectsStore = useProjectsStore()
const { items: projects, loading, error } = storeToRefs(projectsStore)

const open = ref(false)
const editOpen = ref(false)
const saving = ref(false)
const editSaving = ref(false)
const formError = ref("")
const form = ref({ key: "", name: "", description: "" })
const editing = ref<Project | null>(null)
const editForm = ref({ name: "", description: "" })

function canEdit(p: Project) {
  return p.myRole === "owner" || p.myRole === "admin"
}

async function createProject() {
  saving.value = true
  formError.value = ""
  try {
    const p = await projectsStore.create(form.value)
    open.value = false
    form.value = { key: "", name: "", description: "" }
    router.push(`/projects/${p.id}`)
  } catch (e) {
    formError.value = e instanceof Error ? e.message : "创建失败"
  } finally {
    saving.value = false
  }
}

function openEdit(p: Project) {
  editing.value = p
  editForm.value = { name: p.name, description: p.description || "" }
  editOpen.value = true
}

async function saveProject() {
  if (!editing.value) return
  const name = editForm.value.name.trim()
  if (!name) {
    toast.warning("项目名称不能为空")
    return
  }
  editSaving.value = true
  formError.value = ""
  try {
    await projectsStore.update(editing.value.id, {
      name,
      description: editForm.value.description.trim(),
    })
    editOpen.value = false
    editing.value = null
    toast.success("项目已更新")
  } catch (e) {
    formError.value = e instanceof Error ? e.message : "更新失败"
  } finally {
    editSaving.value = false
  }
}

onMounted(() => {
  void projectsStore.fetchAll().catch(() => {})
})
</script>

<template>
  <div class="page">
    <div class="toolbar">
      <div class="toolbar-copy">
        <p class="lead">项目列表</p>
      </div>
      <NpButton variant="primary" @click="open = true">新建项目</NpButton>
    </div>

    <p v-if="error || formError" class="error">{{ formError || error }}</p>
    <p v-if="loading" class="muted">加载中…</p>

    <div v-else class="table-wrap">
      <table class="table">
        <thead>
          <tr>
            <th>标识</th>
            <th>名称</th>
            <th>我的角色</th>
            <th>描述</th>
            <th class="num">事项</th>
            <th>操作</th>
          </tr>
        </thead>
        <tbody>
          <tr
            v-for="p in projects"
            :key="p.id"
            class="click-row"
            @click="router.push(`/projects/${p.id}`)"
          >
            <td>
              <NpBadge tone="accent">{{ p.key }}</NpBadge>
            </td>
            <td class="name">{{ p.name }}</td>
            <td>
              <span class="role">{{ p.myRole ? ROLE_LABELS[p.myRole] || p.myRole : "—" }}</span>
            </td>
            <td class="desc">{{ p.description || "—" }}</td>
            <td class="num">{{ p.issueCount }}</td>
            <td>
              <NpButton
                v-if="canEdit(p)"
                size="sm"
                variant="ghost"
                @click.stop="openEdit(p)"
              >
                编辑项目
              </NpButton>
              <span v-else class="muted">—</span>
            </td>
          </tr>
        </tbody>
      </table>
      <div v-if="!projects.length" class="empty">
        <p class="empty-title">还没有项目</p>
        <NpButton variant="primary" @click="open = true">新建项目</NpButton>
      </div>
    </div>

    <NpModal :open="open" title="新建项目" @close="open = false">
      <div class="form">
        <NpField label="项目标识">
          <NpInput v-model="form.key" placeholder="DEMO" />
        </NpField>
        <NpField label="项目名称">
          <NpInput v-model="form.name" placeholder="例如：宁平台" />
        </NpField>
        <NpField label="描述">
          <NpTextarea v-model="form.description" placeholder="项目目标与范围" :rows="3" />
        </NpField>
      </div>
      <template #footer>
        <NpButton variant="ghost" @click="open = false">取消</NpButton>
        <NpButton variant="primary" :disabled="saving" @click="createProject">
          {{ saving ? "创建中…" : "创建" }}
        </NpButton>
      </template>
    </NpModal>

    <NpModal :open="editOpen" title="编辑项目" @close="editOpen = false">
      <div class="form">
        <NpField label="项目标识">
          <NpInput :model-value="editing?.key || ''" disabled />
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
  </div>
</template>

<style scoped>
.page {
  display: flex;
  flex-direction: column;
  gap: 18px;
}

.toolbar {
  display: flex;
  justify-content: space-between;
  gap: 12px;
  align-items: flex-end;
}

.toolbar-copy {
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

.name {
  font-weight: 650;
  color: var(--np-text);
}

.role {
  display: inline-flex;
  padding: 2px 8px;
  border-radius: 999px;
  background: color-mix(in srgb, var(--np-bg-sunken) 70%, #fff);
  color: var(--np-text-secondary);
  font-size: var(--np-text-xs);
  font-weight: 600;
}

.num {
  text-align: right;
  font-variant-numeric: tabular-nums;
  font-weight: 600;
  color: var(--np-text-secondary);
}

.click-row {
  cursor: pointer;
  transition: background var(--np-duration-fast) var(--np-ease);
}

.desc {
  color: var(--np-text-secondary);
  max-width: 28rem;
}

.muted {
  margin: 0;
  color: var(--np-text-muted);
  font-size: var(--np-text-sm);
}

.empty {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 10px;
  padding: 36px 20px;
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
}

.form {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
</style>
