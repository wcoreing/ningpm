<script setup lang="ts">
withDefaults(
  defineProps<{
    modelValue?: string | number
    disabled?: boolean
    size?: "sm" | "md" | "lg"
  }>(),
  { modelValue: "", disabled: false, size: "md" },
)

const emit = defineEmits<{
  "update:modelValue": [value: string]
}>()
</script>

<template>
  <select
    class="np-select"
    :class="`np-select--${size}`"
    :value="modelValue"
    :disabled="disabled"
    @change="emit('update:modelValue', ($event.target as HTMLSelectElement).value)"
  >
    <slot />
  </select>
</template>

<style scoped>
.np-select {
  width: 100%;
  border: 1px solid var(--np-border);
  border-radius: var(--np-radius-sm);
  color: var(--np-text);
  outline: none;
  cursor: pointer;
  appearance: none;
  background-color: var(--np-bg-elevated);
  background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='12' height='12' viewBox='0 0 12 12' fill='none'%3E%3Cpath d='M3 4.5L6 7.5L9 4.5' stroke='%236b7c93' stroke-width='1.5' stroke-linecap='round' stroke-linejoin='round'/%3E%3C/svg%3E");
  background-repeat: no-repeat;
  background-position: right 8px center;
  background-size: 12px;
  transition:
    border-color var(--np-duration-fast) var(--np-ease),
    box-shadow var(--np-duration-fast) var(--np-ease);
}

.np-select:hover:not(:disabled) {
  border-color: var(--np-border-strong);
}

.np-select:focus {
  border-color: var(--np-accent);
  box-shadow: 0 0 0 3px var(--np-accent-muted);
}

.np-select:disabled {
  background-color: var(--np-bg-sunken);
  cursor: not-allowed;
  opacity: 0.7;
}

.np-select--sm {
  height: var(--np-control-h-sm);
  padding: 0 28px 0 8px;
  font-size: var(--np-text-xs);
  background-position: right 8px center;
}

.np-select--md {
  height: var(--np-control-h-md);
  padding: 0 30px 0 10px;
  font-size: var(--np-text-sm);
  background-position: right 10px center;
}

.np-select--lg {
  height: var(--np-control-h-lg);
  padding: 0 32px 0 12px;
  font-size: var(--np-text-md);
  background-position: right 12px center;
}
</style>
