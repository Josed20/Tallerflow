<script setup lang="ts">
import { RouterLink } from 'vue-router'
import BrandLogo from '../components/BrandLogo.vue'
</script>

<template>
  <main class="auth-layout">
    <section class="auth-card" aria-labelledby="auth-title">
      <div class="auth-card__content">
        <RouterLink class="auth-brand" to="/" aria-label="TallerFlow, ir al inicio"><BrandLogo :size="34" decorative /></RouterLink>
        <slot />
        <footer class="auth-footer">Cada orden visible, de Corte a Entrega · Lima, Gamarra</footer>
      </div>
    </section>
  </main>
</template>
