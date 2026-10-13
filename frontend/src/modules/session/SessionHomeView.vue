<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { ArrowRight, CalendarDays, ClipboardList, Plus, RefreshCw } from '@lucide/vue'
import { useSessionStore } from './session.store'
import { ordersApi, type DashboardMetrics, type Order } from '../orders/api'
import UiAlert from '../../components/UiAlert.vue'

const session = useSessionStore()
const metrics = ref<DashboardMetrics>({ active: 0, on_time: 0, at_risk: 0, late: 0, completed: 0 })
const orders = ref<Order[]>([])
const loading = ref(true)
const error = ref('')

const activeOrders = computed(() => orders.value.filter((order) => order.status === 'ACTIVE'))
const todayKey = () => new Date().toLocaleDateString('en-CA')
const dueToday = computed(() => activeOrders.value.filter((order) => new Date(order.promised_at).toLocaleDateString('en-CA') === todayKey()))
const urgentOrders = computed(() => activeOrders.value
  .filter((order) => order.traffic_light === 'LATE' || order.traffic_light === 'AT_RISK')
  .sort((a, b) => new Date(a.promised_at).getTime() - new Date(b.promised_at).getTime())
  .slice(0, 5))
const upcomingOrders = computed(() => activeOrders.value
  .filter((order) => order.traffic_light !== 'LATE' && order.traffic_light !== 'AT_RISK')
  .sort((a, b) => new Date(a.promised_at).getTime() - new Date(b.promised_at).getTime())
  .slice(0, 5))
const attentionOrders = computed(() => urgentOrders.value.length ? urgentOrders.value : upcomingOrders.value)
const stageOrder = ['CORTE', 'COSTURA', 'ACABADO', 'CONTROL DE CALIDAD', 'PLANCHADO', 'EMPAQUE', 'ENTREGA']
const stageCounts = computed(() => stageOrder.map((name) => ({
  name: name.charAt(0) + name.slice(1).toLocaleLowerCase('es-PE'),
  count: activeOrders.value.filter((order) => order.current_stage?.toUpperCase() === name).length,
})))
const maxStageCount = computed(() => Math.max(1, ...stageCounts.value.map((stage) => stage.count)))
const dateLabel = computed(() => new Intl.DateTimeFormat('es-PE', { weekday: 'long', day: 'numeric', month: 'long' }).format(new Date()))

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
  return new Intl.DateTimeFormat('es-PE', { day: '2-digit', month: 'short', hour: '2-digit', minute: '2-digit' }).format(new Date(value))
}

onMounted(loadDashboardData)
</script>

