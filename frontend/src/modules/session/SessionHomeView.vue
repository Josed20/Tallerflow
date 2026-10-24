<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { ArrowRight, Check, Download, ClipboardList, Plus, RefreshCw, Smartphone, Users } from '@lucide/vue'
import { useSessionStore } from './session.store'
import { api } from './api'
import { ordersApi, type DashboardMetrics, type Order } from '../orders/api'
import { clientsApi } from '../clients/api'
import { teamApi } from '../team/api'
import { completedStations, productionStations } from '../orders/order-display'
import UiAlert from '../../components/UiAlert.vue'
import UiButton from '../../components/UiButton.vue'
import OrderTicket from '../../components/OrderTicket.vue'

const session = useSessionStore()
const metrics = ref<DashboardMetrics>({ active: 0, on_time: 0, at_risk: 0, late: 0, completed: 0 })
const orders = ref<Order[]>([])
const loading = ref(true)
const error = ref('')

// Exportar datos del taller (Sprint 4 - Lucero)
const showExportModal = ref(false)
const exportingOrders = ref(false)
const exportingClients = ref(false)
const exportError = ref('')

const isManager = computed(() => session.principal?.role === 'OWNER' || session.principal?.role === 'ADMIN')

async function downloadExport(type: 'orders' | 'clients') {
  exportError.value = ''
  if (type === 'orders') exportingOrders.value = true
  else exportingClients.value = true

  try {
    const blob = await api.getBlob(`/api/v1/exports/${type}.csv`)
    const url = window.URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `${type === 'orders' ? 'ordenes' : 'clientes'}_${new Date().toISOString().slice(0, 10)}.csv`
    document.body.appendChild(a)
    a.click()
    document.body.removeChild(a)
    window.URL.revokeObjectURL(url)
  } catch (err: any) {
    if (err?.status === 403) {
      exportError.value = 'No tienes permisos para descargar los datos.'
    } else {
      exportError.value = err?.message || 'Error al exportar los datos.'
    }
  } finally {
    if (type === 'orders') exportingOrders.value = false
    else exportingClients.value = false
  }
}


// Sprint 3 - Lucero: Primeros pasos
const clientsCount = ref(0)
const teamCount = ref(0)
const firstStepsDismissed = ref(false)
const appDownloaded = ref(false)

const workshopId = computed(() => session.principal?.workshop.id ?? 'default')
const hasClients = computed(() => clientsCount.value > 0)
const hasOrders = computed(() => orders.value.length > 0)
const hasTeam = computed(() => teamCount.value > 1)
const hasDownloadedApp = computed(() => appDownloaded.value)
const allStepsCompleted = computed(() => hasClients.value && hasOrders.value && hasTeam.value && hasDownloadedApp.value)
const showFirstSteps = computed(() => {
  if (firstStepsDismissed.value || allStepsCompleted.value) return false
  return session.principal?.role === 'OWNER' || session.principal?.role === 'ADMIN'
})

function dismissFirstSteps() {
  firstStepsDismissed.value = true
  try {
    localStorage.setItem(`tallerflow_first_steps_dismissed_${workshopId.value}`, 'true')
  } catch {}
}

function markAppDownloaded() {
  appDownloaded.value = true
  try {
    localStorage.setItem(`tallerflow_app_downloaded_${workshopId.value}`, 'true')
  } catch {}
}

