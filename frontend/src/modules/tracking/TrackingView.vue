<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { Check, Clock, RefreshCw, X } from '@lucide/vue'
import { fetchTrackingOrder, TrackingNotFoundError, type TrackingOrder } from './api'
import { formatOrderQuantity } from '../orders/order-display'
import UiAlert from '../../components/UiAlert.vue'
import BrandLogo from '../../components/BrandLogo.vue'
import StatusChip from '../../components/StatusChip.vue'
import StitchSeam from '../../components/StitchSeam.vue'

const REFRESH_MS = 60_000

const route = useRoute()
const order = ref<TrackingOrder | null>(null)
const loading = ref(true)
const refreshing = ref(false)
const checkedAt = ref<string | null>(null)
const refreshNotice = ref('')
let disposed = false
const notFound = ref(false)
const error = ref('')
const previewPhoto = ref<TrackingOrder['photos'][number] | null>(null)
let timer: ReturnType<typeof setInterval> | null = null

const currentStage = computed(() => order.value?.stages.find((stage) => stage.status === 'IN_PROGRESS') ?? null)
const completedCount = computed(() => order.value?.stages.filter((stage) => stage.status === 'COMPLETED').length ?? 0)
const stationHeadline = computed(() => {
  if (!order.value) return ''
  if (order.value.status === 'COMPLETED') return 'Completado'
  if (order.value.status === 'CANCELLED') return 'Cancelado'
  return currentStage.value ? stageLabel(currentStage.value.name) : 'En preparación'
})
const statusLabel = computed(() => {
  if (!order.value) return ''
  if (order.value.status === 'CANCELLED') return 'Pedido cancelado'
  return ({ ON_TIME: 'A tiempo', AT_RISK: 'Entrega próxima', LATE: 'Con retraso', COMPLETED: 'Pedido completado' })[order.value.traffic_light]
})

async function load(showLoading = false) {
  if (refreshing.value || disposed) return
  refreshing.value = true
  refreshNotice.value = 'Consultando el estado…'
  if (showLoading) loading.value = true
  error.value = ''
  const requestedToken = String(route.params.token)
  try {
    const result = await fetchTrackingOrder(requestedToken)
    if (disposed || requestedToken !== String(route.params.token)) return
    order.value = result
    checkedAt.value = new Date().toISOString()
    refreshNotice.value = 'Estado actualizado.'
    notFound.value = false
  } catch (cause) {
    if (disposed || requestedToken !== String(route.params.token)) return
    if (cause instanceof TrackingNotFoundError) {
      notFound.value = true
      order.value = null
      previewPhoto.value = null
      stopRefresh()
    } else {
      refreshNotice.value = ''
      error.value = 'No pudimos actualizar el estado. Revisa tu conexión e inténtalo de nuevo.'
    }
  } finally {
    loading.value = false
    refreshing.value = false
    if (!disposed && requestedToken !== String(route.params.token)) void load(true)
  }
}

function stopRefresh() {
  if (timer) clearInterval(timer)
  timer = null
}

function onVisibility() {
  if (document.visibilityState === 'visible' && !notFound.value) load()
}

function stageLabel(name: string) {
  const lower = name.toLocaleLowerCase('es-PE')
  return lower.charAt(0).toLocaleUpperCase('es-PE') + lower.slice(1)
}

function stageDescription(stage: TrackingOrder['stages'][number]) {
  const key = stage.name.trim().toLocaleLowerCase('es-PE')
  if (stage.client_description?.trim() && !(key === 'acabado' && stage.client_description === 'Realizamos los acabados de tus prendas.')) return stage.client_description
  return ({
    corte: 'Se cortan las piezas de tela del pedido.',
    costura: 'Se unen y cosen las piezas de cada prenda.',
    acabado: 'Se retiran hilos sobrantes y se realizan los detalles finales de las prendas. Todavía puede haber etapas pendientes.',
    planchado: 'Se planchan las prendas.',
    empaque: 'Se prepara y protege el pedido para su entrega.',
    'control de calidad': 'Se revisa la calidad de las prendas.',
  } as Record<string, string>)[key] ?? 'Etapa del proceso definido por el taller. Su avance se muestra debajo.'
}

