<script setup lang="ts">
import { computed, nextTick, ref, onMounted, onUnmounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ordersApi, type Order, type OrderStage, type StageStatus, type StagePhoto } from './api'
import UiAlert from '../../components/UiAlert.vue'
import { useSessionStore } from '../session/session.store'
import { canAdvanceProduction } from '../session/role-policy'
import { compressImageFile } from '../../utils/image'
import { capturePhoto } from '../../native/camera'
import { isNativeApp } from '../../native/platform'
import { Camera, Check, Copy, ExternalLink, Link2, Pencil, Printer, RotateCcw, X } from '@lucide/vue'
import { clientsApi, type Client } from '../clients/api'
import UiField from '../../components/UiField.vue'
import UiButton from '../../components/UiButton.vue'
import { canManageOperations } from '../session/role-policy'
import { completedStations, formatOrderQuantity } from './order-display'
import StatusChip from '../../components/StatusChip.vue'
import StitchSeam from '../../components/StitchSeam.vue'
import StageAdvanceDialog from './StageAdvanceDialog.vue'

const route = useRoute()
const router = useRouter()
const session = useSessionStore()
const nativeApp = isNativeApp()

const order = ref<Order | null>(null)
const loading = ref(true)
const error = ref('')
const notice = ref('')
const updatingStage = ref<number | null>(null)
const uploadingPhotoStage = ref<number | null>(null)
const showTicketModal = ref(false)
const previewPhoto = ref<StagePhoto | null>(null)
const correctionStage = ref<OrderStage | null>(null)
const correctionReason = ref('')
const correctionError = ref('')
const correcting = ref(false)
const pendingAdvanceStage = ref<OrderStage | null>(null)
const showSharePanel = ref(false)
const trackingUrl = ref('')
const generatingLink = ref(false)
const shareError = ref('')
const copied = ref(false)
const trackingInput = ref<HTMLInputElement | null>(null)
const clock = ref(Date.now())
const undoingPosition = ref<number | null>(null)
const showCancelDialog = ref(false)
const cancellationReason = ref('')
const cancellationError = ref('')
const cancelling = ref(false)

// Estado para edición de orden (Sprint 4 - Lucero)
const showEditModal = ref(false)
const editingOrder = ref(false)
const editError = ref('')
const clients = ref<Client[]>([])
const loadingClients = ref(false)
const editForm = ref({
  clientId: '',
  product: '',
  quantity: 0,
  promisedDate: '',
  promisedTime: '18:00',
})

const canAdvance = computed(() => canAdvanceProduction(session.principal?.role))
const canCorrect = computed(() => canManageOperations(session.principal?.role))
const lastCompletedPosition = computed(() => Math.max(0, ...(order.value?.stages?.filter((stage) => stage.status === 'COMPLETED').map((stage) => stage.position) ?? [])))

function canReopen(stage: OrderStage) {
  if (!canCorrect.value || stage.status !== 'COMPLETED' || stage.position !== lastCompletedPosition.value) return false
  if (stage.position === 7) return order.value?.status === 'COMPLETED'
  const next = order.value?.stages?.find((item) => item.position === stage.position + 1)
  return next?.status === 'IN_PROGRESS' && getStagePhotos(stage.position + 1).length === 0
}

function canUndo(stage: OrderStage) {
  if (!stage.completed_at || !stage.completed_by || stage.completed_by !== session.principal?.id) return false
  if (stage.position !== lastCompletedPosition.value) return false
  return clock.value - new Date(stage.completed_at).getTime() <= 5 * 60 * 1000
}

function undoSecondsLeft(stage: OrderStage) {
  if (!stage.completed_at) return 0
  return Math.max(0, Math.ceil((new Date(stage.completed_at).getTime() + 5 * 60 * 1000 - clock.value) / 1000))
}

async function handleUndo(stage: OrderStage) {
  if (!order.value || undoingPosition.value !== null || !session.csrfToken) return
  undoingPosition.value = stage.position
  error.value = ''
  try {
    await ordersApi.undoStage(order.value.id, stage.position, session.csrfToken)
    if (await loadOrder(false)) notice.value = `El avance se deshizo. La orden volvió a ${stage.name}.`
  } catch (err: any) {
    await loadOrder(false)
    error.value = err?.message || 'Ya no se puede deshacer este avance.'
  } finally {
    undoingPosition.value = null
  }
}

async function openEditModal() {
  if (!order.value) return
  editError.value = ''
  let localDate = ''
  let localTime = '18:00'
  if (order.value.promised_at) {
    const d = new Date(order.value.promised_at)
    const yyyy = d.getFullYear()
    const mm = String(d.getMonth() + 1).padStart(2, '0')
    const dd = String(d.getDate()).padStart(2, '0')
    localDate = `${yyyy}-${mm}-${dd}`
    const hh = String(d.getHours()).padStart(2, '0')
    const min = String(d.getMinutes()).padStart(2, '0')
    localTime = `${hh}:${min}`
  }

  editForm.value = {
    clientId: order.value.client_id,
    product: order.value.product,
    quantity: order.value.quantity,
    promisedDate: localDate,
    promisedTime: localTime,
  }
  showEditModal.value = true
  if (clients.value.length === 0) {
    loadingClients.value = true
    try {
      const res = await clientsApi.list()
      clients.value = res.data || []
    } catch {
      // client select will fallback to current client
    } finally {
      loadingClients.value = false
    }
  }
}

