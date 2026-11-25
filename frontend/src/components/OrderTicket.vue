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
  total?: number
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
      <StitchSeam :completed="completed" :total="total ?? 7" :tone="late ? 'late' : 'default'" />
    </span>
  </component>
</template>

<style scoped>
.ticket{display:flex;flex-direction:column;min-width:0;color:var(--tf-ink);text-decoration:none;border:1px solid var(--tf-border-soft);border-radius:14px;background:white;box-shadow:0 2px 3px #152b4b04;overflow:hidden;transition:border-color .18s,box-shadow .18s,transform .18s}
.ticket__in{display:grid;gap:.6rem;padding:1rem;align-content:start;flex:1}.ticket--late{border-left:3px solid var(--tf-late)}.ticket__top{display:flex;align-items:center;justify-content:space-between;gap:.5rem}.ticket__code{font-family:var(--tf-font-body);color:var(--tf-text-subtle);font-size:.75rem;font-weight:600;letter-spacing:.02em}.ticket__product{font-size:.98rem;font-weight:700;line-height:1.4;overflow-wrap:anywhere}.ticket__meta{display:flex;flex-wrap:wrap;gap:.25rem .7rem;color:var(--tf-text-subtle);font-size:.8rem}.ticket__meta b{color:var(--tf-primary);font-weight:600}a.ticket:hover{border-color:var(--tf-primary-border);box-shadow:0 6px 20px #09234c0c;transform:translateY(-1px)}a.ticket:focus-visible{outline:3px solid var(--tf-focus);outline-offset:3px}
</style>
