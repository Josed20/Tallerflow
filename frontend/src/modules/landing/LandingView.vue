<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { RouterLink } from 'vue-router'
import BrandLogo from '../../components/BrandLogo.vue'
import OrderTicket from '../../components/OrderTicket.vue'
import StatusChip from '../../components/StatusChip.vue'
import StitchSeam from '../../components/StitchSeam.vue'

const stations = ['Corte', 'Costura', 'Acabado', 'Control de calidad', 'Planchado', 'Empaque', 'Entrega']

// The page's one authored moment: the on-time ticket sews its latest stitch once,
// the same motion the app plays when a station is advanced (reduced motion skips it).
const sewn = ref(4)
onMounted(() => {
  window.setTimeout(() => { sewn.value = 5 }, 700)
})
</script>

<template>
  <div class="landing">
    <header class="landing-header">
      <div class="landing-wrap landing-header__inner">
        <RouterLink to="/" class="landing-header__brand" aria-label="TallerFlow, inicio"><BrandLogo :size="30" decorative /></RouterLink>
        <nav class="landing-nav" aria-label="Secciones">
          <a class="landing-nav__section" href="#como-funciona">Cómo funciona</a>
          <a class="landing-nav__section" href="#seguimiento">Para tu cliente</a>
          <RouterLink class="landing-nav__login" to="/login">Ingresar</RouterLink>
          <RouterLink class="landing-button landing-button--primary landing-nav__signup" to="/onboarding">Registrar mi taller</RouterLink>
        </nav>
      </div>
    </header>

    <main>
      <section class="landing-hero" aria-labelledby="hero-title">
        <div class="landing-wrap landing-hero__inner">
          <div class="landing-hero__copy">
            <h1 id="hero-title">Cada orden visible, de Corte a Entrega.</h1>
            <p>Abre el celular y en 2 segundos sabes qué pedido está atrasado. Tus operarios registran el avance con una foto, y tu cliente sigue su pedido con un link, sin llamarte.</p>
            <div class="landing-actions">
              <RouterLink class="landing-button landing-button--primary" to="/onboarding">Registrar mi taller</RouterLink>
              <a class="landing-button landing-button--secondary" href="#como-funciona">Ver cómo funciona</a>
            </div>
          </div>

          <div class="landing-hero__panel" aria-label="Así se ve tu panel">
            <div class="landing-alarm">
              <strong>2 ATRASADAS</strong>
              <span>3 salen hoy · 14 en producción</span>
            </div>
            <OrderTicket code="OT-000031" product="240 polos box cuello redondo" state="LATE" :completed="1" station="Costura" client="Moda Lucía" due="vence 19/10" />
            <OrderTicket code="OT-000034" product="80 casacas drill" state="AT_RISK" :completed="3" station="Control de calidad" client="Urbano PE" due="vence 21/10" />
            <OrderTicket code="OT-000029" product="500 poleras french terry" state="ON_TIME" :completed="sewn" station="Empaque" client="Kuna Store" due="entrega 23/10" />
          </div>
        </div>
      </section>

      <section class="landing-problem" aria-labelledby="problem-title">
        <div class="landing-wrap landing-split">
          <div>
            <h2 id="problem-title">El atraso se descubre el día de la entrega.</h2>
            <p class="landing-lead">En la mayoría de talleres, cada pedido vive repartido entre WhatsApp, llamadas, el cuaderno y la memoria de alguien.</p>
          </div>
          <dl class="landing-rows">
            <div><dt>Sin visibilidad</dt><dd>No sabes en qué estación va cada orden hasta que preguntas.</dd></div>
            <div><dt>Llamadas</dt><dd>El cliente escribe "¿cómo va mi pedido?" y alguien deja de producir para responder.</dd></div>
            <div><dt>Doble registro</dt><dd>Lo anotado en el cuaderno no coincide con lo que se dijo por WhatsApp.</dd></div>
            <div><dt>Sin evidencia</dt><dd>Ante un reclamo, no hay cómo mostrar el estado de la prenda en cada etapa.</dd></div>
          </dl>
        </div>
      </section>

      <section id="como-funciona" class="landing-how" aria-labelledby="how-title">
        <div class="landing-wrap">
          <h2 id="how-title">Una ficha viaja con las prendas. TallerFlow la sigue.</h2>
          <ol class="landing-stations" aria-label="Las 7 estaciones">
            <li v-for="name in stations" :key="name">{{ name }}</li>
          </ol>
          <div class="landing-steps">
            <div>
              <h3>Registras la orden</h3>
              <p>Cliente, prenda, cantidad y fecha prometida. TallerFlow le asigna un código como <code>OT-000003</code> e imprimes su ficha.</p>
            </div>
            <div>
              <h3>El operario avanza con su celular</h3>
              <p>Escanea la ficha, toca "Pasar a…" y sube una foto como evidencia. Sin capacitaciones largas.</p>
            </div>
            <div>
              <h3>Tú ves el semáforo</h3>
              <p>Cada orden aparece a tiempo, en riesgo o atrasada. Detectas el retraso cuando todavía hay margen.</p>
            </div>
          </div>
        </div>
      </section>

      <section id="seguimiento" class="landing-tracking" aria-labelledby="tracking-title">
        <div class="landing-wrap landing-split landing-split--center">
          <div>
            <h2 id="tracking-title">Menos "¿cómo va mi pedido?"</h2>
            <p class="landing-lead">Comparte un link privado por orden. Tu cliente ve la estación actual, la fecha prometida y las fotos del avance, sin instalar nada ni crear cuenta.</p>
          </div>
          <div class="landing-phone" aria-label="Así lo ve tu cliente">
            <p class="landing-phone__made"><span>Hecho en</span> <strong>Confecciones Lucía</strong></p>
            <p class="landing-phone__eyebrow">OT-000034 · Urbano PE</p>
            <p class="landing-phone__station">Costura</p>
            <StitchSeam :completed="2" />
            <div class="landing-phone__facts">
              <span>2 de 7 estaciones</span>
              <span>Entrega <b>vie 24/10</b></span>
            </div>
            <StatusChip state="ON_TIME" />
          </div>
        </div>
      </section>

      <section class="landing-roles" aria-labelledby="roles-title">
        <div class="landing-wrap">
          <h2 id="roles-title">Una sola fuente de verdad para todo el taller.</h2>
          <div class="landing-steps">
            <div><h3>Dueño o administrador</h3><p>Un panel que dice cuántas órdenes van atrasadas, cuáles salen hoy y cuánta carga tiene cada estación.</p></div>
            <div><h3>Operario</h3><p>Una pantalla simple en el celular para avanzar la orden, subir una foto y escanear la ficha.</p></div>
            <div><h3>Cliente final</h3><p>Un link privado con la estación, la fecha prometida y las fotos de avance.</p></div>
          </div>
        </div>
      </section>

      <section class="landing-security" aria-labelledby="security-title">
        <div class="landing-wrap landing-split">
          <h2 id="security-title">Cada taller ve solo sus datos.</h2>
          <ul>
            <li>Aislamiento por taller en la base de datos, incluso ante un error en el código.</li>
            <li>Links de seguimiento con códigos aleatorios: sin el link exacto, nadie ve la orden.</li>
            <li>Las acciones sensibles, como corregir un avance o cambiar un rol, quedan registradas.</li>
          </ul>
        </div>
      </section>

      <section class="landing-final" aria-labelledby="final-title">
        <div class="landing-wrap">
          <h2 id="final-title">Deja el cuaderno. Sigue cada orden desde el celular.</h2>
          <p class="landing-lead">Web y app Android. No instalas nada en el taller.</p>
          <div class="landing-actions">
            <RouterLink class="landing-button landing-button--primary" to="/onboarding">Registrar mi taller</RouterLink>
            <RouterLink class="landing-button landing-button--secondary" to="/login">Ya tengo cuenta</RouterLink>
          </div>
        </div>
      </section>
    </main>

    <footer class="landing-footer">
      <div class="landing-wrap landing-footer__inner">
        <BrandLogo :size="24" />
        <nav class="landing-footer__links" aria-label="Legal y contacto">
          <RouterLink to="/terminos">Términos de servicio</RouterLink>
          <RouterLink to="/privacidad">Política de privacidad</RouterLink>
          <a href="mailto:soporte@tallerflowpe.com">soporte@tallerflowpe.com</a>
        </nav>
        <p>Lima, Perú · © 2026 TallerFlow</p>
      </div>
    </footer>
  </div>
