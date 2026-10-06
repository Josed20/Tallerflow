<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useSessionStore } from './session.store'
import { canManageOperations } from './role-policy'
import { ordersApi, type Order, type DashboardMetrics } from '../orders/api'
import UiButton from '../../components/UiButton.vue'
import UiAlert from '../../components/UiAlert.vue'

const session = useSessionStore()
const router = useRouter()

const metrics = ref<DashboardMetrics>({
  active: 0,
  on_time: 0,
  at_risk: 0,
  late: 0,
  completed: 0,
})
const recentOrders = ref<Order[]>([])
const loading = ref(true)
const error = ref('')

const isManager = computed(() => {
  return canManageOperations(session.principal?.role)
})

async function loadDashboardData() {
  loading.value = true
  error.value = ''
  try {
    const [metricsRes, ordersRes] = await Promise.all([
      ordersApi.getDashboard(),
      ordersApi.list(),
    ])
    metrics.value = metricsRes.data
    recentOrders.value = (ordersRes.data || []).slice(0, 5) // Show top 5 active/recent
  } catch (err: any) {
    error.value = 'No se pudieron cargar las métricas operativas del taller.'
  } finally {
    loading.value = false
  }
}

function formatDate(iso: string) {
  if (!iso) return ''
  const d = new Date(iso)
  return d.toLocaleDateString('es-PE', { day: '2-digit', month: 'short' })
}

onMounted(() => {
  loadDashboardData()
})
</script>

<template>
    <!-- Header del Taller -->
    <header class="workshop-header">
      <div>
        <p class="eyebrow">Taller de Confección</p>
        <h1>{{ session.principal?.workshop.name }}</h1>
        <p class="user-badge">
          {{ session.principal?.displayName || session.principal?.email }} ·
          <strong>{{ session.principal?.role }}</strong>
        </p>
      </div>
    </header>

    <main class="dashboard-page">

    <UiAlert v-if="error" :message="error" />

    <!-- Semáforo Operativo (Métricas Clave) -->
    <section class="metrics-grid">
      <div class="metric-card card-total" @click="router.push('/app/orders')">
        <span class="metric-num">{{ metrics.active }}</span>
        <span class="metric-label">Órdenes Activas</span>
      </div>

      <div class="metric-card card-ontime" @click="router.push('/app/orders')">
        <span class="metric-num">🟢 {{ metrics.on_time }}</span>
        <span class="metric-label">A tiempo</span>
      </div>

      <div class="metric-card card-atrisk" @click="router.push('/app/orders')">
        <span class="metric-num">🟡 {{ metrics.at_risk }}</span>
        <span class="metric-label">En riesgo (&lt;48h)</span>
      </div>

      <div class="metric-card card-late" @click="router.push('/app/orders')">
        <span class="metric-num">🔴 {{ metrics.late }}</span>
        <span class="metric-label">Atrasadas</span>
      </div>
    </section>

    <!-- Accesos Rápidos Principales -->
    <section class="quick-nav-section">
      <h2>Acciones Rápidas</h2>
      <div class="quick-buttons-grid">
        <button class="quick-nav-btn primary" @click="router.push('/app/orders')">
          📋 <span>Ver Órdenes de Trabajo</span>
        </button>

        <button v-if="isManager" class="quick-nav-btn accent" @click="router.push('/app/orders/new')">
          ➕ <span>Nueva Orden de Trabajo</span>
        </button>

        <button v-if="isManager" class="quick-nav-btn" @click="router.push('/app/clients')">
          👥 <span>Gestión de Clientes</span>
        </button>

        <button v-if="isManager" class="quick-nav-btn" @click="router.push('/app/team')">
          ⚙️ <span>Equipo y Roles</span>
        </button>
      </div>
    </section>

    <!-- Órdenes que requieren atención inmediata -->
    <section class="recent-orders-section">
      <div class="section-top">
        <h2>Órdenes en Producción</h2>
        <RouterLink to="/app/orders" class="text-link">Ver todas →</RouterLink>
      </div>

      <div v-if="loading" class="loading-state">
        <p>Cargando avance del taller...</p>
      </div>

      <div v-else-if="recentOrders.length === 0" class="empty-state">
        <p>No hay órdenes activas en este momento.</p>
        <UiButton v-if="isManager" @click="router.push('/app/orders/new')">
          + Crear primera orden
        </UiButton>
      </div>

      <div v-else class="recent-list">
        <div
          v-for="o in recentOrders"
          :key="o.id"
          class="recent-order-item"
          :class="'item-' + o.traffic_light.toLowerCase()"
          @click="router.push(`/app/orders/${o.id}`)"
        >
          <div class="recent-main">
            <span class="recent-code">{{ o.code }}</span>
            <h4>{{ o.product }}</h4>
            <p class="recent-client">{{ o.client_name }} · {{ o.quantity }} {{ o.unit }}</p>
          </div>
          <div class="recent-side">
            <span class="recent-stage">{{ o.current_stage || 'Corte' }}</span>
            <span class="recent-date">Entrega: {{ formatDate(o.promised_at) }}</span>
          </div>
        </div>
      </div>
    </section>
    </main>
