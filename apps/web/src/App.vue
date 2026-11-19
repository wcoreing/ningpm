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
      <NpNavItem
        v-if="auth.isAdmin"
        href="#/users"
        :active="route.path.startsWith('/users')"
      >
        账号管理
      </NpNavItem>
    </template>

    <template #header>
      <div class="header-title">
        <div class="header-row">
          <NpBadge v-if="headerBadge" tone="accent">{{ headerBadge }}</NpBadge>
          <h1>{{ headerTitle }}</h1>
        </div>
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
  gap: 8px;
  min-width: 0;
}

.header-title {
  min-width: 0;
}

.header-title h1 {
  margin: 0;
  font-family: var(--np-font-display);
  font-size: var(--np-text-md);
  font-weight: 700;
  letter-spacing: -0.03em;
  line-height: 1.2;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.header-actions {
  display: flex;
  align-items: center;
  gap: 2px;
  flex-shrink: 0;
  margin: 0;
  padding: 0;
  border: none;
  background: transparent;
}

.header-actions :deep(.is-on) {
  color: var(--np-accent-ink);
  background: var(--np-accent-muted);
}
</style>
