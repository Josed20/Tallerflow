<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { clientsApi, type Client } from './api'
import { useSessionStore } from '../session/session.store'
import { canManageOperations } from '../session/role-policy'
import UiButton from '../../components/UiButton.vue'
import UiAlert from '../../components/UiAlert.vue'

const router = useRouter()
const session = useSessionStore()

const clients = ref<Client[]>([])
const search = ref('')
const loading = ref(true)
const error = ref('')
const showCreateModal = ref(false)

// Form fields
const name = ref('')
const phone = ref('')
const email = ref('')
const submitting = ref(false)
const formError = ref('')

const isManager = computed(() => canManageOperations(session.principal?.role))

const filteredClients = computed(() => {
  const q = search.value.trim().toLowerCase()
  if (!q) return clients.value
  return clients.value.filter(
    (c) =>
      c.name.toLowerCase().includes(q) ||
      (c.phone && c.phone.includes(q)) ||
      (c.email && c.email.toLowerCase().includes(q))
  )
})

async function fetchClients() {
  loading.value = true
  error.value = ''
  try {
    const res = await clientsApi.list()
    clients.value = res.data || []
  } catch (err: any) {
    error.value = err?.message || 'No se pudieron cargar los clientes.'
  } finally {
    loading.value = false
  }
}

async function handleCreateClient() {
  if (!name.value.trim()) {
    formError.value = 'El nombre del cliente es obligatorio.'
    return
  }
  if (!session.csrfToken) {
    formError.value = 'Tu sesión ya no permite guardar cambios. Actualiza la página e inicia sesión nuevamente.'
    return
  }

  submitting.value = true
  formError.value = ''
  try {
    const res = await clientsApi.create({
      name: name.value.trim(),
      phone: phone.value.trim() || null,
      email: email.value.trim() || null,
    }, session.csrfToken)
    clients.value.unshift(res.data)
    name.value = ''
    phone.value = ''
    email.value = ''
    showCreateModal.value = false
  } catch (err: any) {
    formError.value = err?.message || 'Error al registrar el cliente.'
  } finally {
    submitting.value = false
  }
}

onMounted(() => {
  fetchClients()
})
</script>

<template>
  <main class="clients-page">
    <header class="page-header">
      <div class="header-nav">
        <button class="btn-back" @click="router.push('/app')">← Inicio</button>
        <h1>Clientes</h1>
      </div>
      <UiButton v-if="isManager" @click="showCreateModal = true">
        + Nuevo Cliente
      </UiButton>
    </header>

    <UiAlert v-if="error" :message="error" />

    <section class="search-bar">
      <input
        v-model="search"
        type="search"
        placeholder="Buscar por nombre, teléfono..."
        class="input-search"
      />
    </section>

    <section v-if="loading" class="loading-state">
      <p>Cargando clientes...</p>
    </section>

    <section v-else-if="filteredClients.length === 0" class="empty-state">
      <p v-if="search">No se encontraron clientes con "{{ search }}".</p>
      <div v-else>
        <p>Aún no hay clientes registrados en este taller.</p>
        <UiButton v-if="isManager" @click="showCreateModal = true">
          Registrar el primer cliente
        </UiButton>
      </div>
    </section>

    <section v-else class="clients-list">
      <div v-for="client in filteredClients" :key="client.id" class="client-card">
        <div class="client-info">
          <h3>{{ client.name }}</h3>
          <p v-if="client.phone" class="client-meta">📱 {{ client.phone }}</p>
          <p v-if="client.email" class="client-meta">✉️ {{ client.email }}</p>
        </div>
        <span class="status-badge" :class="{ active: client.is_active }">
          {{ client.is_active ? 'Activo' : 'Inactivo' }}
        </span>
      </div>
    </section>

    <!-- Modal Nuevo Cliente -->
    <div v-if="showCreateModal" class="modal-backdrop" @click.self="showCreateModal = false">
      <div class="modal-card">
        <header class="modal-header">
          <h2>Nuevo Cliente</h2>
          <button class="btn-close" @click="showCreateModal = false">✕</button>
        </header>

        <form @submit.prevent="handleCreateClient" class="form-modal">
          <UiAlert v-if="formError" :message="formError" />

          <div class="field-group">
            <label for="client-name">Nombre o Razón Social *</label>
            <input
              id="client-name"
              v-model="name"
              type="text"
              required
              maxlength="160"
              placeholder="Ej. Confecciones Gamarra SAC"
              class="input-text"
            />
          </div>

          <div class="field-group">
            <label for="client-phone">Teléfono / WhatsApp</label>
            <input
              id="client-phone"
              v-model="phone"
              type="tel"
              placeholder="Ej. 987654321"
              class="input-text"
            />
          </div>

          <div class="field-group">
            <label for="client-email">Correo electrónico (opcional)</label>
            <input
              id="client-email"
              v-model="email"
              type="email"
              placeholder="Ej. contacto@cliente.com"
              class="input-text"
            />
          </div>

          <div class="modal-actions">
            <button
              type="button"
              class="btn-secondary"
              @click="showCreateModal = false"
            >
              Cancelar
            </button>
            <UiButton type="submit" :disabled="submitting">
              {{ submitting ? 'Guardando...' : 'Guardar Cliente' }}
            </UiButton>
          </div>
        </form>
      </div>
    </div>
  </main>
