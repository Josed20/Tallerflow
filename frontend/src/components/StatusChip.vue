<script setup lang="ts">
import { computed } from 'vue'

// Cada estado lleva palabra, forma y color: rojo y verde tienen casi la misma
// luminancia, así que la forma es lo que permite distinguirlos sin color.
export type DeliveryState = 'ON_TIME' | 'AT_RISK' | 'LATE' | 'COMPLETED'

const props = defineProps<{ state: DeliveryState; label?: string }>()
const text = computed(() => props.label ?? ({ ON_TIME: 'A tiempo', AT_RISK: 'En riesgo', LATE: 'Atrasada', COMPLETED: 'Completada' })[props.state])
</script>

<template>
  <span class="status-chip" :class="`status-chip--${state.toLowerCase()}`">
    <svg viewBox="0 0 12 12" aria-hidden="true" focusable="false">
      <circle v-if="state === 'ON_TIME'" cx="6" cy="6" r="5" fill="currentColor" />
      <path v-else-if="state === 'AT_RISK'" d="M6 1l5.2 9.5H.8z" fill="currentColor" />
      <template v-else-if="state === 'LATE'">
        <rect x="1" y="1" width="10" height="10" rx="1.5" fill="currentColor" />
        <rect x="5.2" y="2.8" width="1.6" height="4.4" class="status-chip__mark" /><rect x="5.2" y="8.2" width="1.6" height="1.6" class="status-chip__mark" />
      </template>
      <path v-else d="M2 6.3l2.6 2.6L10 3.4" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round" />
    </svg>
    {{ text }}
  </span>
</template>

<style scoped>
.status-chip { display: inline-flex; width: fit-content; align-items: center; gap: .35rem; padding: .2rem .5rem; border-radius: var(--tf-radius-sm); font-size: .75rem; font-weight: 800; line-height: 1.3; white-space: nowrap; }
.status-chip svg { width: .75rem; height: .75rem; flex: none; }
.status-chip--on_time { color: var(--tf-success); background: var(--tf-success-soft); }
.status-chip--at_risk { color: var(--tf-on-risk); background: var(--tf-risk-fill); }
.status-chip--late { color: var(--tf-on-late); background: var(--tf-late); }
.status-chip--late .status-chip__mark { fill: var(--tf-late); }
.status-chip--completed { color: var(--tf-done); background: var(--tf-done-soft); }
</style>