</template>

<style scoped>
.landing { min-height: 100vh; color: var(--tf-ink); background: var(--tf-page); font-family: var(--tf-font-body); }
.landing-wrap { width: 100%; max-width: 1160px; margin: 0 auto; padding: 0 clamp(1rem, 4vw, 1.5rem); }
.landing h1, .landing h2, .landing h3 { margin: 0; color: var(--tf-ink); font-family: var(--tf-font-display); }
.landing h1, .landing h2 { font-variation-settings: "wdth" 125; text-wrap: balance; }
.landing h2 { font-size: clamp(1.6rem, 3.4vw, 2.4rem); font-weight: 800; line-height: 1.1; }
.landing h3 { font-size: 1.15rem; font-weight: 800; line-height: 1.25; }
.landing p { margin: 0; }
.landing section[id] { scroll-margin-top: 4.75rem; }
.landing-lead { margin-top: 1rem !important; max-width: 34rem; color: var(--tf-text-subtle); font-size: 1.06rem; line-height: 1.6; text-wrap: pretty; }

/* Header */
.landing-header { position: sticky; top: 0; z-index: 40; border-bottom: 1px solid var(--tf-border-soft); background: var(--tf-page); }
.landing-header__inner { display: flex; align-items: center; justify-content: space-between; gap: 1rem; min-height: 4.25rem; }
.landing-header__brand { display: inline-flex; align-items: center; min-height: 44px; border-radius: var(--tf-radius); }
.landing-nav { display: flex; align-items: center; gap: 1.5rem; }
.landing-nav a { color: var(--tf-text-subtle); font-size: .95rem; text-decoration: none; white-space: nowrap; }
.landing-nav a:hover { color: var(--tf-ink); }
.landing-nav .landing-nav__login { display: inline-flex; align-items: center; min-height: 44px; color: var(--tf-primary); font-weight: 700; }