</template>

<style scoped>
.clients-page {
  max-width: 600px;
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

.clients-list {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.client-card {
  background: #fff;
  border: 1px solid var(--color-border, #cbd9df);
  border-radius: 8px;
  padding: 1rem;
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05);
}

.client-info h3 {
  margin: 0 0 0.35rem 0;
  font-size: 1.05rem;
  color: var(--color-text-dark, #10242d);
}

.client-meta {
  margin: 0.15rem 0;
  font-size: 0.85rem;
  color: #556973;
}

.status-badge {
  font-size: 0.75rem;
  font-weight: 600;
  padding: 0.2rem 0.5rem;
  border-radius: 4px;
  background: #e2e8f0;
  color: #475569;
}

.status-badge.active {
  background: #dcfce7;
  color: #166534;
}

.loading-state,
.empty-state {
  text-align: center;
  padding: 2rem 1rem;
  color: #64748b;
}

/* Modal */
.modal-backdrop {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: flex-end;
  justify-content: center;
  z-index: 100;
}

@media (min-width: 600px) {
  .modal-backdrop {
    align-items: center;
  }
}

.modal-card {
  background: #fff;
  width: 100%;
  max-width: 480px;
  border-top-left-radius: 16px;
  border-top-right-radius: 16px;
  padding: 1.25rem;
  box-shadow: 0 -4px 16px rgba(0, 0, 0, 0.15);
}

@media (min-width: 600px) {
  .modal-card {
    border-radius: 12px;
  }
}

.modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 1rem;
}

.modal-header h2 {
  font-size: 1.2rem;
  margin: 0;
  color: var(--color-primary, #0a5a70);
}

.btn-close {
  background: none;
  border: none;
  font-size: 1.2rem;
  cursor: pointer;
  color: #64748b;
}

.form-modal {
  display: flex;
  flex-direction: column;
  gap: 0.85rem;
}

.field-group {
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
}

.field-group label {
  font-size: 0.85rem;
  font-weight: 600;
  color: #334155;
}

.input-text {
  width: 100%;
  padding: 0.65rem 0.85rem;
  border: 1px solid var(--color-border, #cbd9df);
  border-radius: 8px;
  font-size: 0.95rem;
}

.modal-actions {
  display: flex;
  justify-content: flex-end;
  gap: 0.75rem;
  margin-top: 0.5rem;
}

.btn-secondary {
  background: none;
  border: 1px solid #cbd9df;
  padding: 0.6rem 1rem;
  border-radius: 8px;
  font-weight: 600;
  color: #475569;
  cursor: pointer;
}
</style>
