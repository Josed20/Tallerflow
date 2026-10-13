<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import { clientPortalApi } from './api'
import type { ClientPortalOrder } from './types'
import { Cog } from '@lucide/vue'
import { formatOrderQuantity } from '../orders/order-display'
import BrandLogo from '../../components/BrandLogo.vue'

const route = useRoute()
const order = ref<ClientPortalOrder | null>(null)
const loading = ref(true)
const error = ref('')

const completedStagesCount = computed(() => {
  if (!order.value?.stages) return 0
  return order.value.stages.filter((s) => s.status === 'COMPLETED').length
})

const totalStagesCount = computed(() => {
  return order.value?.stages?.length || 7
})

const progressPercent = computed(() => {
  if (!totalStagesCount.value) return 0
  return Math.round((completedStagesCount.value / totalStagesCount.value) * 100)
})

const currentActiveStage = computed(() => {
  if (!order.value?.stages) return null
  return order.value.stages.find((s) => s.status === 'IN_PROGRESS') || order.value.stages[0]
})

function formatDate(iso: string) {
  if (!iso) return ''
  const d = new Date(iso)
  return d.toLocaleDateString('es-PE', { day: '2-digit', month: 'short', year: 'numeric' })
}

function formatTime(iso: string) {
  if (!iso) return ''
  const d = new Date(iso)
  return d.toLocaleTimeString('es-PE', { hour: '2-digit', minute: '2-digit' })
}

async function loadOrder() {
  loading.value = true
  error.value = ''
  try {
    const res = await clientPortalApi.getOrder(String(route.params.id))
    order.value = res.data
  } catch {
    error.value = 'No pudimos mostrar este pedido. El enlace puede haber vencido o no existe.'
  } finally {
    loading.value = false
  }
}

onMounted(loadOrder)
</script>