const activeOrders = computed(() => orders.value.filter((order) => order.status === 'ACTIVE'))
const todayKey = () => new Date().toLocaleDateString('en-CA')
const dueToday = computed(() => activeOrders.value.filter((order) => new Date(order.promised_at).toLocaleDateString('en-CA') === todayKey()))
const byPromise = (a: Order, b: Order) => new Date(a.promised_at).getTime() - new Date(b.promised_at).getTime()
const urgentOrders = computed(() => activeOrders.value.filter((order) => order.traffic_light === 'LATE' || order.traffic_light === 'AT_RISK').sort(byPromise).slice(0, 5))
const upcomingOrders = computed(() => activeOrders.value.filter((order) => order.traffic_light === 'ON_TIME').sort(byPromise).slice(0, 5))
const attentionOrders = computed(() => urgentOrders.value.length ? urgentOrders.value : upcomingOrders.value)
const stageCounts = computed(() => productionStations.map((name) => ({
  name: name === 'CONTROL DE CALIDAD' ? 'Calidad' : name.charAt(0) + name.slice(1).toLocaleLowerCase('es-PE'),
  count: activeOrders.value.filter((order) => order.current_stage?.toUpperCase() === name).length,
})))
const busiestCount = computed(() => Math.max(0, ...stageCounts.value.map((stage) => stage.count)))
const dateLabel = computed(() => new Intl.DateTimeFormat('es-PE', { weekday: 'long', day: 'numeric', month: 'long' }).format(new Date()))
const alarmDetail = computed(() => {
  const parts = [`${dueToday.value.length} ${dueToday.value.length === 1 ? 'sale' : 'salen'} hoy`]
  if (metrics.value.at_risk) parts.push(`${metrics.value.at_risk} en riesgo`)
  parts.push(`${metrics.value.active} en producción`)
  return parts.join(' · ')
})

async function loadDashboardData() {
  loading.value = true
  error.value = ''
  try {
    const [metricsResponse, ordersResponse] = await Promise.all([ordersApi.getDashboard(), ordersApi.list()])
    metrics.value = metricsResponse.data
    orders.value = ordersResponse.data ?? []
  } catch (cause: any) {
    error.value = cause?.message || 'No se pudo cargar el estado del taller.'
  } finally {
    loading.value = false
  }
}

function formatDate(value: string) {
  return new Intl.DateTimeFormat('es-PE', { weekday: 'short', day: 'numeric', month: 'short', hour: '2-digit', minute: '2-digit' }).format(new Date(value))
}

function formatTime(value: string) {
  return new Intl.DateTimeFormat('es-PE', { hour: '2-digit', minute: '2-digit' }).format(new Date(value))
}

function stationLabel(order: Order) {
  if (order.status === 'COMPLETED') return 'Completada'
  const name = order.current_stage ?? ''
  return name ? name.charAt(0) + name.slice(1).toLocaleLowerCase('es-PE') : 'En producción'
}

onMounted(() => {
  loadDashboardData()
  try {
    firstStepsDismissed.value = localStorage.getItem(`tallerflow_first_steps_dismissed_${workshopId.value}`) === 'true'
    appDownloaded.value = localStorage.getItem(`tallerflow_app_downloaded_${workshopId.value}`) === 'true'
  } catch {}

  if (session.principal?.role === 'OWNER' || session.principal?.role === 'ADMIN') {
    clientsApi.list().then(res => { clientsCount.value = res.data?.length ?? 0 }).catch(() => {})
    teamApi.list().then(res => { teamCount.value = (res.data?.members?.length ?? 0) + (res.data?.invitations?.length ?? 0) }).catch(() => {})
  }
})
</script>

