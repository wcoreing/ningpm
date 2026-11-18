<script setup lang="ts">
import { NpSurface } from "@ningpm/ui"

defineProps<{ tip: string }>()
</script>

<template>
  <NpSurface>
    <p class="tip">{{ tip }}</p>
  </NpSurface>
</template>

<style scoped>
.tip {
  margin: 0;
  color: var(--np-text-secondary);
}
</style>
