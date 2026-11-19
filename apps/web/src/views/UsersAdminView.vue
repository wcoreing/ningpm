<script setup lang="ts">
import { onMounted, onUnmounted, ref } from "vue"
import { storeToRefs } from "pinia"
import { NpBadge, NpButton, NpField, NpInput, NpModal, NpSelect, toast } from "@ningpm/ui"
import { useAuthStore } from "../stores/auth"
import { usePageHeaderStore } from "../stores/pageHeader"
import { useUsersAdminStore, type AdminUser } from "../stores/usersAdmin"

const auth = useAuthStore()
const pageHeader = usePageHeaderStore()
const usersStore = useUsersAdminStore()
const { items: users, loading, error } = storeToRefs(usersStore)

const open = ref(false)
const editOpen = ref(false)
const saving = ref(false)
const editSaving = ref(false)
const formError = ref("")
const form = ref({ name: "", account: "", password: "", isAdmin: "0" })
const editing = ref<AdminUser | null>(null)
const editForm = ref({ name: "", account: "", password: "", isAdmin: "0" })

function formatTime(iso?: string) {
  if (!iso) return "—"
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return iso
  return d.toLocaleString("zh-CN", { hour12: false })
}

function openCreate() {
  form.value = { name: "", account: "", password: "", isAdmin: "0" }
  formError.value = ""
  open.value = true
}

async function createUser() {
  const name = form.value.name.trim()
  const account = form.value.account.trim()
  const password = form.value.password
  if (!name) {
    formError.value = "请填写用户名"
    return
  }
  if (password.length < 6) {
    formError.value = "密码至少 6 位"
    return
  }
  saving.value = true
  formError.value = ""
  try {
    const created = await usersStore.create({
      name,
      account,
      password,
      isAdmin: form.value.isAdmin === "1",
    })
    open.value = false
    toast.success(`账号已创建：${created.account}（请将初始密码告知对方）`)
  } catch (e) {
    formError.value = e instanceof Error ? e.message : "创建失败"
  } finally {
    saving.value = false
  }
}

function openEdit(u: AdminUser) {
  editing.value = u
  editForm.value = {
    name: u.name,
    account: u.account,
    password: "",
    isAdmin: u.isAdmin ? "1" : "0",
  }
  formError.value = ""
  editOpen.value = true
}

async function saveUser() {
  if (!editing.value) return
  const name = editForm.value.name.trim()
  const account = editForm.value.account.trim()
  if (!name) {
    formError.value = "请填写用户名"
    return
  }
  if (!account) {
    formError.value = "请填写登录账号"
    return
  }
  const password = editForm.value.password
  if (password && password.length < 6) {
    formError.value = "密码至少 6 位"
    return
  }
  editSaving.value = true
  formError.value = ""
  try {
    const payload: { name: string; account: string; password?: string; isAdmin: boolean } = {
      name,
      account,
      isAdmin: editForm.value.isAdmin === "1",
    }
    if (password) payload.password = password
    await usersStore.update(editing.value.id, payload)
    editOpen.value = false
    editing.value = null
    toast.success("账号已更新")
    if (auth.user?.id) {
      await auth.fetchMe()
    }
  } catch (e) {
    formError.value = e instanceof Error ? e.message : "更新失败"
  } finally {
    editSaving.value = false
  }
}

async function removeUser(u: AdminUser) {
  if (u.id === auth.user?.id) {
    toast.warning("不能删除当前登录账号")
    return
  }
  if (!window.confirm(`确认删除账号「${u.name}」？`)) return
  try {
    await usersStore.remove(u.id)
    toast.success("账号已删除")
  } catch (e) {
    toast.error(e instanceof Error ? e.message : "删除失败")
  }
}

onMounted(() => {
  pageHeader.set({
    title: "账号管理",
    subtitle: "创建系统账号，分配管理员或普通角色",
  })
  void usersStore.fetchAll().catch(() => {})
})
onUnmounted(() => pageHeader.clear())
</script>

<template>
  <div class="page">
    <div class="toolbar">
      <div class="toolbar-copy">
        <p class="lead">系统账号</p>
        <p class="muted">管理员可在此新增、编辑、删除登录账号</p>
      </div>
      <NpButton variant="primary" @click="openCreate">新增账号</NpButton>
    </div>

    <p v-if="error || formError" class="error">{{ formError || error }}</p>
    <p v-if="loading" class="muted">加载中…</p>

    <div v-else class="table-wrap">
      <table class="table">
        <thead>
          <tr>
            <th>用户名</th>
            <th>账号</th>
            <th>角色</th>
            <th>创建时间</th>
            <th>操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="u in users" :key="u.id">
            <td class="name">{{ u.name }}</td>
            <td>{{ u.account }}</td>
            <td>
              <NpBadge :tone="u.isAdmin ? 'accent' : 'neutral'">
                {{ u.isAdmin ? "系统管理员" : "普通账号" }}
              </NpBadge>
            </td>
            <td class="time">{{ formatTime(u.createdAt) }}</td>
            <td class="actions">
              <NpButton size="sm" variant="ghost" @click="openEdit(u)">编辑</NpButton>
              <NpButton
                size="sm"
                variant="ghost"
                :disabled="u.id === auth.user?.id"
                @click="removeUser(u)"
              >
                删除
              </NpButton>
            </td>
          </tr>
        </tbody>
      </table>
      <div v-if="!users.length" class="empty">
        <p class="empty-title">还没有账号</p>
        <NpButton variant="primary" @click="openCreate">新增账号</NpButton>
      </div>
    </div>

    <NpModal :open="open" title="新增账号" @close="open = false">
      <div class="form">
        <NpField label="用户名">
          <NpInput v-model="form.name" placeholder="显示名" />
        </NpField>
        <NpField label="账号（选填）">
          <NpInput v-model="form.account" type="text" autocomplete="off" placeholder="登录账号，不填则按用户名生成" />
        </NpField>
        <NpField label="初始密码">
          <NpInput v-model="form.password" type="password" placeholder="至少 6 位" />
        </NpField>
        <NpField label="角色">
          <NpSelect v-model="form.isAdmin">
            <option value="0">普通账号</option>
            <option value="1">系统管理员</option>
          </NpSelect>
        </NpField>
      </div>
      <template #footer>
        <NpButton variant="ghost" @click="open = false">取消</NpButton>
        <NpButton variant="primary" :disabled="saving" @click="createUser">
          {{ saving ? "创建中…" : "创建" }}
        </NpButton>
      </template>
    </NpModal>

    <NpModal :open="editOpen" title="编辑账号" @close="editOpen = false">
      <div class="form">
        <NpField label="用户名">
          <NpInput v-model="editForm.name" placeholder="显示名" />
        </NpField>
        <NpField label="账号">
          <NpInput v-model="editForm.account" type="text" autocomplete="off" placeholder="登录账号" />
        </NpField>
        <NpField label="新密码">
          <NpInput v-model="editForm.password" type="password" placeholder="留空则不修改" />
        </NpField>
        <NpField label="角色">
          <NpSelect v-model="editForm.isAdmin">
            <option value="0">普通账号</option>
            <option value="1">系统管理员</option>
          </NpSelect>
        </NpField>
      </div>
      <template #footer>
        <NpButton variant="ghost" @click="editOpen = false">取消</NpButton>
        <NpButton variant="primary" :disabled="editSaving" @click="saveUser">
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

.time {
  color: var(--np-text-secondary);
  font-variant-numeric: tabular-nums;
  font-size: var(--np-text-sm);
}

.actions {
  display: flex;
  gap: 4px;
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