async function handleEditOrder() {
  if (!order.value || editingOrder.value) return
  if (!editForm.value.product.trim()) {
    editError.value = 'El nombre de la prenda es obligatorio.'
    return
  }
  if (Number(editForm.value.quantity) <= 0) {
    editError.value = 'La cantidad debe ser mayor a 0.'
    return
  }
  if (!editForm.value.promisedDate) {
    editError.value = 'La fecha prometida es obligatoria.'
    return
  }
  if (!session.csrfToken) {
    editError.value = 'Tu sesión necesita renovarse. Vuelve a iniciar sesión.'
    return
  }

  editingOrder.value = true
  editError.value = ''
  const timeStr = editForm.value.promisedTime || '18:00'
  const combinedISO = new Date(`${editForm.value.promisedDate}T${timeStr}:00`).toISOString()

  try {
    const res = await ordersApi.update(
      order.value.id,
      {
        client_id: editForm.value.clientId,
        product: editForm.value.product.trim(),
        quantity: Number(editForm.value.quantity),
        promised_at: combinedISO,
        version: order.value.version,
      },
      session.csrfToken
    )
    order.value = res.data
    showEditModal.value = false
    notice.value = 'Los datos de la orden fueron actualizados exitosamente.'
  } catch (err: any) {
    if (err && (err.status === 409 || err.code === 'ORDER_VERSION_CONFLICT')) {
      editError.value = 'La orden fue modificada recientemente por otro usuario. Recarga la página para ver los cambios actualizados.'
      await loadOrder(false)
    } else {
      editError.value = (err && err.message) || 'No pudimos actualizar la orden. Verifica los datos ingresados.'
    }
  } finally {
    editingOrder.value = false
  }
}

async function handleCancelOrder() {
  if (!order.value || cancelling.value) return
  if (cancellationReason.value.trim().length < 5) { cancellationError.value = 'Escribe un motivo de al menos 5 caracteres.'; return }
  if (!session.csrfToken) { cancellationError.value = 'Tu sesión necesita renovarse.'; return }
  cancelling.value = true
  cancellationError.value = ''
  try {
    order.value = (await ordersApi.cancel(order.value.id, cancellationReason.value.trim(), session.csrfToken)).data
    showCancelDialog.value = false
    notice.value = 'La orden fue cancelada y salió del flujo de producción.'
  } catch (err: any) {
    cancellationError.value = err?.message || 'No pudimos cancelar la orden.'
  } finally {
    cancelling.value = false
  }
}

async function handleCorrection() {
  if (!order.value || !correctionStage.value || correcting.value) return
  if (correctionReason.value.trim().length < 5) {
    correctionError.value = 'Escribe un motivo de al menos 5 caracteres.'
    return
  }
  if (!session.csrfToken) {
    correctionError.value = 'Tu sesión necesita renovarse. Vuelve a iniciar sesión.'
    return
  }
  correcting.value = true
  error.value = ''
  correctionError.value = ''
  const stageName = correctionStage.value.name
  try {
    await ordersApi.reopenStage(order.value.id, correctionStage.value.position, correctionReason.value.trim(), session.csrfToken)
    correctionStage.value = null
    correctionReason.value = ''
    if (await loadOrder(false)) notice.value = `La orden volvió a ${stageName}. La corrección quedó registrada.`
  } catch (err: any) {
    if (err?.status === 409) {
      correctionStage.value = null
      await loadOrder(false)
      error.value = 'La orden cambió o la siguiente etapa ya tiene evidencia. Mostramos el estado más reciente.'
    } else {
      correctionError.value = err?.message || 'No pudimos corregir la etapa.'
    }
  } finally {
    correcting.value = false
  }
}

