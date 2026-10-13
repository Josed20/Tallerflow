<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ordersApi, type Order } from './api'
import UiAlert from '../../components/UiAlert.vue'
import { matchesOrderSearch, orderNumberFromQuery } from './order-search'
import { formatOrderQuantity } from './order-display'

const router = useRouter()
const orders = ref<Order[]>([])
const loading = ref(true)
const error = ref('')
const search = ref('')

// QR Scanner state
const showScannerModal = ref(false)
const scannerError = ref('')
const manualCodeInput = ref('')
const isScanning = ref(false)
const videoRef = ref<HTMLVideoElement | null>(null)
let mediaStream: MediaStream | null = null
let scanInterval: ReturnType<typeof setInterval> | null = null

const allActiveOrders = computed(() => orders.value.filter((order) => order.status === 'ACTIVE'))
const activeOrders = computed(() => allActiveOrders.value.filter((order) => matchesOrderSearch(order, search.value))
  .sort((a, b) => new Date(a.promised_at).getTime() - new Date(b.promised_at).getTime()))

const onTimeCount = computed(() => allActiveOrders.value.filter((o) => o.traffic_light === 'ON_TIME').length)
const atRiskCount = computed(() => allActiveOrders.value.filter((o) => o.traffic_light === 'AT_RISK' || o.traffic_light === 'LATE').length)
const dueTodayCount = computed(() => allActiveOrders.value.filter((order) => new Date(order.promised_at).toLocaleDateString('en-CA') === new Date().toLocaleDateString('en-CA')).length)

async function loadQueue() {
  loading.value = true
  error.value = ''
  try {
    const response = await ordersApi.list()
    orders.value = response.data ?? []
  } catch (cause: any) {
    error.value = cause?.message || 'No se pudo cargar la cola de producción.'
  } finally {
    loading.value = false
  }
}

function resolveScannedText(rawText: string): string | null {
  const text = rawText.trim()
  if (!text) return null

  // Case 1: Full URL like http://.../app/orders/:id or /app/orders/:id
  const urlMatch = text.match(/\/app\/orders\/([a-zA-Z0-9_-]+)/)
  if (urlMatch && urlMatch[1]) {
    return urlMatch[1]
  }

  // Case 2: Direct UUID (e.g. 019623e8-...)
  if (/^[0-9a-fA-F-]{16,}$/.test(text)) {
    return text
  }

  // A printed code or its short numeric form both identify the same order.
  const orderNumber = orderNumberFromQuery(text)
  const found = orders.value.find((o) => o.code.toUpperCase() === text.toUpperCase() || (orderNumber !== null && o.order_number === orderNumber))
  if (found) {
    return found.id
  }

  return null
}

async function openScanner() {
  showScannerModal.value = true
  scannerError.value = ''
  manualCodeInput.value = ''
  isScanning.value = true
  await nextTick()

  if (!('BarcodeDetector' in window)) {
    scannerError.value = 'Este navegador no admite lectura automática. Ingresa el código de la orden manualmente.'
    isScanning.value = false
    return
  }

  // Start video stream if camera available
  if (navigator?.mediaDevices?.getUserMedia) {
    try {
      mediaStream = await navigator.mediaDevices.getUserMedia({
        video: { facingMode: 'environment', width: { ideal: 1280 }, height: { ideal: 720 } }
      })
      if (videoRef.value) {
        videoRef.value.srcObject = mediaStream
        await videoRef.value.play()
        startBarcodeDetection()
      } else {
        scannerError.value = 'No se pudo iniciar la vista de cámara. Ingresa el código manualmente.'
        isScanning.value = false
      }
    } catch (err: any) {
      scannerError.value = 'No se pudo acceder a la cámara. Puedes escribir el código de la orden manualmente.'
      isScanning.value = false
    }
  } else {
    scannerError.value = 'Cámara no soportada en este navegador. Ingresa el código manualmente.'
    isScanning.value = false
  }
}

