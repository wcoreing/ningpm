<script setup lang="ts">
withDefaults(
  defineProps<{
    variant?: "primary" | "secondary" | "ghost" | "danger"
    size?: "sm" | "md" | "lg"
    disabled?: boolean
    block?: boolean
    type?: "button" | "submit" | "reset"
  }>(),
  {
    variant: "secondary",
    size: "md",
    disabled: false,
    block: false,
    type: "button",
  },
)
</script>

<template>
  <button
    class="np-btn"
    :class="[`np-btn--${variant}`, `np-btn--${size}`, { 'np-btn--block': block }]"
    :type="type"
    :disabled="disabled"
  >
    <slot />
  </button>
</template>

<style scoped>
.np-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  border: 1px solid transparent;
  border-radius: var(--np-radius-sm);
  font-weight: 600;
  cursor: pointer;
  transition:
    background var(--np-duration-fast) var(--np-ease),
    border-color var(--np-duration-fast) var(--np-ease),
    color var(--np-duration-fast) var(--np-ease),
    transform var(--np-duration-fast) var(--np-ease);
  user-select: none;
  white-space: nowrap;
}

.np-btn:active:not(:disabled) {
  transform: translateY(1px);
}

.np-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.np-btn--block {
  width: 100%;
}

.np-btn--sm {
  height: var(--np-control-h-sm);
  padding: 0 10px;
  font-size: var(--np-text-xs);
}

.np-btn--md {
  height: var(--np-control-h-md);
  padding: 0 14px;
  font-size: var(--np-text-sm);
}

.np-btn--lg {
  height: var(--np-control-h-lg);
  padding: 0 18px;
  font-size: var(--np-text-md);
}

.np-btn--primary {
  background: var(--np-accent);
  color: #fff;
}

.np-btn--primary:hover:not(:disabled) {
  background: var(--np-accent-hover);
}

.np-btn--primary:active:not(:disabled) {
  background: var(--np-accent-active);
}

.np-btn--secondary {
  background: var(--np-bg-elevated);
  border-color: var(--np-border);
  color: var(--np-text);
  box-shadow: var(--np-shadow-sm);
}

.np-btn--secondary:hover:not(:disabled) {
  border-color: var(--np-border-strong);
  background: var(--np-bg);
}

.np-btn--ghost {
  background: transparent;
  color: var(--np-text-secondary);
}

.np-btn--ghost:hover:not(:disabled) {
  background: var(--np-accent-muted);
  color: var(--np-accent-ink);
}

.np-btn--danger {
  background: var(--np-danger);
  color: #fff;
}

.np-btn--danger:hover:not(:disabled) {
  filter: brightness(0.95);
}
</style>
