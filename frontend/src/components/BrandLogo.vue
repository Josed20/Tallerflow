<script setup lang="ts">
defineProps<{
  inverse?: boolean
  symbolOnly?: boolean
}>()
</script>

<template>
  <img
    :src="symbolOnly ? (inverse ? '/brand/symbol-on-dark.svg' : '/brand/symbol.svg') : (inverse ? '/brand/logo-on-dark.svg' : '/brand/logo.svg')"
    :width="symbolOnly ? 64 : 310"
    :height="symbolOnly ? 64 : 72"
    alt="TallerFlow"
    class="brand-logo"
  />
</template>

<style scoped>
.brand-logo { display: block; max-width: 100%; height: auto; }
</style>
