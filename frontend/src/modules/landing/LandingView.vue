<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue'
import { ArrowLeft, ArrowRight, Check, ClipboardList, Camera, Link2, LayoutDashboard, Columns3, Smartphone, ScanLine, Play, X } from '@lucide/vue'
import BrandLogo from '../../components/BrandLogo.vue'
import UiDialog from '../../components/UiDialog.vue'
const playURL='https://play.google.com/store/apps/details?id=pe.tallerflow.app&pcampaignid=web_share'
const screen=ref(0)
const enlarged=ref(false)
const androidOpen = ref(false)
const screens=[
 {title:'Panel del taller',description:'Identifica las entregas próximas, las órdenes atrasadas y la carga de trabajo por etapa.',image:'/product/panel-escritorio.jpg',icon:LayoutDashboard},
 {title:'Tablero de órdenes',description:'Consulta cada pedido en su etapa y registra el avance de producción.',image:'/product/tablero-escritorio.jpg',icon:Columns3},
 {title:'Seguimiento del cliente',description:'Tu cliente consulta el avance y las fotografías con un enlace privado, sin crear una cuenta.',image:'/product/seguimiento-celular.jpg',icon:Smartphone},
]
const current=computed(()=>screens[screen.value]!)
function moveScreen(delta: number, focusTab = false) {
  screen.value = (screen.value + delta + screens.length) % screens.length
  if (focusTab) nextTick(() => document.getElementById(`screen-tab-${screen.value}`)?.focus())
}
const gallery = ref<HTMLElement | null>(null)
const inView = ref(false)
const hovered = ref(false)
const focusWithin = ref(false)
const autoplay = ref(false)
const pageVisible = ref(true)
const canRotate = computed(() =>
  autoplay.value && inView.value && pageVisible.value && !hovered.value && !focusWithin.value && !enlarged.value,
)

let observer: IntersectionObserver | undefined
let motionPreference: MediaQueryList | undefined
function syncMotionPreference() { autoplay.value = !motionPreference?.matches }
function syncPageVisibility() { pageVisible.value = document.visibilityState === 'visible' }
function leaveGallery(event: FocusEvent) {
  if (!gallery.value?.contains(event.relatedTarget as Node | null)) focusWithin.value = false
}

