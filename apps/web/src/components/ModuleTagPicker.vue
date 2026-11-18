<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref } from "vue"
import { api } from "../lib/api"
import type { Module } from "../types"

const props = withDefaults(
  defineProps<{
    projectId: string
    modules: Module[]
    modelValue: string
    canWrite?: boolean
    disabled?: boolean
  }>(),
  { canWrite: false, disabled: false },
)

const emit = defineEmits<{
  "update:modelValue": [value: string]
  "update:modules": [modules: Module[]]
}>()

const open = ref(false)
const draft = ref("")
const busy = ref(false)
const error = ref("")
const root = ref<HTMLElement | null>(null)
const inputEl = ref<HTMLInputElement | null>(null)

const selected = computed(() => props.modules.find((m) => m.id === props.modelValue) || null)

const filtered = computed(() => {
  const q = draft.value.trim().toLowerCase()
  if (!q) return props.modules
  return props.modules.filter((m) => m.name.toLowerCase().includes(q))
})

const canCreate = computed(() => {
  const q = draft.value.trim()
  if (!q || !props.canWrite) return false
  return !props.modules.some((m) => m.name.toLowerCase() === q.toLowerCase())
})

function focusInput() {
  if (props.disabled) return
  open.value = true
  void nextTick(() => inputEl.value?.focus())
}

function clearSelected(ev?: Event) {
  ev?.stopPropagation()
  if (props.disabled) return
  emit("update:modelValue", "")
  focusInput()
}

function pick(md: Module) {
  if (props.disabled) return
  emit("update:modelValue", md.id)
  draft.value = ""
  open.value = false
}

async function ensureTag(name: string) {
  const trimmed = name.trim()
  if (!trimmed || props.disabled || !props.canWrite) return
  busy.value = true
  error.value = ""
  try {
    const md = await api<Module>(`/api/projects/${props.projectId}/modules`, {
      method: "POST",
      body: JSON.stringify({ name: trimmed }),
    })
    const exists = props.modules.some((m) => m.id === md.id)
    const next = exists ? props.modules.map((m) => (m.id === md.id ? md : m)) : [...props.modules, md]
    emit("update:modules", next)
    emit("update:modelValue", md.id)
    draft.value = ""
    open.value = false
  } catch (e) {
    error.value = e instanceof Error ? e.message : "创建失败"
  } finally {
    busy.value = false
  }
}

async function deleteTag(md: Module, ev: Event) {
  ev.stopPropagation()
  if (!props.canWrite || props.disabled) return
  const tip =
    md.issueCount > 0
      ? `删除标签「${md.name}」？已挂在 ${md.issueCount} 个事项上的归属会清空。`
      : `删除标签「${md.name}」？`
  if (!window.confirm(tip)) return
  busy.value = true
  error.value = ""
  try {
    await api(`/api/projects/${props.projectId}/modules/${md.id}`, { method: "DELETE" })
    emit(
      "update:modules",
      props.modules.filter((m) => m.id !== md.id),
    )
    if (props.modelValue === md.id) emit("update:modelValue", "")
  } catch (e) {
    error.value = e instanceof Error ? e.message : "删除失败"
  } finally {
    busy.value = false
  }
}

function onKeydown(ev: KeyboardEvent) {
  if (ev.key === "Enter") {
    ev.preventDefault()
    if (canCreate.value) {
      void ensureTag(draft.value)
      return
    }
    if (filtered.value[0]) pick(filtered.value[0])
  } else if (ev.key === "Backspace" && !draft.value && selected.value) {
    clearSelected()
  } else if (ev.key === "Escape") {
    open.value = false
  }
}

function onDocPointer(ev: PointerEvent) {
  if (!root.value?.contains(ev.target as Node)) open.value = false
}

onMounted(() => document.addEventListener("pointerdown", onDocPointer))
onBeforeUnmount(() => document.removeEventListener("pointerdown", onDocPointer))
</script>

