<script setup lang="ts">
import { watch } from "vue"

const props = withDefaults(
  defineProps<{
    open: boolean
    title?: string
    width?: string
  }>(),
  { title: "", width: "480px" },
)

const emit = defineEmits<{
  close: []
}>()

watch(
  () => props.open,
  (open) => {
    document.body.style.overflow = open ? "hidden" : ""
  },
)
</script>

<template>
  <Teleport to="body">
    <div v-if="open" class="np-drawer">
      <div class="np-drawer__mask" @click="emit('close')" />
      <aside class="np-drawer__panel" :style="{ width }" role="dialog" aria-modal="true">
        <header class="np-drawer__head">
          <div class="np-drawer__titles">
            <p v-if="$slots.eyebrow" class="np-drawer__eyebrow"><slot name="eyebrow" /></p>
            <h3>{{ title }}</h3>
          </div>
          <button class="np-drawer__x" type="button" aria-label="关闭" @click="emit('close')">×</button>
        </header>
        <div class="np-drawer__body">
          <slot />
        </div>
        <footer v-if="$slots.footer" class="np-drawer__foot">
          <slot name="footer" />
        </footer>
      </aside>
    </div>
  </Teleport>
</template>

<style scoped>
.np-drawer {
  position: fixed;
  inset: 0;
  z-index: 60;
}

.np-drawer__mask {
  position: absolute;
  inset: 0;
  background: rgb(21 32 29 / 36%);
  backdrop-filter: blur(3px);
  animation: np-fade var(--np-duration) var(--np-ease);
}

.np-drawer__panel {
  position: absolute;
  top: 0;
  right: 0;
  bottom: 0;
  display: flex;
  flex-direction: column;
  max-width: 100%;
  background: var(--np-bg-elevated);
  border-left: 1px solid var(--np-border);
  box-shadow: var(--np-shadow-md);
  animation: np-slide var(--np-duration) var(--np-ease);
}

.np-drawer__head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
  padding: 18px 20px 12px;
  border-bottom: 1px solid var(--np-border);
}

.np-drawer__titles {
  min-width: 0;
}

.np-drawer__eyebrow {
  margin: 0 0 4px;
  font-family: var(--np-font-mono);
  font-size: var(--np-text-xs);
  color: var(--np-text-muted);
}

.np-drawer__head h3 {
  margin: 0;
  font-size: var(--np-text-lg);
  letter-spacing: -0.02em;
  line-height: 1.35;
  word-break: break-word;
}

.np-drawer__x {
  border: none;
  background: transparent;
  color: var(--np-text-muted);
  font-size: 24px;
  line-height: 1;
  cursor: pointer;
  flex-shrink: 0;
}

.np-drawer__body {
  flex: 1;
  overflow: auto;
  padding: 16px 20px 24px;
}

.np-drawer__foot {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
  padding: 12px 20px;
  border-top: 1px solid var(--np-border);
}

@keyframes np-fade {
  from {
    opacity: 0;
  }
  to {
    opacity: 1;
  }
}

@keyframes np-slide {
  from {
    transform: translateX(16px);
    opacity: 0.6;
  }
  to {
    transform: translateX(0);
    opacity: 1;
  }
}

@media (max-width: 640px) {
  .np-drawer__panel {
    width: 100% !important;
  }
}
</style>
