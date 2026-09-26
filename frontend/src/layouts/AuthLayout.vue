<template>
  <main class="auth-layout">
    <section class="auth-card" aria-labelledby="auth-title">
      <div class="auth-card__glow auth-card__glow--top" aria-hidden="true" />
      <div class="auth-card__glow auth-card__glow--bottom" aria-hidden="true" />
      <div class="auth-card__content">
        <div class="brand-mark" aria-hidden="true">TF</div>
        <slot />
        <footer class="auth-footer">
          <span class="auth-footer__dot" aria-hidden="true" />
          Líneas activas: 100% <span aria-hidden="true">•</span> Lima · Gamarra
        </footer>
      </div>
    </section>
  </main>
</template>
