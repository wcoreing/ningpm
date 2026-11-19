<script setup lang="ts">
import { computed, ref, watch } from "vue"
import { getToken } from "../lib/api"

const props = withDefaults(
  defineProps<{
    name?: string
    account?: string
    email?: string
    avatarUrl?: string
    size?: "sm" | "md" | "lg"
  }>(),
  {
    name: "",
    account: "",
    email: "",
    avatarUrl: "",
    size: "md",
  },
)

const imgFailed = ref(false)
const objectUrl = ref("")

const identity = computed(() => props.account || props.email || "")

const initials = computed(() => {
  const raw = (props.name || identity.value || "?").trim()
  if (!raw) return "?"
  const parts = raw.split(/[\s@._-]+/).filter(Boolean)
  if (parts.length >= 2) {
    return (parts[0][0] + parts[1][0]).toUpperCase()
  }
  return raw.slice(0, 2).toUpperCase()
})

const palette = [
  ["#2563eb", "#dbeafe"],
  ["#0891b2", "#cffafe"],
  ["#059669", "#d1fae5"],
  ["#d97706", "#fef3c7"],
  ["#dc2626", "#fee2e2"],
  ["#7c3aed", "#ede9fe"],
  ["#db2777", "#fce7f3"],
]

const colors = computed(() => {
  const key = identity.value || props.name || "u"
  let h = 0
  for (let i = 0; i < key.length; i++) h = (h * 31 + key.charCodeAt(i)) >>> 0
  return palette[h % palette.length]
})

const showImg = computed(() => Boolean(props.avatarUrl) && !imgFailed.value)

async function loadImage() {
  imgFailed.value = false
  if (objectUrl.value) {
    URL.revokeObjectURL(objectUrl.value)
    objectUrl.value = ""
  }
  if (!props.avatarUrl) return
  try {
    const res = await fetch(props.avatarUrl, {
      headers: { Authorization: `Bearer ${getToken()}` },
    })
    if (!res.ok) throw new Error("avatar missing")
    const blob = await res.blob()
    objectUrl.value = URL.createObjectURL(blob)
  } catch {
    imgFailed.value = true
  }
}

watch(
  () => props.avatarUrl,
  () => {
    void loadImage()
  },
  { immediate: true },
)
</script>

<template>
  <span
    class="avatar"
    :class="`avatar--${size}`"
    :style="showImg ? undefined : { color: colors[0], background: colors[1] }"
    :title="name || account || email"
  >
    <img v-if="showImg && objectUrl" :src="objectUrl" alt="" />
    <span v-else>{{ initials }}</span>
  </span>
</template>

<style scoped>
.avatar {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  overflow: hidden;
  border-radius: 999px;
  font-weight: 700;
  line-height: 1;
  user-select: none;
}

.avatar img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.avatar--sm {
  width: 24px;
  height: 24px;
  font-size: 10px;
}

.avatar--md {
  width: 32px;
  height: 32px;
  font-size: 12px;
}

.avatar--lg {
  width: 72px;
  height: 72px;
  font-size: 24px;
}
</style>