<template>
  <main class="overview-page">
    <header class="overview-header">
      <div>
        <h1>{{ session.principal?.workshop.name }}</h1>
        <p class="overview-date">{{ dateLabel }}</p>
      </div>
      <div class="overview-header__actions">
        <button v-if="isManager" class="ops-button" type="button" title="Descargar mis datos" @click="showExportModal = true">
          <Download :size="17" aria-hidden="true" /> Descargar mis datos
        </button>
        <button class="ops-icon-button" type="button" title="Actualizar panel" aria-label="Actualizar panel" :disabled="loading" @click="loadDashboardData"><RefreshCw :size="19" /></button>
        <RouterLink class="ops-button ops-button--primary" to="/app/orders/new"><Plus :size="18" /> Nueva orden</RouterLink>
      </div>
    </header>

    <UiAlert v-if="error" :message="error" />

    <RouterLink v-if="loading" to="/app/orders" class="alarm alarm--loading" aria-live="polite"><strong>Revisando el taller…</strong></RouterLink>
    <RouterLink v-else-if="metrics.late" to="/app/orders?filter=LATE" class="alarm alarm--late">
      <strong>{{ metrics.late }} {{ metrics.late === 1 ? 'ATRASADA' : 'ATRASADAS' }}</strong>
      <span>{{ alarmDetail }}</span>
      <small>Ver atrasadas <ArrowRight :size="15" aria-hidden="true" /></small>
    </RouterLink>
    <RouterLink v-else to="/app/orders" class="alarm alarm--calm">
      <strong>TODO A TIEMPO</strong>
      <span>{{ alarmDetail }}</span>
      <small>Ver tablero <ArrowRight :size="15" aria-hidden="true" /></small>
    </RouterLink>

    <!-- Primeros pasos (Sprint 3 - Lucero) -->
    <section v-if="showFirstSteps" class="first-steps" aria-labelledby="first-steps-title">
      <div class="first-steps__header">
        <div>
          <h2 id="first-steps-title" class="first-steps__title">Primeros pasos en tu taller</h2>
          <p class="first-steps__subtitle">Completa esta guía para empezar a operar tu producción con TallerFlow.</p>
        </div>
        <button type="button" class="first-steps__dismiss" @click="dismissFirstSteps">
          Ocultar
        </button>
      </div>

      <div class="first-steps__items">
        <RouterLink to="/app/clients" class="first-step" :class="{ 'first-step--done': hasClients }">
          <div class="first-step__check">
            <Check v-if="hasClients" :size="16" />
            <span v-else>1</span>
          </div>
          <div class="first-step__text">
            <strong>Crea tu primer cliente</strong>
            <span>Registra la marca de prendas.</span>
          </div>
          <ArrowRight :size="16" class="first-step__arrow" />
        </RouterLink>

        <RouterLink to="/app/orders/new" class="first-step" :class="{ 'first-step--done': hasOrders }">
          <div class="first-step__check">
            <Check v-if="hasOrders" :size="16" />
            <span v-else>2</span>
          </div>
          <div class="first-step__text">
            <strong>Crea tu primera orden</strong>
            <span>Genera la ficha viajera del lote.</span>
          </div>
          <ArrowRight :size="16" class="first-step__arrow" />
        </RouterLink>

        <RouterLink to="/app/team" class="first-step" :class="{ 'first-step--done': hasTeam }">
          <div class="first-step__check">
            <Check v-if="hasTeam" :size="16" />
            <span v-else>3</span>
          </div>
          <div class="first-step__text">
            <strong>Invita a tu equipo</strong>
            <span>Suma a tus operarios por correo.</span>
          </div>
          <ArrowRight :size="16" class="first-step__arrow" />
        </RouterLink>

        <div class="first-step first-step--clickable" :class="{ 'first-step--done': hasDownloadedApp }" @click="markAppDownloaded">
          <div class="first-step__check">
            <Check v-if="hasDownloadedApp" :size="16" />
            <span v-else>4</span>
          </div>
          <div class="first-step__text">
            <strong>Descarga la app</strong>
            <span>Para escanear y tomar fotos en planta.</span>
          </div>
          <Check v-if="hasDownloadedApp" :size="16" class="first-step__arrow" />
          <ArrowRight v-else :size="16" class="first-step__arrow" />
        </div>
      </div>
    </section>

    <div class="overview-grid">
      <section class="overview-section" aria-labelledby="priority-title">
        <div class="overview-section__head"><h2 id="priority-title">{{ urgentOrders.length ? 'Requieren atención' : 'Próximas entregas' }}</h2><RouterLink to="/app/orders">Ver todas <ArrowRight :size="16" /></RouterLink></div>
        <div v-if="loading" class="ops-empty">Cargando pedidos…</div>
        <div v-else-if="!attentionOrders.length" class="ops-empty"><ClipboardList :size="24" /><strong>Aún no hay pedidos activos</strong><span>Al crear una orden aparecerá aquí y en el tablero.</span></div>
        <div v-else class="overview-tickets">
          <OrderTicket
            v-for="order in attentionOrders"
            :key="order.id"
            :to="`/app/orders/${order.id}`"
            :code="order.code"
            :product="order.product"
            :state="order.traffic_light"
            :completed="completedStations(order)"
            :station="stationLabel(order)"
            :client="order.client_name"
            :due="formatDate(order.promised_at)"
          />
        </div>
      </section>

      <section class="overview-section" aria-labelledby="today-title">
        <div class="overview-section__head"><h2 id="today-title">Salen hoy</h2><RouterLink to="/app/orders?due=today">Ver del día <ArrowRight :size="16" /></RouterLink></div>
        <p class="overview-today-count"><strong>{{ loading ? '—' : dueToday.length }}</strong> <span>{{ dueToday.length === 1 ? 'pedido con entrega hoy' : 'pedidos con entrega hoy' }}</span></p>
        <ul v-if="!loading && dueToday.length" class="overview-today-list">
          <li v-for="order in dueToday.slice(0, 5)" :key="order.id"><RouterLink :to="`/app/orders/${order.id}`"><span class="mono">{{ order.code }}</span><span class="overview-today-list__product">{{ order.product }}</span><strong class="mono">{{ formatTime(order.promised_at) }}</strong></RouterLink></li>
        </ul>
        <p v-else-if="!loading" class="overview-today-note">No hay entregas programadas para hoy.</p>
      </section>
    </div>

    <section class="overview-section" aria-labelledby="flow-title">
      <div class="overview-section__head"><h2 id="flow-title">Trabajo por estación</h2><RouterLink to="/app/orders?view=board">Abrir tablero <ArrowRight :size="16" /></RouterLink></div>
      <ol class="overview-flow">
        <li v-for="stage in stageCounts" :key="stage.name" :class="{ 'is-busiest': !loading && stage.count > 0 && stage.count === busiestCount }">
          <strong>{{ loading ? '—' : stage.count }}</strong><span>{{ stage.name }}</span>
        </li>
      </ol>
    </section>
  
    <!-- Modal Exportar Datos (Sprint 4 - Lucero) -->
    <Teleport to="body">
      <div v-if="showExportModal" class="export-modal-backdrop" @click.self="showExportModal = false">
        <section class="export-modal" role="dialog" aria-modal="true" aria-labelledby="export-dialog-title" @keydown.esc="showExportModal = false">
          <div class="export-modal__head">
            <h2 id="export-dialog-title">Exportar datos del taller</h2>
            <button class="export-modal__close" type="button" aria-label="Cerrar diálogo" @click="showExportModal = false">×</button>
          </div>
          <p class="export-modal__desc">Descarga los registros en formato CSV compatible con Microsoft Excel (UTF-8) para reportes y contabilidad.</p>

          <UiAlert v-if="exportError" :message="exportError" />

          <div class="export-options">
            <div class="export-card">
              <div>
                <strong>Órdenes de producción</strong>
                <span>Incluye código, cliente, prenda, cantidad, fecha de entrega y estado.</span>
              </div>
              <UiButton type="button" :loading="exportingOrders" @click="downloadExport('orders')">
                <Download :size="16" aria-hidden="true" /> Descargar CSV
              </UiButton>
            </div>

            <div class="export-card">
              <div>
                <strong>Clientes y marcas</strong>
                <span>Incluye nombre de la marca, teléfono, correo de contacto y estado.</span>
              </div>
              <UiButton type="button" variant="secondary" :loading="exportingClients" @click="downloadExport('clients')">
                <Download :size="16" aria-hidden="true" /> Descargar CSV
              </UiButton>
            </div>
          </div>

          <div class="export-modal__actions">
            <button type="button" class="btn-secondary" @click="showExportModal = false">Cerrar</button>
          </div>
        </section>
      </div>
    </Teleport>
  </main>