function startBarcodeDetection() {
  // Check if standard BarcodeDetector is available
  if ('BarcodeDetector' in window) {
    try {
      const barcodeDetector = new (window as any).BarcodeDetector({ formats: ['qr_code', 'code_128', 'code_39'] })
      scanInterval = setInterval(async () => {
        if (!videoRef.value || videoRef.value.readyState < 2) return
        try {
          const barcodes = await barcodeDetector.detect(videoRef.value)
          if (barcodes.length > 0) {
            const rawValue = barcodes[0].rawValue
            if (rawValue) {
              handleCodeScanned(rawValue)
            }
          }
        } catch (_) {
          // ignore detection frame errors
        }
      }, 300)
      return
    } catch (_) {
      // Fallback
    }
  }
  scannerError.value = 'Este navegador no admite lectura automática. Ingresa el código de la orden manualmente.'
  isScanning.value = false
}

async function handleCodeScanned(text: string) {
  const orderId = resolveScannedText(text)
  if (orderId) {
    closeScanner()
    await router.push(`/app/orders/${orderId}`)
  } else {
    scannerError.value = `No se encontró la orden con código o enlace: "${text}"`
  }
}

async function handleManualSubmit() {
  if (!manualCodeInput.value.trim()) return
  await handleCodeScanned(manualCodeInput.value.trim())
}

function closeScanner() {
  if (scanInterval) {
    clearInterval(scanInterval)
    scanInterval = null
  }
  if (mediaStream) {
    mediaStream.getTracks().forEach((track) => track.stop())
    mediaStream = null
  }
  isScanning.value = false
  showScannerModal.value = false
}

onMounted(loadQueue)
onUnmounted(closeScanner)
</script>

<template>
  <main class="production-queue">
    <header class="production-queue__header">
      <div>
        <p class="production-queue__eyebrow">Producción</p>
        <h1>Cola del taller</h1>
        <p class="production-queue__subtitle">Órdenes activas, ordenadas por fecha de entrega.</p>
      </div>
      <div class="production-queue__actions">
        <button
          class="production-queue__scan-btn"
          type="button"
          @click="openScanner"
        >
          <span class="icon" aria-hidden="true">▣</span>
          <span>Escanear Ficha Viajera</span>
        </button>
        <button
          class="production-queue__refresh"
          type="button"
          @click="loadQueue"
        >
          ↻ Actualizar
        </button>
      </div>
    </header>

    <UiAlert v-if="error" :message="error" />

    <!-- Metrics Bar -->
    <section class="production-queue__metrics">
      <div class="metric-pill metric-pill--total">
        <span class="metric-pill__num">{{ allActiveOrders.length }}</span>
        <span class="metric-pill__label">Órdenes activas</span>
      </div>
      <div class="metric-pill metric-pill--ontime">
        <span class="metric-pill__num">{{ onTimeCount }}</span>
        <span class="metric-pill__label">A tiempo</span>
      </div>
      <div v-if="atRiskCount > 0" class="metric-pill metric-pill--risk">
        <span class="metric-pill__num">{{ atRiskCount }}</span>
        <span class="metric-pill__label">En riesgo o atrasadas</span>
      </div>
      <div class="metric-pill metric-pill--today"><span class="metric-pill__num">{{ dueTodayCount }}</span><span class="metric-pill__label">Entrega hoy</span></div>
    </section>

    <!-- Search filter -->
    <label class="production-queue__search" for="production-search">
      <span>Buscar en cola de trabajo</span>
      <input
        id="production-search"
        v-model="search"
        type="search"
        placeholder="Código o número (ej. 3), prenda o cliente"
      />
    </label>

    <p v-if="loading" class="production-queue__state">Cargando trabajo activo del taller…</p>
    <section v-else-if="activeOrders.length === 0" class="production-queue__empty" aria-live="polite">
      <h2>No hay órdenes activas</h2>
      <p v-if="search">No encontramos órdenes con esa búsqueda. Prueba con otro código, prenda o cliente.</p>
      <p v-else>Cuando se cree una orden, aparecerá aquí para registrar su avance.</p>
    </section>
    <section v-else class="production-queue__list" aria-label="Órdenes activas">
      <RouterLink
        v-for="order in activeOrders"
        :key="order.id"
        class="production-queue__card"
        :to="`/app/orders/${order.id}`"
        :aria-label="`Abrir ${order.code}`"
      >
        <div class="card-top">
          <div class="code-and-traffic">
            <span class="production-queue__code">{{ order.code }}</span>
            <span
              v-if="order.traffic_light === 'ON_TIME'"
              class="traffic-badge traffic-badge--green"
            >● A tiempo</span>
            <span
              v-else-if="order.traffic_light === 'AT_RISK'"
              class="traffic-badge traffic-badge--yellow"
            >● En riesgo</span>
            <span
              v-else-if="order.traffic_light === 'LATE'"
              class="traffic-badge traffic-badge--red"
            >● Atrasada</span>
          </div>
          <span class="production-queue__stage-tag">
            {{ order.current_stage || 'Pendiente' }}
          </span>
        </div>

        <div class="card-body">
          <strong class="product-title">{{ order.product }}</strong>
          <p class="product-meta">
            <span>{{ formatOrderQuantity(order.quantity, order.unit) }}</span>
            <span class="meta-sep">·</span>
            <span>{{ order.client_name }}</span>
          </p>
        </div>

        <div class="card-footer">
          <span class="advance-hint">Ver orden y registrar avance →</span>
        </div>
      </RouterLink>
    </section>

    <!-- Scanner Modal -->
    <div
      v-if="showScannerModal"
      class="scanner-modal-backdrop"
      role="dialog"
      aria-modal="true"
      aria-labelledby="scanner-title"
    >
      <div class="scanner-modal">
        <header class="scanner-modal__header">
          <div>
            <h2 id="scanner-title">Escanear Ficha Viajera</h2>
            <p class="scanner-modal__subtitle">Apunta al código impreso o a un QR estándar del lote</p>
          </div>
          <button class="scanner-modal__close" type="button" @click="closeScanner" aria-label="Cerrar escáner">
            ✕
          </button>
        </header>

        <div class="scanner-modal__body">
          <div class="video-container">
            <video
              ref="videoRef"
              class="video-preview"
              autoplay
              playsinline
              muted
            ></video>
            <div class="scanner-reticle">
              <div class="reticle-corner reticle-corner--tl"></div>
              <div class="reticle-corner reticle-corner--tr"></div>
              <div class="reticle-corner reticle-corner--bl"></div>
              <div class="reticle-corner reticle-corner--br"></div>
              <div class="scanner-laser"></div>
            </div>
          </div>

          <p v-if="scannerError" class="scanner-error-text">{{ scannerError }}</p>

          <form class="manual-input-form" @submit.prevent="handleManualSubmit">
            <label for="manual-code-input">O ingresa el código o ID de la orden:</label>
            <div class="manual-input-row">
              <input
                id="manual-code-input"
                v-model="manualCodeInput"
                type="text"
                placeholder="Ej. OT-000003, 3 o pega el enlace"
                autocomplete="off"
              />
              <button class="manual-submit-btn" type="submit" @click.prevent="handleManualSubmit">
                Abrir orden
              </button>
            </div>
          </form>
        </div>
      </div>
    </div>
  </main>
