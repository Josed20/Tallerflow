<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { clientPortalApi } from './api'
import type { ClientPortalOrder } from './types'

const router = useRouter()
const orders = ref<ClientPortalOrder[]>([])
const loading = ref(true)
const error = ref('')

function formatDate(iso: string) {
  if (!iso) return ''
  const d = new Date(iso)
  return d.toLocaleDateString('es-PE', { day: '2-digit', month: 'short', year: 'numeric' })
}

async function loadOrders() {
  loading.value = true
  error.value = ''
  try {
    const res = await clientPortalApi.listOrders()
    orders.value = res.data ?? []
  } catch {
    error.value = 'Tu enlace venció o el portal no está disponible. Solicita uno nuevo para continuar.'
  } finally {
    loading.value = false
  }
}

async function signOut() {
  try {
    await clientPortalApi.logout()
  } finally {
    await router.replace('/portal/access')
  }
}

onMounted(loadOrders)
</script>

<template>
  <main class="portal-orders-page">
    <div class="portal-container">
      <header class="portal-orders__header">
        <div class="header-brand">
          <span class="portal-logo-icon">🧵</span>
          <div>
            <p class="portal-brand-title">TallerFlow</p>
            <h1>Mis pedidos en taller</h1>
          </div>
        </div>
        <button type="button" class="btn-logout" @click="signOut">Cerrar acceso</button>
      </header>

      <section class="portal-welcome-banner">
        <p>👋 Bienvenido al portal privado de seguimiento. Aquí puedes ver el avance en tiempo real de cada una de tus órdenes de confección.</p>
      </section>

      <p v-if="loading" class="portal-orders__state">Cargando tus pedidos desde el taller…</p>

      <section v-else-if="error" class="portal-orders__state portal-orders__state--error" role="alert">
        <p>{{ error }}</p>
        <RouterLink to="/portal/access" class="btn-relink">Solicitar otro enlace</RouterLink>
      </section>

      <section v-else-if="orders.length === 0" class="portal-orders__state portal-orders__state--empty">
        <div class="empty-icon">📦</div>
        <h2>Aún no tienes pedidos para mostrar</h2>
        <p>Cuando el taller registre una orden de confección con tu correo, podrás seguir su avance paso a paso aquí.</p>
      </section>

      <section v-else class="portal-orders__list" aria-label="Mis pedidos">
        <RouterLink
          v-for="order in orders"
          :key="order.id"
          :to="`/portal/orders/${order.id}`"
          class="portal-orders__card"
        >
          <div class="card-top">
            <span class="order-code-tag">{{ order.code }}</span>
            <span class="status-pill text-teal">En confección</span>
          </div>

          <strong class="order-product-title">{{ order.product }}</strong>

          <div class="card-meta">
            <span class="meta-qty">{{ order.quantity }} {{ order.unit }}</span>
            <span class="meta-delivery">Entrega prometida: <strong>{{ formatDate(order.promised_at) }}</strong></span>
          </div>

          <div class="card-action">
            <span>Ver ruta de producción en vivo →</span>
          </div>
        </RouterLink>
      </section>
    </div>
  </main>
</template>

<style scoped>
.portal-orders-page {
  min-height: 100vh;
  background: #f8f9ff;
  color: #0b1c30;
  font-family: Manrope, system-ui, sans-serif;
  padding: 1.5rem 1rem 3rem;
}

.portal-container {
  max-width: 580px;
  margin: 0 auto;
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.portal-orders__header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 1rem;
}

.header-brand {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.portal-logo-icon {
  font-size: 1.75rem;
}

.portal-brand-title {
  margin: 0;
  color: #00685f;
  font-size: 0.75rem;
  font-weight: 800;
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

h1 {
  margin: 0;
  font-size: 1.35rem;
  font-weight: 800;
  color: #0b1c30;
}

.btn-logout {
  background: #ffffff;
  border: 1px solid #cbd5e1;
  border-radius: 8px;
  padding: 0.5rem 0.85rem;
  font-size: 0.82rem;
  font-weight: 700;
  color: #565e74;
  cursor: pointer;
}

.portal-welcome-banner {
  background: #e5eeff;
  border: 1px solid #d3e4fe;
  border-radius: 10px;
  padding: 0.85rem 1rem;
  font-size: 0.85rem;
  color: #334155;
  line-height: 1.4;
}

.portal-welcome-banner p {
  margin: 0;
}

.portal-orders__state {
  background: #ffffff;
  border: 1px dashed #cbd5e1;
  border-radius: 12px;
  padding: 3rem 1.5rem;
  text-align: center;
  color: #64748b;
}

.portal-orders__state--error {
  border-color: #fca5a5;
  color: #991b1b;
}

.empty-icon {
  font-size: 2.5rem;
  margin-bottom: 0.5rem;
}

.portal-orders__state--empty h2 {
  margin: 0 0 0.4rem 0;
  font-size: 1.15rem;
  font-weight: 800;
  color: #0b1c30;
}

.portal-orders__state--empty p {
  margin: 0;
  font-size: 0.85rem;
}

.btn-relink {
  display: inline-block;
  margin-top: 1rem;
  color: #00685f;
  font-weight: 700;
  text-decoration: underline;
}

.portal-orders__list {
  display: flex;
  flex-direction: column;
  gap: 0.85rem;
}

.portal-orders__card {
  background: #ffffff;
  border: 1px solid #cbd9df;
  border-left: 4px solid #00685f;
  border-radius: 12px;
  padding: 1.1rem;
  text-decoration: none;
  color: inherit;
  box-shadow: 0 2px 6px rgba(0, 0, 0, 0.04);
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  transition: transform 0.1s ease, box-shadow 0.1s ease;
}

.portal-orders__card:hover {
  transform: translateY(-2px);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.08);
}

.card-top {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.order-code-tag {
  font-size: 0.82rem;
  font-weight: 800;
  color: #00685f;
  background: #f0fdfa;
  padding: 0.15rem 0.5rem;
  border-radius: 4px;
}

.status-pill {
  font-size: 0.72rem;
  font-weight: 700;
  padding: 0.15rem 0.45rem;
  border-radius: 4px;
}

.text-teal {
  background: #ccfbf1;
  color: #0f766e;
}

.order-product-title {
  font-size: 1.1rem;
  font-weight: 800;
  color: #0b1c30;
}

.card-meta {
  display: flex;
  justify-content: space-between;
  font-size: 0.85rem;
  color: #475569;
}

.meta-qty {
  font-weight: 600;
}

.meta-delivery strong {
  color: #0b1c30;
}

.card-action {
  border-top: 1px dashed #e2e8f0;
  padding-top: 0.4rem;
  font-size: 0.82rem;
  font-weight: 700;
  color: #00685f;
  text-align: right;
}
</style>