onMounted(() => {
  motionPreference = window.matchMedia('(prefers-reduced-motion: reduce)')
  syncMotionPreference()
  syncPageVisibility()
  motionPreference.addEventListener('change', syncMotionPreference)
  document.addEventListener('visibilitychange', syncPageVisibility)
  if (!gallery.value) return
  observer = new IntersectionObserver(([entry]) => {
    inView.value = entry?.isIntersecting ?? false
  }, { threshold: 0.4 })
  observer.observe(gallery.value)
})
onUnmounted(() => {
  observer?.disconnect()
  motionPreference?.removeEventListener('change', syncMotionPreference)
  document.removeEventListener('visibilitychange', syncPageVisibility)
})
watch([canRotate, screen], ([rotate], _, onCleanup) => {
  if (!rotate) return
  const timer = window.setTimeout(() => moveScreen(1), 6500)
  onCleanup(() => window.clearTimeout(timer))
})
const steps=[{title:'Registra el pedido',text:'Cliente, prendas, cantidad y fecha de entrega, en una sola orden.',icon:ClipboardList},{title:'Registra el avance',text:'El equipo completa las etapas y adjunta fotografías desde el celular.',icon:Camera},{title:'Comparte el seguimiento',text:'Copia el enlace de la orden y envíalo al cliente por el medio que prefieras.',icon:Link2}]
</script>
<template>
 <div class="landing">
  <header class="landing-header"><div class="landing-wrap nav-inner"><RouterLink to="/" aria-label="TallerFlow, inicio"><BrandLogo :size="28" tone="dark" decorative/></RouterLink><nav aria-label="Secciones"><a class="nav-section" href="#producto">El sistema</a><a class="nav-section" href="#como-funciona">Cómo funciona</a><a class="nav-section" href="#android">App Android</a><RouterLink class="nav-login" to="/login">Ingresar</RouterLink><RouterLink class="landing-button landing-button--primary" to="/onboarding"><span class="nav-register-desktop">Registrar mi taller</span><span class="nav-register-mobile">Crear taller</span> <ArrowRight :size="16"/></RouterLink></nav></div></header>
  <main>
   <section class="hero"><div class="landing-wrap hero-grid"><div class="hero-copy"><p class="hero-category"><span/>Para talleres de confección</p><h1>Gestiona los pedidos de tu taller, de corte a entrega.</h1><p class="hero-lead">Mira qué debe salir primero, registra el avance con fotos y comparte el estado del pedido con tu cliente.</p><div class="hero-actions"><RouterLink class="landing-button landing-button--light" to="/onboarding">Registrar mi taller <ArrowRight :size="18"/></RouterLink><a class="hero-demo" href="#producto"><Play :size="16"/>Conocer el sistema</a></div></div><div class="hero-product"><div class="desktop-frame"><div class="window-bar"><i/><i/><i/><span>TallerFlow · Panel del taller</span></div><img :src="'/product/panel-escritorio.jpg'" width="1440" height="1050" fetchpriority="high" alt="Panel real de TallerFlow con estado de los pedidos y próximas entregas"/></div><div class="hero-phone"><div class="phone-camera"/><img :src="'/product/seguimiento-celular.jpg'" width="390" height="960" alt="Seguimiento real de un pedido en un celular"/></div></div></div></section>

   <section id="producto" class="product-section landing-wrap">
    <div class="section-heading">
      <h2>Qué puedes ver en TallerFlow</h2>
      <p>Consulta qué está pendiente, dónde está cada pedido y qué debe salir primero.</p>
    </div>
    <div ref="gallery" @mouseenter="hovered = true" @mouseleave="hovered = false" @focusin="focusWithin = true" @focusout="leaveGallery">
      <div class="gallery-toolbar">
        <div class="product-tabs" role="tablist" aria-label="Pantallas del producto">
          <button v-for="(item,index) in screens" :id="`screen-tab-${index}`" :key="item.title" type="button" role="tab"
            :aria-selected="screen === index" :tabindex="screen === index ? 0 : -1" aria-controls="screen-panel"
            @click="screen = index" @keydown.right.prevent="moveScreen(1, true)" @keydown.left.prevent="moveScreen(-1, true)">
            <component :is="item.icon" :size="18" aria-hidden="true" />{{ item.title }}
          </button>
        </div>
      </div>
      <div id="screen-panel" class="product-preview" role="tabpanel" :aria-labelledby="`screen-tab-${screen}`">
        <div class="preview-copy">
          <h3>{{ current.title }}</h3>
          <p>{{ current.description }}</p>
          <div class="preview-controls" aria-label="Cambiar pantalla">
            <button type="button" aria-label="Pantalla anterior" @click="moveScreen(-1)"><ArrowLeft :size="18" aria-hidden="true" /></button>
            <span :aria-live="autoplay ? 'off' : 'polite'">{{ screen + 1 }} / {{ screens.length }}</span>
            <button type="button" aria-label="Pantalla siguiente" @click="moveScreen(1)"><ArrowRight :size="18" aria-hidden="true" /></button>
          </div>
        </div>
        <button type="button" :class="['preview-image', { 'preview-image--phone': screen === 2 }]"
          :aria-label="`Ampliar ${current.title}`" @click="enlarged = true">
          <img :key="current.image" :src="current.image" :alt="current.title + ' de TallerFlow'" loading="lazy" />
        </button>
      </div>
    </div>
   </section>
   <section id="como-funciona" class="how-section"><div class="landing-wrap"><div class="section-heading"><div><h2>Así pasa un pedido por tu taller</h2></div><p>Configura las etapas según la forma de trabajar de tu taller.</p></div><div class="how-grid"><article v-for="(step,index) in steps" :key="step.title"><div class="step-top"><span class="step-icon"><component :is="step.icon" :size="25"/></span><span class="step-number">{{index+1}}</span></div><h3>{{step.title}}</h3><p>{{step.text}}</p></article></div></div></section>
   <section class="photo-section landing-wrap"><div class="workshop-photo"><img :src="'/product/taller-costura.jpg'" alt="Máquina de coser y tela en un espacio de confección" loading="lazy" width="1400" height="930"/></div><div class="photo-copy"><h2>Una sola ficha para todo el equipo</h2><p>Una orden reúne las especificaciones, el avance y las fotografías. Cada miembro del equipo accede según su rol.</p><ul><li><Check :size="18"/>Propietario, administrador y operador.</li><li><Check :size="18"/>Responsable y pausas de producción.</li><li><Check :size="18"/>Historial de cambios y exportación a CSV.</li></ul><RouterLink to="/onboarding" class="text-link">Organizar mi taller <ArrowRight :size="17"/></RouterLink></div></section>
   <section id="android" class="android-section">
    <div class="landing-wrap android-grid">
      <div class="android-copy">
        <h2>Actualiza los pedidos desde tu celular</h2>
        <ul class="android-features">
          <li><ScanLine :size="22" aria-hidden="true" />Escanea la ficha para abrir la orden</li>
          <li><Camera :size="22" aria-hidden="true" />Registra el avance con fotos</li>
        </ul>
        <div class="android-store">
          <img class="store-icon" :src="'/android-chrome-192x192.png'" alt="" width="48" height="48" loading="lazy" />
          <div><strong>TallerFlow</strong><span>Aplicación para Android</span></div>
          <a :href="playURL" class="android-badge" target="_blank" rel="noopener noreferrer" aria-label="Ver TallerFlow en Google Play">
            <img :src="'/product/google-play-es.png'" alt="" width="160" height="48" loading="lazy" />
          </a>
        </div>
      </div>
      <figure class="android-visual">
        <button type="button" class="android-phone" aria-label="Ampliar pantalla de la app Android" @click="androidOpen = true">
          <img :src="'/product/ordenes-android.jpg'" alt="Lista de órdenes de Confecciones Ladera en TallerFlow" width="540" height="1093" loading="lazy" />
        </button>
      </figure>
    </div>
   </section>
  </main>
  <footer class="landing-footer">
    <div class="landing-wrap footer-grid">
      <div class="footer-brand">
        <BrandLogo :size="26" tone="dark"/>
        <p>Gestión de pedidos para talleres de confección.</p>
      </div>
      <nav aria-label="Producto">
        <h2>Producto</h2>
        <a href="#producto">El sistema</a>
        <a href="#como-funciona">Cómo funciona</a>
        <a href="#android">App Android</a>
      </nav>
      <nav aria-label="Ayuda y privacidad">
        <h2>Ayuda y privacidad</h2>
        <a href="mailto:tallerflow.pe@gmail.com">Contactar soporte</a>
        <RouterLink to="/privacidad">Política de privacidad</RouterLink>
        <RouterLink to="/terminos">Términos del servicio</RouterLink>
      </nav>
    </div>
    <div class="landing-wrap footer-base"><small>© 2026 TallerFlow · Perú</small></div>
  </footer>
  <UiDialog v-if="enlarged" :title="current.title" wide @close="enlarged=false"><img class="enlarged-screen" :src="current.image" :alt="current.title"/></UiDialog>
 </div>
  <UiDialog v-if="androidOpen" title="Órdenes en la app Android" @close="androidOpen = false">
    <img class="enlarged-screen" :src="'/product/ordenes-android.jpg'" alt="Lista de órdenes de Confecciones Ladera en TallerFlow" />
  </UiDialog>