</template>

<style scoped>
.production-queue {
  max-width: 82rem;
  margin: 0 auto;
  padding: 1.5rem 1rem;
  color: var(--tf-ink);
}

.production-queue__header {
  display: flex;
  justify-content: space-between;
  gap: 1rem;
  align-items: center;
  margin-bottom: 1.25rem;
  flex-wrap: wrap;
}

.production-queue__eyebrow {
  margin: 0 0 .25rem;
  color: var(--tf-primary);
  font-size: .8rem;
  font-weight: 800;
  text-transform: uppercase;
  letter-spacing: .08em;
}

.production-queue__subtitle { margin: .3rem 0 0; color: var(--tf-text-subtle); font-size: .86rem; }

h1 {
  margin: 0;
  font-size: var(--tf-page-title-size);
  font-weight: 800;
  color: var(--tf-ink);
}

.production-queue__actions {
  display: flex;
  gap: .5rem;
  align-items: center;
}

.production-queue__scan-btn {
  display: inline-flex;
  align-items: center;
  gap: .4rem;
  min-height: 44px;
  background: var(--tf-primary);
  color: #ffffff;
  border: none;
  border-radius: .5rem;
  padding: .5rem .875rem;
  font: inherit;
  font-weight: 700;
  cursor: pointer;
  box-shadow: var(--tf-shadow-sm);
  transition: background 0.15s ease;
}

.production-queue__scan-btn:hover {
  background: var(--tf-primary-hover);
}

.production-queue__refresh {
  min-height: 44px;
  border: 1px solid var(--tf-border-soft);
  border-radius: .5rem;
  padding: .5rem .75rem;
  background: #fff;
  color: var(--tf-ink-soft);
  font: inherit;
  font-weight: 700;
  cursor: pointer;
}

