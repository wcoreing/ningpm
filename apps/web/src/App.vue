<script setup lang="ts">
import { computed, onMounted } from "vue"
import { useRoute, useRouter } from "vue-router"
import { NpAppShell, NpBadge, NpButton, NpNavItem, NpToastHost } from "@ningpm/ui"
import NotificationBell from "./components/NotificationBell.vue"
import { useAuthStore } from "./stores/auth"
import { usePageHeaderStore } from "./stores/pageHeader"
import { APP_VERSION } from "./version"

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()
const pageHeader = usePageHeaderStore()

const isLogin = computed(() => route.name === "login")

const headerTitle = computed(
  () => pageHeader.override?.title || String(route.meta.title || "工作台"),
)
const headerBadge = computed(() => pageHeader.override?.badge || "")
const headerSubtitle = computed(
  () => pageHeader.override?.subtitle || String(route.meta.subtitle || ""),
)

onMounted(() => {
  void auth.fetchMe()
})

function logout() {
  auth.logout()
  router.push({ name: "login" })
}
</script>

<template>
  <NpToastHost />
  <RouterView v-if="isLogin" />

  <NpAppShell v-else brand="ningpm" :version="APP_VERSION">
    <template #nav>
      <NpNavItem href="#/" :active="route.path === '/'">工作台</NpNavItem>
      <NpNavItem href="#/projects" :active="route.path.startsWith('/projects')">项目</NpNavItem>
    </template>

    <template #header>
      <div class="header-title">
        <div class="header-row">
          <NpBadge v-if="headerBadge" tone="accent">{{ headerBadge }}</NpBadge>
          <h1>{{ headerTitle }}</h1>
        </div>
        <p v-if="headerSubtitle">{{ headerSubtitle }}</p>
      </div>
      <div class="header-actions">
        <NotificationBell />
        <NpButton
          size="sm"
          variant="ghost"
          :class="{ 'is-on': route.name === 'account' }"
          @click="router.push({ name: 'account' })"
        >
          {{ auth.user?.name || "账号" }}
        </NpButton>
        <NpButton size="sm" variant="ghost" @click="logout">退出</NpButton>
      </div>
    </template>

    <RouterView />
  </NpAppShell>
</template>

<style scoped>
.header-row {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
}

.header-title {
  min-width: 0;
}

.header-title h1 {
  margin: 0;
  font-family: var(--np-font-display);
  font-size: var(--np-text-xl);
  font-weight: 700;
  letter-spacing: -0.04em;
}

.header-title p {
  margin: 4px 0 0;
  color: var(--np-text-secondary);
  font-size: var(--np-text-sm);
}

.header-actions {
  display: flex;
  align-items: center;
  gap: 2px;
  flex-shrink: 0;
  padding: 4px;
  border: 1px solid var(--np-border);
  border-radius: 999px;
  background: color-mix(in srgb, var(--np-bg-elevated) 88%, transparent);
  box-shadow: var(--np-shadow-sm);
}

.header-actions :deep(.is-on) {
  color: var(--np-accent-ink);
  background: var(--np-accent-muted);
}
</style>
