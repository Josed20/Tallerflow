<script setup lang="ts">
import { computed, ref, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ordersApi, type Order, type OrderStage, type StageStatus, type StagePhoto } from './api'
import UiAlert from '../../components/UiAlert.vue'
import { useSessionStore } from '../session/session.store'
import { canAdvanceProduction } from '../session/role-policy'
import { compressImageFile } from '../../utils/image'

const route = useRoute()
const router = useRouter()
const session = useSessionStore()

const order = ref<Order | null>(null)
const loading = ref(true)
const error = ref('')
const updatingStage = ref<number | null>(null)
const uploadingPhotoStage = ref<number | null>(null)
const showTicketModal = ref(false)
const previewPhoto = ref<StagePhoto | null>(null)

const canAdvance = computed(() => canAdvanceProduction(session.principal?.role))

async function loadOrder() {
  const id = route.params.id as string
  loading.value = true
  error.value = ''
  try {
    const res = await ordersApi.getByID(id)
    order.value = res.data
  } catch (err: any) {
    error.value = err?.message || 'No se pudo cargar la orden de trabajo.'
  } finally {
    loading.value = false
  }
}

async function handleAdvanceStage(stage: OrderStage, nextStatus: StageStatus) {
  if (!order.value || !canAdvance.value || stage.status !== 'IN_PROGRESS' || nextStatus !== 'COMPLETED') return
  if (!session.csrfToken) {
    error.value = 'Tu sesión ya no permite actualizar producción. Actualiza la página e inicia sesión nuevamente.'
    return
  }
  updatingStage.value = stage.position
  try {
    await ordersApi.updateStage(order.value.id, stage.position, nextStatus, session.csrfToken)
    await loadOrder()
  } catch (err: any) {
    if (err?.status === 409) {
      error.value = 'Otro usuario actualizó esta orden. Mostramos el avance más reciente.'
      await loadOrder()
    } else {
      error.value = err?.message || 'Error al actualizar el estado de la etapa.'
    }
  } finally {
    updatingStage.value = null
  }
}

async function handlePhotoFileSelected(stagePosition: number, event: Event) {
  const target = event.target as HTMLInputElement
  const file = target.files?.[0]
  if (!file || !order.value) return

  if (!canAdvance.value) {
    error.value = 'Tu rol no puede registrar evidencia de producción.'
    target.value = ''
    return
  }
  if (!session.csrfToken) {
    error.value = 'Tu sesión ya no permite adjuntar evidencia. Actualiza la página e inicia sesión nuevamente.'
    target.value = ''
    return
  }
  if (!file.type.startsWith('image/')) {
    error.value = 'Selecciona una imagen válida.'
    target.value = ''
    return
  }
  if (file.size > 8 * 1024 * 1024) {
    error.value = 'La imagen debe pesar como máximo 8 MB antes de comprimirla.'
    target.value = ''
    return
  }

  uploadingPhotoStage.value = stagePosition
  error.value = ''
  try {
    const compressed = await compressImageFile(file, 1200, 0.8)
    await ordersApi.addStagePhoto(order.value.id, stagePosition, compressed, '', session.csrfToken)
    await loadOrder()
  } catch (err: any) {
    error.value = err?.message || 'No se pudo adjuntar la foto a la etapa.'
  } finally {
    uploadingPhotoStage.value = null
    target.value = ''
  }
}

function getStagePhotos(position: number): StagePhoto[] {
  if (!order.value?.photos) return []
  return order.value.photos.filter((p) => p.stage_position === position)
}

function formatDate(iso: string) {
  if (!iso) return ''
  const d = new Date(iso)
  return d.toLocaleDateString('es-PE', {
    weekday: 'short',
    day: '2-digit',
    month: 'short',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  })
}

function formatShortTime(iso: string | null) {
  if (!iso) return ''
  const d = new Date(iso)
  return d.toLocaleTimeString('es-PE', { hour: '2-digit', minute: '2-digit' })
}

function printTicket() {
  window.print()
}

onMounted(() => {
  loadOrder()
})
</script>