<template>
  <main class="overview-page">
    <header class="overview-header">
      <div>
        <p class="overview-kicker">Panel del taller <span aria-hidden="true">/</span> {{ dateLabel }}</p>
        <h1>{{ session.principal?.workshop.name }}</h1>
        <p>Así marcha la producción en este momento.</p>
      </div>
      <div class="overview-header__actions">
        <button class="ops-icon-button" type="button" title="Actualizar panel" aria-label="Actualizar panel" :disabled="loading" @click="loadDashboardData"><RefreshCw :size="19" /></button>
        <RouterLink class="ops-button ops-button--primary" to="/app/orders/new"><Plus :size="18" /> Nueva orden</RouterLink>
      </div>
    </header>

    <UiAlert v-if="error" :message="error" />

    <section class="overview-metrics" aria-label="Resumen de órdenes">
      <RouterLink to="/app/orders" class="overview-metric overview-metric--main">
        <span>Órdenes activas</span><strong>{{ loading ? '—' : metrics.active }}</strong><small>Ver tablero <ArrowRight :size="15" /></small>
      </RouterLink>
      <RouterLink to="/app/orders?filter=ON_TIME" class="overview-metric">
        <span>A tiempo</span><strong>{{ loading ? '—' : metrics.on_time }}</strong><small class="ops-status ops-status--good">Sin urgencia</small>
      </RouterLink>
      <RouterLink to="/app/orders?filter=AT_RISK" class="overview-metric">
        <span>En riesgo</span><strong>{{ loading ? '—' : metrics.at_risk }}</strong><small class="ops-status" :class="metrics.at_risk ? 'ops-status--warning' : 'ops-status--good'">{{ metrics.at_risk ? 'Menos de 48 h' : 'Sin riesgo' }}</small>
      </RouterLink>
      <RouterLink to="/app/orders?filter=LATE" class="overview-metric">
        <span>Atrasadas</span><strong>{{ loading ? '—' : metrics.late }}</strong><small class="ops-status" :class="metrics.late ? 'ops-status--danger' : 'ops-status--good'">{{ metrics.late ? 'Requieren atención' : 'Sin atrasos' }}</small>
      </RouterLink>
    </section>

    <div class="overview-grid">
      <section class="overview-section overview-section--priority" aria-labelledby="priority-title">
        <div class="overview-section__head"><div><p class="overview-section__kicker">Prioridad</p><h2 id="priority-title">{{ urgentOrders.length ? 'Pedidos que requieren atención' : 'Próximas entregas' }}</h2></div><RouterLink to="/app/orders">Ver todas <ArrowRight :size="16" /></RouterLink></div>
        <div v-if="loading" class="ops-empty">Cargando pedidos...</div>
        <div v-else-if="!attentionOrders.length" class="ops-empty"><ClipboardList :size="24" /><strong>Aún no hay pedidos activos</strong><span>Al crear una orden aparecerá aquí y en el tablero.</span></div>
        <div v-else class="overview-order-list">
          <RouterLink v-for="order in attentionOrders" :key="order.id" :to="`/app/orders/${order.id}`" class="overview-order">
            <span class="overview-order__signal" :class="`overview-order__signal--${order.traffic_light.toLowerCase()}`" aria-hidden="true" />
            <span class="overview-order__identity"><strong>{{ order.product }}</strong><small>{{ order.code }} · {{ order.client_name }}</small></span>
            <span class="overview-order__due"><strong>{{ formatDate(order.promised_at) }}</strong><small>{{ order.current_stage || 'En producción' }}</small></span>
            <ArrowRight :size="17" aria-hidden="true" />
          </RouterLink>
        </div>
      </section>

      <section class="overview-section overview-section--today" aria-labelledby="today-title">
        <div class="overview-section__head"><div><p class="overview-section__kicker">Agenda</p><h2 id="today-title">Entregas de hoy</h2></div><CalendarDays :size="20" aria-hidden="true" /></div>
        <p class="overview-today-count">{{ loading ? '—' : dueToday.length }} <span>{{ dueToday.length === 1 ? 'pedido' : 'pedidos' }} con entrega prevista</span></p>
        <div v-if="!loading && dueToday.length" class="overview-today-list">
          <RouterLink v-for="order in dueToday.slice(0, 4)" :key="order.id" :to="`/app/orders/${order.id}`"><span>{{ order.code }} · {{ order.product }}</span><strong>{{ formatDate(order.promised_at).split(', ').at(-1) }}</strong></RouterLink>
        </div>
        <p v-else-if="!loading" class="overview-today-note">No hay entregas programadas para hoy.</p>
        <RouterLink class="overview-today-link" to="/app/orders?due=today">Revisar pedidos del día <ArrowRight :size="16" /></RouterLink>
      </section>
    </div>

    <section class="overview-flow" aria-labelledby="flow-title">
      <div class="overview-section__head"><div><p class="overview-section__kicker">Producción</p><h2 id="flow-title">Trabajo por estación</h2></div><RouterLink to="/app/orders?view=board">Abrir tablero <ArrowRight :size="16" /></RouterLink></div>
      <div class="overview-flow__grid">
        <div v-for="stage in stageCounts" :key="stage.name" class="overview-flow__item"><span>{{ stage.name }}</span><strong>{{ loading ? '—' : stage.count }}</strong><div class="overview-flow__track"><div :style="{ width: `${stage.count / maxStageCount * 100}%` }" /></div></div>
      </div>
    </section>
  </main>
</template>

