<script setup lang="ts">
import { ref } from "vue"
import { useRouter } from "vue-router"
import { NpButton, NpField, NpInput, NpSurface } from "@ningpm/ui"
import { useAuthStore } from "../stores/auth"

const auth = useAuthStore()
const router = useRouter()
const email = ref("admin@ningpm.local")
const password = ref("admin123")
const error = ref("")
const loading = ref(false)

async function onSubmit() {
  error.value = ""
  loading.value = true
  try {
    await auth.login(email.value.trim(), password.value)
    router.replace("/")
  } catch (e) {
    error.value = e instanceof Error ? e.message : "登录失败"
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="login">
    <div class="login-panel">
      <div class="login-aside" aria-hidden="true">
        <div class="aside-mark" />
        <p class="aside-brand">ningpm</p>
        <p class="aside-copy">把需求、任务与缺陷收进同一条执行流。</p>
      </div>
      <NpSurface class="login-card">
        <div class="brand">
          <div class="mark" />
          <div>
            <h1>欢迎回来</h1>
          </div>
        </div>

        <form class="form" @submit.prevent="onSubmit">
          <NpField label="邮箱">
            <NpInput v-model="email" type="email" placeholder="admin@ningpm.local" />
          </NpField>
          <NpField label="密码">
            <NpInput v-model="password" type="password" placeholder="密码" />
          </NpField>
          <p v-if="error" class="error">{{ error }}</p>
          <NpButton variant="primary" block type="submit" :disabled="loading">
            {{ loading ? "登录中…" : "登录" }}
          </NpButton>
        </form>

        <p class="hint">演示账号：admin@ningpm.local / admin123</p>
      </NpSurface>
    </div>
  </div>
</template>

<style scoped>
.login {
  min-height: 100vh;
  display: grid;
  place-items: center;
  padding: 28px 20px;
}

.login-panel {
  width: min(860px, 100%);
  display: grid;
  grid-template-columns: minmax(0, 0.92fr) minmax(0, 1.08fr);
  gap: 0;
  overflow: hidden;
  border-radius: 16px;
  border: 1px solid var(--np-border);
  box-shadow: var(--np-shadow-md);
  background: var(--np-bg-elevated);
}

.login-aside {
  display: flex;
  flex-direction: column;
  justify-content: flex-end;
  gap: 12px;
  padding: 28px 24px;
  background:
    radial-gradient(420px 240px at 20% 0%, rgb(64 158 255 / 35%), transparent 60%),
    linear-gradient(165deg, #243447 0%, #1e293b 55%, #162032 100%);
  color: #f1f5f9;
  min-height: 360px;
}

.aside-mark {
  width: 36px;
  height: 36px;
  border-radius: 10px;
  margin-bottom: auto;
  background: linear-gradient(145deg, var(--np-accent-hover), var(--np-accent-active));
  box-shadow: inset 0 0 0 1px rgb(255 255 255 / 22%);
}

.aside-brand {
  margin: 0;
  font-family: var(--np-font-display);
  font-size: 28px;
  font-weight: 700;
  letter-spacing: -0.04em;
}

.aside-copy {
  margin: 0;
  max-width: 16rem;
  color: #94a3b8;
  font-size: var(--np-text-sm);
  line-height: 1.55;
}

.login-card {
  width: 100%;
  border: none;
  border-radius: 0;
  box-shadow: none;
  background: #fff;
}

.brand {
  display: flex;
  gap: 14px;
  align-items: center;
  margin-bottom: 22px;
}

.mark {
  width: 40px;
  height: 40px;
  border-radius: 12px;
  background: linear-gradient(145deg, var(--np-accent-hover), var(--np-accent-active));
  box-shadow: inset 0 0 0 1px rgb(255 255 255 / 22%);
}

.brand h1 {
  margin: 0;
  font-family: var(--np-font-display);
  font-size: 26px;
  letter-spacing: -0.04em;
}

.brand p {
  margin: 4px 0 0;
  color: var(--np-text-secondary);
  font-size: var(--np-text-sm);
}

.form {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.error {
  margin: 0;
  color: var(--np-danger);
  font-size: var(--np-text-sm);
}

.hint {
  margin: 16px 0 0;
  color: var(--np-text-muted);
  font-size: var(--np-text-xs);
}

@media (max-width: 720px) {
  .login-panel {
    grid-template-columns: 1fr;
  }

  .login-aside {
    min-height: 160px;
  }
}
</style>