</template>

<style scoped>
.overview-page { width: min(100%, 82rem); margin: 0 auto; padding: clamp(1rem, 2.5vw, 2.25rem); display: grid; gap: 1.25rem; }
.overview-header, .overview-section__head { display: flex; align-items: center; justify-content: space-between; gap: 1rem; }
.overview-header h1 { color: var(--tf-ink); font-size: var(--tf-page-title-size); }
.overview-date { margin: .25rem 0 0; color: var(--tf-text-subtle); font-size: .9rem; }
.overview-date::first-letter { text-transform: uppercase; }
.overview-header__actions { display: flex; gap: .6rem; align-items: center; }

.alarm { display: grid; gap: .3rem; padding: clamp(1rem, 3vw, 1.5rem); border-radius: var(--tf-radius-sm); text-decoration: none; }
.alarm strong { font-family: var(--tf-font-display); font-variation-settings: "wdth" 125; font-weight: 900; font-size: clamp(1.8rem, 8vw, 3.2rem); line-height: 1; letter-spacing: -.01em; }
.alarm span { font-weight: 700; font-size: 1rem; }
.alarm small { display: inline-flex; align-items: center; gap: .3rem; margin-top: .35rem; font-size: .85rem; font-weight: 700; }
.alarm--late { color: var(--tf-on-late); background: var(--tf-late); }
.alarm--late:hover { background: var(--tf-danger-hover); }
.alarm--calm { color: var(--tf-success); background: var(--tf-success-soft); border: 1px solid var(--tf-success-border); }
.alarm--loading { color: var(--tf-text-subtle); background: var(--tf-surface); border: 1px solid var(--tf-border-soft); }
.alarm--loading strong { font-size: 1.4rem; }
.alarm:focus-visible { outline: 3px solid var(--tf-focus); outline-offset: 3px; }

