<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { api } from '../session/api'

type Workshop = { id: string; name: string; owner_email: string; created_at: string; last_activity: string | null; members: number; active_orders: number; photo_bytes: number; status: 'ACTIVE' | 'SUSPENDED' | 'PENDING_DELETION'; deletion_scheduled_for: string | null; status_reason?: string; status_changed_at?: string | null }
type AuditEvent = { id: string; workshop_id: string | null; event_type: string; actor_email: string; workshop_name: string; reason: string; source_ip: string; created_at: string }
type Summary = { active_workshops: number; suspended_workshops: number; pending_deletion_workshops: number; created_last_7_days: number; created_last_30_days: number; active_last_7_days: number; unverified_owners_over_48h: number; orders_last_30_days: number }

const authenticated = ref(false)
const loading = ref(true)
const busy = ref(false)
const error = ref('')
const email = ref('')
const password = ref('')
const code = ref('')
const search = ref('')
const workshops = ref<Workshop[]>([])
const audit = ref<AuditEvent[]>([])
const summary = ref<Summary | null>(null)
const selected = ref<Workshop | null>(null)
const csrf = ref(sessionStorage.getItem('tallerflow_console_csrf') || '')

async function restore() {
  try { await api.get('/api/v1/console/session'); authenticated.value = true; await refresh() }
  catch { authenticated.value = false }
  finally { loading.value = false }
}

async function login() {
  busy.value = true; error.value = ''
  try {
    const response = await api.post<{ data: { csrf_token: string } }>('/api/v1/console/login', { email: email.value, password: password.value, code: code.value })
    csrf.value = response.data.csrf_token; sessionStorage.setItem('tallerflow_console_csrf', csrf.value)
    password.value = ''; code.value = ''; authenticated.value = true; await refresh()
  } catch (err: any) { error.value = err?.message || 'No se pudo ingresar a la consola.' }
  finally { busy.value = false }
}

async function refresh() {
  const [workshopResponse, auditResponse, summaryResponse] = await Promise.all([
    api.get<{ data: Workshop[] }>(`/api/v1/console/workshops?q=${encodeURIComponent(search.value)}`),
    api.get<{ data: AuditEvent[] }>('/api/v1/console/audit'),
    api.get<{ data: Summary }>('/api/v1/console/summary'),
  ])
  workshops.value = workshopResponse.data; audit.value = auditResponse.data; summary.value = summaryResponse.data
}

async function showWorkshop(workshop: Workshop) {
  try { const response = await api.get<{ data: Workshop }>(`/api/v1/console/workshops/${workshop.id}`); selected.value = response.data }
  catch (err: any) { error.value = err?.message || 'No se pudo cargar el taller.' }
}

async function changeStatus(workshop: Workshop, status: Workshop['status']) {
  const action = status === 'ACTIVE' ? 'reactivar' : status === 'SUSPENDED' ? 'suspender' : 'programar la eliminación de'
  const reason = window.prompt(`Motivo para ${action} ${workshop.name}:`)
  if (!reason || reason.trim().length < 5) return
  busy.value = true; error.value = ''
  try { await api.post(`/api/v1/console/workshops/${workshop.id}/status`, { status, reason: reason.trim() }, csrf.value); await refresh() }
  catch (err: any) { error.value = err?.message || 'No se pudo cambiar el estado.' }
  finally { busy.value = false }
}

async function purge(workshop: Workshop) {
  const confirmation = window.prompt(`Escribe exactamente "${workshop.name}" para eliminar el taller definitivamente:`)
  if (confirmation !== workshop.name) return
  const verificationCode = window.prompt('Ingresa un código nuevo de 6 dígitos de tu aplicación autenticadora:')
  if (!verificationCode || !/^[0-9]{6}$/.test(verificationCode)) return
  busy.value = true; error.value = ''
  try { await api.post(`/api/v1/console/workshops/${workshop.id}/purge`, { name: confirmation, code: verificationCode }, csrf.value); await refresh() }
  catch (err: any) { error.value = err?.message || 'La eliminación todavía no está permitida.' }
  finally { busy.value = false }
}

