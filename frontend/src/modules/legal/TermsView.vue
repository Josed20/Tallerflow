<script setup lang="ts">
import { RouterLink } from 'vue-router'
import BrandLogo from '../../components/BrandLogo.vue'
</script>

<template>
  <div class="legal-page">
    <header class="legal-header">
      <div class="legal-container legal-header__inner">
        <BrandLogo :size="32" />
        <RouterLink to="/" class="legal-back-link">← Volver al inicio</RouterLink>
      </div>
    </header>

    <main class="legal-container legal-content">
      <article class="legal-article">
        <header class="legal-article__header">
          <span class="legal-badge">Documento legal</span>
          <h1>Términos y Condiciones de Servicio</h1>
          <p class="legal-date">Vigente a partir de octubre de 2026 · Lima, Perú</p>
        </header>

        <section>
          <h2>1. Identidad del Servicio</h2>
          <p>
            TallerFlow es una plataforma de software como servicio (SaaS) especializada en la trazabilidad de órdenes de confección para talleres textiles, operada desde Lima, Perú.
          </p>
        </section>

        <section>
          <h2>2. Aislamiento y Privacidad de los Talleres</h2>
          <p>
            TallerFlow opera bajo una arquitectura multi-inquilino con aislamiento forzado a nivel de base de datos (Row-Level Security). Los datos de tu taller (miembros, clientes, órdenes, etapas de producción y fotos) están aislados y son inaccesibles para otros talleres de la plataforma.
          </p>
          <p>
            El equipo técnico y de administración de TallerFlow <strong>no inspecciona el contenido de las órdenes, precios ni fichas de producción</strong> de tu taller. Las herramientas de supervisión de plataforma solo manejan métricas técnicas y de estado del servicio.
          </p>
        </section>

        <section>
          <h2>3. Cuentas y Responsabilidad</h2>
          <p>
            El creador inicial del taller asume el rol de Propietario (<code>OWNER</code>) y es responsable de administrar las invitaciones a los miembros de su equipo (Administradores y Operarios). Cada usuario debe mantener la confidencialidad de sus credenciales.
          </p>
        </section>

        <section>
          <h2>4. Suspensión del Taller</h2>
          <p>
            TallerFlow puede suspender temporalmente el acceso a un taller en caso de:
          </p>
          <ul>
            <li>Falta de pago o vencimiento del periodo acordado;</li>
            <li>Abuso de recursos o intentos de vulneración de seguridad;</li>
            <li>Solicitud expresa del titular de la cuenta.</li>
          </ul>
          <p>
            La suspensión deshabilita el inicio de sesión web y móvil y desactiva los enlaces de seguimiento de clientes, <strong>sin eliminar la información registrada</strong>.
          </p>
        </section>

        <section>
          <h2>5. Cancelación y Periodo de Gracia</h2>
          <p>
            Si solicitas la eliminación de tu taller, se aplicará un <strong>periodo de gracia de treinta (30) días calendario</strong> en estado de eliminación programada. Durante este tiempo puedes reactivar la cuenta comunicándote a <a href="mailto:soporte@tallerflowpe.com">soporte@tallerflowpe.com</a>. Vencido el plazo, la supresión de la base de datos es definitiva.
          </p>
        </section>

        <section>
          <h2>6. Ley Aplicable</h2>
          <p>
            Estos términos se rigen e interpretan bajo las leyes de la República del Perú. Cualquier controversia será resuelta ante los jueces y tribunales del distrito judicial de Lima.
          </p>
        </section>
      </article>
    </main>

    <footer class="legal-footer">
      <div class="legal-container">
        <p>© 2026 TallerFlow · <RouterLink to="/privacidad">Política de privacidad</RouterLink> · Contacto: soporte@tallerflowpe.com</p>
      </div>
    </footer>
  </div>
</template>

<style scoped>
.legal-page {
  min-height: 100vh;
  background-color: var(--tf-background, #F3F5F9);
  color: var(--tf-text, #16233D);
  font-family: var(--tf-font-body, -apple-system, sans-serif);
}

.legal-container {
  max-width: 800px;
  margin: 0 auto;
  padding: 0 1.25rem;
}

.legal-header {
  background-color: #FFFFFF;
  border-bottom: 1px solid var(--tf-border, #D5DCE6);
  padding: 1rem 0;
}

.legal-header__inner {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.legal-back-link {
  color: var(--tf-primary, #1F3A68);
  text-decoration: none;
  font-weight: 700;
  font-size: 0.9375rem;
}

.legal-back-link:hover {
  text-decoration: underline;
}

.legal-content {
  padding: 3rem 1.25rem 5rem;
}

.legal-article {
  background-color: #FFFFFF;
  border: 1px solid var(--tf-border, #D5DCE6);
  border-radius: var(--tf-rounded-md, 6px);
  padding: 2.5rem;
}

@media (max-width: 600px) {
  .legal-article {
    padding: 1.5rem;
  }
}

.legal-article__header {
  margin-bottom: 2.5rem;
  padding-bottom: 1.5rem;
  border-bottom: 1px solid var(--tf-border, #D5DCE6);
}

.legal-badge {
  display: inline-block;
  background-color: var(--tf-primary-soft, #E4EBF6);
  color: var(--tf-primary, #1F3A68);
  font-size: 0.75rem;
  font-weight: 700;
  padding: 0.25rem 0.6rem;
  border-radius: 4px;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  margin-bottom: 0.75rem;
}

.legal-article__header h1 {
  font-family: var(--tf-font-headline, Archivo, sans-serif);
  font-weight: 800;
  font-size: clamp(1.5rem, 4vw, 2.2rem);
  color: var(--tf-primary, #1F3A68);
  margin-bottom: 0.5rem;
  line-height: 1.2;
}

.legal-date {
  font-size: 0.875rem;
  color: var(--tf-text-muted, #556074);
}

.legal-article section {
  margin-bottom: 2rem;
  line-height: 1.7;
}

.legal-article h2 {
  font-family: var(--tf-font-headline, Archivo, sans-serif);
  font-weight: 800;
  font-size: 1.25rem;
  color: var(--tf-text, #16233D);
  margin-bottom: 0.75rem;
}

.legal-article p {
  color: var(--tf-text, #16233D);
  margin-bottom: 1rem;
}

.legal-article ul {
  padding-left: 1.5rem;
  margin-bottom: 1rem;
}

.legal-article li {
  margin-bottom: 0.5rem;
}

.legal-article a {
  color: var(--tf-primary, #1F3A68);
  font-weight: 700;
}

.legal-footer {
  border-top: 1px solid var(--tf-border, #D5DCE6);
  padding: 2rem 0;
  text-align: center;
  font-size: 0.875rem;
  color: var(--tf-text-muted, #556074);
  background-color: #FFFFFF;
}

.legal-footer a {
  color: var(--tf-primary, #1F3A68);
  text-decoration: none;
}
</style>