.production-queue__metrics {
  display: flex;
  gap: .75rem;
  margin-bottom: 1.25rem;
  flex-wrap: wrap;
}

.metric-pill {
  display: flex;
  align-items: center;
  gap: .5rem;
  padding: .4rem .75rem;
  border-radius: 9999px;
  font-size: .85rem;
  font-weight: 700;
}

.metric-pill--total {
  background: var(--tf-neutral-soft);
  color: var(--tf-ink-soft);
  border: 1px solid var(--tf-border-soft);
}

.metric-pill--ontime {
  background: var(--tf-success-soft);
  color: var(--tf-success);
  border: 1px solid var(--tf-success-border);
}

.metric-pill--risk {
  background: var(--tf-danger-soft);
  color: var(--tf-danger);
  border: 1px solid var(--tf-danger-border);
}

.metric-pill--today { background: var(--tf-info-soft); color: var(--tf-info); border: 1px solid var(--tf-info-border); }

.metric-pill__num {
  font-size: 1rem;
  font-weight: 800;
}

.production-queue__search {
  display: grid;
  gap: .35rem;
  margin-bottom: 1.25rem;
  font-weight: 700;
  font-size: .9rem;
}

.production-queue__search input {
  min-height: 46px;
  border: 1px solid var(--tf-border);
  border-radius: .5rem;
  padding: .625rem .875rem;
  font: inherit;
  background: #fff;
}

.production-queue__search input:focus-visible {
  outline: 3px solid var(--tf-focus);
  outline-offset: 1px;
}

.production-queue__state,
.production-queue__empty {
  padding: 3rem 1rem;
  text-align: center;
  color: #475569;
}

.production-queue__empty {
  border: 1px dashed #cbd5e1;
  border-radius: .75rem;
  background: #fff;
}

.empty-icon {
  font-size: 2.5rem;
  margin-bottom: .5rem;
}

.production-queue__empty h2 {
  margin: 0 0 .5rem;
  font-size: 1.2rem;
  color: #0f172a;
}

.production-queue__empty p {
  margin: 0;
  font-size: .95rem;
}

.production-queue__list {
  display: grid;
  gap: .875rem;
}

@media (min-width: 1080px) {
  .production-queue__list { grid-template-columns: repeat(2, minmax(0, 1fr)); }
}

.production-queue__card {
  display: grid;
  gap: .625rem;
  border: 1px solid #e2e8f0;
  border-left: .375rem solid var(--tf-primary);
  border-radius: .5rem;
  padding: 1.125rem 1.25rem;
  color: inherit;
  background: #ffffff;
  text-decoration: none;
  box-shadow: 0 1px 3px rgba(15, 23, 42, 0.06);
  transition: transform 0.1s ease, box-shadow 0.1s ease;
}

.production-queue__card:hover {
  transform: translateY(-1px);
  box-shadow: 0 4px 12px rgba(15, 23, 42, 0.08);
}

.production-queue__card:focus-visible {
  outline: 3px solid var(--tf-focus);
  outline-offset: 2px;
}

.card-top {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.code-and-traffic {
  display: flex;
  align-items: center;
  gap: .5rem;
}

.production-queue__code {
  color: var(--tf-primary);
  font-weight: 800;
  font-size: 1.05rem;
}

.traffic-badge {
  font-size: .75rem;
  font-weight: 700;
  padding: .15rem .45rem;
  border-radius: 9999px;
}

.traffic-badge--green {
  background: var(--tf-success-soft);
  color: var(--tf-success);
}

.traffic-badge--yellow {
  background: var(--tf-warning-soft);
  color: var(--tf-warning);
}

.traffic-badge--red {
  background: var(--tf-danger-soft);
  color: var(--tf-danger);
}

.production-queue__stage-tag {
  background: var(--tf-primary-soft);
  color: var(--tf-primary-strong);
  border: 1px solid var(--tf-primary-border);
  font-weight: 700;
  font-size: .8rem;
  padding: .25rem .6rem;
  border-radius: .375rem;
}

.card-body {
  display: grid;
  gap: .2rem;
}

.product-title {
  font-size: 1.15rem;
  color: #0b1c30;
}

.product-meta {
  margin: 0;
  color: #475569;
  font-size: .95rem;
}

.meta-sep {
  margin: 0 .35rem;
}

.card-footer {
  display: flex;
  justify-content: flex-end;
  border-top: 1px solid #f1f5f9;
  padding-top: .5rem;
}

.advance-hint {
  color: var(--tf-primary);
  font-weight: 700;
  font-size: .85rem;
}

/* Scanner Modal Styles */
.scanner-modal-backdrop {
  position: fixed;
  inset: 0;
  background: rgba(15, 23, 42, 0.75);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 999;
  padding: 1rem;
}

.scanner-modal {
  background: #ffffff;
  border-radius: 1rem;
  max-width: 32rem;
  width: 100%;
  overflow: hidden;
  box-shadow: 0 20px 25px -5px rgba(0, 0, 0, 0.2);
}

.scanner-modal__header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  padding: 1.25rem 1.5rem;
  background: var(--tf-ink);
  color: #ffffff;
}