async function logout() {
  try { await api.post('/api/v1/console/logout', undefined, csrf.value) } finally { sessionStorage.removeItem('tallerflow_console_csrf'); csrf.value = ''; authenticated.value = false }
}

function date(value: string | null) { return value ? new Intl.DateTimeFormat('es-PE', { dateStyle: 'medium', timeStyle: 'short' }).format(new Date(value)) : 'Sin actividad' }
function bytes(value: number) { return value < 1024 * 1024 ? `${Math.round(value / 1024)} KB` : `${(value / 1024 / 1024).toFixed(1)} MB` }
onMounted(restore)
</script>

<template>
  <main class="console-page">
    <header class="console-bar"><strong>Consola de plataforma</strong><button v-if="authenticated" type="button" @click="logout">Cerrar sesión</button></header>
    <section v-if="loading" class="card">Verificando sesión…</section>
    <section v-else-if="!authenticated" class="login card">
      <h1>Administración de TallerFlow</h1><p>Acceso exclusivo para administradores de plataforma.</p>
      <form @submit.prevent="login"><label>Correo<input v-model="email" type="email" autocomplete="username" required /></label><label>Contraseña<input v-model="password" type="password" autocomplete="current-password" required /></label><label>Código de 6 dígitos<input v-model="code" inputmode="numeric" pattern="[0-9]{6}" maxlength="6" autocomplete="one-time-code" required /></label><p v-if="error" class="error">{{ error }}</p><button :disabled="busy" type="submit">{{ busy ? 'Ingresando…' : 'Ingresar' }}</button></form>
    </section>
    <template v-else>
      <section class="content"><h2>Resumen de plataforma</h2><div v-if="summary" class="summary-grid"><article><strong>{{ summary.active_workshops }}</strong><span>Talleres activos</span></article><article><strong>{{ summary.suspended_workshops }}</strong><span>Suspendidos</span></article><article><strong>{{ summary.pending_deletion_workshops }}</strong><span>Eliminación programada</span></article><article><strong>{{ summary.created_last_7_days }} / {{ summary.created_last_30_days }}</strong><span>Talleres creados (7 / 30 días)</span></article><article><strong>{{ summary.active_last_7_days }}</strong><span>Activos en los últimos 7 días</span></article><article><strong>{{ summary.unverified_owners_over_48h }}</strong><span>Dueños sin verificar &gt; 48 h</span></article><article><strong>{{ summary.orders_last_30_days }}</strong><span>Órdenes en 30 días</span></article></div></section>
      <section v-if="selected" class="content"><div class="heading"><h2>{{ selected.name }}</h2><button @click="selected = null">Cerrar</button></div><p>Dueño: {{ selected.owner_email }} · Alta: {{ date(selected.created_at) }} · Estado: {{ selected.status }}</p><p>Motivo: {{ selected.status_reason || 'Sin cambios registrados' }}</p><p v-if="selected.deletion_scheduled_for">Eliminación programada: {{ date(selected.deletion_scheduled_for) }}</p></section>
      <section class="content"><div class="heading"><div><h1>Talleres</h1><p>Metadatos operativos; la consola no puede leer órdenes, clientes ni fotografías.</p></div><form @submit.prevent="refresh"><input v-model="search" placeholder="Nombre o correo del dueño" /><button>Buscar</button></form></div><p v-if="error" class="error">{{ error }}</p>
        <div class="table-wrap"><table><thead><tr><th>Taller</th><th>Estado</th><th>Actividad</th><th>Miembros</th><th>Órdenes activas</th><th>Fotos</th><th>Acciones</th></tr></thead><tbody><tr v-for="workshop in workshops" :key="workshop.id"><td><button class="link" @click="showWorkshop(workshop)">{{ workshop.name }}</button><small>{{ workshop.owner_email }}</small></td><td><span :class="`status status--${workshop.status.toLowerCase()}`">{{ workshop.status }}</span></td><td>{{ date(workshop.last_activity) }}</td><td>{{ workshop.members }}</td><td>{{ workshop.active_orders }}</td><td>{{ bytes(workshop.photo_bytes) }}</td><td class="actions"><button v-if="workshop.status !== 'SUSPENDED'" :disabled="busy" @click="changeStatus(workshop, 'SUSPENDED')">Suspender</button><button v-if="workshop.status !== 'ACTIVE'" :disabled="busy" @click="changeStatus(workshop, 'ACTIVE')">{{ workshop.status === 'PENDING_DELETION' ? 'Cancelar eliminación' : 'Reactivar' }}</button><button v-if="workshop.status !== 'PENDING_DELETION'" :disabled="busy" @click="changeStatus(workshop, 'PENDING_DELETION')">Programar eliminación</button><button v-else class="danger" :disabled="busy" @click="purge(workshop)">Eliminar definitivamente</button></td></tr></tbody></table></div>
      </section>
      <section class="content"><h2>Auditoría reciente</h2><ul class="audit"><li v-for="event in audit" :key="event.id"><strong>{{ event.actor_email || 'Administrador' }} · {{ event.event_type }}<small>{{ event.workshop_name }} · {{ event.source_ip }}</small></strong><span>{{ event.reason }}</span><time>{{ date(event.created_at) }}</time></li></ul></section>
    </template>
  </main>
