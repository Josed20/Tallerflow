<script setup lang="ts">
import { ref, computed, onMounted, nextTick } from 'vue'
import { Check, Mail, Pencil, Phone, Plus, Search, X } from '@lucide/vue'
import { clientsApi, type Client } from './api'
import { useSessionStore } from '../session/session.store'
import { canManageOperations } from '../session/role-policy'
import UiButton from '../../components/UiButton.vue'
import UiAlert from '../../components/UiAlert.vue'

const session = useSessionStore()

const clients = ref<Client[]>([])
const search = ref('')
const loading = ref(true)
const error = ref('')
const showCreateModal = ref(false)
const editing = ref<Client | null>(null)
const notice = ref('')
const nameInput = ref<HTMLInputElement | null>(null)

// Form fields
const name = ref('')
const phone = ref('')
const email = ref('')
const submitting = ref(false)
const formError = ref('')

const isManager = computed(() => canManageOperations(session.principal?.role))

async function openForm(client: Client | null = null) {
  editing.value = client
  name.value = client?.name ?? ''
  phone.value = client?.phone ?? ''
  email.value = client?.email ?? ''
  formError.value = ''
  showCreateModal.value = true
  await nextTick()
  nameInput.value?.focus()
}

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
    const payload = {
      name: name.value.trim(),
      phone: phone.value.trim() || null,
      email: email.value.trim() || null,
    }
    if (editing.value) {
      const res = await clientsApi.update(editing.value.id, payload, session.csrfToken)
      clients.value = clients.value.map((client) => client.id === res.data.id ? res.data : client)
      notice.value = `${res.data.name} se actualizó.`
    } else {
      const res = await clientsApi.create(payload, session.csrfToken)
      clients.value.unshift(res.data)
      notice.value = `${res.data.name} se registró.`
    }
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
        <p class="clients-kicker">Directorio</p>
        <h1>Clientes</h1>
        <p class="clients-subtitle">Contactos para registrar y consultar órdenes.</p>
      </div>
      <UiButton v-if="isManager" @click="openForm()">
        <Plus :size="17" aria-hidden="true" /> Nuevo cliente
      </UiButton>
    </header>

    <UiAlert v-if="error" :message="error" />
    <div v-if="notice" class="clients-notice" role="status"><Check :size="17" /> {{ notice }} <button type="button" aria-label="Cerrar aviso" @click="notice = ''"><X :size="16" /></button></div>

    <section class="search-bar">
      <Search :size="17" aria-hidden="true" />
      <input
        v-model="search"
        type="search"
        aria-label="Buscar clientes por nombre, teléfono o correo"
        placeholder="Nombre, teléfono o correo"
        class="input-search"
      />
    </section>

    <section v-if="loading" class="loading-state">
      <p>Cargando clientes...</p>
    </section>

    <section v-else-if="filteredClients.length === 0" class="empty-state">
      <div v-if="search"><p>No se encontraron clientes con "{{ search }}".</p><button type="button" class="ops-button" @click="search = ''">Ver todos los clientes</button></div>
      <div v-else>
        <p>Aún no hay clientes registrados en este taller.</p>
        <UiButton v-if="isManager" @click="openForm()">
          Registrar el primer cliente
        </UiButton>
      </div>
    </section>

    <section v-else class="clients-list">
      <div v-for="client in filteredClients" :key="client.id" class="client-card">
        <div class="client-info">
          <h3>{{ client.name }}</h3>
          <a v-if="client.phone" class="client-meta" :href="`tel:${client.phone}`"><Phone :size="15" aria-hidden="true" /> {{ client.phone }}</a>
          <a v-if="client.email" class="client-meta" :href="`mailto:${client.email}`"><Mail :size="15" aria-hidden="true" /> {{ client.email }}</a>
        </div>
        <span class="status-badge" :class="{ active: client.is_active }">
          {{ client.is_active ? 'Activo' : 'Inactivo' }}
        </span>
        <button v-if="isManager" type="button" class="ops-button client-edit" :aria-label="`Editar ${client.name}`" @click="openForm(client)"><Pencil :size="16" /> Editar</button>
      </div>
    </section>

    <!-- Modal Nuevo Cliente -->
    <div v-if="showCreateModal" class="modal-backdrop" @click.self="showCreateModal = false">
      <div class="modal-card">
        <header class="modal-header">
          <h2>{{ editing ? 'Editar cliente' : 'Nuevo cliente' }}</h2>
          <button class="btn-close" type="button" aria-label="Cerrar formulario" @click="showCreateModal = false"><X :size="18" /></button>
        </header>

        <form @submit.prevent="handleCreateClient" class="form-modal">
          <UiAlert v-if="formError" :message="formError" />

          <div class="field-group">
            <label for="client-name">Nombre o Razón Social *</label>
            <input
              id="client-name"
              ref="nameInput"
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
              maxlength="40"
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
              maxlength="254"
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
              {{ submitting ? 'Guardando...' : editing ? 'Guardar cambios' : 'Guardar cliente' }}
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
  color: var(--color-primary, var(--tf-primary));
  font-size: 0.95rem;
  font-weight: 600;
  cursor: pointer;
  padding: 0.25rem 0.5rem;
}