<template>
  <main class="order-detail-page">
    <header class="page-header">
      <button class="btn-back" @click="router.push('/app/orders')">← Volver a Órdenes</button>
      <div v-if="order" class="header-title-row">
        <div class="header-left">
          <span class="order-code-badge">{{ order.code }}</span>
          <h1>{{ order.product }}</h1>
        </div>
        <button class="btn-ticket" type="button" @click="showTicketModal = true">
          🖨️ Imprimir Ficha Viajera
        </button>
      </div>
    </header>

    <UiAlert v-if="error" :message="error" />

    <section v-if="loading" class="loading-state">
      <p>Cargando detalle de la orden...</p>
    </section>

    <div v-else-if="order" class="detail-container">
      <!-- Banner de Semáforo -->
      <section class="traffic-banner" :class="'banner-' + order.traffic_light.toLowerCase()">
        <div class="traffic-icon">
          <span v-if="order.traffic_light === 'ON_TIME'">🟢</span>
          <span v-else-if="order.traffic_light === 'AT_RISK'">🟡</span>
          <span v-else-if="order.traffic_light === 'LATE'">🔴</span>
          <span v-else>🔵</span>
        </div>
        <div class="traffic-info">
          <h2 v-if="order.traffic_light === 'ON_TIME'">A tiempo para la entrega</h2>
          <h2 v-else-if="order.traffic_light === 'AT_RISK'">En riesgo de retraso (menos de 48h)</h2>
          <h2 v-else-if="order.traffic_light === 'LATE'">Orden atrasada respecto a la fecha prometida</h2>
          <h2 v-else>Orden completada y entregada</h2>
          <p>Fecha prometida: <strong>{{ formatDate(order.promised_at) }}</strong></p>
        </div>
      </section>

      <!-- Ficha de Datos del Pedido -->
      <section class="order-summary-card">
        <div class="summary-grid">
          <div>
            <span class="label">Cliente:</span>
            <span class="val bold">{{ order.client_name }}</span>
          </div>
          <div>
            <span class="label">Cantidad:</span>
            <span class="val bold">{{ order.quantity }} {{ order.unit }}</span>
          </div>
          <div>
            <span class="label">Estado de la Orden:</span>
            <span class="val status-pill" :class="'status-' + order.status.toLowerCase()">
              {{ order.status === 'ACTIVE' ? 'En Producción' : order.status }}
            </span>
          </div>
          <div>
            <span class="label">Fecha de Recepción:</span>
            <span class="val">{{ formatDate(order.created_at) }}</span>
          </div>
        </div>

        <div v-if="order.specification" class="specification-box">
          <span class="label">Especificación / Ficha técnica:</span>
          <p class="spec-text">{{ order.specification }}</p>
        </div>
      </section>

      <!-- Flujo de Etapas de Producción (Micro-MES) con Fotos de Evidencia -->
      <section class="stages-section">
        <div class="stages-header">
          <div>
            <h2>Flujo Productivo por Estaciones</h2>
            <p class="stages-subtitle">Solo puedes completar la estación que está actualmente en proceso.</p>
          </div>
        </div>

        <div class="stages-stepper">
          <div
            v-for="stage in order.stages"
            :key="stage.id"
            class="stage-card"
            :class="'stage-' + stage.status.toLowerCase()"
          >
            <div class="stage-top-row">
              <div class="stage-left">
                <span class="stage-num">{{ stage.position }}</span>
                <div class="stage-text">
                  <h3>{{ stage.name }}</h3>
                  <span class="stage-status-label">
                    <span v-if="stage.status === 'COMPLETED'">✓ Completado {{ formatShortTime(stage.completed_at) }}</span>
                    <span v-else-if="stage.status === 'IN_PROGRESS'">⚙️ En Proceso en este momento</span>
                    <span v-else>⏳ En espera</span>
                  </span>
                </div>
              </div>

              <!-- Botones de Acción (Avance y Foto) -->
              <div class="stage-actions">
                <!-- Botón de Tomar Foto (Cámara celular) -->
                <label
                  v-if="canAdvance && (stage.status === 'IN_PROGRESS' || stage.status === 'COMPLETED')"
                  class="btn-photo-capture"
                  :class="{ disabled: uploadingPhotoStage === stage.position }"
                  title="Tomar foto de evidencia de esta estación"
                >
                  <input
                    type="file"
                    accept="image/*"
                    capture="environment"
                    class="hidden-file-input"
                    :disabled="uploadingPhotoStage === stage.position"
                    @change="handlePhotoFileSelected(stage.position, $event)"
                  />
                  <span>{{ uploadingPhotoStage === stage.position ? 'Subiendo…' : '📷 Foto' }}</span>
                </label>

                <button
                  v-if="stage.status === 'IN_PROGRESS' && canAdvance"
                  class="btn-complete-stage"
                  :disabled="updatingStage === stage.position"
                  @click="handleAdvanceStage(stage, 'COMPLETED')"
                >
                  {{ updatingStage === stage.position ? 'Guardando...' : '✓ Completar Etapa' }}
                </button>

                <span v-else-if="stage.status === 'COMPLETED'" class="done-badge">
                  ✓ Listo
                </span>
              </div>
            </div>

            <!-- Galería de Fotos Adjuntas en esta Estación -->
            <div v-if="getStagePhotos(stage.position).length > 0" class="stage-photos-gallery">
              <span class="photos-label">Evidencias en planta:</span>
              <div class="photos-grid">
                <div
                  v-for="photo in getStagePhotos(stage.position)"
                  :key="photo.id"
                  class="photo-thumb-wrapper"
                  @click="previewPhoto = photo"
                >
                  <img :src="photo.photo_data" alt="Evidencia de estación" class="photo-thumb" />
                  <span class="photo-time">{{ formatShortTime(photo.created_at) }}</span>
                </div>
              </div>
            </div>
          </div>
        </div>
      </section>
    </div>

    <!-- Modal Ficha Viajera imprimible -->
    <div v-if="showTicketModal && order" class="modal-backdrop" @click.self="showTicketModal = false">
      <div class="ticket-card printable-ticket">
        <div class="ticket-header">
          <div class="ticket-title">
            <span class="ticket-brand">TALLERFLOW · FICHA VIAJERA</span>
            <h2>{{ order.code }}</h2>
            <p>{{ order.product }}</p>
          </div>
          <button class="btn-close-modal" @click="showTicketModal = false">✕</button>
        </div>

        <div class="ticket-body">
          <div class="ticket-meta">
            <p><strong>Cliente:</strong> {{ order.client_name }}</p>
            <p><strong>Cantidad:</strong> {{ order.quantity }} {{ order.unit }}</p>
            <p><strong>Fecha Entrega:</strong> {{ formatDate(order.promised_at) }}</p>
            <p v-if="order.specification"><strong>Detalles:</strong> {{ order.specification }}</p>
          </div>

          <div class="ticket-reference">
            <span class="ticket-reference__label">Código de referencia</span>
            <strong>{{ order.code }}</strong>
            <p>Ingresa este código en la cola de producción para abrir la orden.</p>
          </div>
        </div>

        <div class="ticket-actions no-print">
          <button class="btn-print" type="button" @click="printTicket">
            🖨️ Imprimir Hoja de Ruta
          </button>
          <button class="btn-secondary" type="button" @click="showTicketModal = false">
            Cerrar
          </button>
        </div>
      </div>
    </div>

    <!-- Modal Previsualizar Foto Ampliada -->
    <div v-if="previewPhoto" class="modal-backdrop" @click="previewPhoto = null">
      <div class="photo-lightbox-card">
        <img :src="previewPhoto.photo_data" alt="Evidencia ampliada" class="photo-large" />
        <p class="photo-caption">Subido el {{ formatDate(previewPhoto.created_at) }}</p>
      </div>
    </div>
  </main>
