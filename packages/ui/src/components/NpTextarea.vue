<script setup lang="ts">
withDefaults(
  defineProps<{
    modelValue?: string
    placeholder?: string
    disabled?: boolean
    rows?: number
  }>(),
  { modelValue: "", placeholder: "", disabled: false, rows: 4 },
)

const emit = defineEmits<{
  "update:modelValue": [value: string]
}>()
</script>

<template>
  <textarea
    class="np-textarea"
    :rows="rows"
    :value="modelValue"
    :placeholder="placeholder"
    :disabled="disabled"
    @input="emit('update:modelValue', ($event.target as HTMLTextAreaElement).value)"
  />
</template>

<style scoped>
.np-textarea {
  width: 100%;
  padding: 10px 12px;
  border: 1px solid var(--np-border);
  border-radius: var(--np-radius-sm);
  background: var(--np-bg-elevated);
  color: var(--np-text);
  resize: vertical;
  outline: none;
  line-height: 1.55;
  transition:
    border-color var(--np-duration-fast) var(--np-ease),
    box-shadow var(--np-duration-fast) var(--np-ease);
}

.np-textarea::placeholder {
  color: var(--np-text-muted);
}

.np-textarea:hover:not(:disabled) {
  border-color: var(--np-border-strong);
}

.np-textarea:focus {
  border-color: var(--np-accent);
  box-shadow: 0 0 0 3px var(--np-accent-muted);
}

.np-textarea:disabled {
  background: var(--np-bg-sunken);
  cursor: not-allowed;
}
</style>