<template>
  <main class="portal-detail-page">
    <div class="portal-container">
      <!-- Header de Navegación -->
      <nav class="portal-nav">
        <RouterLink class="portal-back-btn" to="/portal/orders">
          ← Volver a mis pedidos
        </RouterLink>
        <div class="portal-brand-badge"><BrandLogo class="portal-brand-logo" /><span>Seguimiento en vivo</span></div>
      </nav>

      <!-- Estado de Carga -->
      <section v-if="loading" class="portal-state-box">
        <div class="spinner"></div>
        <p>Conectando con el taller para obtener el avance…</p>
      </section>

      <!-- Estado de Error -->
      <section v-else-if="error" class="portal-state-box portal-state-box--error" role="alert">
        <p>{{ error }}</p>
        <RouterLink to="/portal/access" class="btn-retry">Solicitar nuevo enlace</RouterLink>
      </section>

      <!-- Detalle de la Orden (Sentirse dentro del taller) -->
      <article v-else-if="order" class="portal-content">
        <!-- Banner Transparencia Taller -->
        <header class="workshop-banner">
          <div class="workshop-meta">
            <div class="workshop-icon">🏭</div>
            <div>
              <h2>Taller de Confección</h2>
              <p class="workshop-subtitle">Actualización en tiempo real desde la línea de producción</p>
            </div>
          </div>
          <span class="live-pill">
            <span class="live-dot"></span> EN VIVO
          </span>
        </header>

        <!-- Tarjeta Resumen del Pedido -->
        <section class="order-hero-card">
          <div class="order-hero-top">
            <div>
              <span class="order-code-badge">{{ order.code }}</span>
              <h1 class="order-title">{{ order.product }}</h1>
            </div>
            <div class="delivery-badge">
              <span class="delivery-label">Entrega prometida</span>
              <strong class="delivery-date">{{ formatDate(order.promised_at) }}</strong>
            </div>
          </div>

          <div class="order-stats-row">
            <div class="stat-item">
              <span class="stat-label">Cantidad solicitada:</span>
              <strong class="stat-value">{{ formatOrderQuantity(order.quantity, order.unit) }}</strong>
            </div>
            <div class="stat-item">
              <span class="stat-label">Estado general:</span>
              <strong class="stat-value text-teal">{{ order.status === 'ACTIVE' ? 'En Confección' : 'Completada' }}</strong>
            </div>
          </div>

          <!-- Barra de Progreso Visual -->
          <div class="progress-container">
            <div class="progress-info">
              <span>Etapa actual: <strong>{{ currentActiveStage?.name || 'Corte' }}</strong></span>
              <span>{{ completedStagesCount }} de {{ totalStagesCount }} etapas ({{ progressPercent }}%)</span>
            </div>
            <div class="progress-bar-track">
              <div class="progress-bar-fill" :style="{ width: `${progressPercent}%` }"></div>
            </div>
          </div>
        </section>

        <!-- Timeline Vertical de las 7 Estaciones Productivas -->
        <section class="timeline-section" aria-labelledby="timeline-title">
          <div class="timeline-header">
            <h2 id="timeline-title">Ruta de Producción en Taller</h2>
            <p>Trazabilidad paso a paso de tus prendas:</p>
          </div>

          <ol class="timeline-stepper">
            <li
              v-for="stage in order.stages"
              :key="stage.position"
              class="timeline-item"
              :class="'timeline-item--' + stage.status.toLowerCase()"
            >
              <div class="timeline-marker">
                <span v-if="stage.status === 'COMPLETED'" class="marker-icon">✓</span>
                <span v-else-if="stage.status === 'IN_PROGRESS'" class="marker-pulse"></span>
                <span v-else class="marker-num">{{ stage.position }}</span>
              </div>

              <div class="timeline-body">
                <div class="timeline-step-top">
                  <h3>{{ stage.position }}. {{ stage.name }}</h3>
                  <span class="status-tag" :class="'tag--' + stage.status.toLowerCase()">
                    {{ stage.status === 'COMPLETED' ? 'Completada' : stage.status === 'IN_PROGRESS' ? 'En confección ahora' : 'En cola' }}
                  </span>
                </div>
                <p v-if="stage.status === 'IN_PROGRESS'" class="stage-active-note">
                  <Cog :size="16" aria-hidden="true" /> Tus prendas están siendo procesadas en esta estación.
                </p>
                <p v-else-if="stage.status === 'COMPLETED'" class="stage-completed-time">
                  Completado {{ formatDate(stage.completed_at || order.updated_at) }}
                </p>
              </div>
            </li>
          </ol>
        </section>

        <!-- Canal Directo WhatsApp con el Taller -->
        <section class="support-card">
          <div class="support-icon">💬</div>
          <h3>¿Consultas o coordinaciones sobre este lote?</h3>
          <p>Comunícate directamente con el taller indicando tu código de orden <strong>{{ order.code }}</strong>.</p>
          <a
            class="btn-whatsapp"
            :href="`https://wa.me/?text=Hola%2C%20quisiera%20consultar%20sobre%20mi%20orden%20${order.code}%20(${encodeURIComponent(order.product)})%20en%20TallerFlow`"
            target="_blank"
            rel="noopener noreferrer"
          >
            Consultar avance por WhatsApp
          </a>
        </section>

        <footer class="portal-footer">
          <p>Trazabilidad transparente garantizada con <strong>TallerFlow</strong></p>
          <p class="footer-time">Última sincronización: {{ formatDate(order.updated_at) }} · {{ formatTime(order.updated_at) }}</p>
        </footer>
      </article>
    </div>
  </main>
</template>

<style scoped>
.portal-detail-page {
  min-height: 100vh;
  background: var(--tf-page);
  color: var(--tf-ink);
  font-family: Manrope, system-ui, sans-serif;
  padding: 1.25rem 1rem 3rem;
}