</template>
<style scoped>
.landing{color:#17304d;background:white;font-family:var(--tf-font-body)}.landing-wrap{width:min(1240px,100%);padding-inline:clamp(1rem,4vw,2rem);margin-inline:auto}.landing h1,.landing h2,.landing h3{font-family:var(--tf-font-body);font-variation-settings:normal;letter-spacing:-.025em}.landing h2{font-size:clamp(1.8rem,3.5vw,2.7rem);line-height:1.15;margin:0;font-weight:700}.landing p{line-height:1.7}.landing-header{background:white;border-bottom:1px solid #e4eaf1;position:relative;z-index:5}.nav-inner{display:flex;align-items:center;justify-content:space-between;min-height:82px;gap:1rem}.nav-inner>a{display:flex;align-items:center}.nav-inner nav{display:flex;align-items:center;gap:1.75rem}.nav-inner a{color:#51657c;text-decoration:none;font-size:.875rem}.nav-inner .nav-login{font-weight:650;color:#143d70}.landing-button{display:inline-flex;align-items:center;justify-content:center;gap:.65rem;padding:.85rem 1.2rem;min-height:48px;border:1px solid transparent;border-radius:12px;font-weight:650;text-decoration:none;font-size:.9rem;cursor:pointer}.landing-button--primary{background:#103c73;color:white!important}.landing-button--primary:hover{background:#1c518d;transform:translateY(-1px)}.landing-button--light{background:white;color:#103c73}.hero{background:#092b55;color:white;overflow:hidden}.hero-grid{display:grid;grid-template-columns:1fr 1.2fr;align-items:center;gap:3rem;padding-top:5rem;padding-bottom:4rem}.hero-copy h1{font-size:clamp(2.5rem,4.6vw,4rem);font-weight:700;line-height:1.08;margin:1.25rem 0 1.4rem;color:white;letter-spacing:-.035em}.hero-category{display:flex;align-items:center;gap:.55rem;font-size:.8rem;color:#b9e8e3;margin:0}.hero-category>span{width:7px;height:7px;background:#58d5c4;border-radius:50%}.hero-lead{color:#c4d4e7;font-size:1rem;max-width:440px;margin:0}.hero-actions{display:flex;gap:1.25rem;align-items:center;flex-wrap:wrap;margin-top:2rem}.hero-demo{display:inline-flex;align-items:center;gap:.5rem;color:white;font-size:.875rem;text-decoration:none;min-height:44px}.hero-platforms{display:flex;gap:.5rem;align-items:center;color:#a9c0db;font-size:.75rem;margin-top:1.4rem}.hero-platforms>span{width:3px;height:3px;background:currentColor;border-radius:50%;margin:0 .25rem}.hero-product{position:relative;min-width:0;padding-bottom:3rem;margin-right:-3rem}.desktop-frame{border:6px solid #35618a;border-radius:17px;background:#f3f6fa;overflow:hidden;box-shadow:0 25px 55px #00152b60}.window-bar{display:flex;gap:5px;align-items:center;padding:9px 12px;background:#e7edf4}.window-bar i{width:6px;height:6px;background:#aebdd0;border-radius:50%}.window-bar span{font-size:8px;letter-spacing:.02em;color:#5c6f88;margin:auto}.desktop-frame>img{display:block;width:100%;height:auto}.hero-phone{position:absolute;bottom:-1rem;right:1rem;width:148px;max-height:360px;border:5px solid #152132;border-radius:27px;overflow:hidden;background:white;box-shadow:0 15px 35px #00152b50}.hero-phone img{width:100%;height:auto;display:block}.phone-camera{position:absolute;z-index:1;width:38%;height:10px;border-radius:0 0 12px 12px;background:#152132;left:31%;top:0}.capture-caption{font-size:.65rem;color:#99b4d2;margin:1rem 160px 0 0;text-align:center}.value-strip{border-bottom:1px solid #e4eaf1;background:#f8fafc}.value-strip>.landing-wrap{display:flex;align-items:center;justify-content:space-between;gap:1rem;min-height:86px}.value-strip span{display:flex;align-items:center;gap:.7rem;font-size:.85rem;font-weight:600}.value-strip svg{color:#168271}.product-section{padding-block:5.5rem}.section-heading{display:flex;justify-content:space-between;align-items:end;gap:2rem;margin-bottom:2rem}.section-heading>p{color:#62728a;font-size:.95rem;max-width:340px;margin:0}.section-label{color:#168271;font-size:.8rem;font-weight:650;margin:0 0 .8rem}.product-tabs{display:flex;gap:.5rem;flex-wrap:wrap;margin-bottom:1rem}.product-tabs button{display:flex;align-items:center;gap:.55rem;min-height:46px;padding:.6rem 1rem;border:1px solid #e0e7ef;border-radius:11px;color:#5e6e82;background:white;font:inherit;font-size:.85rem;cursor:pointer}.product-tabs button[aria-selected="true"]{background:#103c73;color:white;border-color:#103c73}.product-preview{display:grid;grid-template-columns:.65fr 1.35fr;border:1px solid #e0e7ef;border-radius:22px;overflow:hidden;background:#f5f8fb;min-height:400px}.preview-copy{padding:2.5rem;display:flex;flex-direction:column;justify-content:center}.preview-copy h3{font-size:1.4rem;margin:0}.preview-copy p{font-size:.95rem;color:#62728a}.preview-enlarge,.text-link{display:inline-flex;align-items:center;gap:.5rem;border:0;background:transparent;padding:.5rem 0;color:#103c73;font:inherit;font-size:.85rem;font-weight:650;text-decoration:none;cursor:pointer;align-self:start;min-height:44px}.preview-image{display:block;padding:2rem 0 0 2rem;border:0;background:#e8eef6;cursor:zoom-in;min-width:0;overflow:hidden}.preview-image img{display:block;width:100%;height:auto;border-radius:12px 0 0 0;box-shadow:0 6px 25px #16395b20}.preview-image--phone{padding:1.5rem;max-height:500px}.preview-image--phone img{width:220px;margin:auto;border:5px solid #152132;border-radius:23px}.demo-note{font-size:.72rem;color:#708096;margin:.7rem 0}.how-section{padding:5rem 0;background:#f6f9fc}.how-grid{display:grid;grid-template-columns:repeat(3,minmax(0,1fr));gap:1.25rem}.how-grid article{padding:1.75rem;background:white;border:1px solid #e3eaf2;border-radius:18px}.step-top{display:flex;align-items:center;justify-content:space-between}.step-icon{width:50px;height:50px;display:grid;place-items:center;background:#e7f4f1;color:#127a66;border-radius:14px}.step-number{color:#b5c3d2;font-size:.9rem}.how-grid h3{font-size:1.1rem;margin:1.5rem 0 .6rem}.how-grid p{color:#62728a;font-size:.9rem;margin:0}.photo-section{display:grid;grid-template-columns:1.05fr 1fr;align-items:center;gap:4rem;padding-block:5.5rem}.workshop-photo{position:relative;border-radius:22px;overflow:hidden;aspect-ratio:1/1}.workshop-photo img{width:100%;height:100%;object-fit:cover}.workshop-photo>span{position:absolute;bottom:1rem;left:1rem;font-size:.65rem;color:white;background:#10263bb3;border-radius:6px;padding:.35rem .5rem}.photo-copy>p:not(.section-label){color:#62728a;font-size:.95rem}.photo-copy ul{padding:0;list-style:none;display:grid;gap:.75rem;margin:1.5rem 0}.photo-copy li{display:flex;gap:.7rem;align-items:center;font-size:.85rem;color:#4b6179}.photo-copy li svg{color:#168271;flex:none}.android-section{background:#eef7f5;padding:4rem 0;overflow:hidden}.android-grid{display:grid;grid-template-columns:.8fr 1.2fr;align-items:center;gap:5rem}.android-phone{position:relative;width:244px;max-height:420px;border:7px solid #152132;border-bottom:0;border-radius:30px 30px 0 0;background:white;overflow:hidden;margin:auto auto -4rem;box-shadow:0 10px 45px #123c5025}.android-phone img{display:block;width:100%;height:auto}.android-grid>div>p:not(.section-label){max-width:460px;color:#536d79;font-size:.95rem}.play-button{display:inline-flex;align-items:center;gap:.9rem;padding:.75rem 1.25rem;border-radius:13px;background:#102b46;color:white;text-decoration:none;margin-top:1rem}.play-button>svg{width:27px;height:30px}.play-button>svg:last-child{width:18px}.play-button span{display:grid;font-size:.65rem;line-height:1.5}.play-button strong{font-size:1.2rem;font-weight:600}.android-note{font-size:.75rem!important}.final-section{padding-block:5rem;display:flex;align-items:center;justify-content:space-between;gap:2rem}.final-section>div>svg{color:#168271;margin-bottom:1rem}.final-section p{color:#62728a;font-size:.9rem}.final-actions{display:grid;justify-items:center;gap:1rem}.final-actions>a:last-child{font-size:.85rem;color:#62728a;text-decoration:none}.landing-footer{border-top:1px solid #e3eaf2;background:#f9fbfd;padding:1.7rem 0}.footer-inner{display:flex;align-items:center;justify-content:space-between;gap:1rem;flex-wrap:wrap}.footer-inner nav{display:flex;gap:1rem;flex-wrap:wrap}.footer-inner a,.footer-inner>span{font-size:.75rem;color:#62728a;text-decoration:none}.enlarged-screen{display:block;width:100%;max-height:70vh;object-fit:contain}.landing a:focus-visible,.landing button:focus-visible{outline:3px solid #4da7ce;outline-offset:4px}
@media(max-width:1050px){.hero-grid{gap:2rem}.hero-product{margin-right:-2rem}.hero-phone{width:125px;right:.5rem;bottom:0}.capture-caption{margin-right:135px}.nav-inner nav{gap:1rem}.photo-section{gap:2.5rem}.preview-copy{padding:1.5rem}}
@media(max-width:760px){.nav-inner{min-height:72px}.nav-inner .nav-section{display:none}.nav-inner nav{gap:.8rem}.nav-inner .landing-button{font-size:.75rem;padding:.6rem .8rem;min-height:42px}.hero-grid{grid-template-columns:1fr;padding-top:2.8rem;padding-bottom:3rem;gap:3rem}.hero-copy h1{font-size:clamp(2.4rem,9vw,3.2rem);margin:1rem 0}.hero-lead{font-size:.95rem}.hero-product{margin:0 .75rem 0 0;padding-bottom:2rem}.hero-phone{width:112px;max-height:260px;right:-.5rem;bottom:-.5rem;border-width:4px;border-radius:23px}.capture-caption{margin-right:115px;font-size:.6rem}.value-strip>.landing-wrap{flex-wrap:wrap;justify-content:center;gap:.8rem 1.5rem;padding-block:1.3rem}.value-strip span{font-size:.75rem}.value-strip svg{width:18px}.product-section,.how-section{padding-block:3rem}.section-heading{display:block}.section-heading>p{margin-top:1rem}.product-tabs{gap:.4rem}.product-tabs button{font-size:.75rem;padding:.5rem .7rem}.product-tabs svg{width:15px}.product-preview{grid-template-columns:1fr;border-radius:16px;min-height:0}.preview-copy{padding:1.25rem}.preview-copy h3{font-size:1.2rem}.preview-copy p{font-size:.85rem}.preview-image{padding:1rem 0 0 1rem}.preview-image--phone{padding:1rem;max-height:420px}.preview-image--phone img{width:180px}.how-grid{grid-template-columns:1fr;gap:.75rem}.how-grid article{padding:1.3rem}.how-grid h3{margin-top:1rem}.photo-section{grid-template-columns:1fr;padding-block:3rem;gap:2rem}.workshop-photo{aspect-ratio:4/3}.android-grid{display:flex;flex-direction:column-reverse;gap:2rem;align-items:start}.android-grid>div{min-width:0}.android-phone{margin:0 auto -4rem;max-height:340px;width:210px}.final-section{padding-block:3rem;align-items:start;flex-direction:column}.final-actions{justify-items:start}.footer-inner{display:grid;gap:1.25rem}.hero-platforms{font-size:.65rem;gap:.35rem}.hero-actions{gap:1rem}.hero-demo{font-size:.8rem}}

.nav-register-mobile{display:none}@media(max-width:760px){.nav-register-desktop{display:none}.nav-register-mobile{display:inline}.nav-inner nav>.landing-button>svg{display:none}.nav-inner .landing-button{white-space:nowrap}}

.landing-header {
  position: sticky;
  top: 0;
  z-index: 50;
  background: #f7faff;
  border-bottom: 1px solid #d6e2ef;
}
.landing section[id] { scroll-margin-top: 92px; }
.hero { background: var(--tf-shell); }
.nav-inner a,.nav-inner .nav-login{color:var(--tf-primary-strong)}
.nav-inner .nav-section:hover,.nav-inner .nav-login:hover{text-decoration:underline;text-underline-offset:.3em}
.landing-button--primary{border-radius:8px}
.hero-copy h1{font-size:clamp(2.45rem,4.5vw,4rem);max-width:12ch}
.hero-product{padding-bottom:1.5rem}
.how-grid article{border:0;border-top:2px solid var(--tf-accent-text);border-radius:0}
.step-number{color:var(--tf-text-muted)}
.hero-category, .hero-platforms { color: #c4e5da; }
.desktop-frame { border-color: #678e86; }
.store-link {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: .75rem 1.4rem;
  width: fit-content;
  max-width: 100%;
  padding: .8rem 1rem;
  border: 1px solid #b8d7c9;
  border-radius: 10px;
  background: #fff;
  text-decoration: none;
  color: #102d34;
}
.store-link span { display: grid; gap: .15rem; }
.store-link .store-icon { border-radius: 10px; }
.store-link small { color: #496660; }
.store-link img { object-fit: contain; max-width: 100%; }
.store-link:hover { border-color: #1b6358; }

.landing-header {
  background: rgb(12 34 62 / 96%);
  border-color: rgb(183 212 239 / 24%);
  backdrop-filter: blur(14px);
}
.nav-inner nav { gap: clamp(.3rem, 2vw, 1.45rem); }
.nav-inner :is(.nav-section,.nav-login) {
  display: inline-flex;
  align-items: center;
  min-height: 44px;
  padding: .45rem .65rem;
  border-radius: 6px;
  color: #e9f2ff;
  transition: color 160ms ease, background-color 160ms ease;
}
.nav-inner :is(.nav-section,.nav-login):hover {
  color: #91f0e5;
  background: rgb(188 221 255 / 12%);
  text-decoration: none;
}
.nav-inner .landing-button--primary { background: #eef6ff; color: #102e52 !important; }
.nav-inner .landing-button--primary:hover { background: #cce6fa; }
.hero { background: #0b284e; }
.product-preview { background: #f2f7fc; border-color: #cbd9e8; }
.preview-copy { align-items: start; }
.preview-controls { display: flex; align-items: center; gap: .75rem; margin-top: 1.25rem; }
.preview-controls button {
  display: grid; place-items: center; min-width: 44px; min-height: 44px;
  border: 1px solid #b3c6dc; border-radius: 8px;
  background: white; color: #103c73; cursor: pointer;
}
.preview-controls button:hover { background: #103c73; color: white; border-color: #103c73; }
.preview-controls span { font-size: .875rem; color: #34546f; font-variant-numeric: tabular-nums; }
.preview-image img { animation: screen-in 240ms ease-out; }
@keyframes screen-in {
  from { opacity: .5; transform: translateX(8px); }
  to { opacity: 1; transform: translateX(0); }
}
@media(prefers-reduced-motion:reduce) {
  .preview-image img { animation: none; }
  .nav-inner :is(.nav-section,.nav-login) { transition: none; }
}
.landing-footer { background: #0c223e; color: #d5e3f3; border: 0; padding: 0; }
.footer-grid { display: grid; grid-template-columns: 1.5fr 1fr 1fr; gap: 2rem; padding-block: 3rem 2rem; }
.footer-brand p { max-width: 28ch; font-size: .95rem; }
.footer-grid nav { display: grid; align-content: start; justify-items: start; gap: .65rem; }
.footer-grid nav h2 { color: white; font-size: 1rem; }
.footer-grid a { color: #d5e3f3; font-size: .95rem; text-decoration: none; }
.footer-grid a:hover { color: #91f0e5; text-decoration: underline; text-underline-offset: .2em; }
.footer-base { border-top: 1px solid #304761; padding-block: 1rem; }
.footer-base small { font-size: .875rem; }
@media(max-width:600px) {
  .nav-inner :is(.nav-section,.nav-login) { padding-inline: .3rem; }
  .nav-inner .nav-section { display: none; }
  .footer-grid { grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 1.5rem 1rem; padding-block: 2rem; }
  .footer-brand { grid-column: 1 / -1; }
}

.how-section { background: #e9f4f2; }
.how-grid { gap: 1.75rem; }
.how-grid article { background: transparent; border-top-color: #36665f; padding: 1.25rem 0 0; }
.step-top { justify-content: flex-start; gap: .75rem; }
.step-icon { background: #0e574e; color: white; border-radius: 8px; }
.step-number { font-weight: 700; color: #234e49; }
.how-grid h3 { margin-top: .85rem; }
.android-section { background: #e6eef9; }
.android-phone { width: min(285px, 100%); max-height: none; border: 6px solid #152132; border-radius: 26px; margin: 0 auto; }
.android-phone .phone-camera { display: none; }
.android-phone img { width: 100%; }

.gallery-toolbar { display: flex; align-items: start; justify-content: space-between; gap: .5rem; }
.product-preview { min-height: 0; grid-template-columns: minmax(0, 1fr) minmax(0, 2.25fr); }
.preview-copy { padding: clamp(1.1rem, 3vw, 2.1rem); }
.preview-image, .preview-image--phone {
  display: grid; place-items: center; aspect-ratio: 16 / 11;
  max-height: none; padding: clamp(.65rem, 2vw, 1.5rem);
}
.preview-image img {
  width: 100%; height: 100%; min-height: 0; max-height: 100%;
  object-fit: contain; border: 0; border-radius: 8px; box-shadow: none;
}
.preview-image--phone img { width: auto; max-width: 100%; }
.preview-image:hover { background: #dce7f3; }
.preview-image:focus-visible { outline-offset: -4px; }
@media (max-width: 700px) {
  .product-preview { grid-template-columns: 1fr; }
  .preview-copy p { margin-bottom: 0; }
}

.product-section .section-heading { display: grid; justify-content: start; gap: .45rem; }
.product-section .section-heading > p { max-width: 54ch; margin: 0; }
.android-section { background: #def3ed; color: #133d37; padding-block: clamp(2rem, 6vw, 5rem); }
.android-grid { grid-template-columns: minmax(0,1.05fr) minmax(0,1fr); gap: clamp(1.5rem,5vw,5rem); }
.android-copy h2 { max-width: 13ch; }
.android-features { display: grid; gap: .9rem; list-style: none; padding: 0; margin: 1.6rem 0 2rem; }
.android-features li { display: flex; align-items: center; gap: .7rem; }
.android-features svg { color: #137064; flex: none; }
.android-store { display: flex; align-items: center; flex-wrap: wrap; gap: .5rem .85rem; max-width: fit-content; border-top: 1px solid #9fc8bd; padding-top: 1rem; }
.android-store .store-icon { border-radius: 11px; }
.android-store span { display: block; color: #365c55; font-size: .85rem; }
.android-badge { display: grid; align-items: center; min-height: 48px; margin-left: .35rem; }
.android-badge img { display: block; max-width: 100%; height: auto; }
.android-badge:hover { filter: brightness(1.2); }
.android-visual { display: grid; place-items: center; margin: 0; min-width: 0; padding: 1.2rem; border-radius: 24px; background: #16756b; }
.android-phone { display: block; width: min(270px,100%); padding: 0; color: inherit; cursor: zoom-in; }
.android-phone img { height: auto; }
@media (max-width:760px) {
  .android-grid { display:grid; grid-template-columns:minmax(0,1fr); }
  .android-copy h2 { max-width: 18ch; }
  .android-visual { width:100%; }
  .android-phone { width:min(230px,100%); }
}
</style>