const progressDescription = computed(() => {
  if (order.value?.status === 'COMPLETED') return 'El taller marcó el pedido como completado. Coordina la entrega con quien te atendió.'
  if (order.value?.status === 'CANCELLED') return 'El taller canceló este pedido. Contacta a quien te atendió para conocer los detalles.'
  return currentStage.value ? stageDescription(currentStage.value) : 'El taller está preparando el pedido para continuar su producción.'
})

function photosFor(position: number) {
  return order.value?.photos.filter((photo) => photo.stage_position === position) ?? []
}

function formatDate(value: string) {
  return new Intl.DateTimeFormat('es-PE', { weekday: 'long', day: 'numeric', month: 'long', hour: '2-digit', minute: '2-digit' }).format(new Date(value))
}

function formatShort(value: string | null) {
  return value ? new Intl.DateTimeFormat('es-PE', { day: 'numeric', month: 'short', hour: '2-digit', minute: '2-digit' }).format(new Date(value)) : ''
}

// Photo viewer dialog: focus moves to Cerrar, Escape closes it and focus
// returns to the thumbnail that opened it.
const photoCloseButton = ref<HTMLButtonElement | null>(null)
let photoOpener: HTMLElement | null = null

async function openPhoto(photo: TrackingOrder['photos'][number], event: Event) {
  photoOpener = event.currentTarget as HTMLElement
  previewPhoto.value = photo
  await nextTick()
  photoCloseButton.value?.focus()
}

function closePhoto() {
  previewPhoto.value = null
  photoOpener?.focus()
  photoOpener = null
}

function onKeydown(event: KeyboardEvent) {
  if (event.key === 'Escape' && previewPhoto.value) closePhoto()
}

watch(() => route.params.token, () => {
  order.value = null
  previewPhoto.value = null
  notFound.value = false
  checkedAt.value = null
  stopRefresh()
  timer = setInterval(() => { if (document.visibilityState === 'visible') load() }, REFRESH_MS)
  loading.value = true
  void load(true)
})

onMounted(() => {
  load(true)
  timer = setInterval(() => { if (document.visibilityState === 'visible') load() }, REFRESH_MS)
  document.addEventListener('visibilitychange', onVisibility)
  window.addEventListener('keydown', onKeydown)
})

onUnmounted(() => {
  disposed = true
  stopRefresh()
  document.removeEventListener('visibilitychange', onVisibility)
  window.removeEventListener('keydown', onKeydown)
})
</script>

