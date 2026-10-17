<script setup lang="ts">
import { ref, watch } from 'vue'

// Costura de 7 puntadas: 6 segmentos y un nudo final (la Entrega).
// Al subir `completed`, el segmento recién terminado "se cose" de izquierda a derecha.
const props = withDefaults(defineProps<{ completed: number; total?: number; tone?: 'default' | 'late' }>(), { total: 7, tone: 'default' })
const sewing = ref<number | null>(null)

watch(() => props.completed, (now, before) => {
  sewing.value = before !== undefined && now === before + 1 ? now : null
})
</script>

<template>
  <div class="seam" :class="{ 'seam--late': tone === 'late' }" :style="{ gridTemplateColumns: `repeat(${Math.max(1, total - 1)}, minmax(0, 1fr)) auto` }" role="img" :aria-label="`${completed} de ${total} estaciones completadas`">
    <i v-for="index in total - 1" :key="index" class="seam__stitch" :class="{ 'is-done': index <= completed, 'is-sewing': index === sewing }" />
    <i class="seam__knot" :class="{ 'is-done': completed >= total }" />
  </div>
</template>

<style scoped>
.seam { display: grid; grid-template-columns: repeat(6, minmax(0, 1fr)) auto; align-items: center; gap: 3px; }
.seam__stitch { position: relative; height: 3px; overflow: hidden; border-radius: 2px; background: repeating-linear-gradient(90deg, var(--tf-border-soft) 0 4px, transparent 4px 7px); }
.seam__stitch::after { content: ""; position: absolute; inset: 0; background: var(--tf-primary); transform: scaleX(0); transform-origin: left; }
.seam__stitch.is-done::after { transform: scaleX(1); }
.seam__stitch.is-sewing::after { animation: seam-sew .45s ease-out both; }
.seam__knot { width: 7px; height: 7px; border: 2px solid var(--tf-border-soft); border-radius: 50%; }
.seam__knot.is-done { border-color: var(--tf-primary); background: var(--tf-primary); }
.seam--late .seam__stitch { background: repeating-linear-gradient(90deg, var(--tf-danger-border) 0 4px, transparent 4px 7px); }
@keyframes seam-sew { from { transform: scaleX(0); } to { transform: scaleX(1); } }
@media (prefers-reduced-motion: reduce) { .seam__stitch.is-sewing::after { animation: none; transform: scaleX(1); } }
</style>
