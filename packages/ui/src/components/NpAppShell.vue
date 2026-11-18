<script setup lang="ts">
defineProps<{
  brand?: string
  version?: string
}>()
</script>

<template>
  <div class="np-shell">
    <aside class="np-shell__aside">
      <div class="np-shell__brand">
        <div class="np-shell__mark" aria-hidden="true" />
        <div class="np-shell__brand-text">
          <strong>{{ brand || "ningpm" }}</strong>
          <span v-if="version" class="np-shell__version">v{{ version }}</span>
        </div>
      </div>
      <nav class="np-shell__nav">
        <slot name="nav" />
      </nav>
      <div v-if="$slots['aside-foot']" class="np-shell__aside-foot">
        <slot name="aside-foot" />
      </div>
    </aside>

    <div class="np-shell__main">
      <header v-if="$slots.header" class="np-shell__header">
        <slot name="header" />
      </header>
      <main class="np-shell__content">
        <slot />
      </main>
    </div>
  </div>
</template>

<style scoped>
.np-shell {
  display: grid;
  grid-template-columns: 232px minmax(0, 1fr);
  height: 100vh;
  overflow: hidden;
}

.np-shell__aside {
  display: flex;
  flex-direction: column;
  gap: 18px;
  min-height: 0;
  height: 100%;
  padding: 22px 14px;
  border-right: 1px solid var(--np-shell-aside-border);
  background:
    linear-gradient(180deg, var(--np-shell-aside-bg-soft) 0%, var(--np-shell-aside-bg) 55%),
    var(--np-shell-aside-bg);
  color: var(--np-shell-aside-text);
  overflow: hidden;
}

.np-shell__brand {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 6px 10px;
  flex-shrink: 0;
}

.np-shell__mark {
  width: 30px;
  height: 30px;
  border-radius: 9px;
  background:
    linear-gradient(145deg, var(--np-accent-hover), var(--np-accent-active) 72%),
    var(--np-accent);
  box-shadow:
    inset 0 0 0 1px rgb(255 255 255 / 22%),
    0 6px 16px rgb(64 158 255 / 28%);
}

.np-shell__brand-text {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}

.np-shell__brand-text strong {
  font-family: var(--np-font-display);
  font-size: var(--np-text-lg);
  font-weight: 700;
  letter-spacing: -0.04em;
  line-height: 1.1;
  color: var(--np-shell-aside-text);
}

.np-shell__version {
  font-family: var(--np-font-mono);
  font-size: 11px;
  color: var(--np-shell-aside-muted);
}

.np-shell__nav {
  display: flex;
  flex-direction: column;
  gap: 4px;
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  overscroll-behavior: contain;
  padding-right: 2px;
}

.np-shell__aside-foot {
  flex-shrink: 0;
  padding: 10px 10px 0;
  border-top: 1px solid var(--np-shell-aside-border);
  color: var(--np-shell-aside-muted);
  font-size: var(--np-text-xs);
}

.np-shell__main {
  display: flex;
  flex-direction: column;
  min-width: 0;
  min-height: 0;
  height: 100%;
  overflow: hidden;
}

.np-shell__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  flex-shrink: 0;
  min-height: 68px;
  padding: 14px 28px;
  border-bottom: 1px solid color-mix(in srgb, var(--np-border) 80%, transparent);
  background: color-mix(in srgb, var(--np-bg-elevated) 82%, transparent);
  backdrop-filter: blur(10px);
  -webkit-backdrop-filter: blur(10px);
}

.np-shell__content {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  overscroll-behavior: contain;
  padding: 28px;
  background: transparent;
}

@media (max-width: 860px) {
  .np-shell {
    grid-template-columns: 1fr;
    height: auto;
    min-height: 100vh;
    overflow: visible;
  }

  .np-shell__aside {
    height: auto;
    overflow: visible;
  }

  .np-shell__nav {
    overflow: visible;
  }

  .np-shell__aside {
    border-right: none;
    border-bottom: 1px solid var(--np-shell-aside-border);
  }

  .np-shell__main {
    height: auto;
    overflow: visible;
  }

  .np-shell__content {
    overflow: visible;
    padding: 20px 16px;
  }
}
</style>
