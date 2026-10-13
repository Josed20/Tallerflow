<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import { AlertTriangle, Check, Clock, RefreshCw } from '@lucide/vue'
import { fetchTrackingOrder, TrackingNotFoundError, type TrackingOrder } from './api'
import { formatOrderQuantity } from '../orders/order-display'
import UiAlert from '../../components/UiAlert.vue'

const REFRESH_MS = 60_000

const route = useRoute()
const order = ref<TrackingOrder | null>(null)
const loading = ref(true)
const notFound = ref(false)
const error = ref('')
const previewPhoto = ref<TrackingOrder['photos'][number] | null>(null)
let timer: ReturnType<typeof setInterval> | null = null

const currentStage = computed(() => order.value?.stages.find((stage) => stage.status === 'IN_PROGRESS') ?? null)
const completedCount = computed(() => order.value?.stages.filter((stage) => stage.status === 'COMPLETED').length ?? 0)
const progress = computed(() => order.value?.stages.length ? Math.round((completedCount.value / order.value.stages.length) * 100) : 0)
const statusLabel = computed(() => {
  if (!order.value) return ''
  if (order.value.status === 'CANCELLED') return 'Pedido cancelado'
  return ({ ON_TIME: 'A tiempo', AT_RISK: 'Entrega próxima', LATE: 'Con retraso', COMPLETED: 'Pedido completado' })[order.value.traffic_light]
})

