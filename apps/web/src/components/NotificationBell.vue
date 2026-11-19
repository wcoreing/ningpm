<script setup lang="ts">
import { onMounted, onUnmounted, ref } from "vue"
import { storeToRefs } from "pinia"
import { useRouter } from "vue-router"
import { NpButton } from "@ningpm/ui"
import type { AppNotification } from "../types"
import { useNotificationsStore } from "../stores/notifications"

const router = useRouter()
const notif = useNotificationsStore()
const { items, unread, loading, unreadLabel } = storeToRefs(notif)

const open = ref(false)
let pollTimer: ReturnType<typeof setInterval> | null = null

function kindText(n: AppNotification) {
  switch (n.kind) {
    case "assigned":
      return "把事项指派给你"
    case "commented":
      return "评论了事项"
    default:
      return "更新了事项"
  }
}

function timeLabel(iso: string) {
  const t = new Date(iso).getTime()
  if (Number.isNaN(t)) return ""
  const diff = Date.now() - t
  const m = Math.floor(diff / 60000)
  if (m < 1) return "刚刚"
  if (m < 60) return `${m} 分钟前`
  const h = Math.floor(m / 60)
  if (h < 24) return `${h} 小时前`
  const d = Math.floor(h / 24)
  if (d < 7) return `${d} 天前`
  return new Date(iso).toLocaleDateString("zh-CN")
}

async function toggle() {
  open.value = !open.value
  if (open.value) await notif.fetchList()
}

async function openItem(n: AppNotification) {
  if (!n.readAt) {
    try {
      await notif.markRead(n.id)
    } catch {
      /* still navigate */
    }
  }
  open.value = false
  await router.push({
    name: "project-detail",
    params: { id: n.projectId },
    query: { issueId: n.issueId },
  })
}

function onDocClick(e: MouseEvent) {
  const root = (e.target as HTMLElement | null)?.closest?.(".notif-root")
  if (!root) open.value = false
}

onMounted(() => {
  void notif.fetchCount()
  pollTimer = setInterval(() => void notif.fetchCount(), 30000)
  document.addEventListener("click", onDocClick)
})

onUnmounted(() => {
  if (pollTimer) clearInterval(pollTimer)
  document.removeEventListener("click", onDocClick)
})
</script>

<template>
  <div class="notif-root">
    <NpButton
      size="sm"
      variant="ghost"
      class="bell"
      :aria-expanded="open"
      :aria-label="unread > 0 ? `通知 ${unreadLabel}` : '通知'"
      @click.stop="toggle"
    >
      <svg class="bell-icon" viewBox="0 0 16 16" width="16" height="16" aria-hidden="true">
        <path
          fill="currentColor"
          d="M8 1.4a3.4 3.4 0 0 0-3.4 3.4v1.3c0 .7-.2 1.4-.6 2L3.2 9.4A1 1 0 0 0 4 11h8a1 1 0 0 0 .8-1.6l-.8-1.3c-.4-.6-.6-1.3-.6-2V4.8A3.4 3.4 0 0 0 8 1.4zm0 12.2a1.6 1.6 0 0 0 1.5-1H6.5a1.6 1.6 0 0 0 1.5 1z"
        />
      </svg>
      <span v-if="unread > 0" class="bell-badge">{{ unreadLabel }}</span>
    </NpButton>

    <div v-if="open" class="panel" role="dialog" aria-label="通知列表" @click.stop>
      <header class="panel-head">
        <strong>通知</strong>
        <NpButton size="sm" variant="ghost" :disabled="!unread" @click="notif.markAll()">全部已读</NpButton>
      </header>
      <p v-if="loading" class="muted">加载中…</p>
      <p v-else-if="!items.length" class="muted empty">暂无通知</p>
      <ul v-else class="list">
        <li v-for="n in items" :key="n.id">
          <button type="button" class="item" :class="{ unread: !n.readAt }" @click="openItem(n)">
            <div class="item-top">
              <span class="actor">{{ n.actorName }}</span>
              <span class="when">{{ timeLabel(n.createdAt) }}</span>
            </div>
            <div class="item-main">
              {{ kindText(n) }}
              <span class="key">{{ n.issueKey }}</span>
              {{ n.issueTitle }}
            </div>
            <div v-if="n.body" class="snippet">{{ n.body }}</div>
            <div class="project">{{ n.projectName }}</div>
          </button>
        </li>
      </ul>
    </div>
  </div>
</template>

<style scoped>
.notif-root {
  position: relative;
}

.bell {
  position: relative;
  min-width: 32px;
  padding-inline: 8px;
}

.bell :deep(.np-button__label),
.bell {
  display: inline-flex;
  align-items: center;
  justify-content: center;
}

.bell-icon {
  display: block;
}

.bell-badge {
  position: absolute;
  top: 0;
  right: 0;
  min-width: 16px;
  height: 16px;
  padding: 0 4px;
  border-radius: 999px;
  background: var(--np-danger);
  color: #fff;
  font-size: 10px;
  font-weight: 700;
  line-height: 16px;
  text-align: center;
  font-variant-numeric: tabular-nums;
  transform: translate(20%, -10%);
  pointer-events: none;
}

.panel {
  position: absolute;
  top: calc(100% + 8px);
  right: 0;
  z-index: 40;
  width: min(360px, calc(100vw - 32px));
  max-height: 420px;
  overflow: auto;
  border: 1px solid var(--np-border);
  border-radius: var(--np-radius-lg);
  background: var(--np-bg-elevated);
  box-shadow: var(--np-shadow-md);
}

.panel-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding: 10px 12px;
  border-bottom: 1px solid var(--np-border);
  position: sticky;
  top: 0;
  background: var(--np-bg-elevated);
}

.list {
  list-style: none;
  margin: 0;
  padding: 6px;
}

.item {
  display: flex;
  flex-direction: column;
  gap: 4px;
  width: 100%;
  padding: 10px;
  border: none;
  border-radius: var(--np-radius-md);
  background: transparent;
  text-align: left;
  cursor: pointer;
  color: inherit;
}

.item:hover {
  background: var(--np-accent-muted);
}

.item.unread {
  background: color-mix(in srgb, var(--np-accent-muted) 55%, transparent);
}

.item-top {
  display: flex;
  justify-content: space-between;
  gap: 8px;
  font-size: var(--np-text-xs);
  color: var(--np-text-muted);
}

.actor {
  font-weight: 600;
  color: var(--np-text);
}

.item-main {
  font-size: var(--np-text-sm);
  line-height: 1.4;
}

.key {
  font-family: var(--np-font-mono);
  font-size: var(--np-text-xs);
  color: var(--np-accent);
  margin: 0 4px;
}

.snippet {
  font-size: var(--np-text-xs);
  color: var(--np-text-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.project {
  font-size: var(--np-text-xs);
  color: var(--np-text-muted);
}

.muted {
  margin: 0;
  padding: 16px;
  color: var(--np-text-muted);
  font-size: var(--np-text-sm);
}

.empty {
  text-align: center;
}
</style>
