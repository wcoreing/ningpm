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
      aria-label="通知"
      @click.stop="toggle"
    >
      通知{{ unread > 0 ? ` · ${unreadLabel}` : "" }}
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
