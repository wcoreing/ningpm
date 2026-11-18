<script setup lang="ts">
export type NpTabItem = { value: string; label: string }

withDefaults(
  defineProps<{
    modelValue: string
    items: NpTabItem[]
    /** pills = 分段按钮；line = 页面级视图切换 */
    variant?: "pills" | "line"
  }>(),
  { variant: "pills" },
)

const emit = defineEmits<{
  "update:modelValue": [value: string]
}>()
</script>

<template>
  <div class="np-tabs" :class="`np-tabs--${variant}`" role="tablist">
    <button
      v-for="item in items"
      :key="item.value"
      type="button"
      class="np-tabs__item"
      :class="{ 'np-tabs__item--active': modelValue === item.value }"
      role="tab"
      :aria-selected="modelValue === item.value"
      @click="emit('update:modelValue', item.value)"
    >
      {{ item.label }}
    </button>
  </div>
</template>

<style scoped>
.np-tabs {
  display: inline-flex;
  gap: 4px;
}

.np-tabs--pills {
  padding: 4px;
  border-radius: var(--np-radius-md);
  background: var(--np-bg-sunken);
}

.np-tabs--line {
  gap: 2px;
  width: 100%;
  border-bottom: 1px solid var(--np-border);
}

.np-tabs__item {
  border: none;
  background: transparent;
  color: var(--np-text-secondary);
  height: 32px;
  padding: 0 12px;
  border-radius: var(--np-radius-sm);
  font-size: var(--np-text-sm);
  font-weight: 600;
  cursor: pointer;
  transition:
    background var(--np-duration-fast) var(--np-ease),
    color var(--np-duration-fast) var(--np-ease),
    box-shadow var(--np-duration-fast) var(--np-ease);
}

.np-tabs--line .np-tabs__item {
  height: 40px;
  padding: 0 14px;
  border-radius: 0;
  border-bottom: 2px solid transparent;
  margin-bottom: -1px;
}

.np-tabs__item:hover {
  color: var(--np-text);
}

.np-tabs--pills .np-tabs__item--active {
  background: var(--np-bg-elevated);
  color: var(--np-accent-ink);
  box-shadow: var(--np-shadow-sm);
}

.np-tabs--line .np-tabs__item--active {
  color: var(--np-accent-ink);
  border-bottom-color: var(--np-accent);
}
</style>