</template>

<style scoped>
.dashboard-page {
  max-width: 680px;
  margin: 0 auto;
  padding: 1rem;
}

.workshop-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 1.25rem;
  background: #fff;
  padding: 1rem;
  border-radius: 12px;
  border: 1px solid var(--color-border, #cbd9df);
}

.eyebrow {
  font-size: 0.75rem;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: #64748b;
  margin: 0;
}

h1 {
  font-size: 1.35rem;
  color: var(--color-primary, #0a5a70);
  margin: 0.15rem 0 0.25rem 0;
}

.user-badge {
  font-size: 0.85rem;
  color: #475569;
  margin: 0;
}

.btn-logout {
  background: none;
  border: 1px solid #cbd9df;
  border-radius: 6px;
  padding: 0.35rem 0.65rem;
  font-size: 0.8rem;
  color: #64748b;
  cursor: pointer;
}

/* Semáforo de Métricas */
.metrics-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 0.65rem;
  margin-bottom: 1.25rem;
}

@media (min-width: 480px) {
  .metrics-grid {
    grid-template-columns: repeat(4, 1fr);
  }
}

.metric-card {
  background: #fff;
  border: 1px solid var(--color-border, #cbd9df);
  border-radius: 10px;
  padding: 0.85rem 0.5rem;
  text-align: center;
  cursor: pointer;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.04);
  transition: transform 0.1s ease;
}

.metric-card:active {
  transform: scale(0.98);
}

.metric-num {
  display: block;
  font-size: 1.35rem;
  font-weight: 800;
  margin-bottom: 0.2rem;
}

.metric-label {
  font-size: 0.75rem;
  font-weight: 600;
  color: #64748b;
}

.card-total .metric-num { color: #0284c7; }
.card-ontime { border-top: 4px solid #22c55e; }
.card-atrisk { border-top: 4px solid #eab308; }
.card-late { border-top: 4px solid #ef4444; }

/* Accesos rápidos */
.quick-nav-section {
  margin-bottom: 1.25rem;
}

.quick-nav-section h2 {
  font-size: 1.1rem;
  margin: 0 0 0.65rem 0;
  color: var(--color-text-dark, #10242d);
}

.quick-buttons-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 0.65rem;
}

.quick-nav-btn {
  background: #fff;
  border: 1px solid var(--color-border, #cbd9df);
  border-radius: 10px;
  padding: 0.85rem 0.75rem;
  font-size: 0.9rem;
  font-weight: 600;
  color: #334155;
  cursor: pointer;
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 0.35rem;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05);
}

.quick-nav-btn.primary {
  background: #f0f9ff;
  border-color: #7dd3fc;
  color: #0369a1;
}

.quick-nav-btn.accent {
  background: #ecfdf5;
  border-color: #6ee7b7;
  color: #047857;
}

/* Órdenes recientes */
.recent-orders-section {
  background: #fff;
  border: 1px solid var(--color-border, #cbd9df);
  border-radius: 12px;
  padding: 1rem;
}

.section-top {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 0.75rem;
}

.section-top h2 {
  font-size: 1.1rem;
  margin: 0;
  color: var(--color-text-dark, #10242d);
}

.text-link {
  font-size: 0.85rem;
  font-weight: 600;
  color: var(--color-primary, #0a5a70);
  text-decoration: none;
}

.recent-list {
  display: flex;
  flex-direction: column;
  gap: 0.6rem;
}

.recent-order-item {
  border: 1px solid #e2e8f0;
  border-left: 4px solid #cbd9df;
  border-radius: 8px;
  padding: 0.75rem;
  display: flex;
  justify-content: space-between;
  align-items: center;
  cursor: pointer;
  background: #fafafa;
}

.recent-order-item.item-on_time { border-left-color: #22c55e; }
.recent-order-item.item-at_risk { border-left-color: #eab308; }
.recent-order-item.item-late { border-left-color: #ef4444; }

.recent-code {
  font-size: 0.75rem;
  font-weight: 700;
  color: #0284c7;
}

.recent-main h4 {
  margin: 0.1rem 0;
  font-size: 0.95rem;
  color: #1e293b;
}

.recent-client {
  margin: 0;
  font-size: 0.8rem;
  color: #64748b;
}

.recent-side {
  text-align: right;
  display: flex;
  flex-direction: column;
  gap: 0.2rem;
}

.recent-stage {
  font-size: 0.8rem;
  font-weight: 700;
  background: #e2e8f0;
  padding: 0.15rem 0.45rem;
  border-radius: 4px;
  color: #334155;
}

.recent-date {
  font-size: 0.75rem;
  color: #64748b;
}

.loading-state,
.empty-state {
  text-align: center;
  padding: 2rem 1rem;
  color: #64748b;
}
</style>
