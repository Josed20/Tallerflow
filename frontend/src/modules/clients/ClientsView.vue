<script setup lang="ts">
import { ref, computed, onMounted, nextTick } from 'vue'
import { Pencil, Plus, Search, X } from '@lucide/vue'
import { clientsApi, type Client } from './api'
import { useSessionStore } from '../session/session.store'
import { canManageOperations } from '../session/role-policy'
import UiButton from '../../components/UiButton.vue'
import UiAlert from '../../components/UiAlert.vue'
import UiNotice from '../../components/UiNotice.vue'

const session = useSessionStore()

const clients = ref<Client[]>([])
const search = ref('')
const loading = ref(true)
const error = ref('')
const showCreateModal = ref(false)
const editing = ref<Client | null>(null)
const notice = ref('')
const archived = ref(false)
const archiveCandidate = ref<Client | null>(null)
async function archiveClient() {
  const client = archiveCandidate.value
  if (!client || submitting.value || !session.csrfToken) return
  submitting.value = true
  try { const response = await clientsApi.update(client.id, { is_active: !client.is_active }, session.csrfToken); clients.value = clients.value.map(item => item.id === client.id ? response.data : item); notice.value = client.is_active ? 'Cliente archivado. Sus órdenes se conservan.' : 'Cliente restaurado.'; archiveCandidate.value = null }
  catch { error.value = 'No se pudo actualizar el cliente. Inténtalo nuevamente.' }
  finally { submitting.value = false }
}
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
  const visible = clients.value.filter(client => archived.value ? !client.is_active : client.is_active)
  if (!q) return visible
  return visible.filter(
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
        <h1>Clientes</h1>
        <p class="clients-subtitle">Contactos para registrar y consultar órdenes.</p>
      </div>
      <UiButton v-if="isManager" @click="openForm()">
        <Plus :size="17" aria-hidden="true" /> Nuevo cliente
      </UiButton>
    </header>

    <UiAlert v-if="error" :message="error" />
    <UiNotice :message="notice" />

    <label class="archive-filter"><input v-model="archived" type="checkbox" /> Ver clientes archivados</label>
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
        <p>{{ archived ? 'No hay clientes archivados.' : 'Aún no hay clientes registrados en este taller.' }}</p>
        <UiButton v-if="isManager && !archived" @click="openForm()">
          Registrar el primer cliente
        </UiButton>
      </div>
    </section>

    <table v-else class="clients-list">
      <caption class="sr-only">Clientes del taller</caption>
      <thead><tr><th scope="col">Cliente</th><th scope="col">Teléfono</th><th scope="col">Correo</th><th v-if="isManager" scope="col"><span class="sr-only">Acciones</span></th></tr></thead>
      <tbody>
        <tr v-for="client in filteredClients" :key="client.id" class="client-row">
          <th scope="row" class="client-name">{{ client.name }}</th>
          <td data-label="Teléfono"><span v-if="client.phone" class="client-contact">{{ client.phone }}</span><span v-else class="missing-contact">Sin teléfono</span></td>
          <td data-label="Correo"><span v-if="client.email" class="client-contact">{{ client.email }}</span><span v-else class="missing-contact">Sin correo</span></td>

          <td v-if="isManager" class="client-actions"><button type="button" class="ops-button client-edit" :aria-label="`Editar ${client.name}`" @click="openForm(client)"><Pencil :size="16" aria-hidden="true" /> Editar</button><button type="button" class="ops-button" @click="archiveCandidate = client">{{ client.is_active ? 'Archivar' : 'Restaurar' }}</button></td>
        </tr>
      </tbody>
    </table>

    <div v-if="archiveCandidate" class="modal-backdrop"><section class="modal-card" role="alertdialog" aria-modal="true" aria-labelledby="archive-title"><h2 id="archive-title">{{ archiveCandidate.is_active ? 'Archivar cliente' : 'Restaurar cliente' }}</h2><p>{{ archiveCandidate.name }}</p><p>Sus órdenes y el historial se conservan. Un cliente archivado no se utiliza para pedidos nuevos.</p><div class="modal-actions"><button type="button" class="ops-button" @click="archiveCandidate = null">Cancelar</button><UiButton :disabled="submitting" @click="archiveClient">{{ submitting ? 'Guardando…' : 'Confirmar' }}</UiButton></div></section></div>
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
.clients-subtitle { margin: .3rem 0 0; color: var(--tf-text-subtle); font-size: .86rem; }
.btn-back { color: var(--tf-primary); min-height: 44px; }
.input-search { max-width: 25rem; min-height: 46px; border-color: var(--tf-border); border-radius: .4rem; }
.search-bar { display: flex; align-items: center; gap: .55rem; width: min(100%, 26rem); min-height: 46px; padding: 0 .75rem; border: 1px solid var(--tf-border); border-radius: .4rem; background: var(--tf-surface); color: var(--tf-text-subtle); }
.search-bar:focus-within { border-color: var(--tf-primary); box-shadow: 0 0 0 1px var(--tf-primary); }
.search-bar .input-search { width: 100%; min-height: 42px; padding: 0; border: 0; outline: 0; }
.input-text:focus-visible { outline: 2px solid var(--tf-focus); border-color: var(--tf-primary); }
.status-badge.active { color: var(--tf-primary-strong); background: var(--tf-success-soft); }
.status-badge:not(.active) { color: var(--tf-done); background: var(--tf-done-soft); }
.clients-notice { display: flex; align-items: center; gap: .5rem; padding: .75rem; border: 1px solid var(--tf-success-border); border-radius: .4rem; color: var(--tf-primary-strong); background: var(--tf-success-soft); font-size: .85rem; font-weight: 700; }
.clients-notice button { display: grid; margin-left: auto; place-items: center; border: 0; color: inherit; background: transparent; cursor: pointer; }
.modal-card { border-radius: .5rem; }
.modal-header h2 { color: var(--tf-ink); }
.input-text { min-height: 46px; border-color: var(--tf-border); border-radius: .4rem; }
.btn-secondary { min-height: 44px; color: var(--tf-primary); border-color: var(--tf-primary-border); }
@media (max-width: 560px) { .page-header { align-items: flex-start; flex-direction: column; } .modal-actions { flex-direction: column-reverse; } .modal-actions > * { width: 100%; } }

.clients-list { width: 100%; border-collapse: collapse; table-layout: fixed; background: var(--tf-surface); }
.clients-list th, .clients-list td { padding: .75rem; border-bottom: 1px solid var(--tf-border); text-align: left; vertical-align: middle; overflow-wrap: anywhere; }
.clients-list thead th { color: var(--tf-text-subtle); font-size: .85rem; }
.clients-list thead th:first-child { width: 25%; }
.clients-list thead th:nth-child(2) { width: 18%; }
.clients-list thead th:nth-child(3) { width: 27%; }
.clients-list thead th:nth-child(4) { width: 14%; }
.client-name { color: var(--tf-ink); font-weight: 700; }
.client-meta { display: flex; align-items: center; gap: .4rem; min-height: 48px; color: var(--tf-primary); }
.client-meta svg { flex: 0 0 auto; }
.client-meta:focus-visible { outline: 3px solid var(--tf-focus); outline-offset: 2px; }
.missing-contact { color: var(--tf-text-subtle); font-size: .85rem; }
.client-edit { min-height: 48px; }
.search-bar { min-width: 0; box-sizing: border-box; min-height: 48px; border-radius: var(--tf-radius); }
.search-bar .input-search { min-width: 0; max-width: none; outline: none; box-shadow: none; }
@media (max-width: 760px) {
  .clients-list, .clients-list tbody { display: block; }
  .clients-list thead { position: absolute; width: 1px; height: 1px; overflow: hidden; clip-path: inset(50%); }
  .clients-list .client-row { display: grid; grid-template-columns: minmax(0, 1fr) auto; gap: .5rem; padding: 1rem; border-bottom: 1px solid var(--tf-border); }
  .clients-list th, .clients-list td { width: auto; padding: 0; border: 0; }
  .client-name, .clients-list td[data-label="Teléfono"], .clients-list td[data-label="Correo"] { grid-column: 1 / -1; }
  .clients-list td[data-label="Estado"] { align-self: center; }
  .client-actions { justify-self: end; }
}

.search-bar .input-search:focus, .search-bar .input-search:focus-visible { border: 0; outline: none; box-shadow: none; appearance: none; background: transparent; }
.clients-list thead th:first-child { width: 29%; }
.clients-list thead th:nth-child(2) { width: 17%; }
.clients-list thead th:nth-child(3) { width: 29%; }
.clients-list thead th:nth-child(4) { width: 25%; }
.client-actions { white-space: nowrap; }
.client-actions button + button { margin-left: .5rem; }
.archive-filter { display: flex; align-items: center; gap: .5rem; min-height: 44px; color: var(--tf-text-subtle); font-size: .875rem; }
.archive-filter input { width: 18px; height: 18px; accent-color: var(--tf-primary); }
@media(max-width:760px) { .client-actions { grid-column: 1 / -1; padding-top: .5rem !important; } }
</style>