/* Buttons */
.landing-button { display: inline-flex; align-items: center; justify-content: center; min-height: 48px; padding: 0 1.4rem; border: 1.5px solid transparent; border-radius: var(--tf-radius); font-size: 1rem; font-weight: 700; text-decoration: none; white-space: nowrap; transition: background-color .15s ease-out, border-color .15s ease-out; }
.landing-nav .landing-button--primary, .landing-button--primary { color: var(--tf-on-primary); background: var(--tf-primary); }
.landing-nav .landing-button--primary:hover, .landing-button--primary:hover { color: var(--tf-on-primary); background: var(--tf-primary-hover); }
.landing-button--secondary { border-color: var(--tf-primary); color: var(--tf-primary); background: var(--tf-surface); }
.landing-button--secondary:hover { background: var(--tf-primary-soft); }
.landing a:focus-visible { outline: 3px solid var(--tf-focus); outline-offset: 2px; }
.landing-actions { display: flex; flex-wrap: wrap; gap: .75rem; margin-top: 2rem; }

/* Hero */
.landing-hero { padding: clamp(2.5rem, 7vw, 4.5rem) 0 clamp(3rem, 8vw, 5.5rem); }
.landing-hero__inner { display: grid; grid-template-columns: repeat(auto-fit, minmax(min(100%, 26rem), 1fr)); gap: clamp(2rem, 5vw, 3.5rem); align-items: center; }
.landing-hero h1 { font-size: clamp(2.2rem, 5.6vw, 3.9rem); font-weight: 900; line-height: 1.02; letter-spacing: -.005em; }
.landing-hero__copy > p { margin-top: 1.5rem; max-width: 32rem; color: var(--tf-text-subtle); font-size: clamp(1.05rem, 2vw, 1.2rem); line-height: 1.55; text-wrap: pretty; }
.landing-hero__panel { display: grid; gap: .6rem; width: 100%; max-width: 28rem; justify-self: center; }
.landing-alarm { display: grid; gap: .25rem; padding: 1.1rem 1.25rem; border-radius: var(--tf-radius-sm); color: var(--tf-on-late); background: var(--tf-late); }
.landing-alarm strong { font-family: var(--tf-font-display); font-size: clamp(1.55rem, 4vw, 2rem); font-variation-settings: "wdth" 125; font-weight: 900; line-height: 1; }
.landing-alarm span { font-size: .95rem; }

/* Problem */
.landing-problem { padding: clamp(3rem, 8vw, 5rem) 0; border-block: 1px solid var(--tf-border-soft); background: var(--tf-surface); }
.landing-split { display: grid; grid-template-columns: repeat(auto-fit, minmax(min(100%, 22rem), 1fr)); gap: clamp(2rem, 5vw, 3rem); }
.landing-split--center { align-items: center; }
.landing-rows { margin: 0; }
.landing-rows div { display: grid; grid-template-columns: minmax(8rem, 10rem) 1fr; gap: 1rem; padding: 1.1rem 0; border-top: 1px solid var(--tf-border-soft); }
.landing-rows div:last-child { border-bottom: 1px solid var(--tf-border-soft); }
.landing-rows dt { font-weight: 700; }
.landing-rows dd { margin: 0; color: var(--tf-text-subtle); line-height: 1.5; }