<style scoped>
.overview-page { width: min(100%, 82rem); margin: 0 auto; padding: clamp(1rem, 2.5vw, 2.25rem); display: grid; gap: 1.5rem; }
.overview-header, .overview-section__head { display: flex; align-items: start; justify-content: space-between; gap: 1rem; }
.overview-header h1 { margin: .2rem 0; color: var(--tf-ink); font-size: var(--tf-page-title-size); }
.overview-header p:last-child { margin: .2rem 0 0; color: var(--tf-text-subtle); }
.overview-kicker, .overview-section__kicker { margin: 0; color: var(--tf-role-ink); font-size: .77rem; font-weight: 800; text-transform: uppercase; }
.overview-kicker span { color: var(--tf-border); padding: 0 .35rem; }
.overview-header__actions { display: flex; gap: .6rem; align-items: center; }
.overview-metrics { display: grid; grid-template-columns: repeat(4, minmax(0, 1fr)); gap: .75rem; }
.overview-metric { min-height: 9rem; display: flex; flex-direction: column; gap: .35rem; padding: 1.15rem; border: 1px solid var(--tf-border-soft); border-radius: .5rem; background: #fff; color: var(--tf-ink); text-decoration: none; }
.overview-metric:hover, .overview-order:hover { border-color: var(--tf-primary-border); }
.overview-metric--main { background: var(--tf-primary); color: #fff; border-color: var(--tf-primary); }
.overview-metric > span { font-size: .86rem; font-weight: 700; }
.overview-metric > strong { font-size: 2rem; line-height: 1.1; }
.overview-metric > small { display: inline-flex; align-items: center; gap: .3rem; margin-top: auto; font-size: .75rem; font-weight: 700; }
.overview-grid { display: grid; grid-template-columns: minmax(0, 1.7fr) minmax(17rem, 1fr); gap: 1rem; }
.overview-section, .overview-flow { padding: 1.2rem; border: 1px solid var(--tf-border-soft); border-radius: .5rem; background: #fff; }
.overview-section__head { align-items: center; margin-bottom: 1rem; }
.overview-section__head h2 { margin: .2rem 0 0; font-size: 1.125rem; }
.overview-section__head a, .overview-today-link { display: inline-flex; align-items: center; gap: .25rem; color: var(--tf-primary); font-size: .83rem; font-weight: 800; text-decoration: none; white-space: nowrap; }
.overview-section__head a:hover, .overview-today-link:hover { text-decoration: underline; }
.overview-order-list { display: grid; gap: .45rem; }
.overview-order { min-width: 0; display: grid; grid-template-columns: .3rem minmax(0, 1fr) auto auto; gap: .75rem; align-items: center; padding: .65rem .5rem; border-bottom: 1px solid var(--tf-border-soft); color: var(--tf-ink); text-decoration: none; }
.overview-order:last-child { border-bottom: 0; }
.overview-order__signal { width: .3rem; height: 2.65rem; border-radius: 1rem; background: var(--tf-info); }
.overview-order__signal--late { background: var(--tf-danger); }
.overview-order__signal--at_risk { background: var(--tf-warning); }
.overview-order__identity, .overview-order__due { min-width: 0; display: grid; gap: .15rem; }
.overview-order__identity strong, .overview-order__identity small { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.overview-order__identity small, .overview-order__due small { color: var(--tf-text-subtle); font-size: .75rem; }
.overview-order__due { text-align: right; font-size: .8rem; }
.overview-today-count { margin: .5rem 0 1rem; font-size: 2rem; font-weight: 800; }
.overview-today-count span { display: block; color: var(--tf-text-subtle); font-size: .8rem; font-weight: 500; }
.overview-today-list { display: grid; gap: .4rem; margin-bottom: 1rem; }
.overview-today-list a { display: flex; justify-content: space-between; gap: .7rem; color: var(--tf-ink); text-decoration: none; font-size: .82rem; }
.overview-today-list a span { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.overview-today-note { min-height: 3rem; color: var(--tf-text-subtle); font-size: .85rem; }
.overview-flow__grid { display: grid; grid-template-columns: repeat(7, minmax(0, 1fr)); gap: .8rem; }
.overview-flow__item { min-width: 0; display: grid; gap: .4rem; }
.overview-flow__item span { color: var(--tf-text-subtle); font-size: .76rem; overflow-wrap: anywhere; }
.overview-flow__item strong { font-size: 1.2rem; }
.overview-flow__track { height: .35rem; border-radius: 1rem; background: var(--tf-neutral-soft); overflow: hidden; }
.overview-flow__track div { height: 100%; background: var(--tf-primary); border-radius: inherit; }
@media (max-width: 980px) { .overview-flow__grid { grid-template-columns: repeat(4, minmax(0, 1fr)); } }
@media (max-width: 760px) { .overview-metrics { grid-template-columns: repeat(2, minmax(0, 1fr)); } .overview-grid { grid-template-columns: 1fr; } }
@media (max-width: 480px) { .overview-header { flex-direction: column; } .overview-header__actions { width: 100%; } .overview-header__actions .ops-button { flex: 1; } .overview-flow__grid { grid-template-columns: repeat(2, minmax(0, 1fr)); } .overview-order { grid-template-columns: .3rem minmax(0, 1fr) auto; } .overview-order__due { grid-column: 2; grid-row: 2; text-align: left; } }
</style>