</template>

<style scoped>
.order-detail-page {
  max-width: 680px;
  margin: 0 auto;
  padding: 1rem;
}

.page-header {
  margin-bottom: 1rem;
}

.btn-back {
  background: none;
  border: none;
  color: var(--color-primary, #0a5a70);
  font-size: 0.95rem;
  font-weight: 600;
  cursor: pointer;
  padding: 0.25rem 0;
  margin-bottom: 0.4rem;
}

.header-title-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 0.6rem;
  flex-wrap: wrap;
}

.header-left {
  display: flex;
  align-items: center;
  gap: 0.6rem;
}

.order-code-badge {
  font-size: 0.95rem;
  font-weight: 800;
  background: var(--color-primary, #0a5a70);
  color: #fff;
  padding: 0.2rem 0.6rem;
  border-radius: 6px;
}

h1 {
  font-size: 1.35rem;
  color: var(--color-text-dark, #10242d);
  margin: 0;
}

.btn-ticket {
  background: #f8fafc;
  border: 1px solid #cbd5e1;
  border-radius: 8px;
  padding: 0.45rem 0.85rem;
  font-size: 0.85rem;
  font-weight: 700;
  color: #334155;
  cursor: pointer;
  transition: all 0.1s ease;
}

.btn-ticket:hover {
  background: #f1f5f9;
  border-color: #94a3b8;
}

.detail-container {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

/* Banner de Semáforo */
.traffic-banner {
  border-radius: 12px;
  padding: 1rem;
  display: flex;
  gap: 0.85rem;
  align-items: center;
  border: 1px solid transparent;
}

.traffic-icon {
  font-size: 2rem;
  line-height: 1;
}

.traffic-info h2 {
  font-size: 1.1rem;
  margin: 0 0 0.2rem 0;
}

.traffic-info p {
  margin: 0;
  font-size: 0.9rem;
}

.banner-on_time {
  background: #f0fdf4;
  border-color: #86efac;
  color: #14532d;
}

.banner-at_risk {
  background: #fefce8;
  border-color: #fde047;
  color: #713f12;
}

.banner-late {
  background: #fef2f2;
  border-color: #fca5a5;
  color: #7f1d1d;
}

.banner-completed {
  background: #eff6ff;
  border-color: #93c5fd;
  color: #1e3a8a;
}

/* Ficha de datos */
.order-summary-card {
  background: #fff;
  border: 1px solid var(--color-border, #cbd9df);
  border-radius: 10px;
  padding: 1rem;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05);
}

.summary-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 0.75rem;
}

.label {
  display: block;
  font-size: 0.75rem;
  color: #64748b;
  margin-bottom: 0.15rem;
}

.val {
  font-size: 0.95rem;
  color: #1e293b;
}

.val.bold {
  font-weight: 700;
}

.status-pill {
  display: inline-block;
  font-size: 0.8rem;
  font-weight: 700;
  padding: 0.15rem 0.5rem;
  border-radius: 4px;
}

.status-active {
  background: #e0f2fe;
  color: #0369a1;
}

.status-completed {
  background: #dcfce7;
  color: #15803d;
}

.specification-box {
  margin-top: 0.75rem;
  padding-top: 0.75rem;
  border-top: 1px dashed #e2e8f0;
}

.spec-text {
  margin: 0.25rem 0 0 0;
  font-size: 0.9rem;
  color: #334155;
  white-space: pre-wrap;
}

/* Estaciones */
.stages-section {
  background: #fff;
  border: 1px solid var(--color-border, #cbd9df);
  border-radius: 10px;
  padding: 1rem;
}

.stages-header h2 {
  font-size: 1.15rem;
  margin: 0 0 0.2rem 0;
  color: var(--color-primary, #0a5a70);
}

.stages-subtitle {
  font-size: 0.85rem;
  color: #64748b;
  margin: 0 0 1rem 0;
}

.stages-stepper {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.stage-card {
  border: 1px solid #e2e8f0;
  border-radius: 8px;
  padding: 0.85rem 1rem;
  display: flex;
  flex-direction: column;
  gap: 0.65rem;
  transition: all 0.15s ease;
}

.stage-card.stage-in_progress {
  border-color: #3b82f6;
  background: #f8faff;
  box-shadow: 0 0 0 2px rgba(59, 130, 246, 0.15);
}

.stage-card.stage-completed {
  background: #f8fafc;
  opacity: 0.9;
}

.stage-top-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.stage-left {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.stage-num {
  width: 28px;
  height: 28px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-weight: 700;
  font-size: 0.85rem;
  background: #e2e8f0;
  color: #475569;
}

.stage-in_progress .stage-num {
  background: #3b82f6;
  color: #fff;
}

.stage-completed .stage-num {
  background: #22c55e;
  color: #fff;
}

.stage-text h3 {
  margin: 0 0 0.15rem 0;
  font-size: 0.95rem;
  color: #1e293b;
}

.stage-status-label {
  font-size: 0.78rem;
  color: #64748b;
}

.stage-in_progress .stage-status-label {
  color: #2563eb;
  font-weight: 600;
}

.stage-actions {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.btn-complete-stage {
  background: #22c55e;
  color: #fff;
  border: none;
  font-weight: 700;
  font-size: 0.9rem;
  padding: 0.65rem 1.1rem;
  border-radius: 8px;
  cursor: pointer;
  box-shadow: 0 2px 4px rgba(34, 197, 94, 0.3);
}

.btn-complete-stage:active {
  transform: scale(0.97);
}

.btn-photo-capture {
  display: inline-flex;
  align-items: center;
  background: #f1f5f9;
  border: 1px solid #cbd5e1;
  padding: 0.55rem 0.75rem;
  border-radius: 8px;
  font-size: 0.82rem;
  font-weight: 700;
  color: #334155;
  cursor: pointer;
  position: relative;
}

.btn-photo-capture.disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.hidden-file-input {
  position: absolute;
  inset: 0;
  opacity: 0;
  width: 100%;
  height: 100%;
  cursor: pointer;
}

.done-badge {
  font-size: 0.85rem;
  font-weight: 700;
  color: #16a34a;
}

/* Fotos de la etapa */
.stage-photos-gallery {
  border-top: 1px dashed #e2e8f0;
  padding-top: 0.5rem;
}

.photos-label {
  font-size: 0.75rem;
  font-weight: 700;
  color: #64748b;
  display: block;
  margin-bottom: 0.35rem;
}

.photos-grid {
  display: flex;
  gap: 0.5rem;
  flex-wrap: wrap;
}

.photo-thumb-wrapper {
  position: relative;
  width: 64px;
  height: 64px;
  border-radius: 6px;
  overflow: hidden;
  border: 1px solid #cbd5e1;
  cursor: pointer;
}

.photo-thumb {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.photo-time {
  position: absolute;
  bottom: 0;
  left: 0;
  right: 0;
  background: rgba(0, 0, 0, 0.65);
  color: #ffffff;
  font-size: 0.65rem;
  text-align: center;
  padding: 0.1rem 0;
}

/* Modales */
.modal-backdrop {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.6);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 100;
  padding: 1rem;
}

.ticket-card {
  background: #ffffff;
  border-radius: 14px;
  max-width: 480px;
  width: 100%;
  padding: 1.5rem;
  box-shadow: 0 10px 30px rgba(0, 0, 0, 0.2);
}

.ticket-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  border-bottom: 2px solid #00685f;
  padding-bottom: 0.75rem;
  margin-bottom: 1rem;
}

.ticket-brand {
  font-size: 0.72rem;
  font-weight: 800;
  color: #00685f;
  letter-spacing: 0.05em;
  display: block;
}

.ticket-header h2 {
  font-size: 1.5rem;
  font-weight: 800;
  margin: 0.2rem 0 0 0;
}

.btn-close-modal {
  background: none;
  border: none;
  font-size: 1.25rem;
  cursor: pointer;
  color: #64748b;
}

.ticket-body {
  display: flex;
  flex-direction: column;
  gap: 1rem;
  align-items: center;
}

.ticket-meta {
  width: 100%;
  font-size: 0.9rem;
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
}

.ticket-meta p {
  margin: 0;
}

.ticket-reference {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0.5rem;
  padding: 1rem;
  background: #f8fafc;
  border-radius: 12px;
  border: 1px solid #e2e8f0;
}

.ticket-reference__label,
.ticket-reference p {
  font-size: 0.75rem;
  color: #64748b;
  text-align: center;
  margin: 0;
}

.ticket-reference strong {
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 1.35rem;
  letter-spacing: 0.08em;
}

.ticket-actions {
  display: flex;
  justify-content: flex-end;
  gap: 0.75rem;
  margin-top: 1.25rem;
}

.btn-print {
  background: #00685f;
  color: #ffffff;
  border: none;
  border-radius: 8px;
  padding: 0.65rem 1.1rem;
  font-weight: 700;
  cursor: pointer;
}

.photo-lightbox-card {
  max-width: 90vw;
  max-height: 85vh;
  background: #ffffff;
  border-radius: 12px;
  padding: 0.75rem;
  display: flex;
  flex-direction: column;
  align-items: center;
}

.photo-large {
  max-width: 100%;
  max-height: 75vh;
  object-fit: contain;
  border-radius: 8px;
}

.photo-caption {
  margin: 0.5rem 0 0 0;
  font-size: 0.8rem;
  color: #64748b;
}

@media print {
  .no-print, .page-header, .stages-section, .traffic-banner {
    display: none !important;
  }
  .modal-backdrop {
    position: static;
    background: none;
    padding: 0;
  }
  .printable-ticket {
    box-shadow: none;
    border: 1px solid #000;
    max-width: 100%;
  }
}
</style>