.overview-grid { display: grid; grid-template-columns: minmax(0, 1.6fr) minmax(16rem, 1fr); gap: 1rem; align-items: start; }
.overview-section { padding: 1.1rem; border: 1px solid var(--tf-border-soft); border-radius: var(--tf-radius-sm); background: var(--tf-surface); }
.overview-section__head { margin-bottom: .9rem; }
.overview-section__head h2 { font-size: var(--tf-section-title-size); color: var(--tf-ink); }
.overview-section__head a { display: inline-flex; min-height: 44px; align-items: center; gap: .25rem; color: var(--tf-primary); font-size: .85rem; font-weight: 700; text-decoration: none; white-space: nowrap; }
.overview-section__head a:hover { text-decoration: underline; }
.overview-tickets { display: grid; gap: .6rem; }

.overview-today-count { display: flex; align-items: baseline; gap: .5rem; margin: 0 0 .75rem; }
.overview-today-count strong { font-family: var(--tf-font-display); font-variation-settings: "wdth" 125; font-weight: 900; font-size: 2.2rem; line-height: 1; }
.overview-today-count span { color: var(--tf-text-subtle); font-size: .85rem; }
.overview-today-list { display: grid; margin: 0; padding: 0; list-style: none; }
.overview-today-list a { display: grid; grid-template-columns: auto minmax(0, 1fr) auto; gap: .6rem; align-items: baseline; min-height: 44px; padding: .55rem 0; border-top: 1px solid var(--tf-border-soft); color: var(--tf-ink); font-size: .85rem; text-decoration: none; }
.overview-today-list a:hover .overview-today-list__product { text-decoration: underline; }
.overview-today-list__product { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.overview-today-note { margin: 0; color: var(--tf-text-subtle); font-size: .88rem; }
.mono { font-family: var(--tf-font-mono); font-variant-numeric: tabular-nums; }

.overview-flow { display: grid; grid-template-columns: repeat(7, minmax(0, 1fr)); gap: .5rem; margin: 0; padding: 0; list-style: none; }
.overview-flow li { display: grid; gap: .2rem; min-width: 0; padding: .6rem .25rem; border-radius: var(--tf-radius-sm); background: var(--tf-surface-subtle); text-align: center; }
.overview-flow strong { font-family: var(--tf-font-display); font-variation-settings: "wdth" 112; font-weight: 800; font-size: 1.5rem; line-height: 1; }
.overview-flow span { color: var(--tf-text-subtle); font-size: .72rem; font-weight: 700; overflow-wrap: anywhere; }
.overview-flow li.is-busiest { background: var(--tf-primary-soft); }
.overview-flow li.is-busiest strong { color: var(--tf-primary); }

@media (max-width: 760px) { .overview-grid { grid-template-columns: 1fr; } }
@media (max-width: 520px) { .overview-header { flex-direction: column; align-items: stretch; } .overview-header__actions .ops-button { flex: 1; } .overview-flow { grid-template-columns: repeat(4, minmax(0, 1fr)); } }

/* Primeros pasos */
.first-steps {
  background: var(--tf-surface);
  border: 1px solid var(--tf-border-soft);
  border-left: 4px solid var(--tf-primary);
  border-radius: var(--tf-radius-sm);
  padding: 1.25rem;
  display: grid;
  gap: 1rem;
}
.first-steps__header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 1rem;
}
.first-steps__title {
  margin: 0;
  font-size: 1.15rem;
  font-weight: 700;
  color: var(--tf-ink);
}
.first-steps__subtitle {
  margin: 0.25rem 0 0;
  font-size: 0.875rem;
  color: var(--tf-text-subtle);
}
.first-steps__dismiss {
  background: transparent;
  border: none;
  color: var(--tf-text-subtle);
  font-size: 0.8rem;
  font-weight: 600;
  cursor: pointer;
  padding: 0.25rem 0.5rem;
  border-radius: 4px;
}
.first-steps__dismiss:hover {
  color: var(--tf-ink);
  background: var(--tf-surface-subtle);
}
.first-steps__items {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
  gap: 0.75rem;
}
.first-step {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 0.75rem 0.9rem;
  background: var(--tf-surface-subtle);
  border: 1px solid var(--tf-border-soft);
  border-radius: var(--tf-radius-sm);
  text-decoration: none;
  color: var(--tf-ink);
  min-height: 48px;
  transition: border-color 0.15s, background-color 0.15s;
}
.first-step:hover {
  border-color: var(--tf-primary);
  background: var(--tf-surface);
}
.first-step--clickable {
  cursor: pointer;
}
.first-step__check {
  width: 28px;
  height: 28px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 0.85rem;
  font-weight: 700;
  background: var(--tf-surface);
  border: 2px solid var(--tf-border-soft);
  color: var(--tf-text-subtle);
  flex-shrink: 0;
}
.first-step--done {
  background: var(--tf-surface);
  opacity: 0.85;
}
.first-step--done .first-step__check {
  background: var(--tf-success, #16A34A);
  border-color: var(--tf-success, #16A34A);
  color: #fff;
}
.first-step__text {
  flex: 1;
  min-width: 0;
  display: grid;
  gap: 0.1rem;
}
.first-step__text strong {
  font-size: 0.875rem;
  color: var(--tf-ink);
  line-height: 1.2;
}
.first-step__text span {
  font-size: 0.75rem;
  color: var(--tf-text-subtle);
  line-height: 1.2;
}
.first-step__arrow {
  color: var(--tf-text-subtle);
  flex-shrink: 0;
}
.first-step:hover .first-step__arrow {
  color: var(--tf-primary);
}

/* Modal de Exportación (Sprint 4 - Lucero) */
.export-modal-backdrop {
  position: fixed;
  inset: 0;
  background: rgba(22, 35, 61, 0.48);
  display: grid;
  place-items: center;
  padding: 1rem;
  z-index: 100;
}
.export-modal {
  background: var(--tf-surface);
  border: 1px solid var(--tf-border-soft);
  border-radius: var(--tf-radius, 8px);
  width: min(100%, 30rem);
  padding: 1.5rem;
  display: grid;
  gap: 1rem;
  box-shadow: 0 18px 50px rgba(22, 35, 61, 0.28);
}
.export-modal__head {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.export-modal__head h2 {
  margin: 0;
  font-size: 1.25rem;
  color: var(--tf-ink);
}
.export-modal__close {
  background: transparent;
  border: 0;
  font-size: 1.5rem;
  color: var(--tf-text-subtle);
  cursor: pointer;
  line-height: 1;
}
.export-modal__desc {
  margin: 0;
  font-size: 0.875rem;
  color: var(--tf-text-subtle);
}
.export-options {
  display: grid;
  gap: 0.75rem;
}
.export-card {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 1rem;
  padding: 0.875rem 1rem;
  border: 1px solid var(--tf-border-soft);
  border-radius: 6px;
  background: var(--tf-surface-subtle);
}
.export-card strong {
  display: block;
  font-size: 0.9375rem;
  color: var(--tf-ink);
}
.export-card span {
  display: block;
  font-size: 0.75rem;
  color: var(--tf-text-subtle);
  margin-top: 0.15rem;
}
.export-modal__actions {
  display: flex;
  justify-content: flex-end;
  margin-top: 0.5rem;
}
</style>
