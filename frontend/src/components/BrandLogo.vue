<script setup lang="ts">
// Etiqueta colgante cruzada por 7 puntadas (las 7 estaciones; la última, la Entrega).
// Usa currentColor para funcionar en tinta sobre claro o en blanco sobre azul mezclilla.
withDefaults(defineProps<{ size?: number; wordmark?: boolean }>(), { size: 32, wordmark: true })
</script>

<template>
  <span class="brand-logo">
    <svg :width="size" :height="size" viewBox="0 0 48 48" aria-hidden="true" focusable="false">
      <path d="M9 6h22l9 9v27H9z" fill="none" stroke="currentColor" stroke-width="3.2" stroke-linejoin="round" />
      <circle cx="17" cy="14" r="3.2" fill="none" stroke="currentColor" stroke-width="2.6" />
      <g fill="currentColor">
        <rect x="12" y="30" width="2.6" height="2.8" rx="1" /><rect x="15.9" y="30" width="2.6" height="2.8" rx="1" />
        <rect x="19.8" y="30" width="2.6" height="2.8" rx="1" /><rect x="23.7" y="30" width="2.6" height="2.8" rx="1" />
        <rect x="27.6" y="30" width="2.6" height="2.8" rx="1" /><rect x="31.5" y="30" width="2.6" height="2.8" rx="1" />
        <circle cx="36" cy="31.4" r="2.1" />
      </g>
    </svg>
    <span v-if="wordmark" class="brand-logo__word">TallerFlow</span>
  </span>
</template>

<style scoped>
.brand-logo { display: inline-flex; align-items: center; gap: .55rem; color: inherit; }
.brand-logo svg { flex: none; }
.brand-logo__word { font-family: var(--tf-font-display); font-variation-settings: "wdth" 125; font-weight: 800; font-size: 1.15em; letter-spacing: -.01em; line-height: 1; }
</style>