<template>
  <main class="tracking">
    <p v-if="loading" class="tracking__state">Cargando el estado de tu pedido…</p>

    <section v-else-if="notFound" class="tracking__state tracking__state--empty">
      <h1>Este enlace ya no está activo</h1>
      <p>El enlace puede haber vencido, sido reemplazado o revocado. Pide un enlace actualizado al taller.</p>
    </section>

    <template v-else-if="order">
      <header class="tracking__header">
        <div class="tracking__title">
          <h1>{{ order.product }}</h1>
          <p class="tracking__workshop"><span>Taller</span> <strong>{{ order.workshop_name }}</strong></p>
          <span class="tracking__code">{{ order.code }}</span>
          <p>{{ formatOrderQuantity(order.quantity, order.unit) }}</p>
        </div>
      </header>

      <section class="tracking__summary" aria-label="Resumen del pedido">
        <div class="tracking__headline">
          <span class="tracking__eyebrow">{{ order.status === 'ACTIVE' ? 'Etapa actual de tu pedido' : 'Estado de tu pedido' }}</span>
          <p class="tracking__station">{{ stationHeadline }}</p>
          <p class="tracking__explanation">{{ progressDescription }}</p>
        </div>
        <dl class="tracking__facts">
          <div><dt>Fecha de entrega prometida</dt><dd>{{ formatDate(order.promised_at) }}</dd></div>
          <div><dt>Estado</dt><dd><StatusChip :state="order.status === 'CANCELLED' ? 'CANCELLED' : order.traffic_light" :label="statusLabel" /></dd></div>
        </dl>
        <StitchSeam v-if="order.stages.length" :completed="completedCount" :total="order.stages.length" :tone="order.traffic_light === 'LATE' ? 'late' : 'default'" />
        <p class="tracking__progress-label">{{ completedCount }} de {{ order.stages.length }} etapas completadas</p>
      </section>

      <section v-if="order.specification" class="tracking__route" aria-labelledby="description-title">
        <h2 id="description-title">Descripción del pedido</h2>
        <p class="tracking__specification">{{ order.specification }}</p>
      </section>

      <UiAlert v-if="error" :message="error" />

      <section class="tracking__route" aria-labelledby="route-title">
        <h2 id="route-title">Recorrido en el taller</h2>
        <p v-if="!order.stages.length">El taller todavía no registró etapas para este pedido.</p>
        <ol>
          <li v-for="stage in order.stages" :key="stage.position" class="tracking__stage" :class="`tracking__stage--${stage.status.toLowerCase()}`" :aria-current="stage.status === 'IN_PROGRESS' ? 'step' : undefined">
            <span class="tracking__dot" aria-hidden="true">
              <Check v-if="stage.status === 'COMPLETED'" :size="14" />
              <template v-else>{{ stage.position }}</template>
            </span>
            <div class="tracking__stage-body">
              <strong>{{ stageLabel(stage.name) }}</strong>
              <p class="tracking__explanation">{{ stageDescription(stage) }}</p>
              <span v-if="stage.status === 'COMPLETED'">Terminado {{ formatShort(stage.completed_at) }}</span>
              <span v-else-if="stage.status === 'IN_PROGRESS'">En proceso ahora</span>
              <span v-else>Pendiente</span>
              <div v-if="photosFor(stage.position).length" class="tracking__photos">
                <button v-for="photo in photosFor(stage.position)" :key="photo.created_at" type="button" class="tracking__photo" :aria-label="`Ver foto de ${stageLabel(stage.name)}`" @click="openPhoto(photo, $event)">
                  <img :src="photo.photo_data" :alt="`Evidencia de ${stage.name}`" loading="lazy" />
                </button>
              </div>
            </div>
          </li>
        </ol>
      </section>

      <footer class="tracking__footer">
        <span><Clock :size="14" aria-hidden="true" /> Último cambio del taller: {{ formatShort(order.updated_at) }}</span>
        <button type="button" :disabled="refreshing" :aria-busy="refreshing" @click="load()"><RefreshCw :size="14" aria-hidden="true" /> {{ refreshing ? 'Actualizando…' : 'Actualizar' }}</button>
        <p>Se actualiza automáticamente cada minuto mientras esta página está visible.</p>
        <p v-if="checkedAt">Última consulta: {{ formatShort(checkedAt) }}</p>
        <p role="status" aria-live="polite">{{ refreshNotice }}</p>
      </footer>
    </template>

    <section v-else-if="error" class="tracking__state"><p role="alert">{{ error }}</p><button type="button" :disabled="refreshing" @click="load(true)">Reintentar</button></section>

    <!-- Shown on dead links too, so a customer always knows which service sent them. -->
    <p v-if="!loading" class="tracking__brand"><span>Seguimiento con</span> <BrandLogo :size="16" /></p>

    <div v-if="previewPhoto" class="tracking__lightbox" role="dialog" aria-modal="true" aria-label="Foto de evidencia" @click.self="closePhoto">
      <button ref="photoCloseButton" type="button" class="tracking__lightbox-close" @click="closePhoto"><X :size="18" aria-hidden="true" /> Cerrar</button>
      <img :src="previewPhoto.photo_data" alt="Evidencia ampliada" />
      <p>{{ previewPhoto.caption || 'Evidencia registrada' }} · {{ formatShort(previewPhoto.created_at) }}</p>
    </div>
  </main>
