<script setup lang="ts">
import { computed } from 'vue'
import { RouterLink } from 'vue-router'
import BrandLogo from '../../components/BrandLogo.vue'
import { useSessionStore } from '../session/session.store'

const session = useSessionStore()
const isAuthenticated = computed(() => session.isAuthenticated)
</script>

<template>
  <div class="landing-page">
    <header class="landing-header">
      <div class="landing-container landing-header__inner">
        <BrandLogo :size="36" />
        <nav class="landing-nav">
          <RouterLink v-if="isAuthenticated" to="/app" class="landing-btn landing-btn--primary">
            Ir a mi panel
          </RouterLink>
          <template v-else>
            <RouterLink to="/login" class="landing-btn landing-btn--ghost">
              Iniciar sesión
            </RouterLink>
            <RouterLink to="/onboarding" class="landing-btn landing-btn--primary">
              Crear mi taller
            </RouterLink>
          </template>
        </nav>
      </div>
    </header>

    <main>
      <!-- Hero -->
      <section class="landing-hero">
        <div class="landing-container landing-hero__inner">
          <div class="landing-badge">
            <span class="landing-badge__dot"></span>
            Ficha viajera · Control de producción para talleres
          </div>
          <h1 class="landing-title">
            En 2 segundos sabes qué pedido está atrasado
          </h1>
          <p class="landing-subtitle">
            TallerFlow conecta al dueño, los operarios y tus marcas clientes en un flujo simple de 7 estaciones. Diseñado especialmente para la realidad textil de Gamarra.
          </p>
          <div class="landing-actions">
            <RouterLink to="/onboarding" class="landing-btn landing-btn--cta">
              Crear mi taller gratis
            </RouterLink>
            <RouterLink to="/login" class="landing-btn landing-btn--secondary">
              Ingresar al sistema
            </RouterLink>
            <a href="#descargar-app" class="landing-btn landing-btn--ghost">
              Descargar app Android
            </a>
          </div>
        </div>
      </section>

      <!-- Value Props -->
      <section class="landing-features">
        <div class="landing-container">
          <h2 class="landing-section-title">Hecho para el ritmo real del taller</h2>
          <div class="landing-grid">
            <article class="landing-card">
              <div class="landing-card__tag">01</div>
              <h3 class="landing-card__title">Trazabilidad en 7 estaciones</h3>
              <p class="landing-card__text">
                De Corte a Entrega. Cada orden avanza como una ficha viajera con su costura visual de puntadas y fotos de evidencia en cada etapa.
              </p>
            </article>

            <article class="landing-card">
              <div class="landing-card__tag">02</div>
              <h3 class="landing-card__title">App móvil para operarios</h3>
              <p class="landing-card__text">
                Botones grandes de 48 px, escaneo rápido de fichas y opción de deshacer de inmediato si alguien avanza una orden por error.
              </p>
            </article>

            <article class="landing-card">
              <div class="landing-card__tag">03</div>
              <h3 class="landing-card__title">Tranquilidad para tu cliente</h3>
              <p class="landing-card__text">
                Comparte un enlace público de seguimiento con tu marca cliente. Ellos ven el avance en tiempo real sin llamarte a cada hora.
              </p>
            </article>
          </div>
        </div>
      </section>

      <!-- App Download Section -->
      <section id="descargar-app" class="landing-download">
        <div class="landing-container landing-download__box">
          <div class="landing-download__info">
            <h2>Lleva TallerFlow en el celular</h2>
            <p>
              Instala la aplicación en los teléfonos de planta para que los operarios registren avances y fotos sin interrumpir la costura.
            </p>
            <div class="landing-download__links">
              <a href="/app" class="landing-btn landing-btn--primary">
                Abrir versión web móvil
              </a>
              <span class="landing-download__hint">App Android disponible para el piloto de talleres</span>
            </div>
          </div>
        </div>
      </section>
    </main>

    <!-- Footer -->
    <footer class="landing-footer">
      <div class="landing-container landing-footer__inner">
        <div class="landing-footer__brand">
          <BrandLogo :size="28" />
          <p>Plataforma de trazabilidad y gestión para talleres de confección textil.</p>
        </div>
        <div class="landing-footer__links">
          <div>
            <h4>Legal</h4>
            <ul>
              <li><RouterLink to="/terminos">Términos de servicio</RouterLink></li>
              <li><RouterLink to="/privacidad">Política de privacidad</RouterLink></li>
            </ul>
          </div>
          <div>
            <h4>Soporte</h4>
            <ul>
              <li><a href="mailto:soporte@tallerflowpe.com">soporte@tallerflowpe.com</a></li>
              <li><span>Lima, Perú</span></li>
            </ul>
          </div>
        </div>
      </div>
      <div class="landing-container landing-footer__bottom">
        <p>© 2026 TallerFlow. Todos los derechos reservados.</p>
      </div>
    </footer>
  </div>
