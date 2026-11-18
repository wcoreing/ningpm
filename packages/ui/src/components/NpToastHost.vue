<script setup lang="ts">
import { dismissToast, useToastState } from "../toast"

const state = useToastState()
</script>

<template>
  <Teleport to="body">
    <div class="np-toast-host" aria-live="polite" aria-relevant="additions">
      <TransitionGroup name="np-toast">
        <div
          v-for="item in state.items"
          :key="item.id"
          class="np-toast"
          :class="`np-toast--${item.tone}`"
          role="status"
        >
          <span class="np-toast__msg">{{ item.message }}</span>
          <button
            type="button"
            class="np-toast__x"
            aria-label="关闭"
            @click="dismissToast(item.id)"
          >
            ×
          </button>
        </div>
      </TransitionGroup>
    </div>
  </Teleport>
</template>

<style scoped>
.np-toast-host {
  position: fixed;
  top: 18px;
  left: 50%;
  z-index: 80;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  width: min(420px, calc(100vw - 24px));
  transform: translateX(-50%);
  pointer-events: none;
}

.np-toast {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  width: 100%;
  padding: 10px 12px;
  border: 1px solid var(--np-border);
  border-radius: var(--np-radius-md);
  background: var(--np-bg-elevated);
  box-shadow: var(--np-shadow-md);
  pointer-events: auto;
}

.np-toast__msg {
  flex: 1;
  min-width: 0;
  font-size: var(--np-text-sm);
  font-weight: 600;
  line-height: 1.45;
  color: var(--np-text);
}

.np-toast__x {
  flex-shrink: 0;
  border: none;
  background: transparent;
  color: var(--np-text-muted);
  font-size: 18px;
  line-height: 1;
  cursor: pointer;
}

.np-toast--success {
  border-color: color-mix(in srgb, var(--np-success) 45%, var(--np-border));
  background: color-mix(in srgb, var(--np-success-muted) 70%, var(--np-bg-elevated));
}

.np-toast--success .np-toast__msg {
  color: var(--np-success);
}

.np-toast--error {
  border-color: color-mix(in srgb, var(--np-danger) 45%, var(--np-border));
  background: color-mix(in srgb, var(--np-danger-muted) 70%, var(--np-bg-elevated));
}

.np-toast--error .np-toast__msg {
  color: var(--np-danger);
}

.np-toast--warning {
  border-color: color-mix(in srgb, var(--np-warning) 45%, var(--np-border));
  background: color-mix(in srgb, var(--np-warning-muted) 70%, var(--np-bg-elevated));
}

.np-toast--warning .np-toast__msg {
  color: var(--np-warning);
}

.np-toast--info {
  border-color: color-mix(in srgb, var(--np-accent) 40%, var(--np-border));
  background: color-mix(in srgb, var(--np-accent-muted) 65%, var(--np-bg-elevated));
}

.np-toast--info .np-toast__msg {
  color: var(--np-accent-ink);
}

.np-toast-enter-active,
.np-toast-leave-active {
  transition:
    opacity var(--np-duration) var(--np-ease),
    transform var(--np-duration) var(--np-ease);
}

.np-toast-enter-from,
.np-toast-leave-to {
  opacity: 0;
  transform: translateY(-8px);
}

.np-toast-move {
  transition: transform var(--np-duration) var(--np-ease);
}
</style>
