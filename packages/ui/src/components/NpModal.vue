<script setup lang="ts">
import { watch } from "vue"

const props = defineProps<{
  open: boolean
  title?: string
}>()

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
    <div v-if="open" class="np-modal" @click.self="emit('close')">
      <div class="np-modal__panel" role="dialog" aria-modal="true">
        <header class="np-modal__head">
          <h3>{{ title }}</h3>
          <button class="np-modal__x" type="button" aria-label="关闭" @click="emit('close')">×</button>
        </header>
        <div class="np-modal__body">
          <slot />
        </div>
        <footer v-if="$slots.footer" class="np-modal__foot">
          <slot name="footer" />
        </footer>
      </div>
    </div>
  </Teleport>
</template>

<style scoped>
.np-modal {
  position: fixed;
  inset: 0;
  z-index: 50;
  display: grid;
  place-items: center;
  padding: 20px;
  background: rgb(21 32 29 / 42%);
  backdrop-filter: blur(4px);
  animation: np-fade var(--np-duration) var(--np-ease);
}

.np-modal__panel {
  width: min(520px, 100%);
  background: var(--np-bg-elevated);
  border: 1px solid var(--np-border);
  border-radius: var(--np-radius-lg);
  box-shadow: var(--np-shadow-md);
  animation: np-rise var(--np-duration) var(--np-ease);
}

.np-modal__head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 16px 18px 0;
}

.np-modal__head h3 {
  margin: 0;
  font-size: var(--np-text-lg);
  letter-spacing: -0.02em;
}

.np-modal__x {
  border: none;
  background: transparent;
  color: var(--np-text-muted);
  font-size: 22px;
  line-height: 1;
  cursor: pointer;
}

.np-modal__body {
  padding: 16px 18px;
}

.np-modal__foot {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
  padding: 0 18px 18px;
}

@keyframes np-fade {
  from {
    opacity: 0;
  }
  to {
    opacity: 1;
  }
}

@keyframes np-rise {
  from {
    opacity: 0;
    transform: translateY(8px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}
</style>
