<script setup lang="ts">
import { RouterLink } from 'vue-router'
import BrandLogo from '../../components/BrandLogo.vue'
</script>

<template>
  <div class="legal-page">
    <header class="legal-header"><div class="legal-container legal-header__inner">
      <BrandLogo :size="32" />
      <RouterLink to="/login" class="legal-back-link">Volver a TallerFlow</RouterLink>
    </div></header>
    <main class="legal-container legal-content"><article class="legal-article">
      <header class="legal-article__header">
        <span class="legal-badge">Tu cuenta y tus datos</span>
        <h1>Solicitar la eliminación de tu cuenta de TallerFlow</h1>
        <p>No necesitas iniciar sesión ni tener instalada la app para solicitarlo.</p>
      </header>
      <section>
        <h2>Cómo solicitarlo</h2>
        <ol>
          <li>Escribe a <a href="mailto:tallerflow.pe@gmail.com?subject=Eliminar%20mi%20cuenta%20de%20TallerFlow">tallerflow.pe@gmail.com</a> desde el correo registrado en TallerFlow.</li>
          <li>Usa el asunto <strong>Eliminar mi cuenta de TallerFlow</strong> e indica el nombre de tu taller y si solicitas eliminar tu cuenta personal o, como propietario, el taller completo.</li>
          <li>El equipo verificará tu identidad y el alcance de la solicitud, y te confirmará por correo su tramitación. Si perdiste acceso a tu correo registrado, indícalo para coordinar la verificación.</li>
        </ol>
        <p>No envíes contraseñas. Abrir el correo no elimina nada automáticamente: debes enviar la solicitud y recibir la confirmación del equipo.</p>
      </section>
      <section>
        <h2>Qué datos se eliminan</h2>
        <p>La solicitud incluye tu perfil de usuario (nombre, correo, teléfono si se registró), tus credenciales y sesiones. Si eres propietario y solicitas borrar el taller completo, incluye también sus miembros, clientes, órdenes, avances y fotografías de evidencia.</p>
        <p>Eliminar la cuenta de un integrante no equivale a eliminar el taller de otros usuarios. Las órdenes y evidencias compartidas del taller pueden conservarse como registros operativos; el equipo te informará qué información permanece y su relación con tu cuenta antes de tramitar la solicitud.</p>
      </section>
      <section>
        <h2>Plazos y datos que se conservan</h2>
        <p>Para la eliminación del taller se aplica el periodo de gracia de <strong>30 días calendario</strong> indicado en nuestra política de privacidad. Durante ese periodo puedes solicitar cancelar la eliminación por el mismo correo. Después se realiza la supresión definitiva de los datos del taller.</p>
        <p>Las copias de seguridad pueden conservar los datos durante un máximo de <strong>4 semanas adicionales</strong>, según la política de privacidad. Si alguna información debe conservarse por una obligación aplicable, te informaremos los datos, el motivo y el plazo correspondiente al responder tu solicitud.</p>
      </section>
      <section><h2>Más información</h2><p>Consulta nuestra <RouterLink to="/privacidad">política de privacidad</RouterLink> y los <RouterLink to="/terminos">términos de servicio</RouterLink>.</p></section>
    </article></main>
    <footer class="legal-footer"><div class="legal-container"><p>TallerFlow · Contacto: <a href="mailto:tallerflow.pe@gmail.com">tallerflow.pe@gmail.com</a></p></div></footer>
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
  color: var(--tf-primary, #0A346F);
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
  background-color: var(--tf-primary-soft, #E7EEF8);
  color: var(--tf-primary, #0A346F);
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
  color: var(--tf-primary, #0A346F);
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

.legal-article ul, .legal-article ol {
  padding-left: 1.5rem;
  margin-bottom: 1rem;
}

.legal-article li {
  margin-bottom: 0.5rem;
}

.legal-article a {
  color: var(--tf-primary, #0A346F);
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
  color: var(--tf-primary, #0A346F);
  text-decoration: none;
}
</style>
