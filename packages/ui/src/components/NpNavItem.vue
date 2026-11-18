<script setup lang="ts">
withDefaults(
  defineProps<{
    active?: boolean
    href?: string
  }>(),
  { active: false, href: "#" },
)
</script>

<template>
  <a class="np-nav-item" :class="{ 'np-nav-item--active': active }" :href="href">
    <span class="np-nav-item__dot" aria-hidden="true" />
    <span><slot /></span>
  </a>
</template>

<style scoped>
.np-nav-item {
  display: flex;
  align-items: center;
  gap: 10px;
  height: 38px;
  padding: 0 12px;
  border-radius: var(--np-radius-md);
  color: var(--np-shell-aside-muted);
  text-decoration: none;
  font-size: var(--np-text-sm);
  font-weight: 600;
  transition:
    background var(--np-duration-fast) var(--np-ease),
    color var(--np-duration-fast) var(--np-ease);
}

.np-nav-item__dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: color-mix(in srgb, var(--np-shell-aside-muted) 55%, transparent);
  transition: background var(--np-duration-fast) var(--np-ease);
}

.np-nav-item:hover {
  background: var(--np-shell-aside-hover);
  color: var(--np-shell-aside-text);
}

.np-nav-item--active {
  background: var(--np-shell-aside-active);
  color: #fff;
  box-shadow: inset 3px 0 0 var(--np-accent);
}

.np-nav-item--active .np-nav-item__dot {
  background: var(--np-accent-hover);
}
</style>