/* How it works: the 7 stations on a navy line, the thread marking the start */
.landing-how, .landing-roles { padding: clamp(3rem, 8vw, 5.5rem) 0; }
.landing-how h2, .landing-roles h2 { max-width: 40rem; }
.landing-stations { display: grid; grid-template-columns: repeat(7, minmax(0, 1fr)); gap: .5rem; margin: 2.5rem 0 0; padding: 1rem 0 0; border-top: 3px solid var(--tf-primary); list-style: none; counter-reset: station; }
.landing-stations li { position: relative; counter-increment: station; font-family: var(--tf-font-display); font-size: 1rem; font-variation-settings: "wdth" 112; font-weight: 800; line-height: 1.2; }
.landing-stations li::before { content: counter(station, decimal-leading-zero); display: block; margin-bottom: .25rem; color: var(--tf-text-subtle); font-family: var(--tf-font-mono); font-size: .8rem; font-weight: 400; }
.landing-stations li:first-child::after { content: ""; position: absolute; top: calc(-1rem - 3px); left: 0; width: 100%; height: 3px; background: var(--tf-accent); }
.landing-steps { display: grid; grid-template-columns: repeat(auto-fit, minmax(min(100%, 16rem), 1fr)); gap: 2rem; margin-top: 3rem; }
.landing-steps > div { display: grid; gap: .6rem; align-content: start; padding-top: 1.1rem; border-top: 1px solid var(--tf-border-soft); }
.landing-steps p { color: var(--tf-text-subtle); line-height: 1.55; }
.landing-steps code { color: var(--tf-ink); font-family: var(--tf-font-mono); }

/* Tracking: the navy band with the turquoise selvedge */
.landing-tracking { padding: clamp(3rem, 8vw, 5.5rem) 0; border-bottom: 4px solid var(--tf-accent); background: var(--tf-shell); }
.landing-tracking h2 { color: var(--tf-on-shell); }
.landing-tracking .landing-lead { color: var(--tf-on-shell-muted); }
.landing-phone { display: grid; gap: .8rem; width: 100%; max-width: 24rem; justify-self: center; padding: 1.5rem; border-radius: var(--tf-radius); background: var(--tf-surface); }
.landing-phone__made { display: flex; flex-wrap: wrap; align-items: center; gap: .45rem; color: var(--tf-text-subtle); font-size: .8rem; }
.landing-phone__made strong { padding: .2rem .6rem .15rem; border-bottom: 2px solid var(--tf-accent); border-radius: var(--tf-radius-sm); color: var(--tf-on-primary); background: var(--tf-primary); font-family: var(--tf-font-display); font-size: .72rem; letter-spacing: .07em; text-transform: uppercase; }
.landing-phone__eyebrow { color: var(--tf-text-subtle); font-family: var(--tf-font-mono); font-size: .85rem; }
.landing-phone__station { color: var(--tf-primary); font-family: var(--tf-font-display); font-size: 2.4rem; font-variation-settings: "wdth" 125; font-weight: 900; line-height: 1; }
.landing-phone__facts { display: flex; flex-wrap: wrap; justify-content: space-between; gap: .5rem; color: var(--tf-text-subtle); font-size: .9rem; }
.landing-phone__facts b { color: var(--tf-ink); font-family: var(--tf-font-mono); }
.landing-phone :deep(.status-chip) { justify-self: start; }

/* Security */
.landing-security { padding: clamp(3rem, 7vw, 4.5rem) 0; border-block: 1px solid var(--tf-border-soft); background: var(--tf-surface); }
.landing-security h2 { font-size: clamp(1.4rem, 2.8vw, 2rem); }
.landing-security ul { display: grid; gap: .9rem; margin: 0; padding-left: 1.1rem; color: var(--tf-text-subtle); line-height: 1.5; }

/* Final CTA */
.landing-final { padding: clamp(4rem, 10vw, 6rem) 0; }
.landing-final h2 { max-width: 46rem; font-size: clamp(2rem, 5vw, 3.4rem); font-weight: 900; line-height: 1.04; }

/* Footer */
.landing-footer { border-top: 1px solid var(--tf-border-soft); background: var(--tf-surface); }
.landing-footer__inner { display: flex; flex-wrap: wrap; align-items: center; justify-content: space-between; gap: 1rem 2rem; padding-block: 1.75rem; color: var(--tf-text-subtle); font-size: .9rem; }
.landing-footer__links { display: flex; flex-wrap: wrap; gap: .25rem 1.5rem; }
.landing-footer__links a { display: inline-flex; align-items: center; min-height: 44px; color: var(--tf-text-subtle); text-decoration: underline; text-decoration-color: var(--tf-accent); text-decoration-thickness: 2px; text-underline-offset: 3px; }
.landing-footer__links a:hover { color: var(--tf-primary); }

@media (max-width: 899px) {
  .landing-nav .landing-nav__section { display: none; }
  .landing-stations { grid-template-columns: repeat(auto-fit, minmax(7.5rem, 1fr)); row-gap: 1rem; }
}
/* Below ~560px the logo, "Ingresar" and the CTA do not fit; the hero repeats the CTA right below. */
@media (max-width: 559px) {
  .landing-nav .landing-nav__signup { display: none; }
  .landing-rows div { grid-template-columns: 1fr; gap: .25rem; }
}
</style>
