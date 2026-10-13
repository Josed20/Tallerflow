<script setup lang="ts">
import BrandLogo from '../components/BrandLogo.vue'
</script>

<template>
  <main class="auth-layout">
    <section class="auth-card" aria-labelledby="auth-title">
      <div class="auth-card__content">
        <BrandLogo class="auth-brand-logo" />
        <slot />
      </div>
    </section>
  </main>
</template>
