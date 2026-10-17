<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { ArrowRight, ClipboardList, Plus, RefreshCw } from '@lucide/vue'
import { useSessionStore } from './session.store'
import { ordersApi, type DashboardMetrics, type Order } from '../orders/api'
import { completedStations, productionStations } from '../orders/order-display'
import UiAlert from '../../components/UiAlert.vue'
import OrderTicket from '../../components/OrderTicket.vue'

const session = useSessionStore()
const metrics = ref<DashboardMetrics>({ active: 0, on_time: 0, at_risk: 0, late: 0, completed: 0 })
const orders = ref<Order[]>([])
const loading = ref(true)
const error = ref('')

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

onMounted(loadDashboardData)
</script>

<template>
  <main class="overview-page">
    <header class="overview-header">
      <div>
        <h1>{{ session.principal?.workshop.name }}</h1>
        <p class="overview-date">{{ dateLabel }}</p>
      </div>
      <div class="overview-header__actions">
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
.overview-flow li { display: grid; gap: .2rem; min-width: 0; padding: .6rem .5rem; border-radius: var(--tf-radius-sm); background: var(--tf-surface-subtle); text-align: center; }
.overview-flow strong { font-family: var(--tf-font-display); font-variation-settings: "wdth" 112; font-weight: 800; font-size: 1.5rem; line-height: 1; }
.overview-flow span { overflow: hidden; color: var(--tf-text-subtle); font-size: .72rem; font-weight: 700; text-overflow: ellipsis; white-space: nowrap; }
.overview-flow li.is-busiest { background: var(--tf-primary-soft); }
.overview-flow li.is-busiest strong { color: var(--tf-primary); }

@media (max-width: 760px) { .overview-grid { grid-template-columns: 1fr; } }
@media (max-width: 520px) { .overview-header { flex-direction: column; align-items: stretch; } .overview-header__actions .ops-button { flex: 1; } .overview-flow { grid-template-columns: repeat(4, minmax(0, 1fr)); } }
</style>
