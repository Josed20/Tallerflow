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
          <span class="legal-badge">Protección de datos</span>
          <h1>Política de Privacidad</h1>
          <p class="legal-date">Conforme a la Ley N° 29733 (Perú) · Octubre de 2026</p>
        </header>

        <section>
          <h2>1. Compromiso de Privacidad</h2>
          <p>
            En TallerFlow entendemos que la información de tus órdenes, modelos de prendas y marcas clientes representa el activo más valioso de tu taller. Nos comprometemos a proteger tus datos personales conforme a la <strong>Ley N° 29733, Ley de Protección de Datos Personales del Perú</strong>, y su Reglamento.
          </p>
        </section>

        <section>
          <h2>2. Datos que Tratamos y Finalidad</h2>
          <p>Tratamos únicamente los datos necesarios para brindar el servicio de trazabilidad textil:</p>
          <ul>
            <li><strong>Datos del taller y equipo:</strong> Razón social o nombre comercial, nombre y correo del propietario y miembros (operarios y administradores), para gestionar accesos y perfiles de trabajo.</li>
            <li><strong>Clientes de confección:</strong> Nombre comercial o de contacto, y teléfono de coordinación para generar los enlaces de seguimiento.</li>
            <li><strong>Órdenes y fotos de evidencia:</strong> Fichas de lote, cantidades, estaciones de avance y fotografías tomadas en planta para sustentar el control de calidad.</li>
          </ul>
        </section>

        <section>
          <h2>3. Principio de No Invasión</h2>
          <p>
            El personal de TallerFlow <strong>no accede al contenido de las órdenes, precios pactados ni listas de clientes</strong> de los talleres registrados. Las consolas de plataforma solo muestran métricas agregadas y metadatos técnicos para asegurar la disponibilidad del sistema.
          </p>
        </section>

        <section>
          <h2>4. Dónde se Alojan tus Datos</h2>
          <p>
            Nuestros servicios operan en un servidor privado virtual (VPS) configurado con cifrado en reposo y en tránsito (HTTPS / TLS 1.3). Empleamos proveedores confiables de infraestructura de red (Cloudflare) y pasarelas de entrega de correo transaccional sujetas a acuerdos de confidencialidad.
          </p>
        </section>

        <section>
          <h2>5. Retención, Eliminación y Copias de Seguridad</h2>
          <p>
            Conservamos los datos de tu taller mientras mantengas una cuenta activa. En caso de solicitar la eliminación definitiva:
          </p>
          <ul>
            <li>Se inicia un <strong>periodo de gracia de treinta (30) días calendario</strong> durante el cual puedes revertir la decisión.</li>
            <li>Al término de los 30 días, los datos del taller se purgan permanentemente de la base de datos principal.</li>
            <li>Las copias de seguridad automáticas (backups) conservan registros cifrados por un <strong>plazo máximo de cuatro (4) semanas</strong>, tras el cual se eliminan de forma automatizada e irrecuperable.</li>
          </ul>
        </section>

        <section>
          <h2>6. Derechos ARCO y Contacto</h2>
          <p>
            Puedes ejercer en cualquier momento tus derechos de Acceso, Rectificación, Cancelación y Oposición (ARCO) enviando un mensaje a <a href="mailto:soporte@tallerflow.pe">soporte@tallerflow.pe</a> con el asunto "Derechos ARCO", indicando el correo de tu cuenta registrada.
          </p>
        </section>
      </article>
    </main>

    <footer class="legal-footer">
      <div class="legal-container">
        <p>© 2026 TallerFlow · <RouterLink to="/terminos">Términos de servicio</RouterLink> · Contacto: soporte@tallerflow.pe</p>
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
