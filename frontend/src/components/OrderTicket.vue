<script setup lang="ts">
import { computed } from 'vue'
import StatusChip, { type DeliveryState } from './StatusChip.vue'
import StitchSeam from './StitchSeam.vue'

// Ficha viajera: la forma propia del sistema, solo para órdenes.
const props = defineProps<{
  code: string
  product: string
  state: DeliveryState
  completed: number
  station?: string
  client?: string
  due?: string
  to?: string
}>()

const late = computed(() => props.state === 'LATE')
</script>

<template>
  <component :is="to ? 'RouterLink' : 'article'" :to="to" class="ticket" :class="{ 'ticket--late': late }">
    <span class="ticket__in">
      <span class="ticket__top"><span class="ticket__code">{{ code }}</span><StatusChip :state="state" /></span>
      <span class="ticket__product">{{ product }}</span>
      <span class="ticket__meta">
        <b v-if="station">{{ station }}</b>
        <span v-if="client">{{ client }}</span>
        <span v-if="due">{{ due }}</span>
      </span>
      <slot />
      <StitchSeam :completed="completed" :tone="late ? 'late' : 'default'" />
    </span>
  </component>
</template>

<style scoped>
.ticket { display: block; min-width: 0; padding: 1px; border-radius: var(--tf-radius-sm); background: var(--tf-border-soft); color: var(--tf-ink); text-decoration: none; clip-path: polygon(0 0, calc(100% - var(--tf-notch)) 0, 100% var(--tf-notch), 100% 100%, 0 100%); }
.ticket__in { display: grid; gap: .3rem; padding: .7rem .75rem .65rem; border-radius: calc(var(--tf-radius-sm) - 1px); background: var(--tf-surface); clip-path: polygon(0 0, calc(100% - var(--tf-notch) + .5px) 0, 100% calc(var(--tf-notch) - .5px), 100% 100%, 0 100%); }
.ticket--late { background: var(--tf-late); }
.ticket--late .ticket__in { background: var(--tf-late-soft); }
a.ticket:hover { background: var(--tf-primary); }
a.ticket.ticket--late:hover { background: var(--tf-danger-hover); }
a.ticket:focus-visible { outline: 3px solid var(--tf-focus); outline-offset: 2px; clip-path: none; }
.ticket__top { display: flex; align-items: center; justify-content: space-between; gap: .5rem; padding-right: .7rem; }
.ticket__code { font-family: var(--tf-font-mono); font-size: .85rem; font-weight: 700; font-variant-numeric: tabular-nums; }
.ticket__product { overflow-wrap: anywhere; font-size: .98rem; font-weight: 700; line-height: 1.3; }
.ticket__meta { display: flex; flex-wrap: wrap; gap: .15rem .75rem; color: var(--tf-text-subtle); font-size: .8rem; }
.ticket__meta b { color: var(--tf-ink); font-weight: 700; }
</style>