h1 {
  font-size: 1.4rem;
  color: var(--color-text-dark, var(--tf-primary-strong));
  margin: 0;
}

.search-bar {
  margin-bottom: 1rem;
}

.input-search {
  width: 100%;
  padding: 0.65rem 0.85rem;
  border: 1px solid var(--color-border, var(--tf-border-soft));
  border-radius: 8px;
  font-size: 0.95rem;
  background: var(--tf-surface);
}

.clients-list {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.client-card {
  background: var(--tf-surface);
  border: 1px solid var(--color-border, var(--tf-border-soft));
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
  color: var(--color-text-dark, var(--tf-primary-strong));
}

.client-meta {
  margin: 0.15rem 0;
  font-size: 0.85rem;
  color: var(--tf-text-subtle);
}

.status-badge {
  font-size: 0.75rem;
  font-weight: 600;
  padding: 0.2rem 0.5rem;
  border-radius: 4px;
  background: var(--tf-border-soft);
  color: var(--tf-text-subtle);
}

.status-badge.active {
  background: var(--tf-success-soft);
  color: var(--tf-success);
}

.loading-state,
.empty-state {
  text-align: center;
  padding: 2rem 1rem;
  color: var(--tf-text-subtle);
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
  background: var(--tf-surface);
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
  color: var(--color-primary, var(--tf-primary));
}

.btn-close {
  background: none;
  border: none;
  font-size: 1.2rem;
  cursor: pointer;
  color: var(--tf-text-subtle);
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
  color: var(--tf-ink-soft);
}

.input-text {
  width: 100%;
  padding: 0.65rem 0.85rem;
  border: 1px solid var(--color-border, var(--tf-border-soft));
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
  border: 1px solid var(--tf-border-soft);
  padding: 0.6rem 1rem;
  border-radius: 8px;
  font-weight: 600;
  color: var(--tf-text-subtle);
  cursor: pointer;
}

.clients-page { max-width: 72rem; padding: clamp(1rem, 2.5vw, 2rem); }
.clients-page { display: grid; gap: 1rem; }
.page-header { margin-bottom: 1.5rem; }
.page-header h1 { font-size: var(--tf-page-title-size); color: var(--tf-ink); }
.header-nav { display: block; }
.clients-kicker { margin: 0; color: var(--tf-primary); font-size: .75rem; font-weight: 800; text-transform: uppercase; }
.clients-subtitle { margin: .3rem 0 0; color: var(--tf-text-subtle); font-size: .86rem; }
.btn-back { color: var(--tf-primary); min-height: 44px; }
.input-search { max-width: 25rem; min-height: 46px; border-color: var(--tf-border); border-radius: .4rem; }
.search-bar { display: flex; align-items: center; gap: .55rem; width: min(100%, 26rem); min-height: 46px; padding: 0 .75rem; border: 1px solid var(--tf-border); border-radius: .4rem; background: var(--tf-surface); color: var(--tf-text-subtle); }
.search-bar:focus-within { border-color: var(--tf-primary); outline: 3px solid var(--tf-focus); outline-offset: 2px; }
.search-bar .input-search { width: 100%; min-height: 42px; padding: 0; border: 0; outline: 0; }
.input-search:focus-visible, .input-text:focus-visible { outline: 2px solid var(--tf-focus); border-color: var(--tf-primary); }
.clients-list { display: grid; grid-template-columns: 1fr; gap: .5rem; }
.client-card { min-height: 9rem; gap: .75rem; border-color: var(--tf-border-soft); border-radius: .5rem; box-shadow: none; }
.client-card { min-height: 6rem; align-items: center; }
.client-info { min-width: 0; }
.client-info h3 { color: var(--tf-ink); overflow-wrap: anywhere; }
.client-meta { color: var(--tf-text-subtle); overflow-wrap: anywhere; }
.client-meta { display: flex; align-items: center; gap: .4rem; text-decoration: none; }
.client-meta:hover { color: var(--tf-primary); }
.client-edit { min-width: 6rem; }
.status-badge.active { color: var(--tf-primary-strong); background: var(--tf-success-soft); }
.status-badge:not(.active) { color: var(--tf-done); background: var(--tf-done-soft); }
.clients-notice { display: flex; align-items: center; gap: .5rem; padding: .75rem; border: 1px solid var(--tf-success-border); border-radius: .4rem; color: var(--tf-primary-strong); background: var(--tf-success-soft); font-size: .85rem; font-weight: 700; }
.clients-notice button { display: grid; margin-left: auto; place-items: center; border: 0; color: inherit; background: transparent; cursor: pointer; }
.modal-card { border-radius: .5rem; }
.modal-header h2 { color: var(--tf-ink); }
.input-text { min-height: 46px; border-color: var(--tf-border); border-radius: .4rem; }
.btn-secondary { min-height: 44px; color: var(--tf-primary); border-color: var(--tf-primary-border); }
@media (max-width: 560px) { .page-header { align-items: flex-start; flex-direction: column; } .client-card { align-items: flex-start; flex-wrap: wrap; } .client-info { flex: 1 1 100%; } .modal-actions { flex-direction: column-reverse; } .modal-actions > * { width: 100%; } }
</style>