</template>

<style scoped>
.landing-page {
  min-height: 100vh;
  background-color: var(--tf-background, #F3F5F9);
  color: var(--tf-text, #16233D);
  font-family: var(--tf-font-body, -apple-system, sans-serif);
}

.landing-container {
  max-width: 1120px;
  margin: 0 auto;
  padding: 0 1.25rem;
}

.landing-header {
  background-color: #FFFFFF;
  border-bottom: 1px solid var(--tf-border, #D5DCE6);
  position: sticky;
  top: 0;
  z-index: 40;
}

.landing-header__inner {
  display: flex;
  align-items: center;
  justify-content: space-between;
  height: 4.25rem;
}

.landing-nav {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.landing-hero {
  padding: 4rem 0 3.5rem;
  text-align: center;
}

.landing-hero__inner {
  max-width: 800px;
}

.landing-badge {
  display: inline-flex;
  align-items: center;
  gap: 0.5rem;
  background-color: var(--tf-primary-soft, #E4EBF6);
  color: var(--tf-primary, #1F3A68);
  font-size: 0.8125rem;
  font-weight: 700;
  padding: 0.35rem 0.85rem;
  border-radius: 9999px;
  margin-bottom: 1.5rem;
  text-transform: uppercase;
  letter-spacing: 0.04em;
}

.landing-badge__dot {
  width: 8px;
  height: 8px;
  background-color: var(--tf-primary, #1F3A68);
  border-radius: 50%;
}

.landing-title {
  font-family: var(--tf-font-display, Archivo, sans-serif);
  font-variation-settings: "wdth" 125;
  font-weight: 900;
  font-size: clamp(2rem, 5.5vw, 3.4rem);
  line-height: 1.1;
  letter-spacing: -0.02em;
  color: var(--tf-primary, #1F3A68);
  margin-bottom: 1.25rem;
}

.landing-subtitle {
  font-size: clamp(1.05rem, 2.5vw, 1.25rem);
  line-height: 1.6;
  color: var(--tf-text-muted, #556074);
  margin: 0 auto 2.5rem;
}

.landing-actions {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: center;
  gap: 0.875rem;
}

.landing-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 0.75rem 1.5rem;
  font-size: 1rem;
  font-weight: 700;
  border-radius: var(--tf-rounded-md, 6px);
  text-decoration: none;
  min-height: 48px;
  box-sizing: border-box;
  transition: background-color 0.15s ease, border-color 0.15s ease;
}

.landing-btn--primary,
.landing-btn--cta {
  background-color: var(--tf-primary, #1F3A68);
  color: #FFFFFF;
}

.landing-btn--primary:hover,
.landing-btn--cta:hover {
  background-color: var(--tf-primary-hover, #2A4C85);
}

.landing-btn--secondary {
  background-color: #FFFFFF;
  color: var(--tf-primary, #1F3A68);
  border: 1.5px solid var(--tf-primary, #1F3A68);
}

.landing-btn--secondary:hover {
  background-color: var(--tf-primary-soft, #E4EBF6);
}

.landing-btn--ghost {
  color: var(--tf-primary, #1F3A68);
  background-color: transparent;
  border: 1px solid var(--tf-border, #D5DCE6);
}

.landing-btn--ghost:hover {
  background-color: #FFFFFF;
}

.landing-features {
  padding: 4rem 0;
  background-color: #FFFFFF;
  border-top: 1px solid var(--tf-border, #D5DCE6);
  border-bottom: 1px solid var(--tf-border, #D5DCE6);
}

.landing-section-title {
  font-family: var(--tf-font-headline, Archivo, sans-serif);
  font-weight: 800;
  font-size: 1.75rem;
  text-align: center;
  margin-bottom: 2.5rem;
  color: var(--tf-primary, #1F3A68);
}

.landing-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
  gap: 1.5rem;
}

.landing-card {
  background-color: var(--tf-background, #F3F5F9);
  border: 1px solid var(--tf-border, #D5DCE6);
  border-radius: var(--tf-rounded-md, 6px);
  padding: 1.75rem;
  position: relative;
}

.landing-card__tag {
  font-family: var(--tf-font-mono, monospace);
  font-weight: 800;
  font-size: 0.875rem;
  color: var(--tf-primary, #1F3A68);
  background-color: var(--tf-primary-soft, #E4EBF6);
  padding: 0.2rem 0.5rem;
  border-radius: 4px;
  display: inline-block;
  margin-bottom: 1rem;
}

.landing-card__title {
  font-family: var(--tf-font-headline, Archivo, sans-serif);
  font-weight: 800;
  font-size: 1.25rem;
  margin-bottom: 0.75rem;
  color: var(--tf-text, #16233D);
}

.landing-card__text {
  color: var(--tf-text-muted, #556074);
  line-height: 1.6;
}

.landing-download {
  padding: 4rem 0;
}

.landing-download__box {
  background-color: #16294A;
  color: #FFFFFF;
  border-radius: 8px;
  padding: 3rem 2rem;
  text-align: center;
}

.landing-download__info h2 {
  font-family: var(--tf-font-headline, Archivo, sans-serif);
  font-weight: 800;
  font-size: 1.75rem;
  margin-bottom: 1rem;
}

.landing-download__info p {
  color: #A1AEC2;
  max-width: 600px;
  margin: 0 auto 2rem;
  line-height: 1.6;
}

.landing-download__links {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0.75rem;
}

.landing-download__hint {
  font-size: 0.8125rem;
  color: #A1AEC2;
}

.landing-footer {
  background-color: #FFFFFF;
  border-top: 1px solid var(--tf-border, #D5DCE6);
  padding: 3rem 0 1.5rem;
}

.landing-footer__inner {
  display: flex;
  flex-wrap: wrap;
  justify-content: space-between;
  gap: 2rem;
  margin-bottom: 2.5rem;
}

.landing-footer__brand p {
  margin-top: 0.75rem;
  color: var(--tf-text-muted, #556074);
  max-width: 320px;
  line-height: 1.5;
}

.landing-footer__links {
  display: flex;
  gap: 3rem;
}

.landing-footer__links h4 {
  font-size: 0.875rem;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: var(--tf-text, #16233D);
  margin-bottom: 0.75rem;
}

.landing-footer__links ul {
  list-style: none;
  padding: 0;
  margin: 0;
}

.landing-footer__links li {
  margin-bottom: 0.5rem;
}

.landing-footer__links a {
  color: var(--tf-text-muted, #556074);
  text-decoration: none;
}

.landing-footer__links a:hover {
  color: var(--tf-primary, #1F3A68);
}

.landing-footer__bottom {
  border-top: 1px solid var(--tf-border, #D5DCE6);
  padding-top: 1.5rem;
  text-align: center;
  font-size: 0.875rem;
  color: var(--tf-text-muted, #556074);
}
</style>
