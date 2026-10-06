<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { ordersApi, type Order, type TrafficLight } from './api'
import { useSessionStore } from '../session/session.store'
import { canManageOperations } from '../session/role-policy'
import UiButton from '../../components/UiButton.vue'
import UiAlert from '../../components/UiAlert.vue'

const router = useRouter()
const session = useSessionStore()

const orders = ref<Order[]>([])
const loading = ref(true)
const error = ref('')
const search = ref('')
const activeFilter = ref<'ALL' | TrafficLight>('ALL')

const isManager = computed(() => canManageOperations(session.principal?.role))

const filteredOrders = computed(() => {
  let list = orders.value

  if (activeFilter.value !== 'ALL') {
    list = list.filter((o) => o.traffic_light === activeFilter.value)
  }

  const q = search.value.trim().toLowerCase()
  if (!q) return list

  return list.filter(
    (o) =>
      o.code.toLowerCase().includes(q) ||
      o.product.toLowerCase().includes(q) ||
      o.client_name.toLowerCase().includes(q)
  )
})

const countOnTime = computed(() => orders.value.filter((o) => o.traffic_light === 'ON_TIME').length)
const countAtRisk = computed(() => orders.value.filter((o) => o.traffic_light === 'AT_RISK').length)
const countLate = computed(() => orders.value.filter((o) => o.traffic_light === 'LATE').length)
const countCompleted = computed(() => orders.value.filter((o) => o.traffic_light === 'COMPLETED').length)

async function fetchOrders() {
  loading.value = true
  error.value = ''
  try {
    const res = await ordersApi.list()
    orders.value = res.data || []
  } catch (err: any) {
    error.value = err?.message || 'No se pudieron cargar las órdenes de trabajo.'
  } finally {
    loading.value = false
  }
}

function formatDate(iso: string) {
  if (!iso) return ''
  const d = new Date(iso)
  return d.toLocaleDateString('es-PE', {
    day: '2-digit',
    month: 'short',
    hour: '2-digit',
    minute: '2-digit',
  })
}

function getTrafficLightLabel(tl: TrafficLight) {
  switch (tl) {
    case 'ON_TIME':
      return 'A tiempo'
    case 'AT_RISK':
      return 'En riesgo (<48h)'
    case 'LATE':
      return 'Atrasada'
    case 'COMPLETED':
      return 'Completada'
  }
}

function openOrder(id: string) {
  router.push(`/app/orders/${id}`)
}

onMounted(() => {
  fetchOrders()
})
</script>

<template>
  <main class="orders-page">
    <header class="page-header">
      <div class="header-nav">
        <button class="btn-back" @click="router.push('/app')">← Inicio</button>
        <h1>Órdenes de Trabajo</h1>
      </div>
      <UiButton v-if="isManager" @click="router.push('/app/orders/new')">
        + Nueva Orden
      </UiButton>
    </header>

    <UiAlert v-if="error" :message="error" />

    <!-- Filtros por Semáforo -->
    <section class="traffic-light-tabs">
      <button
        class="tab-btn"
        :class="{ active: activeFilter === 'ALL' }"
        @click="activeFilter = 'ALL'"
      >
        Todas ({{ orders.length }})
      </button>
      <button
        class="tab-btn tab-on-time"
        :class="{ active: activeFilter === 'ON_TIME' }"
        @click="activeFilter = 'ON_TIME'"
      >
        🟢 A tiempo ({{ countOnTime }})
      </button>
      <button
        class="tab-btn tab-at-risk"
        :class="{ active: activeFilter === 'AT_RISK' }"
        @click="activeFilter = 'AT_RISK'"
      >
        🟡 En riesgo ({{ countAtRisk }})
      </button>
      <button
        class="tab-btn tab-late"
        :class="{ active: activeFilter === 'LATE' }"
        @click="activeFilter = 'LATE'"
      >
        🔴 Atrasadas ({{ countLate }})
      </button>
      <button
        class="tab-btn tab-completed"
        :class="{ active: activeFilter === 'COMPLETED' }"
        @click="activeFilter = 'COMPLETED'"
      >
        ✓ Listas ({{ countCompleted }})
      </button>
    </section>

    <!-- Barra de Búsqueda -->
    <section class="search-bar">
      <input
        v-model="search"
        type="search"
        placeholder="Buscar por código (OT-000001), cliente o producto..."
        class="input-search"
      />
    </section>

    <!-- Estados -->
    <section v-if="loading" class="loading-state">
      <p>Cargando órdenes del taller...</p>
    </section>

    <section v-else-if="filteredOrders.length === 0" class="empty-state">
      <p v-if="search">No hay órdenes que coincidan con la búsqueda.</p>
      <div v-else>
        <p>No hay órdenes registradas en esta categoría.</p>
        <UiButton v-if="isManager" @click="router.push('/app/orders/new')">
          Crear la primera orden
        </UiButton>
      </div>
    </section>

    <!-- Lista de Órdenes -->
    <section v-else class="orders-list">
      <article
        v-for="order in filteredOrders"
        :key="order.id"
        class="order-card"
        :class="'card-' + order.traffic_light.toLowerCase()"
        @click="openOrder(order.id)"
      >
        <div class="order-top">
          <span class="order-code">{{ order.code }}</span>
          <span class="traffic-badge" :class="'badge-' + order.traffic_light.toLowerCase()">
            {{ getTrafficLightLabel(order.traffic_light) }}
          </span>
        </div>

        <h3 class="order-product">{{ order.product }}</h3>
        <p class="order-client">Cliente: <strong>{{ order.client_name }}</strong></p>

        <div class="order-meta-grid">
          <div>
            <span class="meta-label">Cantidad:</span>
            <span class="meta-val">{{ order.quantity }} {{ order.unit }}</span>
          </div>
          <div>
            <span class="meta-label">Entrega prometida:</span>
            <span class="meta-val">{{ formatDate(order.promised_at) }}</span>
          </div>
        </div>

        <div class="order-footer">
          <span class="stage-pill">
            📍 Estación: <strong>{{ order.current_stage || 'Corte' }}</strong>
          </span>
          <span class="action-hint">Ver avance →</span>
        </div>
      </article>
    </section>
  </main>
