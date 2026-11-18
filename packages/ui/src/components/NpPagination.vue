<script setup lang="ts">
import { computed } from "vue"

const props = withDefaults(
  defineProps<{
    total: number
    page?: number
    pageSize?: number
    pageSizes?: number[]
    disabled?: boolean
  }>(),
  {
    page: 1,
    pageSize: 20,
    pageSizes: () => [10, 20, 50],
    disabled: false,
  },
)

const emit = defineEmits<{
  "update:page": [page: number]
  "update:pageSize": [size: number]
  change: [payload: { page: number; pageSize: number }]
}>()

const pageCount = computed(() => Math.max(1, Math.ceil(props.total / props.pageSize) || 1))

const current = computed(() => Math.min(Math.max(1, props.page), pageCount.value))

const from = computed(() => {
  if (props.total <= 0) return 0
  return (current.value - 1) * props.pageSize + 1
})

const to = computed(() => Math.min(props.total, current.value * props.pageSize))

const pages = computed(() => {
  const total = pageCount.value
  const cur = current.value
  if (total <= 7) {
    return Array.from({ length: total }, (_, i) => i + 1)
  }
  const set = new Set<number>([1, total, cur - 1, cur, cur + 1])
  if (cur <= 3) {
    set.add(2)
    set.add(3)
    set.add(4)
  }
  if (cur >= total - 2) {
    set.add(total - 1)
    set.add(total - 2)
    set.add(total - 3)
  }
  return [...set].filter((n) => n >= 1 && n <= total).sort((a, b) => a - b)
})

function go(page: number) {
  if (props.disabled) return
  const next = Math.min(Math.max(1, page), pageCount.value)
  if (next === props.page) return
  emit("update:page", next)
  emit("change", { page: next, pageSize: props.pageSize })
}

function onSizeChange(ev: Event) {
  const size = Number((ev.target as HTMLSelectElement).value)
  if (!size || size === props.pageSize) return
  emit("update:pageSize", size)
  emit("update:page", 1)
  emit("change", { page: 1, pageSize: size })
}
</script>

<template>
  <nav class="np-pager" aria-label="分页">
    <span class="np-pager__total">共 {{ total }} 条</span>
    <span v-if="total > 0" class="np-pager__range">{{ from }}–{{ to }}</span>

    <label class="np-pager__size">
      <select :value="pageSize" :disabled="disabled" @change="onSizeChange">
        <option v-for="s in pageSizes" :key="s" :value="s">{{ s }} 条/页</option>
      </select>
    </label>

    <div class="np-pager__btns">
      <button type="button" class="np-pager__btn" :disabled="disabled || current <= 1" @click="go(current - 1)">
        上一页
      </button>
      <template v-for="(p, idx) in pages" :key="p">
        <span v-if="idx > 0 && p - pages[idx - 1]! > 1" class="np-pager__ellipsis">…</span>
        <button
          type="button"
          class="np-pager__btn"
          :class="{ 'is-on': p === current }"
          :disabled="disabled"
          :aria-current="p === current ? 'page' : undefined"
          @click="go(p)"
        >
          {{ p }}
        </button>
      </template>
      <button
        type="button"
        class="np-pager__btn"
        :disabled="disabled || current >= pageCount"
        @click="go(current + 1)"
      >
        下一页
      </button>
    </div>
  </nav>
</template>

<style scoped>
.np-pager {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px 12px;
  color: var(--np-text-secondary);
  font-size: var(--np-text-xs);
}

.np-pager__total,
.np-pager__range {
  font-variant-numeric: tabular-nums;
}

.np-pager__size select {
  height: var(--np-control-h-sm);
  padding: 0 28px 0 8px;
  border: 1px solid var(--np-border);
  border-radius: var(--np-radius-sm);
  color: var(--np-text);
  font: inherit;
  font-size: var(--np-text-xs);
  cursor: pointer;
  appearance: none;
  background-color: var(--np-bg-elevated);
  background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='12' height='12' viewBox='0 0 12 12' fill='none'%3E%3Cpath d='M3 4.5L6 7.5L9 4.5' stroke='%236b7c93' stroke-width='1.5' stroke-linecap='round' stroke-linejoin='round'/%3E%3C/svg%3E");
  background-repeat: no-repeat;
  background-position: right 8px center;
  background-size: 12px;
}

.np-pager__btns {
  display: inline-flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 4px;
  margin-left: auto;
}

.np-pager__btn {
  min-width: 28px;
  height: 28px;
  padding: 0 8px;
  border: 1px solid var(--np-border);
  border-radius: var(--np-radius-sm);
  background: var(--np-bg-elevated);
  color: var(--np-text-secondary);
  font: inherit;
  font-size: var(--np-text-xs);
  font-weight: 600;
  cursor: pointer;
}

.np-pager__btn:hover:not(:disabled) {
  color: var(--np-accent);
  border-color: var(--np-border-strong);
}

.np-pager__btn.is-on {
  background: var(--np-accent);
  border-color: var(--np-accent);
  color: #fff;
}

.np-pager__btn:disabled {
  opacity: 0.45;
  cursor: not-allowed;
}

.np-pager__ellipsis {
  padding: 0 2px;
  color: var(--np-text-muted);
}
</style>