async function load(showLoading = false) {
  if (showLoading) loading.value = true
  error.value = ''
  try {
    order.value = await fetchTrackingOrder(String(route.params.token))
    notFound.value = false
  } catch (cause) {
    if (cause instanceof TrackingNotFoundError) {
      notFound.value = true
      order.value = null
      stopRefresh()
    } else {
      error.value = 'No pudimos actualizar el estado. Revisa tu conexión e inténtalo de nuevo.'
    }
  } finally {
    loading.value = false
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

function photosFor(position: number) {
  return order.value?.photos.filter((photo) => photo.stage_position === position) ?? []
}

function formatDate(value: string) {
  return new Intl.DateTimeFormat('es-PE', { weekday: 'long', day: 'numeric', month: 'long', hour: '2-digit', minute: '2-digit' }).format(new Date(value))
}

function formatShort(value: string | null) {
  return value ? new Intl.DateTimeFormat('es-PE', { day: 'numeric', month: 'short', hour: '2-digit', minute: '2-digit' }).format(new Date(value)) : ''
}

onMounted(() => {
  load(true)
  timer = setInterval(() => { if (document.visibilityState === 'visible') load() }, REFRESH_MS)
  document.addEventListener('visibilitychange', onVisibility)
})

onUnmounted(() => {
  stopRefresh()
  document.removeEventListener('visibilitychange', onVisibility)
})
</script>

<template>
  <main class="tracking">
    <p v-if="loading" class="tracking__state">Cargando el estado de tu pedido…</p>

    <section v-else-if="notFound" class="tracking__state tracking__state--empty">
      <h1>Este enlace ya no está activo</h1>
      <p>Puede que el taller haya generado un enlace nuevo. Pídele el enlace actualizado a quien te atendió.</p>
    </section>

    <template v-else-if="order">
      <header class="tracking__header">
        <p class="tracking__workshop">{{ order.workshop_name }}</p>
        <div class="tracking__title">
          <span class="tracking__code">{{ order.code }}</span>
          <h1>{{ order.product }}</h1>
          <p>{{ formatOrderQuantity(order.quantity, order.unit) }}</p>
        </div>
      </header>

      <section class="tracking__summary" aria-label="Resumen del pedido">
        <div class="tracking__status" :class="`tracking__status--${order.status === 'CANCELLED' ? 'late' : order.traffic_light.toLowerCase()}`">
          <Check v-if="order.traffic_light === 'ON_TIME' || order.traffic_light === 'COMPLETED'" :size="18" aria-hidden="true" />
          <AlertTriangle v-else :size="18" aria-hidden="true" />
          <span>{{ statusLabel }}</span>
        </div>
        <dl class="tracking__facts">
          <div><dt>Etapa actual</dt><dd>{{ order.status === 'COMPLETED' ? 'Entregado' : (currentStage ? stageLabel(currentStage.name) : '—') }}</dd></div>
          <div><dt>Entrega prometida</dt><dd>{{ formatDate(order.promised_at) }}</dd></div>
        </dl>
        <div class="tracking__progress" role="progressbar" :aria-valuenow="progress" aria-valuemin="0" aria-valuemax="100" :aria-label="`${completedCount} de ${order.stages.length} etapas completadas`">
          <span :style="{ width: `${progress}%` }"></span>
        </div>
        <p class="tracking__progress-label">{{ completedCount }} de {{ order.stages.length }} etapas completadas</p>
      </section>

      <UiAlert v-if="error" :message="error" />

      <section class="tracking__route" aria-labelledby="route-title">
        <h2 id="route-title">Recorrido en el taller</h2>
        <ol>
          <li v-for="stage in order.stages" :key="stage.position" class="tracking__stage" :class="`tracking__stage--${stage.status.toLowerCase()}`" :aria-current="stage.status === 'IN_PROGRESS' ? 'step' : undefined">
            <span class="tracking__dot" aria-hidden="true">
              <Check v-if="stage.status === 'COMPLETED'" :size="14" />
              <template v-else>{{ stage.position }}</template>
            </span>
            <div class="tracking__stage-body">
              <strong>{{ stageLabel(stage.name) }}</strong>
              <span v-if="stage.status === 'COMPLETED'">Terminado {{ formatShort(stage.completed_at) }}</span>
              <span v-else-if="stage.status === 'IN_PROGRESS'">En proceso ahora</span>
              <span v-else>Pendiente</span>
              <div v-if="photosFor(stage.position).length" class="tracking__photos">
                <button v-for="photo in photosFor(stage.position)" :key="photo.created_at" type="button" class="tracking__photo" :aria-label="`Ver foto de ${stageLabel(stage.name)}`" @click="previewPhoto = photo">
                  <img :src="photo.photo_data" :alt="`Evidencia de ${stage.name}`" loading="lazy" />
                </button>
              </div>
            </div>
          </li>
        </ol>
      </section>

      <footer class="tracking__footer">
        <span><Clock :size="14" aria-hidden="true" /> Actualizado {{ formatShort(order.updated_at) }}</span>
        <button type="button" @click="load()"><RefreshCw :size="14" aria-hidden="true" /> Actualizar</button>
      </footer>
      <p class="tracking__brand">Seguimiento con <strong>TallerFlow</strong></p>
    </template>

    <p v-else-if="error" class="tracking__state">{{ error }}</p>

    <div v-if="previewPhoto" class="tracking__lightbox" role="dialog" aria-modal="true" aria-label="Foto de evidencia" @click="previewPhoto = null" @keydown.esc="previewPhoto = null">
      <img :src="previewPhoto.photo_data" alt="Evidencia ampliada" />
      <p>{{ previewPhoto.caption || 'Evidencia registrada' }} · {{ formatShort(previewPhoto.created_at) }}</p>
    </div>
  </main>
</template>

<style scoped>
.tracking { width: min(40rem, 100%); margin: 0 auto; padding: 1.5rem 1rem 2.5rem; display: grid; gap: 1.1rem; color: var(--tf-ink); }
.tracking__state { padding: 2rem 0; color: var(--tf-text-subtle); text-align: center; }
.tracking__state--empty { display: grid; gap: .6rem; padding: 3rem 1.25rem; border: 1px solid var(--tf-border-soft); border-radius: .6rem; background: var(--tf-surface); }
.tracking__state--empty h1 { margin: 0; color: var(--tf-ink); font-size: 1.35rem; }
.tracking__state--empty p { margin: 0; }
.tracking__header { display: grid; gap: .6rem; }
.tracking__workshop { margin: 0; color: var(--tf-primary); font-size: .78rem; font-weight: 800; letter-spacing: .06em; text-transform: uppercase; }
.tracking__title { display: grid; gap: .25rem; }
.tracking__code { width: fit-content; padding: .2rem .55rem; border-radius: .3rem; color: #fff; background: var(--tf-primary); font-size: .78rem; font-weight: 800; font-variant-numeric: tabular-nums; }
.tracking__title h1 { margin: 0; font-size: var(--tf-page-title-size); line-height: 1.2; text-wrap: balance; }
.tracking__title p { margin: 0; color: var(--tf-text-subtle); font-size: .92rem; }
.tracking__summary { display: grid; gap: .8rem; padding: 1.1rem; border: 1px solid var(--tf-border-soft); border-radius: .6rem; background: var(--tf-surface); }
.tracking__status { display: inline-flex; align-items: center; gap: .45rem; width: fit-content; padding: .35rem .7rem; border-radius: 99px; font-size: .88rem; font-weight: 800; }
.tracking__status--on_time, .tracking__status--completed { color: var(--tf-success); background: var(--tf-success-soft); }
.tracking__status--at_risk { color: var(--tf-warning); background: var(--tf-warning-soft); }
.tracking__status--late { color: var(--tf-danger); background: var(--tf-danger-soft); }
.tracking__facts { display: grid; grid-template-columns: repeat(auto-fit, minmax(12rem, 1fr)); gap: .8rem; margin: 0; }
.tracking__facts div { display: grid; gap: .15rem; }
.tracking__facts dt { color: var(--tf-text-subtle); font-size: .78rem; font-weight: 700; }
.tracking__facts dd { margin: 0; font-size: 1rem; font-weight: 800; text-transform: none; }
.tracking__facts dd::first-letter { text-transform: uppercase; }
.tracking__progress { height: .55rem; border-radius: 99px; background: var(--tf-neutral-soft); overflow: hidden; }
.tracking__progress span { display: block; height: 100%; border-radius: inherit; background: var(--tf-primary); transition: width .4s ease; }
.tracking__progress-label { margin: -.3rem 0 0; color: var(--tf-text-subtle); font-size: .8rem; font-variant-numeric: tabular-nums; }
.tracking__route { display: grid; gap: .8rem; padding: 1.1rem; border: 1px solid var(--tf-border-soft); border-radius: .6rem; background: var(--tf-surface); }
.tracking__route h2 { margin: 0; font-size: var(--tf-section-title-size); }
.tracking__route ol { margin: 0; padding: 0; list-style: none; display: grid; }
.tracking__stage { position: relative; display: grid; grid-template-columns: 2rem minmax(0, 1fr); gap: .8rem; padding-bottom: 1.1rem; }
.tracking__stage:not(:last-child)::before { content: ""; position: absolute; top: 2rem; bottom: 0; left: calc(1rem - 1px); width: 2px; background: var(--tf-border-soft); }
.tracking__stage--completed:not(:last-child)::before { background: var(--tf-success); }
.tracking__dot { display: grid; place-items: center; width: 2rem; height: 2rem; border: 2px solid var(--tf-border); border-radius: 50%; color: var(--tf-text-subtle); background: var(--tf-surface); font-size: .8rem; font-weight: 800; }
.tracking__stage--completed .tracking__dot { border-color: var(--tf-success); color: #fff; background: var(--tf-success); }
.tracking__stage--in_progress .tracking__dot { border-color: var(--tf-primary); color: #fff; background: var(--tf-primary); box-shadow: 0 0 0 4px var(--tf-primary-soft); }
.tracking__stage-body { display: grid; gap: .1rem; padding-top: .25rem; min-width: 0; }
.tracking__stage-body strong { font-size: .98rem; }
.tracking__stage-body > span { color: var(--tf-text-subtle); font-size: .84rem; }
.tracking__stage--in_progress .tracking__stage-body > span { color: var(--tf-primary-strong); font-weight: 700; }
.tracking__stage--pending .tracking__stage-body strong { color: var(--tf-text-subtle); font-weight: 600; }
.tracking__photos { display: flex; flex-wrap: wrap; gap: .45rem; margin-top: .45rem; }
.tracking__photo { width: 4.5rem; height: 4.5rem; padding: 0; border: 1px solid var(--tf-border-soft); border-radius: .4rem; overflow: hidden; background: var(--tf-neutral-soft); cursor: zoom-in; }
.tracking__photo img { width: 100%; height: 100%; object-fit: cover; }
.tracking__footer { display: flex; justify-content: space-between; align-items: center; gap: .75rem; flex-wrap: wrap; color: var(--tf-text-subtle); font-size: .82rem; }
.tracking__footer span { display: inline-flex; align-items: center; gap: .35rem; }
.tracking__footer button { display: inline-flex; align-items: center; gap: .35rem; min-height: 40px; padding: .4rem .8rem; border: 1px solid var(--tf-primary-border); border-radius: .4rem; color: var(--tf-primary); background: var(--tf-surface); font: inherit; font-weight: 700; cursor: pointer; }
.tracking__brand { margin: 0; color: var(--tf-text-subtle); font-size: .78rem; text-align: center; }
.tracking__lightbox { position: fixed; inset: 0; z-index: 100; display: grid; place-content: center; gap: .6rem; padding: 1rem; background: rgb(11 28 48 / 82%); cursor: zoom-out; }
.tracking__lightbox img { max-width: min(100%, 48rem); max-height: 80vh; border-radius: .5rem; }
.tracking__lightbox p { margin: 0; color: #fff; font-size: .85rem; text-align: center; }
.tracking button:focus-visible { outline: 3px solid var(--tf-focus); outline-offset: 2px; }
@media (prefers-reduced-motion: reduce) { .tracking__progress span { transition: none; } }
</style>