async function loadOrder(showLoading = true) {
  const id = route.params.id as string
  if (showLoading) loading.value = true
  error.value = ''
  try {
    const res = await ordersApi.getByID(id)
    order.value = res.data
    return true
  } catch (err: any) {
    error.value = err?.message || 'No se pudo cargar la orden de trabajo.'
    return false
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
  notice.value = ''
  try {
    const latest = await ordersApi.getByID(order.value.id)
    if (latest.data.stages?.find((item) => item.position === stage.position)?.status !== 'IN_PROGRESS') {
      order.value = latest.data
      error.value = 'La orden cambió mientras la veías. Revisa la etapa actual antes de avanzar.'
      return
    }
    await ordersApi.updateStage(order.value.id, stage.position, nextStatus, session.csrfToken)
    if (await loadOrder(false)) {
      const next = order.value?.stages?.find((item) => item.status === 'IN_PROGRESS')
      notice.value = next ? `${stage.name} completada. La orden pasó a ${next.name}.` : `${stage.name} completada. La orden está lista.`
    }
  } catch (err: any) {
    if (err?.status === 409) {
      await loadOrder(false)
      error.value = 'Otro usuario actualizó esta orden. Mostramos el avance más reciente.'
    } else {
      error.value = err?.message || 'Error al actualizar el estado de la etapa.'
    }
  } finally {
    updatingStage.value = null
    pendingAdvanceStage.value = null
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
    if (await loadOrder(false)) notice.value = 'La foto quedó guardada en la etapa.'
  } catch (err: any) {
    error.value = err?.message || 'No se pudo adjuntar la foto a la etapa.'
  } finally {
    uploadingPhotoStage.value = null
    target.value = ''
  }
}

async function handleNativePhoto(stagePosition: number) {
  if (!nativeApp || !order.value || !canAdvance.value || !session.isAuthenticated || uploadingPhotoStage.value) return
  uploadingPhotoStage.value = stagePosition
  error.value = ''
  try {
    const photo = await capturePhoto()
    if (!photo) return
    const dataUrl = await new Promise<string>((resolve, reject) => {
      const reader = new FileReader()
      reader.onerror = () => reject(new Error('No se pudo leer la foto.'))
      reader.onload = () => resolve(String(reader.result))
      reader.readAsDataURL(photo)
    })
    await ordersApi.addStagePhoto(order.value.id, stagePosition, dataUrl, '', session.csrfToken ?? undefined)
    if (await loadOrder(false)) notice.value = 'La foto quedó guardada en la etapa.'
  } catch (err: any) {
    error.value = err?.message || 'No se pudo adjuntar la foto a la etapa.'
  } finally {
    uploadingPhotoStage.value = null
  }
}

function getStagePhotos(position: number): StagePhoto[] {
  if (!order.value?.photos) return []
  return order.value.photos.filter((p) => p.stage_position === position)
}

function nextStageName(position: number) {
  if (position >= 7) return null
  return order.value?.stages?.find((stage) => stage.position === position + 1)?.name
    ?? ['Corte', 'Costura', 'Acabado', 'Control de calidad', 'Planchado', 'Empaque', 'Entrega'][position]
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

const whatsappUrl = computed(() => order.value && trackingUrl.value
  ? `https://wa.me/?text=${encodeURIComponent(`Hola, puedes ver el avance de tu pedido ${order.value.code} (${order.value.product}) aquí: ${trackingUrl.value}`)}`
  : '')

async function generateTrackingLink() {
  if (!order.value || generatingLink.value) return
  if (!session.csrfToken) {
    shareError.value = 'Tu sesión necesita renovarse. Vuelve a iniciar sesión.'
    return
  }
  generatingLink.value = true
  shareError.value = ''
  copied.value = false
  try {
    const response = await ordersApi.createTrackingLink(order.value.id, session.csrfToken)
    trackingUrl.value = `${window.location.origin}/seguimiento/${response.data.token}`
  } catch (err: any) {
    shareError.value = err?.message || 'No pudimos generar el enlace.'
  } finally {
    generatingLink.value = false
  }
}

async function copyTrackingLink() {
  try {
    await navigator.clipboard.writeText(trackingUrl.value)
    copied.value = true
  } catch {
    trackingInput.value?.select()
    shareError.value = 'No pudimos copiar automáticamente. El enlace quedó seleccionado: cópialo con Ctrl+C.'
  }
}

function printTicket() {
  window.print()
}

// The photo viewer is a dialog: focus moves to Cerrar, Escape closes it and
// focus returns to the thumbnail that opened it.
const photoCloseButton = ref<HTMLButtonElement | null>(null)
let photoOpener: HTMLElement | null = null
let clockTimer: number | undefined

async function openPhoto(photo: StagePhoto, event: Event) {
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

onMounted(() => {
  loadOrder()
  clockTimer = window.setInterval(() => { clock.value = Date.now() }, 1000)
  window.addEventListener('keydown', onKeydown)
})

onUnmounted(() => {
  if (clockTimer !== undefined) window.clearInterval(clockTimer)
  window.removeEventListener('keydown', onKeydown)
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
        <div class="header-actions">
          <button v-if="canCorrect" class="btn-ticket" type="button" :aria-expanded="showSharePanel" @click="showSharePanel = !showSharePanel">
            <Link2 :size="17" aria-hidden="true" /> Compartir con el cliente
          </button>
          <button class="btn-ticket" type="button" @click="showTicketModal = true">
            <Printer :size="17" aria-hidden="true" /> Imprimir ficha viajera
          </button>
          <button v-if="canCorrect && order.status !== 'CANCELLED' && order.status !== 'COMPLETED'" class="btn-ticket btn-edit-order" type="button" @click="openEditModal">
            <Pencil :size="17" aria-hidden="true" /> Editar orden
          </button>
          <button v-if="canCorrect && order.status !== 'CANCELLED'" class="btn-ticket btn-cancel-order" type="button" @click="showCancelDialog = true; cancellationReason = ''; cancellationError = ''">
            <X :size="17" aria-hidden="true" /> Cancelar orden
          </button>
        </div>
      </div>
    </header>

    <section v-if="showSharePanel && order" class="share-panel" aria-labelledby="share-title">
      <div class="share-panel__head">
        <h2 id="share-title">Enlace de seguimiento para el cliente</h2>
        <button class="share-panel__close" type="button" aria-label="Cerrar" @click="showSharePanel = false"><X :size="18" /></button>
      </div>
      <p class="share-panel__text">El cliente verá la estación actual, la fecha de entrega, el semáforo y las fotos de evidencia de <strong>{{ order.code }}</strong>, sin crear una cuenta. No verá datos internos ni otras órdenes.</p>
      <template v-if="trackingUrl">
        <label class="share-panel__label" for="tracking-url">Enlace privado</label>
        <div class="share-panel__row">
          <input id="tracking-url" ref="trackingInput" class="share-panel__input" type="text" readonly :value="trackingUrl" @focus="($event.target as HTMLInputElement).select()" />
          <button class="share-panel__primary" type="button" @click="copyTrackingLink"><Copy :size="16" aria-hidden="true" /> {{ copied ? 'Copiado' : 'Copiar' }}</button>
        </div>
        <div class="share-panel__links">
          <a :href="whatsappUrl" target="_blank" rel="noopener noreferrer">Enviar por WhatsApp</a>
          <a :href="trackingUrl" target="_blank" rel="noopener noreferrer"><ExternalLink :size="14" aria-hidden="true" /> Ver como cliente</a>
        </div>
        <p class="share-panel__note">Guárdalo ahora: por seguridad no se puede volver a mostrar. Si lo pierdes, genera uno nuevo.</p>
      </template>
      <template v-else>
        <p class="share-panel__note">Si ya enviaste un enlace de esta orden antes, generar uno nuevo lo desactiva.</p>
        <button class="share-panel__primary" type="button" :disabled="generatingLink" @click="generateTrackingLink">
          <Link2 :size="16" aria-hidden="true" /> {{ generatingLink ? 'Generando...' : 'Generar enlace' }}
        </button>
      </template>
      <p v-if="shareError" class="share-panel__error" role="alert">{{ shareError }}</p>
    </section>

    <UiAlert v-if="error" :message="error" />
    <p v-if="notice" class="detail-notice" role="status">{{ notice }}</p>

    <section v-if="loading" class="loading-state">
      <p>Cargando detalle de la orden...</p>
    </section>

    <div v-else-if="order" class="detail-container">
      <div class="order-delivery-line">
        <StatusChip :state="order.traffic_light" />
        <span class="order-delivery-line__date">Entrega prometida <strong>{{ formatDate(order.promised_at) }}</strong></span>
        <StitchSeam class="order-delivery-line__seam" :completed="completedStations(order)" :tone="order.traffic_light === 'LATE' ? 'late' : 'default'" />
      </div>

      <!-- Ficha de Datos del Pedido -->
      <section class="order-summary-card">
        <div class="summary-grid">
          <div>
            <span class="label">Cliente:</span>
            <span class="val bold">{{ order.client_name }}</span>
          </div>
          <div>
            <span class="label">Cantidad:</span>
            <span class="val bold">{{ formatOrderQuantity(order.quantity, order.unit) }}</span>
          </div>
          <div>
            <span class="label">Estado de la Orden:</span>
            <span class="val status-pill" :class="'status-' + order.status.toLowerCase()">
              {{ order.status === 'ACTIVE' ? 'En producción' : order.status === 'COMPLETED' ? 'Completada' : 'Cancelada' }}
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
            <h2>Etapas de producción</h2>
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
                    <span v-if="stage.status === 'COMPLETED'">Completada {{ formatShortTime(stage.completed_at) }}</span>
                    <span v-else-if="stage.status === 'IN_PROGRESS'">En proceso</span>
                    <span v-else>En espera</span>
                  </span>
                </div>
              </div>

              <!-- Botones de Acción (Avance y Foto) -->
              <div class="stage-actions">
                <!-- Botón de Tomar Foto (Cámara celular) -->
                <button
                  v-if="nativeApp && canAdvance && (stage.status === 'IN_PROGRESS' || stage.status === 'COMPLETED')"
                  type="button"
                  class="btn-photo-capture"
                  :disabled="uploadingPhotoStage === stage.position"
                  @click="handleNativePhoto(stage.position)"
                >
                  <Camera :size="16" aria-hidden="true" /> {{ uploadingPhotoStage === stage.position ? 'Subiendo...' : 'Añadir foto' }}
                </button>
                <label
                  v-else-if="canAdvance && (stage.status === 'IN_PROGRESS' || stage.status === 'COMPLETED')"
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
                  <span><Camera :size="16" aria-hidden="true" /> {{ uploadingPhotoStage === stage.position ? 'Subiendo...' : 'Añadir foto' }}</span>
                </label>

                <button
                  v-if="stage.status === 'IN_PROGRESS' && canAdvance"
                  class="btn-complete-stage"
                  :disabled="updatingStage === stage.position"
                  @click="pendingAdvanceStage = stage"
                >
                  {{ updatingStage === stage.position ? 'Guardando...' : nextStageName(stage.position) ? `Completar etapa · pasar a ${nextStageName(stage.position)?.toLocaleLowerCase('es-PE')}` : 'Completar etapa y pedido' }}
                </button>

                <span v-else-if="stage.status === 'COMPLETED'" class="done-badge"><Check :size="15" aria-hidden="true" /> Listo</span>
                <button v-if="canUndo(stage)" class="btn-correct-stage" type="button" :disabled="undoingPosition === stage.position" @click="handleUndo(stage)"><RotateCcw :size="15" aria-hidden="true" /> {{ undoingPosition === stage.position ? 'Deshaciendo...' : `Deshacer (${undoSecondsLeft(stage)} s)` }}</button>
                <button v-if="canReopen(stage)" class="btn-correct-stage" type="button" @click="correctionStage = stage; correctionReason = ''; correctionError = ''"><RotateCcw :size="15" aria-hidden="true" /> Corregir avance</button>
              </div>
            </div>

            <!-- Galería de Fotos Adjuntas en esta Estación -->
            <div v-if="getStagePhotos(stage.position).length > 0" class="stage-photos-gallery">
              <span class="photos-label">Evidencias en planta:</span>
              <div class="photos-grid">
                <button
                  v-for="photo in getStagePhotos(stage.position)"
                  :key="photo.id"
                  type="button"
                  class="photo-thumb-wrapper"
                  :aria-label="`Ver foto de ${stage.name.toLocaleLowerCase('es-PE')}, ${formatShortTime(photo.created_at)}`"
                  @click="openPhoto(photo, $event)"
                >
                  <img :src="photo.photo_data" alt="" class="photo-thumb" />
                  <span class="photo-time" aria-hidden="true">{{ formatShortTime(photo.created_at) }}</span>
                </button>
              </div>
            </div>
          </div>
        </div>
      </section>
    </div>

    <StageAdvanceDialog v-if="pendingAdvanceStage && order" :code="order.code" :stage="pendingAdvanceStage.name" :next-stage="nextStageName(pendingAdvanceStage.position)" :busy="updatingStage !== null" @cancel="pendingAdvanceStage = null" @confirm="handleAdvanceStage(pendingAdvanceStage, 'COMPLETED')" />

    <!-- Modal Ficha Viajera imprimible -->
    <div v-if="showTicketModal && order" class="modal-backdrop" @click.self="showTicketModal = false">
      <div class="ticket-card printable-ticket">
        <div class="ticket-header">
          <div class="ticket-title">
            <span class="ticket-brand">TALLERFLOW · FICHA VIAJERA</span>
            <h2>{{ order.code }}</h2>
            <p>{{ order.product }}</p>
          </div>
          <button class="btn-close-modal no-print" type="button" aria-label="Cerrar ficha" @click="showTicketModal = false"><X :size="18" /></button>
        </div>

        <div class="ticket-body">
          <div class="ticket-meta">
            <p><strong>Cliente:</strong> {{ order.client_name }}</p>
            <p><strong>Cantidad:</strong> {{ formatOrderQuantity(order.quantity, order.unit) }}</p>
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
            <Printer :size="17" aria-hidden="true" /> Imprimir hoja de ruta
          </button>
          <button class="btn-secondary" type="button" @click="showTicketModal = false">
            Cerrar
          </button>
        </div>
      </div>
    </div>

    <div v-if="correctionStage" class="modal-backdrop" @click.self="correctionStage = null">
      <div class="correction-dialog" role="dialog" aria-modal="true" aria-labelledby="correction-title">
        <h2 id="correction-title">Volver a {{ correctionStage.name }}</h2>
        <p>Solo se puede corregir la última etapa si la siguiente aún no tiene evidencia. Esta acción quedará registrada.</p>
        <form @submit.prevent="handleCorrection">
          <UiAlert v-if="correctionError" :message="correctionError" />
          <label for="correction-reason">Motivo de la corrección</label>
          <textarea id="correction-reason" v-model="correctionReason" required minlength="5" maxlength="300" rows="3" placeholder="Ej. Se marcó por error antes de terminar la revisión" />
          <div class="correction-dialog__actions"><button type="button" class="ops-button" @click="correctionStage = null">Mantener etapa</button><button type="submit" class="ops-button ops-button--primary" :disabled="correcting">{{ correcting ? 'Corrigiendo...' : 'Confirmar corrección' }}</button></div>
        </form>
      </div>
    </div>

        <!-- Modal de Editar Orden (Sprint 4 - Lucero) -->
    <div v-if="showEditModal && order" class="modal-backdrop" @click.self="!editingOrder && (showEditModal = false)">
      <div class="edit-dialog" role="dialog" aria-modal="true" aria-labelledby="edit-order-title">
        <div class="edit-dialog__head">
          <h2 id="edit-order-title">Editar orden {{ order.code }}</h2>
          <button class="btn-close-modal" type="button" aria-label="Cerrar" :disabled="editingOrder" @click="showEditModal = false">
            <X :size="18" />
          </button>
        </div>
        <p class="edit-dialog__desc">Modifica los datos generales de la orden de producción. Los cambios se reflejarán de inmediato.</p>

        <form @submit.prevent="handleEditOrder" class="edit-order-form">
          <UiAlert v-if="editError" :message="editError" />

          <div class="form-group">
            <label for="edit-client">Cliente / Marca</label>
            <select id="edit-client" v-model="editForm.clientId" required :disabled="loadingClients || editingOrder">
              <option v-for="c in clients" :key="c.id" :value="c.id">{{ c.name }}</option>
              <option v-if="!clients.some(c => c.id === editForm.clientId)" :value="editForm.clientId">{{ order.client_name }}</option>
            </select>
          </div>

          <div class="form-group">
            <label for="edit-product">Prenda / Producto</label>
            <input id="edit-product" v-model="editForm.product" type="text" required maxlength="120" placeholder="Ej. Polo Jersey 30/1 cuello redondo" :disabled="editingOrder" />
          </div>

          <div class="form-group">
            <label for="edit-quantity">Cantidad ({{ order.unit || 'unidades' }})</label>
            <input id="edit-quantity" v-model.number="editForm.quantity" type="number" step="any" min="0.01" required :disabled="editingOrder" />
          </div>

          <div class="form-row-2">
            <div class="form-group">
              <label for="edit-date">Fecha de entrega</label>
              <input id="edit-date" v-model="editForm.promisedDate" type="date" required :disabled="editingOrder" />
            </div>
            <div class="form-group">
              <label for="edit-time">Hora</label>
              <input id="edit-time" v-model="editForm.promisedTime" type="time" required :disabled="editingOrder" />
            </div>
          </div>

          <div class="edit-dialog__actions">
            <button type="button" class="btn-secondary" :disabled="editingOrder" @click="showEditModal = false">
              Cancelar
            </button>
            <button type="submit" class="btn-primary" :disabled="editingOrder">
              {{ editingOrder ? 'Guardando cambios...' : 'Guardar cambios' }}
            </button>
          </div>
        </form>
      </div>
    </div>

<div v-if="showCancelDialog && order" class="modal-backdrop" @click.self="!cancelling && (showCancelDialog = false)">
      <div class="correction-dialog" role="dialog" aria-modal="true" aria-labelledby="cancel-order-title">
        <h2 id="cancel-order-title">Cancelar {{ order.code }}</h2>
        <p>La orden saldrá del tablero y el cliente verá que fue cancelada. Esta acción quedará registrada.</p>
        <form @submit.prevent="handleCancelOrder">
          <UiAlert v-if="cancellationError" :message="cancellationError" />
          <label for="cancellation-reason">Motivo de la cancelación</label>
          <textarea id="cancellation-reason" v-model="cancellationReason" required minlength="5" maxlength="300" rows="3" />
          <div class="correction-dialog__actions"><button type="button" class="ops-button" :disabled="cancelling" @click="showCancelDialog = false">Mantener orden</button><button type="submit" class="ops-button ops-button--primary" :disabled="cancelling">{{ cancelling ? 'Cancelando...' : 'Confirmar cancelación' }}</button></div>
        </form>
      </div>
    </div>

    <!-- Modal Previsualizar Foto Ampliada -->
    <div v-if="previewPhoto" class="modal-backdrop" @click.self="closePhoto">
      <div class="photo-lightbox-card" role="dialog" aria-modal="true" aria-label="Foto de evidencia">
        <button ref="photoCloseButton" type="button" class="photo-lightbox-close" @click="closePhoto"><X :size="18" aria-hidden="true" /> Cerrar</button>
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
  color: var(--tf-primary);
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
  font-family: var(--tf-font-mono);
  font-size: 0.95rem;
  font-weight: 800;
  background: var(--tf-primary);
  color: var(--tf-on-primary);
  padding: 0.2rem 0.6rem;
  border-radius: 6px;
}

h1 {
  font-size: 1.35rem;
  color: var(--tf-ink);
  margin: 0;
}

.btn-ticket {
  background: var(--tf-surface-subtle);
  border: 1px solid var(--tf-border-soft);
  border-radius: 8px;
  padding: 0.45rem 0.85rem;
  font-size: 0.85rem;
  font-weight: 700;
  color: var(--tf-ink-soft);
  cursor: pointer;
  transition: all 0.1s ease;
}

.btn-ticket:hover {
  background: var(--tf-neutral-soft);
  border-color: var(--tf-border);
}

.detail-container {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.order-delivery-line { display: flex; align-items: center; flex-wrap: wrap; gap: .55rem 1rem; padding: .3rem 0 .75rem; border-bottom: 1px solid var(--tf-border-soft); }
.order-delivery-line__date { color: var(--tf-text-subtle); font-size: .84rem; }
.order-delivery-line__date strong { color: var(--tf-ink); font-weight: 700; }
.order-delivery-line__seam { flex: 1 1 100%; max-width: 22rem; }

/* Ficha de datos */
.order-summary-card {
  background: var(--tf-surface);
  border: 1px solid var(--tf-border-soft);
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
  color: var(--tf-text-subtle);
  margin-bottom: 0.15rem;
}

.val {
  font-size: 0.95rem;
  color: var(--tf-ink-soft);
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
  background: var(--tf-info-soft);
  color: var(--tf-info);
}

.status-completed {
  background: var(--tf-done-soft);
  color: var(--tf-done);
}

.status-cancelled { background: var(--tf-error-soft); color: var(--tf-error); }
.btn-cancel-order { color: var(--tf-error); border-color: var(--tf-error); }
.btn-edit-order {
  color: var(--tf-primary);
  border-color: var(--tf-primary-border);
}
.btn-edit-order:hover {
  background: var(--tf-primary-soft);
  border-color: var(--tf-primary);
}

/* Modal de edición (Lucero - Sprint 4) */
.edit-dialog {
  background: var(--tf-surface);
  border: 1px solid var(--tf-border-soft);
  border-radius: var(--tf-radius, 8px);
  width: min(100%, 32rem);
  padding: 1.5rem;
  display: grid;
  gap: 1rem;
  box-shadow: 0 10px 30px rgba(0, 0, 0, 0.15);
  max-height: 90vh;
  overflow-y: auto;
}
.edit-dialog__head {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.edit-dialog__head h2 {
  margin: 0;
  font-size: 1.25rem;
  color: var(--tf-ink);
}
.edit-dialog__desc {
  margin: 0;
  font-size: 0.875rem;
  color: var(--tf-text-subtle);
}
.edit-order-form {
  display: grid;
  gap: 0.9rem;
}
.edit-order-form .form-group {
  display: grid;
  gap: 0.35rem;
}
.edit-order-form label {
  font-size: 0.8125rem;
  font-weight: 700;
  color: var(--tf-ink);
}
.edit-order-form input,
.edit-order-form select {
  width: 100%;
  min-height: 48px;
  padding: 0.625rem 0.75rem;
  border: 1px solid var(--tf-border);
  border-radius: 6px;
  background: var(--tf-surface);
  color: var(--tf-ink);
  font: inherit;
  font-size: 0.9rem;
  box-sizing: border-box;
}
.edit-order-form input:focus,
.edit-order-form select:focus {
  outline: 3px solid var(--tf-focus);
  outline-offset: 2px;
  border-color: var(--tf-primary);
}
.form-row-2 {
  display: grid;
  grid-template-columns: 1.5fr 1fr;
  gap: 0.75rem;
}
.edit-dialog__actions {
  display: flex;
  justify-content: flex-end;
  gap: 0.75rem;
  margin-top: 0.5rem;
  padding-top: 0.75rem;
  border-top: 1px solid var(--tf-border-soft);
}


.specification-box {
  margin-top: 0.75rem;
  padding-top: 0.75rem;
  border-top: 1px dashed var(--tf-border-soft);
}

.spec-text {
  margin: 0.25rem 0 0 0;
  font-size: 0.9rem;
  color: var(--tf-ink-soft);
  white-space: pre-wrap;
}

/* Estaciones */
.stages-section {
  background: var(--tf-surface);
  border: 1px solid var(--color-border, var(--tf-border-soft));
  border-radius: 10px;
  padding: 1rem;
}

.stages-header h2 {
  font-size: var(--tf-section-title-size);
  margin: 0 0 0.2rem 0;
  color: var(--tf-ink);
}

.stages-subtitle {
  font-size: 0.85rem;
  color: var(--tf-text-subtle);
  margin: 0 0 1rem 0;
}

.stages-stepper {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.stage-card {
  border: 1px solid var(--tf-border-soft);
  border-radius: 8px;
  padding: 0.85rem 1rem;
  display: flex;
  flex-direction: column;
  gap: 0.65rem;
  transition: all 0.15s ease;
}

.stage-card.stage-in_progress {
  border-color: var(--tf-primary-border);
  background: var(--tf-primary-soft);
  box-shadow: 0 0 0 2px rgb(31 58 104 / 12%);
}

.stage-card.stage-completed {
  background: var(--tf-surface-subtle);
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
  background: var(--tf-border-soft);
  color: var(--tf-text-subtle);
}

.stage-in_progress .stage-num {
  background: var(--tf-primary);
  color: var(--tf-on-primary);
}

.stage-completed .stage-num {
  background: var(--tf-primary);
  color: var(--tf-on-primary);
}

.stage-text h3 {
  text-transform: lowercase;
  margin: 0 0 0.15rem 0;
  font-size: 0.95rem;
  color: var(--tf-ink-soft);
}

.stage-text h3::first-letter { text-transform: uppercase; }

.stage-status-label {
  font-size: 0.78rem;
  color: var(--tf-text-subtle);
}

.stage-in_progress .stage-status-label {
  color: var(--tf-primary);
  font-weight: 600;
}

.stage-actions {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.btn-complete-stage {
  background: var(--tf-primary);
  color: var(--tf-on-primary);
  border: none;
  font-weight: 700;
  font-size: 0.9rem;
  padding: 0.65rem 1.1rem;
  border-radius: 8px;
  cursor: pointer;
  box-shadow: var(--tf-shadow-sm);
}

.btn-complete-stage:active {
  transform: scale(0.97);
}

.btn-photo-capture {
  display: inline-flex;
  align-items: center;
  background: var(--tf-neutral-soft);
  border: 1px solid var(--tf-border-soft);
  padding: 0.55rem 0.75rem;
  border-radius: 8px;
  font-size: 0.82rem;
  font-weight: 700;
  color: var(--tf-ink-soft);
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
  color: var(--tf-success);
}

/* Fotos de la etapa */
.stage-photos-gallery {
  border-top: 1px dashed var(--tf-border-soft);
  padding-top: 0.5rem;
}

.photos-label {
  font-size: 0.75rem;
  font-weight: 700;
  color: var(--tf-text-subtle);
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
  padding: 0;
  border-radius: 6px;
  overflow: hidden;
  border: 1px solid var(--tf-border-soft);
  background: var(--tf-neutral-soft);
  font: inherit;
  cursor: zoom-in;
  touch-action: manipulation;
}

.photo-thumb-wrapper:focus-visible, .photo-lightbox-close:focus-visible {
  outline: 3px solid var(--tf-focus);
  outline-offset: 2px;
}

.photo-lightbox-close {
  display: inline-flex;
  align-items: center;
  gap: .35rem;
  min-height: 44px;
  margin-left: auto;
  padding: .5rem .8rem;
  border: 1px solid var(--tf-border);
  border-radius: var(--tf-radius);
  color: var(--tf-ink);
  background: var(--tf-surface);
  font: inherit;
  font-weight: 700;
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
  color: var(--tf-on-primary);
  font-family: inherit;
  font-size: .68rem;
  font-variant-numeric: tabular-nums;
  line-height: 1.2;
  text-align: center;
  padding: 0.16rem 0.12rem;
  white-space: nowrap;
  letter-spacing: 0;
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
  overscroll-behavior: contain;
}

.ticket-card {
  background: var(--tf-surface);
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
  border-bottom: 2px solid var(--tf-primary);
  padding-bottom: 0.75rem;
  margin-bottom: 1rem;
}

.ticket-brand {
  font-size: 0.72rem;
  font-weight: 800;
  color: var(--tf-primary);
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
  color: var(--tf-text-subtle);
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
  background: var(--tf-surface-subtle);
  border-radius: 12px;
  border: 1px solid var(--tf-border-soft);
}

.ticket-reference__label,
.ticket-reference p {
  font-size: 0.75rem;
  color: var(--tf-text-subtle);
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
  background: var(--tf-primary);
  color: var(--tf-on-primary);
  border: none;
  border-radius: 8px;
  padding: 0.65rem 1.1rem;
  font-weight: 700;
  cursor: pointer;
}

.photo-lightbox-card {
  max-width: 90vw;
  max-height: calc(100vh - 2rem);
  background: var(--tf-surface);
  border-radius: 12px;
  padding: 0.75rem;
  display: flex;
  flex-direction: column;
  align-items: center;
  overflow: auto;
}

.photo-large {
  max-width: 100%;
  max-height: calc(100vh - 7.5rem);
  object-fit: contain;
  border-radius: 8px;
  display: block;
}

.photo-caption {
  flex: none;
  width: 100%;
  margin: 0.65rem 0 0;
  padding: 0.25rem 0.5rem 0;
  text-align: center;
  font-size: 0.85rem;
  line-height: 1.35;
  font-variant-numeric: tabular-nums;
  color: var(--tf-text-subtle);
}

@media (max-width: 480px) {
  .photo-lightbox-card { max-width: calc(100vw - 1rem); padding: 0.5rem; }
  .photo-large { max-height: calc(100vh - 7rem); }
}

.order-detail-page { max-width: 82rem; padding: clamp(1rem, 2.5vw, 2rem); }
.detail-notice { margin: 0 0 1rem; padding: .8rem 1rem; border: 1px solid var(--tf-success-border); border-radius: .4rem; color: var(--tf-primary-strong); background: var(--tf-success-soft); font-size: .88rem; font-weight: 700; }
.btn-back { min-height: 44px; color: var(--tf-primary); }
.order-code-badge { background: var(--tf-primary); }
.header-left { flex-wrap: wrap; }
.header-left h1 { font-size: var(--tf-page-title-size); color: var(--tf-ink); }
.btn-ticket { min-height: 44px; color: var(--tf-primary); border-color: var(--tf-primary-border); background: var(--tf-surface); display: inline-flex; align-items: center; gap: .45rem; }
.header-actions { display: flex; flex-wrap: wrap; gap: .5rem; }
.share-panel { display: grid; gap: .75rem; margin: 0 0 1rem; padding: 1rem 1.1rem; border: 1px solid var(--tf-primary-border); border-radius: .5rem; background: var(--tf-primary-soft); }
.share-panel__head { display: flex; justify-content: space-between; align-items: center; gap: .75rem; }
.share-panel__head h2 { margin: 0; color: var(--tf-ink); font-size: var(--tf-section-title-size); }
.share-panel__close { display: grid; place-items: center; width: 44px; height: 44px; border: 0; border-radius: .4rem; color: var(--tf-ink-soft); background: transparent; cursor: pointer; }
.share-panel__text, .share-panel__note { margin: 0; color: var(--tf-text-subtle); font-size: .88rem; line-height: 1.5; }
.share-panel__label { color: var(--tf-ink); font-size: .82rem; font-weight: 700; }
.share-panel__row { display: flex; gap: .5rem; }
.share-panel__input { flex: 1; min-width: 0; min-height: 44px; padding: .5rem .7rem; border: 1px solid var(--tf-border); border-radius: .4rem; color: var(--tf-ink); background: var(--tf-surface); font: inherit; font-size: .85rem; }
.share-panel__primary { display: inline-flex; align-items: center; justify-content: center; gap: .4rem; min-height: 44px; padding: .5rem 1rem; border: 0; border-radius: .4rem; color: var(--tf-on-primary); background: var(--tf-primary); font: inherit; font-size: .88rem; font-weight: 750; cursor: pointer; justify-self: start; white-space: nowrap; }
.share-panel__primary:hover:not(:disabled) { background: var(--tf-primary-hover); }
.share-panel__primary:disabled { opacity: .65; cursor: wait; }
.share-panel__links { display: flex; flex-wrap: wrap; gap: 1.1rem; }
.share-panel__links a { display: inline-flex; align-items: center; gap: .3rem; min-height: 44px; color: var(--tf-primary-strong); font-size: .88rem; font-weight: 700; }
.share-panel__error { margin: 0; color: var(--tf-error); font-size: .85rem; font-weight: 700; }
.share-panel button:focus-visible, .share-panel a:focus-visible, .share-panel__input:focus-visible { outline: 3px solid var(--tf-focus); outline-offset: 2px; }
@media (max-width: 480px) { .share-panel__row { flex-direction: column; } .header-actions, .header-actions .btn-ticket { width: 100%; justify-content: center; } }
.order-summary-card, .stages-section { border-radius: .5rem; box-shadow: none; }
.stage-card.stage-in_progress { border-color: var(--tf-primary-border); background: var(--tf-primary-soft); box-shadow: none; }
.stage-in_progress .stage-num { background: var(--tf-primary); }
.stage-completed .stage-num { background: var(--tf-primary); }
.btn-complete-stage { min-height: 44px; color: var(--tf-on-primary); background: var(--tf-primary); box-shadow: none; }
.btn-complete-stage:hover:not(:disabled) { background: var(--tf-primary-hover); }
.btn-complete-stage:disabled { opacity: .6; cursor: wait; }
.btn-photo-capture { min-height: 44px; color: var(--tf-primary); border-color: var(--tf-primary-border); background: var(--tf-surface); }
.stage-top-row { gap: 1rem; }
.stage-actions { flex-wrap: wrap; justify-content: flex-end; }
.btn-ticket, .btn-print, .btn-photo-capture, .done-badge, .btn-correct-stage { display: inline-flex; align-items: center; justify-content: center; gap: .4rem; }
.btn-correct-stage { min-height: 44px; padding: .5rem .7rem; border: 1px solid var(--tf-primary); border-radius: var(--tf-radius); color: var(--tf-primary); background: var(--tf-surface); font: inherit; font-size: .79rem; font-weight: 700; cursor: pointer; }
.btn-correct-stage:hover { background: var(--tf-primary-soft); }
.correction-dialog { width: min(100%, 29rem); padding: 1.35rem; border: 1px solid var(--tf-border-soft); border-radius: .5rem; background: var(--tf-surface); box-shadow: var(--tf-shadow-md); }
.correction-dialog h2 { margin: 0 0 .45rem; font-size: 1.15rem; color: var(--tf-ink); }
.correction-dialog p { margin: 0 0 1rem; color: var(--tf-text-subtle); font-size: .87rem; }
.correction-dialog form { display: grid; gap: .6rem; }
.correction-dialog label { font-size: .83rem; font-weight: 700; }
.correction-dialog textarea { width: 100%; padding: .7rem; border: 1px solid var(--tf-border); border-radius: .4rem; font: inherit; resize: vertical; }
.correction-dialog textarea:focus-visible { outline: 2px solid var(--tf-focus); border-color: var(--tf-primary); }
.correction-dialog__actions { display: flex; justify-content: flex-end; flex-wrap: wrap; gap: .5rem; margin-top: .45rem; }
@media (min-width: 980px) { .detail-container { display: grid; grid-template-columns: minmax(0, 1fr) minmax(16rem, 19rem); align-items: start; } .order-delivery-line { grid-column: 1 / -1; } .stages-section { grid-column: 1; grid-row: 2; } .order-summary-card { grid-column: 2; grid-row: 2; } .summary-grid { grid-template-columns: 1fr; } }
@media (max-width: 580px) { .stage-top-row { align-items: stretch; flex-direction: column; } .stage-actions { justify-content: flex-start; } .btn-complete-stage { width: 100%; } .summary-grid { grid-template-columns: 1fr; } }

@media print {
  @page {
    size: A4 portrait;
    margin: 12mm;
  }

  :global(html), :global(body) {
    background: #fff !important;
  }

  :global(body) * {
    visibility: hidden !important;
  }

  .printable-ticket, .printable-ticket * {
    visibility: visible !important;
  }

  .no-print, .page-header, .stages-section, .order-delivery-line,
  .order-summary-card, .detail-notice {
    display: none !important;
  }

  .modal-backdrop {
    position: absolute;
    inset: 0;
    display: block;
    background: none;
    padding: 0;
  }

  .printable-ticket {
    box-sizing: border-box;
    width: 100%;
    max-width: none;
    min-height: 0;
    margin: 0;
    padding: 0;
    border: 1px solid #172b4d;
    border-radius: 0;
    box-shadow: none;
    color: #172b4d;
    background: #fff;
  }

  .ticket-header {
    border-bottom-color: #172b4d;
  }

  .ticket-reference {
    border-color: #b8c5d6;
    background: #f4f7fb;
  }

  .ticket-actions {
    display: none !important;
  }
}
</style>