</template>

<style scoped>
.orders-page {
  max-width: 680px;
  margin: 0 auto;
  padding: 1rem;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 1rem;
  gap: 0.5rem;
}

.header-nav {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.btn-back {
  background: none;
  border: none;
  color: var(--color-primary, #0a5a70);
  font-size: 0.95rem;
  font-weight: 600;
  cursor: pointer;
  padding: 0.25rem 0.5rem;
}

h1 {
  font-size: 1.4rem;
  color: var(--color-text-dark, #10242d);
  margin: 0;
}

.traffic-light-tabs {
  display: flex;
  gap: 0.35rem;
  overflow-x: auto;
  padding-bottom: 0.5rem;
  margin-bottom: 0.75rem;
  -webkit-overflow-scrolling: touch;
}

.tab-btn {
  white-space: nowrap;
  padding: 0.4rem 0.75rem;
  border: 1px solid var(--color-border, #cbd9df);
  background: #fff;
  border-radius: 20px;
  font-size: 0.82rem;
  font-weight: 600;
  color: #475569;
  cursor: pointer;
  transition: all 0.15s ease;
}

.tab-btn.active {
  background: var(--color-primary, #0a5a70);
  color: #fff;
  border-color: var(--color-primary, #0a5a70);
}

.search-bar {
  margin-bottom: 1rem;
}

.input-search {
  width: 100%;
  padding: 0.65rem 0.85rem;
  border: 1px solid var(--color-border, #cbd9df);
  border-radius: 8px;
  font-size: 0.95rem;
  background: #fff;
}

.orders-list {
  display: flex;
  flex-direction: column;
  gap: 0.85rem;
}

.order-card {
  background: #fff;
  border: 1px solid var(--color-border, #cbd9df);
  border-radius: 10px;
  padding: 1rem;
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.06);
  cursor: pointer;
  transition: transform 0.1s ease, box-shadow 0.1s ease;
  border-left: 5px solid #cbd9df;
}

.order-card:active {
  transform: scale(0.99);
}

.order-card.card-on_time {
  border-left-color: #22c55e;
}

.order-card.card-at_risk {
  border-left-color: #eab308;
}

.order-card.card-late {
  border-left-color: #ef4444;
}

.order-card.card-completed {
  border-left-color: #3b82f6;
  opacity: 0.85;
}

.order-top {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 0.4rem;
}

.order-code {
  font-weight: 700;
  font-size: 0.95rem;
  color: var(--color-primary, #0a5a70);
  background: #f0f9ff;
  padding: 0.15rem 0.45rem;
  border-radius: 4px;
}

.traffic-badge {
  font-size: 0.75rem;
  font-weight: 700;
  padding: 0.2rem 0.5rem;
  border-radius: 12px;
}

.badge-on_time {
  background: #dcfce7;
  color: #166534;
}

.badge-at_risk {
  background: #fef9c3;
  color: #854d0e;
}

.badge-late {
  background: #fee2e2;
  color: #991b1b;
}

.badge-completed {
  background: #dbeafe;
  color: #1e40af;
}

.order-product {
  margin: 0 0 0.25rem 0;
  font-size: 1.1rem;
  color: var(--color-text-dark, #10242d);
}

.order-client {
  margin: 0 0 0.5rem 0;
  font-size: 0.9rem;
  color: #475569;
}

.order-meta-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 0.5rem;
  background: #f8fafc;
  padding: 0.5rem 0.75rem;
  border-radius: 6px;
  font-size: 0.85rem;
  margin-bottom: 0.6rem;
}

.meta-label {
  display: block;
  color: #64748b;
  font-size: 0.75rem;
}

.meta-val {
  font-weight: 600;
  color: #1e293b;
}

.order-footer {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding-top: 0.4rem;
  border-top: 1px dashed #e2e8f0;
}

.stage-pill {
  font-size: 0.85rem;
  color: #334155;
}

.action-hint {
  font-size: 0.85rem;
  font-weight: 600;
  color: var(--color-primary, #0a5a70);
}

.loading-state,
.empty-state {
  text-align: center;
  padding: 3rem 1rem;
  color: #64748b;
}
</style>