</template>

<style scoped>
.tracking { width: min(40rem, 100%); margin: 0 auto; padding: 1.5rem 1rem 2.5rem; display: grid; gap: 1.1rem; color: var(--tf-ink); }
.tracking__explanation { margin: .4rem 0; color: var(--tf-text-subtle); line-height: 1.55; font-size: .9rem; }
.tracking__specification { margin: 0; white-space: pre-wrap; overflow-wrap: anywhere; }
.tracking__title h1, .tracking__station, .tracking__stage-body strong { overflow-wrap: anywhere; }
.tracking__footer p { margin: 0; }
.tracking__footer button:disabled { opacity: .65; cursor: wait; }
.tracking__state { padding: 2rem 0; color: var(--tf-text-subtle); text-align: center; }
.tracking__state--empty { display: grid; gap: .6rem; padding: 3rem 1.25rem; border: 1px solid var(--tf-border-soft); border-radius: var(--tf-radius-sm); background: var(--tf-surface); text-align: left; }
.tracking__state--empty h1 { margin: 0; color: var(--tf-ink); font-size: 1.35rem; }
.tracking__state--empty p { margin: 0; }
.tracking__header { display: grid; gap: .6rem; }
.tracking__workshop { display: flex; flex-wrap: wrap; align-items: center; gap: .45rem; margin: 0; color: var(--tf-text-subtle); font-size: .8rem; }
/* The workshop's woven label (DESIGN.md, Components). */
.tracking__workshop strong { padding: .2rem .6rem .15rem; border-bottom: 2px solid var(--tf-accent); border-radius: var(--tf-radius-sm); color: var(--tf-on-primary); background: var(--tf-primary); font-family: var(--tf-font-display); font-size: .72rem; font-weight: 700; letter-spacing: .07em; text-transform: uppercase; overflow-wrap: anywhere; }
.tracking__title { display: grid; gap: .25rem; }
.tracking__code { width: fit-content; padding: .2rem .55rem; border-radius: var(--tf-radius-sm); color: var(--tf-on-primary); background: var(--tf-primary); font-family: var(--tf-font-mono); font-size: .8rem; font-weight: 700; font-variant-numeric: tabular-nums; }
.tracking__title h1 { margin: 0; font-size: var(--tf-page-title-size); line-height: 1.2; text-wrap: balance; }
.tracking__title p { margin: 0; color: var(--tf-text-subtle); font-size: .92rem; }
.tracking__summary { display: grid; gap: .9rem; padding: 1.1rem; border: 1px solid var(--tf-border-soft); border-radius: var(--tf-radius-sm); background: var(--tf-surface); }
.tracking__headline { display: grid; gap: .15rem; }
.tracking__eyebrow { color: var(--tf-text-subtle); font-family: var(--tf-font-display); font-size: .72rem; font-weight: 700; letter-spacing: .07em; text-transform: uppercase; }
.tracking__station { margin: 0; color: var(--tf-primary); font-family: var(--tf-font-display); font-variation-settings: "wdth" 125; font-size: clamp(2rem, 9vw, 2.8rem); font-weight: 900; line-height: 1; letter-spacing: -.01em; }
.tracking__facts { display: grid; grid-template-columns: repeat(auto-fit, minmax(12rem, 1fr)); gap: .8rem; margin: 0; }
.tracking__facts div { display: grid; gap: .15rem; }
.tracking__facts dt { color: var(--tf-text-subtle); font-size: .78rem; font-weight: 700; }
.tracking__facts dd { margin: 0; font-size: 1rem; font-weight: 800; text-transform: none; }
.tracking__facts dd::first-letter { text-transform: uppercase; }
.tracking__progress-label { margin: -.4rem 0 0; color: var(--tf-text-subtle); font-size: .8rem; font-variant-numeric: tabular-nums; }
.tracking__route { display: grid; gap: .8rem; padding: 1.1rem; border: 1px solid var(--tf-border-soft); border-radius: var(--tf-radius-sm); background: var(--tf-surface); }
.tracking__route h2 { margin: 0; font-size: var(--tf-section-title-size); }
.tracking__route ol { margin: 0; padding: 0; list-style: none; display: grid; }
.tracking__stage { position: relative; display: grid; grid-template-columns: 2rem minmax(0, 1fr); gap: .8rem; padding-bottom: 1.1rem; }
.tracking__stage:not(:last-child)::before { content: ""; position: absolute; top: 2rem; bottom: 0; left: calc(1rem - 1.5px); width: 3px; background: repeating-linear-gradient(180deg, var(--tf-border-soft) 0 5px, transparent 5px 9px); }
.tracking__stage--completed:not(:last-child)::before { background: var(--tf-primary); }
.tracking__dot { display: grid; place-items: center; width: 2rem; height: 2rem; border: 2px solid var(--tf-border-soft); border-radius: 50%; color: var(--tf-text-subtle); background: var(--tf-surface); font-size: .8rem; font-weight: 800; }
.tracking__stage--completed .tracking__dot { border-color: var(--tf-primary); color: var(--tf-on-primary); background: var(--tf-primary); }
.tracking__stage--in_progress .tracking__dot { border-color: var(--tf-primary); color: var(--tf-primary); background: var(--tf-surface); box-shadow: 0 0 0 4px var(--tf-primary-soft); }
.tracking__stage-body { display: grid; gap: .1rem; padding-top: .25rem; min-width: 0; }
.tracking__stage-body strong { font-size: .98rem; }
.tracking__stage-body > span { color: var(--tf-text-subtle); font-size: .84rem; }
.tracking__stage--in_progress .tracking__stage-body > span { color: var(--tf-primary-strong); font-weight: 700; }
.tracking__stage--pending .tracking__stage-body strong { color: var(--tf-text-subtle); font-weight: 600; }
.tracking__photos { display: flex; flex-wrap: wrap; gap: .45rem; margin-top: .45rem; }
.tracking__photo { width: 4.5rem; height: 4.5rem; padding: 0; border: 1px solid var(--tf-border-soft); border-radius: var(--tf-radius-sm); overflow: hidden; background: var(--tf-neutral-soft); cursor: zoom-in; }
.tracking__photo img { width: 100%; height: 100%; object-fit: cover; }
.tracking__footer { display: flex; justify-content: space-between; align-items: center; gap: .75rem; flex-wrap: wrap; color: var(--tf-text-subtle); font-size: .82rem; }
.tracking__footer span { display: inline-flex; align-items: center; gap: .35rem; }
.tracking__footer button { display: inline-flex; align-items: center; gap: .35rem; min-height: 44px; padding: .4rem .8rem; border: 1px solid var(--tf-primary); border-radius: var(--tf-radius); color: var(--tf-primary); background: var(--tf-surface); font: inherit; font-weight: 700; cursor: pointer; }
.tracking__brand { display: flex; align-items: center; justify-content: center; gap: .4rem; margin: 0; color: var(--tf-text-subtle); font-size: .78rem; }
.tracking__lightbox { position: fixed; inset: 0; z-index: 100; display: grid; place-content: center; gap: .6rem; padding: 1rem; background: rgb(22 35 61 / 82%); cursor: zoom-out; }
.tracking__lightbox-close { display: inline-flex; align-items: center; gap: .35rem; justify-self: end; min-height: 44px; padding: .5rem .8rem; border: 1px solid var(--tf-on-primary); border-radius: var(--tf-radius); color: var(--tf-on-primary); background: transparent; font: inherit; font-weight: 700; cursor: pointer; }
.tracking__lightbox-close:focus-visible { outline: 3px solid var(--tf-shell-accent); outline-offset: 2px; }
.tracking__lightbox img { max-width: min(100%, 48rem); max-height: 80vh; border-radius: .5rem; }
.tracking__lightbox p { margin: 0; color: var(--tf-on-primary); font-size: .85rem; text-align: center; }
.tracking button:focus-visible { outline: 3px solid var(--tf-focus); outline-offset: 2px; }
</style>
