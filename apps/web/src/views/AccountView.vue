<script setup lang="ts">
import { onMounted, ref } from "vue"
import { storeToRefs } from "pinia"
import { useRouter } from "vue-router"
import { NpBadge, NpButton, NpField, NpInput, toast } from "@ningpm/ui"
import UserAvatar from "../components/UserAvatar.vue"
import { ROLE_LABELS } from "../lib/labels"
import { useAuthStore } from "../stores/auth"
import { useProjectsStore } from "../stores/projects"

const auth = useAuthStore()
const projectsStore = useProjectsStore()
const { items: projects, loading: projectsLoading, error } = storeToRefs(projectsStore)
const router = useRouter()

const tab = ref<"profile" | "password" | "projects">("profile")
const sideItems = [
  { value: "profile" as const, label: "资料" },
  { value: "password" as const, label: "密码" },
  { value: "projects" as const, label: "项目与角色" },
]

const name = ref("")
const profileSaving = ref(false)
const avatarSaving = ref(false)
const fileInput = ref<HTMLInputElement | null>(null)

const currentPassword = ref("")
const newPassword = ref("")
const confirmPassword = ref("")
const pwdSaving = ref(false)

async function saveProfile() {
  const n = name.value.trim()
  if (!n) {
    toast.warning("显示名不能为空")
    return
  }
  profileSaving.value = true
  try {
    await auth.updateProfile(n)
    toast.success("资料已保存")
  } catch (e) {
    toast.error(e instanceof Error ? e.message : "保存失败")
  } finally {
    profileSaving.value = false
  }
}

async function changePassword() {
  if (newPassword.value.length < 6) {
    toast.warning("新密码至少 6 位")
    return
  }
  if (newPassword.value !== confirmPassword.value) {
    toast.warning("两次新密码不一致")
    return
  }
  pwdSaving.value = true
  try {
    await auth.changePassword(currentPassword.value, newPassword.value)
    currentPassword.value = ""
    newPassword.value = ""
    confirmPassword.value = ""
    toast.success("密码已更新")
  } catch (e) {
    toast.error(e instanceof Error ? e.message : "修改失败")
  } finally {
    pwdSaving.value = false
  }
}

function pickAvatar() {
  fileInput.value?.click()
}

async function onAvatarPicked(ev: Event) {
  const input = ev.target as HTMLInputElement
  const file = input.files?.[0]
  input.value = ""
  if (!file) return
  avatarSaving.value = true
  try {
    await auth.uploadAvatar(file)
    toast.success("头像已更新")
  } catch (e) {
    toast.error(e instanceof Error ? e.message : "上传失败")
  } finally {
    avatarSaving.value = false
  }
}

async function resetAvatar() {
  avatarSaving.value = true
  try {
    await auth.clearAvatar()
    toast.success("已恢复默认头像")
  } catch (e) {
    toast.error(e instanceof Error ? e.message : "操作失败")
  } finally {
    avatarSaving.value = false
  }
}

onMounted(() => {
  name.value = auth.user?.name || ""
  void projectsStore.fetchAll().catch(() => {})
})
</script>