<template>
  <div ref="root" class="el-tag-select" :class="{ 'is-disabled': disabled, 'is-open': open }">
    <div class="el-tag-select__wrap" @click="focusInput">
      <span v-if="selected" class="el-tag">
        {{ selected.name }}
        <button
          v-if="!disabled"
          type="button"
          class="el-tag__close"
          aria-label="移除"
          @click="clearSelected($event)"
        >
          ×
        </button>
      </span>
      <input
        ref="inputEl"
        v-model="draft"
        class="el-tag-select__input"
        type="text"
        :disabled="disabled || busy"
        :placeholder="selected ? '' : '选择或创建标签'"
        @focus="open = true"
        @keydown="onKeydown"
      />
    </div>

    <div v-if="open && !disabled" class="el-tag-select__dropdown" role="listbox">
      <button
        v-if="canCreate"
        type="button"
        class="el-tag-select__option el-tag-select__option--create"
        @click="ensureTag(draft)"
      >
        创建「{{ draft.trim() }}」
      </button>
      <button
        v-for="md in filtered"
        :key="md.id"
        type="button"
        class="el-tag-select__option"
        :class="{ 'is-active': md.id === modelValue }"
        @click="pick(md)"
      >
        <span>{{ md.name }}</span>
        <span
          v-if="canWrite"
          class="el-tag-select__delete"
          title="删除标签"
          @click="deleteTag(md, $event)"
        >删除</span>
      </button>
      <p v-if="!filtered.length && !canCreate" class="el-tag-select__empty">无匹配标签</p>
    </div>

    <p v-if="error" class="error">{{ error }}</p>
  </div>
</template>

<style scoped>
.el-tag-select {
  position: relative;
  width: 100%;
}

.el-tag-select__wrap {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 6px;
  min-height: var(--np-control-h-md);
  padding: 4px 8px;
  border: 1px solid var(--np-border);
  border-radius: var(--np-radius-sm);
  background: var(--np-bg-elevated);
  cursor: text;
  transition:
    border-color var(--np-duration-fast) var(--np-ease),
    box-shadow var(--np-duration-fast) var(--np-ease);
}

.is-open .el-tag-select__wrap,
.el-tag-select__wrap:focus-within {
  border-color: var(--np-accent);
  box-shadow: 0 0 0 3px var(--np-accent-muted);
}

.el-tag {
  display: inline-flex;
  align-items: center;
  gap: 2px;
  max-width: 100%;
  height: 24px;
  padding: 0 6px 0 8px;
  border-radius: var(--np-radius-sm);
  background: color-mix(in srgb, var(--np-accent-muted) 70%, var(--np-bg-sunken));
  color: var(--np-accent-ink);
  font-size: var(--np-text-xs);
  font-weight: 600;
  line-height: 1;
}

.el-tag__close {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 14px;
  height: 14px;
  border: none;
  border-radius: 50%;
  background: transparent;
  color: var(--np-text-muted);
  font-size: 12px;
  line-height: 1;
  cursor: pointer;
}

.el-tag__close:hover {
  background: color-mix(in srgb, var(--np-text) 12%, transparent);
  color: var(--np-text);
}

.el-tag-select__input {
  flex: 1;
  min-width: 7rem;
  height: 24px;
  border: none;
  outline: none;
  background: transparent;
  color: var(--np-text);
  font: inherit;
  font-size: var(--np-text-sm);
}

.el-tag-select__dropdown {
  position: absolute;
  z-index: 30;
  top: calc(100% + 4px);
  left: 0;
  right: 0;
  max-height: min(180px, 30vh);
  overflow: auto;
  padding: 6px;
  border: 1px solid var(--np-border);
  border-radius: var(--np-radius-md);
  background: var(--np-bg-elevated);
  box-shadow: var(--np-shadow-md);
}

.el-tag-select__option {
  display: flex;
  width: 100%;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  min-height: 32px;
  padding: 0 10px;
  border: none;
  border-radius: var(--np-radius-sm);
  background: transparent;
  color: var(--np-text);
  font: inherit;
  font-size: var(--np-text-sm);
  text-align: left;
  cursor: pointer;
}

.el-tag-select__option:hover,
.el-tag-select__option.is-active {
  background: var(--np-bg-sunken);
}

.el-tag-select__option--create {
  color: var(--np-accent-ink);
  font-weight: 600;
}

.el-tag-select__delete {
  flex-shrink: 0;
  color: var(--np-text-muted);
  font-size: var(--np-text-xs);
  opacity: 0;
}

.el-tag-select__option:hover .el-tag-select__delete {
  opacity: 1;
}

.el-tag-select__delete:hover {
  color: var(--np-danger);
}

.el-tag-select__empty {
  margin: 0;
  padding: 10px;
  color: var(--np-text-muted);
  font-size: var(--np-text-sm);
  text-align: center;
}

.error {
  margin: 6px 0 0;
  color: var(--np-danger);
  font-size: var(--np-text-xs);
}

.is-disabled {
  opacity: 0.65;
  pointer-events: none;
}
</style>
