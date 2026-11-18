<script setup lang="ts">
withDefaults(
  defineProps<{
    padded?: boolean
    interactive?: boolean
  }>(),
  { padded: true, interactive: false },
)
</script>

<template>
  <div
    class="np-surface"
    :class="{ 'np-surface--padded': padded, 'np-surface--interactive': interactive }"
  >
    <slot />
  </div>
</template>

<style scoped>
.np-surface {
  background: color-mix(in srgb, var(--np-bg-elevated) 88%, transparent);
  border: 1px solid var(--np-border);
  border-radius: var(--np-radius-lg);
  backdrop-filter: blur(10px);
  box-shadow: var(--np-shadow-sm);
}

.np-surface--padded {
  padding: 20px;
}

.np-surface--interactive {
  cursor: pointer;
  transition:
    border-color var(--np-duration-fast) var(--np-ease),
    box-shadow var(--np-duration) var(--np-ease),
    transform var(--np-duration) var(--np-ease);
}

.np-surface--interactive:hover {
  border-color: var(--np-border-strong);
  box-shadow: var(--np-shadow-md);
  transform: translateY(-1px);
}
</style>
