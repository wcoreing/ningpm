<script setup lang="ts">
defineProps<{
  label?: string
  hint?: string
  error?: string
}>()
</script>

<template>
  <label class="np-field">
    <span v-if="label" class="np-field__label">{{ label }}</span>
    <slot />
    <span v-if="error" class="np-field__error">{{ error }}</span>
    <span v-else-if="hint" class="np-field__hint">{{ hint }}</span>
  </label>
</template>

<style scoped>
.np-field {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.np-field__label {
  font-size: var(--np-text-sm);
  font-weight: 600;
  color: var(--np-text);
}

.np-field__hint,
.np-field__error {
  font-size: var(--np-text-xs);
}

.np-field__hint {
  color: var(--np-text-muted);
}

.np-field__error {
  color: var(--np-danger);
}
</style>
