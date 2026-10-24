<script setup lang="ts">
import { computed } from 'vue'

// Rombo tejido de TallerFlow. Los SVG salen de scripts/brand/build_logo.py; nunca se redibuja
// ni se reescribe "TallerFlow" con texto (DESIGN.md, Logo). `size` es la altura del rombo en px.
const props = withDefaults(defineProps<{ size?: number; wordmark?: boolean; tone?: 'light' | 'dark'; decorative?: boolean }>(), {
  size: 32,
  wordmark: true,
  tone: 'light',
  decorative: false,
})

const src = computed(() => {
  const name = props.wordmark ? 'logo-horizontal' : 'logo-symbol'
  return `${import.meta.env.BASE_URL}${name}${props.tone === 'dark' ? '-on-dark' : ''}.svg`
})
// Each file's viewBox ratio, so the image reserves its box before it loads.
const width = computed(() => Math.round(props.size * (props.wordmark ? 4.7183 : 1.3718)))
</script>

<template>
  <img class="brand-logo" :src="src" :width="width" :height="size" :alt="decorative ? '' : 'TallerFlow'" />
</template>

<style scoped>
.brand-logo { display: block; flex: none; max-width: 100%; height: auto; }
</style>