</template>

<style scoped>
.console-page{min-height:100vh;background:#f4f5f7;color:#172033;font-family:Inter,system-ui,sans-serif}.console-bar{position:sticky;top:0;z-index:2;display:flex;justify-content:space-between;align-items:center;padding:.85rem clamp(1rem,4vw,3rem);background:#252a31;color:white}.console-bar button,.heading button,.login button{min-height:44px;padding:.55rem 1rem;border:0;border-radius:6px;background:#087f6d;color:white;font-weight:700;cursor:pointer}.card,.content{width:min(1180px,calc(100% - 2rem));margin:2rem auto;padding:1.25rem;background:white;border:1px solid #dde1e7;border-radius:10px}.login{max-width:430px}.login form,.login label{display:grid;gap:.45rem}.login form{gap:1rem}.login input,.heading input{min-height:44px;padding:.55rem .7rem;border:1px solid #b9c0ca;border-radius:6px;font:inherit}.heading{display:flex;justify-content:space-between;align-items:end;gap:1rem}.heading h1,.content h2{margin:0}.heading p{margin:.35rem 0;color:#5d6573}.heading form{display:flex;gap:.5rem}.table-wrap{overflow:auto;margin-top:1rem}table{width:100%;border-collapse:collapse;font-size:.88rem}th,td{padding:.75rem;text-align:left;border-bottom:1px solid #e5e7eb;vertical-align:top}td small{display:block;color:#687181;margin-top:.2rem}.status{display:inline-block;padding:.25rem .5rem;border-radius:999px;font-size:.72rem;font-weight:800}.status--active{background:#d9f5ea;color:#096b52}.status--suspended{background:#fee2e2;color:#a11919}.status--pending_deletion{background:#fff0c7;color:#855b00}.actions{display:flex;flex-wrap:wrap;gap:.35rem}.actions button{min-height:36px;border:1px solid #aeb6c2;border-radius:5px;background:white;cursor:pointer}.actions .danger{border-color:#b42318;color:#b42318}.error{padding:.7rem;border-radius:6px;background:#fee2e2;color:#991b1b}.audit{list-style:none;padding:0;margin:1rem 0 0;display:grid;gap:.6rem}.audit li{display:grid;grid-template-columns:1fr 2fr auto;gap:1rem;padding:.65rem;border-bottom:1px solid #e5e7eb}.audit time,.audit span{color:#667085}@media(max-width:700px){.heading{align-items:stretch;flex-direction:column}.heading form{display:grid;width:100%}.audit li{grid-template-columns:1fr}.content{width:calc(100% - 1rem);margin:.5rem}.console-bar{position:static}}
</style>