.scanner-modal__header h2 {
  margin: 0 0 .25rem;
  font-size: 1.2rem;
  color: #ffffff;
}

.scanner-modal__subtitle {
  margin: 0;
  font-size: .85rem;
  color: #94a3b8;
}

.scanner-modal__close {
  background: transparent;
  border: none;
  color: #cbd5e1;
  font-size: 1.3rem;
  cursor: pointer;
  padding: .25rem;
}

.scanner-modal__body {
  padding: 1.5rem;
  display: grid;
  gap: 1.25rem;
}

.video-container {
  position: relative;
  width: 100%;
  height: 220px;
  background: #0f172a;
  border-radius: .75rem;
  overflow: hidden;
  display: flex;
  align-items: center;
  justify-content: center;
}

.video-preview {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.scanner-reticle {
  position: absolute;
  width: 150px;
  height: 150px;
  pointer-events: none;
}

.reticle-corner {
  position: absolute;
  width: 20px;
  height: 20px;
  border-color: var(--tf-focus);
}

.reticle-corner--tl { top: 0; left: 0; border-top: 3px solid; border-left: 3px solid; }
.reticle-corner--tr { top: 0; right: 0; border-top: 3px solid; border-right: 3px solid; }
.reticle-corner--bl { bottom: 0; left: 0; border-bottom: 3px solid; border-left: 3px solid; }
.reticle-corner--br { bottom: 0; right: 0; border-bottom: 3px solid; border-right: 3px solid; }

.scanner-laser {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  height: 2px;
  background: var(--tf-danger);
  box-shadow: 0 0 8px var(--tf-danger);
  animation: laser-sweep 2s infinite ease-in-out;
}

@keyframes laser-sweep {
  0% { top: 0; }
  50% { top: 100%; }
  100% { top: 0; }
}

.scanner-error-text {
  margin: 0;
  padding: .75rem;
  background: var(--tf-danger-soft);
  border-left: 3px solid var(--tf-danger);
  border-radius: .375rem;
  color: var(--tf-danger);
  font-size: .85rem;
}

.manual-input-form {
  display: grid;
  gap: .5rem;
}

.manual-input-form label {
  font-size: .85rem;
  font-weight: 700;
  color: #334155;
}

.manual-input-row {
  display: flex;
  gap: .5rem;
}

.manual-input-row input {
  flex: 1;
  min-height: 42px;
  border: 1px solid #cbd5e1;
  border-radius: .5rem;
  padding: .5rem .75rem;
  font: inherit;
}

.manual-submit-btn {
  background: var(--tf-primary);
  color: #ffffff;
  border: none;
  border-radius: .5rem;
  padding: .5rem 1rem;
  font: inherit;
  font-weight: 700;
  cursor: pointer;
  white-space: nowrap;
}

.manual-submit-btn:hover {
  background: var(--tf-primary-hover);
}

@media (max-width: 640px) {
  .production-queue__actions { width: 100%; }
  .production-queue__scan-btn { flex: 1; justify-content: center; }
  .card-top, .code-and-traffic { align-items: flex-start; flex-wrap: wrap; }
  .manual-input-row { flex-direction: column; }
  .manual-submit-btn { min-height: 44px; }
}

@media (prefers-reduced-motion: reduce) {
  .scanner-laser { animation: none; }
  .production-queue__card { transition: none; }
}
</style>