.portal-container {
  max-width: 600px;
  margin: 0 auto;
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.portal-nav {
  display: flex;
  justify-content: space-between;
  align-items: center;
  flex-wrap: wrap;
  gap: 0.5rem;
}

.portal-back-btn {
  color: #00685f;
  font-weight: 700;
  text-decoration: none;
  font-size: 0.9rem;
}

.portal-brand-badge {
  display: flex;
  align-items: center;
  gap: .5rem;
  font-size: 0.75rem;
  font-weight: 700;
  color: var(--tf-text-subtle);
}

.portal-brand-logo { width: 7.5rem; }

.portal-content {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

/* Banner Taller */
.workshop-banner {
  background: var(--tf-primary-soft);
  border: 1px solid var(--tf-primary-border);
  border-radius: 12px;
  padding: 0.85rem 1rem;
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.workshop-meta {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.workshop-icon {
  font-size: 1.5rem;
}

.workshop-meta h2 {
  font-size: 1rem;
  font-weight: 800;
  margin: 0;
  color: #0b1c30;
}

.workshop-subtitle {
  margin: 0;
  font-size: 0.78rem;
  color: #565e74;
}

.live-pill {
  display: flex;
  align-items: center;
  gap: 0.35rem;
  font-size: 0.7rem;
  font-weight: 800;
  background: #dcfce7;
  color: #166534;
  padding: 0.2rem 0.5rem;
  border-radius: 9999px;
  letter-spacing: 0.05em;
}

.live-dot {
  width: 6px;
  height: 6px;
  background: #22c55e;
  border-radius: 50%;
  animation: pulse 1.5s infinite;
}

@keyframes pulse {
  0% { transform: scale(0.95); opacity: 0.8; }
  50% { transform: scale(1.3); opacity: 1; }
  100% { transform: scale(0.95); opacity: 0.8; }
}

/* Hero Card */
.order-hero-card {
  background: #ffffff;
  border: 1px solid #cbd9df;
  border-radius: 14px;
  padding: 1.25rem;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.04);
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.order-hero-top {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 1rem;
}

.order-code-badge {
  font-size: 0.85rem;
  font-weight: 800;
  background: #00685f;
  color: #ffffff;
  padding: 0.2rem 0.6rem;
  border-radius: 6px;
}

.order-title {
  margin: 0.35rem 0 0 0;
  font-size: 1.35rem;
  font-weight: 800;
  color: #0b1c30;
}

.delivery-badge {
  text-align: right;
  background: #f8f9ff;
  border: 1px solid #e5eeff;
  border-radius: 8px;
  padding: 0.45rem 0.75rem;
}

.delivery-label {
  display: block;
  font-size: 0.7rem;
  color: #565e74;
}

.delivery-date {
  font-size: 0.95rem;
  font-weight: 800;
  color: #00685f;
}

.order-stats-row {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 0.75rem;
  background: #f8fafc;
  padding: 0.75rem;
  border-radius: 8px;
}

.stat-label {
  display: block;
  font-size: 0.75rem;
  color: #64748b;
}

.stat-value {
  font-size: 0.95rem;
  color: #1e293b;
}

.text-teal {
  color: #00685f;
}

/* Progress Bar */
.progress-container {
  display: flex;
  flex-direction: column;
  gap: 0.4rem;
}

.progress-info {
  display: flex;
  justify-content: space-between;
  font-size: 0.82rem;
  color: #334155;
}

.progress-bar-track {
  width: 100%;
  height: 8px;
  background: #e2e8f0;
  border-radius: 9999px;
  overflow: hidden;
}

.progress-bar-fill {
  height: 100%;
  background: #00685f;
  border-radius: 9999px;
  transition: width 0.3s ease;
}

/* Timeline */
.timeline-section {
  background: #ffffff;
  border: 1px solid #cbd9df;
  border-radius: 14px;
  padding: 1.25rem;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.04);
}

.timeline-header h2 {
  font-size: 1.15rem;
  font-weight: 800;
  margin: 0 0 0.15rem 0;
  color: #0b1c30;
}

.timeline-header p {
  font-size: 0.82rem;
  color: #64748b;
  margin: 0 0 1rem 0;
}

.timeline-stepper {
  list-style: none;
  padding: 0;
  margin: 0;
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
  position: relative;
}

.timeline-item {
  display: flex;
  gap: 0.85rem;
  align-items: flex-start;
  padding: 0.75rem;
  border-radius: 10px;
  border: 1px solid #f1f5f9;
  background: #f8fafc;
}

.timeline-item--completed {
  background: #f0fdf4;
  border-color: #bbf7d0;
}

.timeline-item--in_progress {
  background: #f0fdfa;
  border-color: #99f6e4;
  box-shadow: 0 0 0 2px rgba(13, 148, 136, 0.2);
}

.timeline-marker {
  width: 28px;
  height: 28px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-weight: 800;
  font-size: 0.8rem;
  shrink: 0;
  background: #e2e8f0;
  color: #64748b;
}

.timeline-item--completed .timeline-marker {
  background: #22c55e;
  color: #ffffff;
}

.timeline-item--in_progress .timeline-marker {
  background: #00685f;
  color: #ffffff;
}

.marker-pulse {
  width: 8px;
  height: 8px;
  background: #ffffff;
  border-radius: 50%;
  animation: pulse 1.2s infinite;
}

.timeline-body {
  flex: 1;
}

.timeline-step-top {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.timeline-step-top h3 {
  margin: 0;
  font-size: 0.95rem;
  font-weight: 700;
  color: #1e293b;
}

.status-tag {
  font-size: 0.72rem;
  font-weight: 700;
  padding: 0.15rem 0.45rem;
  border-radius: 4px;
}

.tag--completed {
  background: #dcfce7;
  color: #166534;
}

.tag--in_progress {
  background: #ccfbf1;
  color: #0f766e;
}

.tag--pending {
  background: #e2e8f0;
  color: #475569;
}

.stage-active-note {
  margin: 0.35rem 0 0 0;
  font-size: 0.82rem;
  color: #0f766e;
  font-weight: 600;
}

.stage-completed-time {
  margin: 0.25rem 0 0 0;
  font-size: 0.75rem;
  color: #64748b;
}

/* Support Card */
.support-card {
  background: #ffffff;
  border: 1px solid #cbd9df;
  border-radius: 14px;
  padding: 1.25rem;
  text-align: center;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0.5rem;
}

.support-icon {
  font-size: 2rem;
}

.support-card h3 {
  margin: 0;
  font-size: 1rem;
  font-weight: 800;
  color: #0b1c30;
}

.support-card p {
  margin: 0;
  font-size: 0.85rem;
  color: #64748b;
  max-width: 360px;
}

.btn-whatsapp {
  margin-top: 0.5rem;
  width: 100%;
  padding: 0.75rem 1rem;
  background: #25d366;
  color: #ffffff;
  font-weight: 700;
  font-size: 0.95rem;
  text-decoration: none;
  border-radius: 8px;
  box-shadow: 0 2px 4px rgba(37, 211, 102, 0.25);
  display: inline-block;
  transition: transform 0.1s ease;
}

.btn-whatsapp:active {
  transform: scale(0.98);
}

.portal-footer {
  text-align: center;
  padding: 1rem 0;
  font-size: 0.78rem;
  color: #64748b;
}

.portal-footer p {
  margin: 0.15rem 0;
}

.footer-time {
  font-size: 0.7rem;
  color: #94a3b8;
}

/* State Box */
.portal-state-box {
  background: #ffffff;
  border: 1px solid #cbd9df;
  border-radius: 12px;
  padding: 3rem 1.5rem;
  text-align: center;
  color: #64748b;
}

.portal-state-box--error {
  border-color: #fca5a5;
  color: #991b1b;
}

.btn-retry {
  display: inline-block;
  margin-top: 1rem;
  color: #00685f;
  font-weight: 700;
  text-decoration: underline;
}

.spinner {
  width: 32px;
  height: 32px;
  border: 3px solid #e2e8f0;
  border-top-color: #00685f;
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
  margin: 0 auto 0.75rem;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}
</style>