<template>
  <div class="page">
    <aside class="side" aria-label="账号菜单">
      <p class="side-label">账号</p>
      <nav class="side-nav">
        <button
          v-for="item in sideItems"
          :key="item.value"
          type="button"
          class="side-item"
          :class="{ 'is-on': tab === item.value }"
          @click="tab = item.value"
        >
          {{ item.label }}
        </button>
      </nav>
    </aside>

    <section class="main">
      <p v-if="error" class="error">{{ error }}</p>

      <div v-if="tab === 'profile'" class="panel">
        <h2>资料</h2>
        <div class="avatar-row">
          <UserAvatar
            :name="auth.user?.name"
            :email="auth.user?.email"
            :avatar-url="auth.user?.avatarUrl"
            size="lg"
          />
          <div class="avatar-actions">
            <div class="actions">
              <NpButton size="sm" variant="secondary" :disabled="avatarSaving" @click="pickAvatar">
                {{ avatarSaving ? "处理中…" : "上传头像" }}
              </NpButton>
              <NpButton
                v-if="auth.user?.avatarUrl"
                size="sm"
                variant="ghost"
                :disabled="avatarSaving"
                @click="resetAvatar"
              >
                恢复默认
              </NpButton>
            </div>
            <input
              ref="fileInput"
              type="file"
              accept="image/png,image/jpeg,image/webp,image/gif"
              hidden
              @change="onAvatarPicked"
            />
          </div>
        </div>
        <div class="form">
          <NpField label="显示名">
            <NpInput v-model="name" placeholder="你的名字" />
          </NpField>
          <NpField label="邮箱">
            <NpInput :model-value="auth.user?.email || ''" disabled />
          </NpField>
          <div class="actions">
            <NpButton variant="primary" :disabled="profileSaving" @click="saveProfile">
              {{ profileSaving ? "保存中…" : "保存资料" }}
            </NpButton>
          </div>
        </div>
      </div>

      <div v-else-if="tab === 'password'" class="panel">
        <h2>修改密码</h2>
        <div class="form">
          <NpField label="当前密码">
            <NpInput v-model="currentPassword" type="password" placeholder="当前密码" />
          </NpField>
          <NpField label="新密码">
            <NpInput v-model="newPassword" type="password" placeholder="新密码（至少 6 位）" />
          </NpField>
          <NpField label="确认新密码">
            <NpInput v-model="confirmPassword" type="password" placeholder="再输一次" />
          </NpField>
          <div class="actions">
            <NpButton variant="primary" :disabled="pwdSaving" @click="changePassword">
              {{ pwdSaving ? "提交中…" : "更新密码" }}
            </NpButton>
          </div>
        </div>
      </div>

      <div v-else class="panel">
        <h2>我的项目与角色</h2>
        <p v-if="projectsLoading" class="muted">加载中…</p>
        <div v-else class="table-wrap">
          <table class="table">
            <thead>
              <tr>
                <th>标识</th>
                <th>名称</th>
                <th>我的角色</th>
                <th></th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="p in projects" :key="p.id">
                <td><NpBadge tone="accent">{{ p.key }}</NpBadge></td>
                <td>{{ p.name }}</td>
                <td>{{ p.myRole ? ROLE_LABELS[p.myRole] || p.myRole : "—" }}</td>
                <td>
                  <NpButton size="sm" variant="ghost" @click="router.push(`/projects/${p.id}`)">
                    进入
                  </NpButton>
                </td>
              </tr>
            </tbody>
          </table>
          <p v-if="!projects.length" class="muted empty">还没有加入任何项目。</p>
        </div>
      </div>
    </section>
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

.main {
  display: flex;
  flex-direction: column;
  gap: 14px;
  min-width: 0;
  padding: 0;
}

.panel {
  max-width: 640px;
  padding: 18px;
  border: 1px solid var(--np-border);
  border-radius: var(--np-radius-lg);
  background:
    linear-gradient(180deg, color-mix(in srgb, #fff 72%, var(--np-bg)) 0%, var(--np-bg-elevated) 100%);
  box-shadow: var(--np-shadow-sm);
}

.panel h2 {
  margin: 0 0 12px;
  font-family: var(--np-font-display);
  font-size: var(--np-text-lg);
  font-weight: 700;
  letter-spacing: -0.03em;
}

.avatar-row {
  display: flex;
  align-items: center;
  gap: 16px;
  margin-bottom: 16px;
  padding-bottom: 16px;
  border-bottom: 1px solid var(--np-border);
}

.avatar-actions {
  min-width: 0;
}

.form {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.actions {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  justify-content: flex-start;
}

.muted {
  margin: 0 0 12px;
  color: var(--np-text-muted);
  font-size: var(--np-text-sm);
}

.empty {
  padding: 12px 0 0;
}

.error {
  color: var(--np-danger);
}

.table-wrap {
  overflow: auto;
  border: 1px solid var(--np-border);
  border-radius: var(--np-radius-md);
  background: #fff;
}

.table {
  width: 100%;
  border-collapse: collapse;
}

.table th,
.table td {
  padding: 10px 12px;
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

@media (max-width: 860px) {
  .page {
    grid-template-columns: 1fr;
    margin: 0;
  }

  .side {
    position: static;
    padding: 10px;
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

  .panel {
    max-width: none;
  }
}
</style>
