<script setup lang="ts">
withDefaults(
  defineProps<{
    tone?: "neutral" | "accent" | "success" | "warning" | "danger"
  }>(),
  { tone: "neutral" },
)
</script>

<template>
  <span class="np-badge" :class="`np-badge--${tone}`">
    <slot />
  </span>
</template>

<style scoped>
.np-badge {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  height: 22px;
  padding: 0 8px;
  border-radius: 999px;
  font-size: var(--np-text-xs);
  font-weight: 600;
  line-height: 1;
}

.np-badge--neutral {
  background: var(--np-bg-sunken);
  color: var(--np-text-secondary);
}

.np-badge--accent {
  background: var(--np-accent-muted);
  color: var(--np-accent-ink);
}

.np-badge--success {
  background: color-mix(in srgb, var(--np-success) 14%, transparent);
  color: var(--np-success);
}

.np-badge--warning {
  background: color-mix(in srgb, var(--np-warning) 16%, transparent);
  color: #7a5c10;
}

.np-badge--danger {
  background: var(--np-danger-muted);
  color: var(--np-danger);
}
</style>
