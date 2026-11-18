<script setup lang="ts">
withDefaults(
  defineProps<{
    modelValue?: string
    type?: string
    placeholder?: string
    disabled?: boolean
    size?: "sm" | "md" | "lg"
  }>(),
  {
    modelValue: "",
    type: "text",
    placeholder: "",
    disabled: false,
    size: "md",
  },
)

const emit = defineEmits<{
  "update:modelValue": [value: string]
  focus: [event: FocusEvent]
  blur: [event: FocusEvent]
  keydown: [event: KeyboardEvent]
}>()
</script>

<template>
  <input
    class="np-input"
    :class="`np-input--${size}`"
    :type="type"
    :value="modelValue"
    :placeholder="placeholder"
    :disabled="disabled"
    @input="emit('update:modelValue', ($event.target as HTMLInputElement).value)"
    @focus="emit('focus', $event)"
    @blur="emit('blur', $event)"
    @keydown="emit('keydown', $event)"
  />
</template>

<style scoped>
.np-input {
  width: 100%;
  border: 1px solid var(--np-border);
  border-radius: var(--np-radius-sm);
  background: var(--np-bg-elevated);
  color: var(--np-text);
  outline: none;
  transition:
    border-color var(--np-duration-fast) var(--np-ease),
    box-shadow var(--np-duration-fast) var(--np-ease);
}

.np-input::placeholder {
  color: var(--np-text-muted);
}

.np-input:hover:not(:disabled) {
  border-color: var(--np-border-strong);
}

.np-input:focus {
  border-color: var(--np-accent);
  box-shadow: 0 0 0 3px var(--np-accent-muted);
}

.np-input:disabled {
  background: var(--np-bg-sunken);
  cursor: not-allowed;
}

.np-input--sm {
  height: var(--np-control-h-sm);
  padding: 0 10px;
  font-size: var(--np-text-xs);
}

.np-input--md {
  height: var(--np-control-h-md);
  padding: 0 12px;
  font-size: var(--np-text-sm);
}

.np-input--lg {
  height: var(--np-control-h-lg);
  padding: 0 14px;
  font-size: var(--np-text-md);
}
</style>
