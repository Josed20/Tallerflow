<script setup lang="ts">
import { computed, nextTick, onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import { ArrowRight, Check, Columns3, List, Plus, RefreshCw, ScanLine, Search } from '@lucide/vue'
import { ordersApi, type Order, type TrafficLight } from './api'
import { useSessionStore } from '../session/session.store'
import { canManageOperations, canAdvanceProduction } from '../session/role-policy'
import UiAlert from '../../components/UiAlert.vue'
import { matchesOrderSearch } from './order-search'
import { formatOrderQuantity } from './order-display'
import StageAdvanceDialog from './StageAdvanceDialog.vue'
import OrderScannerDialog from './OrderScannerDialog.vue'

const route = useRoute()
const session = useSessionStore()
const stages = ['CORTE', 'COSTURA', 'ACABADO', 'CONTROL DE CALIDAD', 'PLANCHADO', 'EMPAQUE', 'ENTREGA']
const filterOptions: { value: 'ALL' | TrafficLight; label: string }[] = [
  { value: 'ALL', label: 'Todas' }, { value: 'ON_TIME', label: 'A tiempo' },
  { value: 'AT_RISK', label: 'En riesgo' }, { value: 'LATE', label: 'Atrasadas' },
  { value: 'COMPLETED', label: 'Completadas' },
]
const orders = ref<Order[]>([])
const loading = ref(true)
const busyOrderId = ref<string | null>(null)
const pendingAdvance = ref<Order | null>(null)
const showScanner = ref(false)
const boardScroll = ref<HTMLElement | null>(null)
const error = ref('')
const notice = ref('')
const search = ref('')
const view = ref<'board' | 'list'>(route.query.view === 'list' ? 'list' : 'board')
const trafficFilter = ref<'ALL' | TrafficLight>(['ON_TIME', 'AT_RISK', 'LATE', 'COMPLETED'].includes(String(route.query.filter)) ? route.query.filter as TrafficLight : 'ALL')
const dueTodayOnly = ref(route.query.due === 'today')
const isManager = computed(() => canManageOperations(session.principal?.role))
const canAdvance = computed(() => canAdvanceProduction(session.principal?.role))
const todayKey = () => new Date().toLocaleDateString('en-CA')

function selectTrafficFilter(value: 'ALL' | TrafficLight) {
  trafficFilter.value = value
  dueTodayOnly.value = false
}

function toggleDueToday() {
  dueTodayOnly.value = !dueTodayOnly.value
  if (dueTodayOnly.value) trafficFilter.value = 'ALL'
}

function clearFilters() {
  trafficFilter.value = 'ALL'
  dueTodayOnly.value = false
  search.value = ''
}

const filteredOrders = computed(() => orders.value.filter((order) => {
  if (trafficFilter.value !== 'ALL' && order.traffic_light !== trafficFilter.value) return false
  if (dueTodayOnly.value && new Date(order.promised_at).toLocaleDateString('en-CA') !== todayKey()) return false
  return matchesOrderSearch(order, search.value)
}))
const columns = computed(() => [
  ...stages.map((name, index) => ({ name, label: name.charAt(0) + name.slice(1).toLocaleLowerCase('es-PE'), index, orders: filteredOrders.value.filter((order) => order.status === 'ACTIVE' && order.current_stage?.toUpperCase() === name) })),
  { name: 'COMPLETED', label: 'Completadas', index: stages.length, orders: filteredOrders.value.filter((order) => order.status === 'COMPLETED') },
])
const counts = computed(() => ({
  ALL: orders.value.length,
  ON_TIME: orders.value.filter((order) => order.traffic_light === 'ON_TIME').length,
  AT_RISK: orders.value.filter((order) => order.traffic_light === 'AT_RISK').length,
  LATE: orders.value.filter((order) => order.traffic_light === 'LATE').length,
  COMPLETED: orders.value.filter((order) => order.traffic_light === 'COMPLETED').length,
}))

async function loadOrders(showLoading = true) {
  if (showLoading) loading.value = true
  error.value = ''
  try {
    const response = await ordersApi.list()
    orders.value = response.data ?? []
    return true
  } catch (cause: any) {
    error.value = cause?.message || 'No se pudieron cargar las órdenes.'
    return false
  } finally {
    loading.value = false
  }
}

async function advance(order: Order) {
  if (!canAdvance.value || busyOrderId.value) return
  if (!session.csrfToken) {
    error.value = 'Tu sesión necesita renovarse para registrar avances. Vuelve a iniciar sesión.'
    return
  }
  busyOrderId.value = order.id
  error.value = ''
  notice.value = ''
  try {
    const response = await ordersApi.getByID(order.id)
    const current = response.data.stages?.find((stage) => stage.status === 'IN_PROGRESS')
    if (!current) throw new Error('Esta orden ya no tiene una etapa en curso. Actualiza el tablero.')
    if (current.name.toUpperCase() !== order.current_stage?.toUpperCase()) {
      await loadOrders(false)
      throw new Error('La orden cambió mientras la veías. Revisa su etapa actual antes de avanzar.')
    }
    await ordersApi.updateStage(order.id, current.position, 'COMPLETED', session.csrfToken)
    if (!await loadOrders(false)) {
      error.value = 'La etapa se guardó, pero no pudimos actualizar el tablero. Pulsa Actualizar para ver el estado más reciente.'
      return
    }
    const next = stages[current.position]
    notice.value = next ? `${order.code} pasó a ${next.toLocaleLowerCase('es-PE')}.` : `${order.code} quedó completada.`
    if (view.value === 'board') {
      await nextTick()
      const target = boardScroll.value?.querySelector<HTMLElement>(`[data-stage-index="${current.position}"]`)
      if (target && boardScroll.value) {
        const container = boardScroll.value.getBoundingClientRect()
        const desiredLeft = boardScroll.value.scrollLeft + target.getBoundingClientRect().left - container.left
        boardScroll.value.scrollTo({ left: desiredLeft, behavior: window.matchMedia('(prefers-reduced-motion: reduce)').matches ? 'auto' : 'smooth' })
      }
    }
  } catch (cause: any) {
    error.value = cause?.status === 409 ? 'La orden cambió mientras la veías. Actualiza el tablero y vuelve a intentarlo.' : cause?.message || 'No se pudo avanzar la orden.'
    if (cause?.status === 409) {
      await loadOrders(false)
      error.value = 'La orden cambió mientras la veías. Revisa el tablero y vuelve a intentarlo.'
    }
  } finally {
    busyOrderId.value = null
    pendingAdvance.value = null
  }
}

function formatDate(value: string) {
  return new Intl.DateTimeFormat('es-PE', { day: '2-digit', month: 'short', hour: '2-digit', minute: '2-digit' }).format(new Date(value))
}

function advanceLabel(order: Order) {
  const index = stages.indexOf(order.current_stage?.toUpperCase() ?? '')
  return index >= 0 && index < stages.length - 1 ? `Pasar a ${stages[index + 1].toLocaleLowerCase('es-PE')}` : 'Completar pedido'
}

function trafficLabel(value: TrafficLight) {
  return ({ ON_TIME: 'A tiempo', AT_RISK: 'En riesgo', LATE: 'Atrasada', COMPLETED: 'Completada' })[value]
}

onMounted(loadOrders)
</script>

<template>
  <main class="board-page">
    <header class="board-header">
      <div><p class="board-kicker">Producción</p><h1>Tablero de órdenes</h1><p>El avance de cada pedido, estación por estación.</p></div>
      <div class="board-header__actions">
        <button class="ops-button" type="button" @click="showScanner = true"><ScanLine :size="17" /> Escanear ficha viajera</button>
        <button class="ops-icon-button" type="button" title="Actualizar órdenes" aria-label="Actualizar órdenes" :disabled="loading" @click="loadOrders()"><RefreshCw :size="18" /></button>
        <RouterLink v-if="isManager" class="ops-button ops-button--primary" to="/app/orders/new"><Plus :size="17" /> Nueva orden</RouterLink>
      </div>
    </header>

    <UiAlert v-if="error" :message="error" />
    <div v-if="notice" class="board-notice" role="status"><Check :size="18" /> {{ notice }} <button type="button" aria-label="Cerrar aviso" @click="notice = ''">×</button></div>

    <div class="board-toolbar">
      <div class="board-view-switch" role="group" aria-label="Vista de órdenes">
        <button type="button" :class="{ active: view === 'board' }" :aria-pressed="view === 'board'" @click="view = 'board'"><Columns3 :size="17" /> Tablero</button>
        <button type="button" :class="{ active: view === 'list' }" :aria-pressed="view === 'list'" @click="view = 'list'"><List :size="17" /> Lista</button>
      </div>
      <label class="board-search"><Search :size="17" aria-hidden="true" /><span class="sr-only">Buscar órdenes por código, número, prenda o cliente</span><input v-model="search" type="search" placeholder="Código o número (ej. 3), prenda, cliente" /></label>
    </div>
    <div class="board-filters" aria-label="Filtros de órdenes">
      <button v-for="option in filterOptions" :key="option.value" type="button" :class="{ active: trafficFilter === option.value && !dueTodayOnly }" :aria-pressed="trafficFilter === option.value && !dueTodayOnly" @click="selectTrafficFilter(option.value)"><span>{{ option.label }}</span><strong>{{ counts[option.value] }}</strong></button>
      <button type="button" :class="{ active: dueTodayOnly }" :aria-pressed="dueTodayOnly" @click="toggleDueToday">Entrega hoy</button>
    </div>

    <p class="board-results" role="status">{{ filteredOrders.length }} {{ filteredOrders.length === 1 ? 'orden' : 'órdenes' }} {{ search || trafficFilter !== 'ALL' || dueTodayOnly ? 'en este resultado' : 'en total' }}</p>

    <div v-if="loading" class="ops-empty">Cargando tablero...</div>
    <div v-else-if="filteredOrders.length === 0" class="ops-empty board-empty"><strong>No hay órdenes que coincidan</strong><span v-if="search || trafficFilter !== 'ALL' || dueTodayOnly">No hay pedidos en esta selección. Puedes quitar los filtros.</span><span v-else>Cuando registres una orden, aparecerá en Corte.</span><button v-if="search || trafficFilter !== 'ALL' || dueTodayOnly" class="ops-button" type="button" @click="clearFilters">Ver todas las órdenes</button><RouterLink v-if="isManager && !orders.length" class="ops-button ops-button--primary" to="/app/orders/new">Crear la primera orden</RouterLink></div>

    <section v-else-if="view === 'board'" ref="boardScroll" class="kanban-scroll" aria-label="Tablero por estaciones">
      <div class="kanban-grid">
        <section v-for="column in columns" :key="column.name" class="kanban-column" :data-stage-index="column.index" :aria-label="`${column.label}, ${column.orders.length} órdenes`">
          <header class="kanban-column__header"><div><span class="kanban-column__number">{{ String(column.index + 1).padStart(2, '0') }}</span><h2>{{ column.label }}</h2></div><strong>{{ column.orders.length }}</strong></header>
          <div class="kanban-column__items">
            <article v-for="order in column.orders" :key="order.id" class="kanban-card" :class="`kanban-card--${order.traffic_light.toLowerCase()}`">
              <div class="kanban-card__top"><span>{{ order.code }}</span><span class="ops-status" :class="`ops-status--${order.traffic_light === 'LATE' ? 'danger' : order.traffic_light === 'AT_RISK' ? 'warning' : 'good'}`">{{ trafficLabel(order.traffic_light) }}</span></div>
              <h3>{{ order.product }}</h3><p>{{ order.client_name }}</p>
              <div class="kanban-card__meta"><span>{{ formatOrderQuantity(order.quantity, order.unit) }}</span><span>Entrega {{ formatDate(order.promised_at) }}</span></div>
              <div class="kanban-card__actions"><RouterLink :to="`/app/orders/${order.id}`">Ver orden <ArrowRight :size="14" /></RouterLink><button v-if="canAdvance && order.status === 'ACTIVE'" type="button" :disabled="busyOrderId !== null" @click="pendingAdvance = order">{{ busyOrderId === order.id ? 'Guardando...' : advanceLabel(order) }}</button></div>
            </article>
            <p v-if="!column.orders.length" class="kanban-column__empty">Sin pedidos</p>
          </div>
        </section>
      </div>
    </section>

    <section v-else class="board-list" aria-label="Lista de órdenes">
      <article v-for="order in filteredOrders" :key="order.id" class="board-list__row">
        <div><span class="board-list__code">{{ order.code }}</span><h2>{{ order.product }}</h2><p>{{ order.client_name }} · {{ formatOrderQuantity(order.quantity, order.unit) }}</p></div>
        <div class="board-list__details"><strong>{{ order.status === 'COMPLETED' ? 'Completada' : order.current_stage || 'En producción' }}</strong><span>Entrega {{ formatDate(order.promised_at) }}</span></div>
        <span class="ops-status" :class="`ops-status--${order.traffic_light === 'LATE' ? 'danger' : order.traffic_light === 'AT_RISK' ? 'warning' : 'good'}`">{{ trafficLabel(order.traffic_light) }}</span>
        <div class="board-list__actions"><RouterLink class="ops-button" :to="`/app/orders/${order.id}`">Ver orden</RouterLink><button v-if="canAdvance && order.status === 'ACTIVE'" class="ops-button ops-button--primary" type="button" :disabled="busyOrderId !== null" @click="pendingAdvance = order">{{ busyOrderId === order.id ? 'Guardando...' : advanceLabel(order) }}</button></div>
      </article>
    </section>
    <StageAdvanceDialog v-if="pendingAdvance" :code="pendingAdvance.code" :stage="pendingAdvance.current_stage || 'etapa actual'" :next-stage="stages[stages.indexOf(pendingAdvance.current_stage?.toUpperCase() ?? '') + 1] || null" :busy="busyOrderId !== null" @cancel="pendingAdvance = null" @confirm="advance(pendingAdvance)" />
    <OrderScannerDialog v-if="showScanner" :orders="orders" @close="showScanner = false" />
  </main>
</template>

<style scoped>
.board-page { width: 100%; min-width: 0; margin: 0 auto; padding: clamp(1rem, 2.5vw, 2rem); display: grid; gap: 1rem; }
.board-header, .board-toolbar { display: flex; align-items: center; justify-content: space-between; gap: 1rem; }
.board-header h1 { margin: .2rem 0; font-size: var(--tf-page-title-size); }
.board-header p:last-child { margin: 0; color: var(--tf-text-subtle); font-size: .88rem; }
.board-kicker { margin: 0; color: var(--tf-primary); font-size: .76rem; font-weight: 800; text-transform: uppercase; }
.board-header__actions { display: flex; align-items: center; gap: .5rem; }
.board-toolbar { padding-top: .3rem; }
.board-view-switch { display: inline-flex; gap: .2rem; padding: .2rem; border: 1px solid var(--tf-border-soft); border-radius: .5rem; background: #fff; }
.board-view-switch button { display: inline-flex; min-height: 38px; align-items: center; gap: .4rem; padding: .35rem .7rem; border: 0; border-radius: .3rem; color: var(--tf-text-subtle); background: transparent; font: inherit; font-size: .82rem; font-weight: 800; cursor: pointer; }
.board-view-switch button.active { color: var(--tf-primary-strong); background: var(--tf-primary-soft); }
.board-search { display: flex; width: min(100%, 21rem); min-height: 44px; align-items: center; gap: .5rem; padding: 0 .75rem; border: 1px solid var(--tf-border-soft); border-radius: .5rem; color: var(--tf-text-subtle); background: #fff; }
.board-search:focus-within { border-color: var(--tf-primary); outline: 2px solid var(--tf-focus); }
.board-search input { min-width: 0; width: 100%; border: 0; outline: 0; background: transparent; font-size: .85rem; }
.board-filters { display: flex; gap: .4rem; overflow-x: auto; padding-bottom: .3rem; }
.board-filters button { display: inline-flex; min-height: 38px; flex: 0 0 auto; align-items: center; gap: .4rem; padding: .35rem .65rem; border: 1px solid var(--tf-border-soft); border-radius: .35rem; color: var(--tf-text-subtle); background: #fff; font: inherit; font-size: .8rem; font-weight: 700; cursor: pointer; }
.board-filters button.active { border-color: var(--tf-primary); color: var(--tf-primary-strong); background: var(--tf-primary-soft); }
.board-filters strong { color: var(--tf-ink); }
.board-results { margin: -.6rem 0 0; color: var(--tf-text-subtle); font-size: .78rem; }
.board-notice { display: flex; align-items: center; gap: .5rem; padding: .7rem .8rem; border: 1px solid var(--tf-success-border); border-left: 4px solid var(--tf-success); border-radius: .4rem; color: var(--tf-primary-strong); background: var(--tf-success-soft); font-size: .86rem; font-weight: 700; }
.board-notice button { margin-left: auto; border: 0; background: transparent; color: inherit; font-size: 1.25rem; cursor: pointer; }
.board-empty { padding: 3rem 1rem; border: 1px dashed var(--tf-border); border-radius: .5rem; background: #fff; }
.kanban-scroll { max-width: 100%; overflow-x: auto; padding-bottom: .75rem; scrollbar-color: var(--tf-border) transparent; }
.kanban-grid { display: grid; width: max-content; grid-template-columns: repeat(8, minmax(15.5rem, 15.5rem)); gap: .7rem; }
.kanban-column { min-height: 24rem; border: 1px solid var(--tf-border-soft); border-radius: .5rem; background: #eef3f3; }
.kanban-column__header { display: flex; align-items: center; justify-content: space-between; gap: .5rem; padding: .8rem; border-bottom: 1px solid var(--tf-border-soft); }
.kanban-column__header > div { display: flex; align-items: center; gap: .5rem; min-width: 0; }
.kanban-column__number { color: var(--tf-primary); font-size: .7rem; font-weight: 800; }
.kanban-column__header h2 { margin: 0; font-size: .83rem; }
.kanban-column__header > strong { display: grid; min-width: 1.55rem; height: 1.55rem; place-items: center; border-radius: .3rem; color: var(--tf-primary-strong); background: #fff; font-size: .72rem; }
.kanban-column__items { display: grid; align-content: start; gap: .6rem; padding: .65rem; }
.kanban-column__empty { margin: 1.2rem 0; color: var(--tf-text-subtle); font-size: .78rem; text-align: center; }
.kanban-card { display: grid; gap: .6rem; padding: .85rem; border: 1px solid var(--tf-border-soft); border-left: 3px solid var(--tf-success); border-radius: .4rem; background: #fff; box-shadow: var(--tf-shadow-sm); }
.kanban-card--at_risk { border-left-color: var(--tf-warning); }
.kanban-card--late { border-left-color: var(--tf-danger); }
.kanban-card--completed { border-left-color: var(--tf-info); }
.kanban-card__top { display: flex; align-items: center; justify-content: space-between; gap: .3rem; }
.kanban-card__top > span:first-child { color: var(--tf-primary-strong); font-size: .74rem; font-weight: 800; }
.kanban-card h3 { margin: 0; color: var(--tf-ink); font-size: .94rem; overflow-wrap: anywhere; }
.kanban-card p { margin: -.3rem 0 0; color: var(--tf-text-subtle); font-size: .78rem; }
.kanban-card__meta { display: grid; gap: .15rem; padding-top: .55rem; border-top: 1px solid var(--tf-border-soft); color: var(--tf-text-subtle); font-size: .75rem; }
.kanban-card__actions { display: grid; gap: .35rem; }
.kanban-card__actions a { display: inline-flex; align-items: center; gap: .2rem; color: var(--tf-primary); font-size: .78rem; font-weight: 800; text-decoration: none; }
.kanban-card__actions button { min-height: 38px; border: 1px solid var(--tf-primary-border); border-radius: .3rem; color: var(--tf-primary-strong); background: var(--tf-primary-soft); font: inherit; font-size: .76rem; font-weight: 800; text-transform: capitalize; cursor: pointer; }
.kanban-card__actions button:hover:not(:disabled) { color: #fff; background: var(--tf-primary); }
.kanban-card__actions button:disabled { opacity: .55; cursor: wait; }
.board-list { display: grid; gap: .55rem; }
.board-list__row { display: grid; grid-template-columns: minmax(10rem, 1.5fr) minmax(9rem, 1fr) auto auto; align-items: center; gap: .8rem; padding: .9rem 1rem; border: 1px solid var(--tf-border-soft); border-radius: .4rem; background: #fff; }
.board-list__code { color: var(--tf-primary); font-size: .75rem; font-weight: 800; }
.board-list__row h2 { margin: .1rem 0; font-size: .94rem; }
.board-list__row p { margin: 0; color: var(--tf-text-subtle); font-size: .8rem; }
.board-list__details { display: grid; gap: .2rem; font-size: .78rem; }
.board-list__details span { color: var(--tf-text-subtle); }
.board-list__actions { display: flex; flex-wrap: wrap; gap: .4rem; justify-content: flex-end; }
.board-list__actions .ops-button { min-height: 38px; font-size: .76rem; }
@media (max-width: 1060px) { .board-list__row { grid-template-columns: minmax(10rem, 1fr) auto; } .board-list__actions { grid-column: 1 / -1; justify-content: flex-start; } }
@media (max-width: 620px) { .board-header { align-items: start; flex-direction: column; } .board-header__actions { width: 100%; } .board-header__actions .ops-button { flex: 1; } .board-toolbar { align-items: stretch; flex-direction: column; } .board-search { width: 100%; } .board-list__row { grid-template-columns: 1fr auto; } .board-list__details { grid-column: 1 / -1; } }
</style>
