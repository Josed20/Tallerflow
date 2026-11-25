<script setup lang="ts">
import UiAlert from '../../components/UiAlert.vue'
import { computed, onMounted, ref } from 'vue'
import { ArrowUpRight, Building2, Activity, PackageCheck, Search, ChevronRight } from '@lucide/vue'
import UiDialog from '../../components/UiDialog.vue'
import UiNotice from '../../components/UiNotice.vue'
import BrandLogo from '../../components/BrandLogo.vue'
import { api } from '../session/api'
import AnalyticsPanel from './AnalyticsPanel.vue'

type Workshop = { id: string; name: string; owner_email: string; created_at: string; last_activity: string | null; members: number; active_orders: number; photo_bytes: number | null; photo_count: number; r2_photo_count: number; is_demo: boolean; status: 'ACTIVE' | 'SUSPENDED' | 'PENDING_DELETION'; deletion_scheduled_for: string | null; status_reason?: string; status_changed_at?: string | null }
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
const includeDemo = ref(false)
const metricsRevision = ref(0)
const workshops = ref<Workshop[]>([])
const audit = ref<AuditEvent[]>([])
const expandedAudit = ref(false)
const visibleAudit = computed(() => expandedAudit.value ? audit.value : audit.value.slice(0,5))
const summary = ref<Summary | null>(null)
const selected = ref<Workshop | null>(null)
const csrf = ref(sessionStorage.getItem('tallerflow_console_csrf') || '')

async function restore() {
  // A new tab has no mutation token; confirm identity instead of showing unusable actions.
  if (!csrf.value) { loading.value = false; return }
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
  error.value = ''
  try {
  const [workshopResponse, auditResponse, summaryResponse] = await Promise.all([
    api.get<{ data: Workshop[] }>(`/api/v1/console/workshops?q=${encodeURIComponent(search.value)}`),
    api.get<{ data: AuditEvent[] }>('/api/v1/console/audit'),
    api.get<{ data: Summary }>(`/api/v1/console/summary?include_demo=${includeDemo.value}`),
  ])
  workshops.value = workshopResponse.data; audit.value = auditResponse.data; summary.value = summaryResponse.data
  } catch { error.value = 'No se pudieron actualizar los datos. Revisa la conexión y vuelve a intentarlo.' }
}

async function showWorkshop(workshop: Workshop) {
  try { const response = await api.get<{ data: Workshop }>(`/api/v1/console/workshops/${workshop.id}`); selected.value = response.data }
  catch (err: any) { error.value = err?.message || 'No se pudo cargar el taller.' }
}

const action = ref<{workshop:Workshop; status?:Workshop['status']; purge?:boolean}|null>(null)
const reason = ref('')
const confirmation = ref('')
const verificationCode = ref('')
const notice = ref('')
const statusFilter = ref('ALL')
const visibleWorkshops = computed(() => workshops.value.filter(workshop => statusFilter.value === 'ALL' || workshop.status === statusFilter.value))
const statusLabels:Record<string,string> = {ACTIVE:'Activo',SUSPENDED:'Suspendido',PENDING_DELETION:'Eliminación programada'}
const actionTitle = computed(() => action.value?.purge ? 'Eliminar taller definitivamente' : action.value?.status === 'ACTIVE' ? 'Reactivar taller' : action.value?.status === 'SUSPENDED' ? 'Suspender taller' : 'Programar eliminación')
function changeStatus(workshop:Workshop,status:Workshop['status']) {error.value='';reason.value='';action.value={workshop,status}}
function purge(workshop:Workshop) {error.value='';confirmation.value='';verificationCode.value='';action.value={workshop,purge:true}}
async function confirmAction() {
  const current=action.value
  if(!current || busy.value)return
  error.value=''; notice.value=''
  if(current.purge && (confirmation.value !== current.workshop.name || !/^[0-9]{6}$/.test(verificationCode.value))) {error.value='Escribe el nombre exacto del taller y un código de 6 dígitos válido.';return}
  if(!current.purge && reason.value.trim().length<5) {error.value='Indica un motivo de al menos 5 caracteres.';return}
  busy.value=true
  try {
    if(current.purge) await api.post(`/api/v1/console/workshops/${current.workshop.id}/purge`,{name:confirmation.value,code:verificationCode.value},csrf.value)
    else await api.post(`/api/v1/console/workshops/${current.workshop.id}/status`,{status:current.status,reason:reason.value.trim()},csrf.value)
    await refresh(); selected.value=null; action.value=null; notice.value=current.purge ? 'Taller eliminado.' : 'Estado del taller actualizado.'
  } catch(err:any) {error.value=err?.message || 'No se pudo completar la acción.'}
  finally {busy.value=false}
}
function eventLabel(value:string) {return ({WORKSHOP_DEMO_CHANGED:'Clasificación demo actualizada',WORKSHOP_STATUS_CHANGED:'Estado del taller actualizado',WORKSHOP_PURGED:'Taller eliminado',CONSOLE_LOGIN:'Acceso a la consola'} as Record<string,string>)[value] || 'Cambio administrativo'}

async function logout() {
  try { await api.post('/api/v1/console/logout', undefined, csrf.value) } finally { sessionStorage.removeItem('tallerflow_console_csrf'); csrf.value = ''; authenticated.value = false }
}

function date(value: string | null) { return value ? new Intl.DateTimeFormat('es-PE', { dateStyle: 'medium', timeStyle: 'short' }).format(new Date(value)) : 'Sin actividad' }
async function toggleDemo(workshop: Workshop) {
  busy.value = true; error.value = ''
  const isDemo = !workshop.is_demo
  try { await api.post(`/api/v1/console/workshops/${workshop.id}/demo`, { is_demo: isDemo }, csrf.value); workshops.value=workshops.value.map(item=>item.id===workshop.id?{...item,is_demo:isDemo}:item); if(selected.value?.id===workshop.id)selected.value={...selected.value,is_demo:isDemo}; await refresh(); if(selected.value?.id === workshop.id) selected.value=workshops.value.find(item=>item.id===workshop.id) || null; metricsRevision.value++; notice.value=isDemo ? 'Taller marcado como demostración.' : 'Taller excluido de demostración.' }
  catch { error.value = 'No se pudo cambiar la clasificación demo.' }
  finally { busy.value = false }
}

function bytes(value: number | null) { if (value === null) return 'Tamaño no disponible'; return value < 1024 * 1024 ? `${Math.round(value / 1024)} KB` : `${(value / 1024 / 1024).toFixed(1)} MB` }
onMounted(restore)
</script>

<template>
  <main class="console-page">
    <header class="console-bar"><div class="console-brand"><BrandLogo :size="25" tone="dark"/><span>Administración</span></div><button v-if="authenticated" type="button" @click="logout">Cerrar sesión</button></header>
    <UiAlert :message="error"/><UiNotice :message="notice"/>
    <section v-if="loading" class="card">Verificando sesión…</section>
    <section v-else-if="!authenticated" class="login card"><h1>Administración de TallerFlow</h1><p>Acceso exclusivo para el equipo de plataforma.</p><form @submit.prevent="login"><label>Correo<input v-model="email" type="email" autocomplete="username" required/></label><label>Contraseña<input v-model="password" type="password" autocomplete="current-password" required/></label><label>Código de 6 dígitos<input v-model="code" inputmode="numeric" pattern="[0-9]{6}" maxlength="6" autocomplete="one-time-code" required/></label><button :disabled="busy">{{busy?'Ingresando…':'Ingresar'}}</button></form></section>
    <div v-else class="console-layout">
      <div class="console-heading"><div><p>Consola de plataforma</p><h1>Vista general</h1><span>Adopción, actividad y estado de los talleres.</span></div><label class="demo-filter"><input v-model="includeDemo" type="checkbox" @change="refresh"/>Incluir talleres de demostración</label></div>
      <section v-if="summary" class="summary-grid" aria-label="Resumen de plataforma">
        <article><span class="stat-icon"><Building2 :size="23"/></span><span>Talleres activos</span><strong>{{summary.active_workshops}}</strong><small>{{summary.created_last_30_days}} creados en 30 días</small></article>
        <article><span class="stat-icon stat-icon--green"><Activity :size="23"/></span><span>Actividad reciente</span><strong>{{summary.active_last_7_days}}</strong><small>Talleres activos en los últimos 7 días</small></article>
        <article><span class="stat-icon stat-icon--violet"><PackageCheck :size="23"/></span><span>Órdenes creadas</span><strong>{{summary.orders_last_30_days}}</strong><small>En los últimos 30 días</small></article>
      </section>
      <section v-if="summary" class="platform-health"><span>Estado de los talleres</span><div class="health-track" role="img" :aria-label="`${summary.active_workshops} activos, ${summary.suspended_workshops} suspendidos, ${summary.pending_deletion_workshops} con eliminación programada`"><i :style="{flex:summary.active_workshops || 0}"/><i class="health-suspended" :style="{flex:summary.suspended_workshops || 0}"/><i class="health-deletion" :style="{flex:summary.pending_deletion_workshops || 0}"/></div><div class="health-legend"><span>{{summary.active_workshops}} activos</span><span>{{summary.suspended_workshops}} suspendidos</span><span>{{summary.pending_deletion_workshops}} con eliminación programada</span><span>{{summary.unverified_owners_over_48h}} propietarios sin verificar &gt; 48 h</span></div></section>
      <section class="content"><AnalyticsPanel :revision="metricsRevision" :csrf="csrf" :include-demo="includeDemo"/></section>
      <section class="content" id="workshops"><div class="heading"><div><h2>Talleres</h2><p>Administra el acceso y consulta su actividad.</p></div><form @submit.prevent="refresh"><label class="console-search"><Search :size="17"/><span class="sr-only">Buscar taller</span><input v-model="search" placeholder="Nombre o correo del propietario"/></label><button>Buscar</button></form></div>
        <div class="workshop-filters" role="group" aria-label="Estado del taller"><button v-for="filter in [{key:'ALL',label:'Todos'},{key:'ACTIVE',label:'Activos'},{key:'SUSPENDED',label:'Suspendidos'},{key:'PENDING_DELETION',label:'Eliminación programada'}]" :key="filter.key" :aria-pressed="statusFilter===filter.key" @click="statusFilter=filter.key">{{filter.label}}</button></div>
        <div class="table-wrap"><table><thead><tr><th>Taller</th><th>Estado</th><th>Última actividad</th><th>Equipo</th><th>Pedidos activos</th><th>Fotos</th><th><span class="sr-only">Gestión</span></th></tr></thead><tbody><tr v-for="workshop in visibleWorkshops" :key="workshop.id"><td><button class="workshop-name" @click="showWorkshop(workshop)">{{workshop.name}}</button><small>{{workshop.owner_email}}</small><span v-if="workshop.is_demo" class="demo-badge">Demostración</span></td><td data-label="Estado"><span :class="`status status--${workshop.status.toLowerCase()}`">{{statusLabels[workshop.status]}}</span></td><td data-label="Última actividad">{{date(workshop.last_activity)}}</td><td data-label="Equipo">{{workshop.members}} {{workshop.members===1?'miembro':'miembros'}}</td><td data-label="Pedidos activos">{{workshop.active_orders}}</td><td data-label="Fotos">{{workshop.photo_count}} {{workshop.photo_count===1?'foto':'fotos'}}<small>{{bytes(workshop.photo_bytes)}}</small></td><td><button class="manage-button" :aria-label="`Administrar ${workshop.name}`" @click="showWorkshop(workshop)">Administrar <ChevronRight :size="16"/></button></td></tr><tr v-if="!visibleWorkshops.length"><td colspan="7" class="empty-workshops">No hay talleres para esta selección.</td></tr></tbody></table></div>
      </section>
      <section class="content"><h2>Actividad administrativa</h2><p class="muted">Cambios recientes realizados por el equipo de TallerFlow.</p><ul class="audit"><li v-for="event in visibleAudit" :key="event.id"><span class="audit-dot"/><div><strong>{{eventLabel(event.event_type)}}</strong><p>{{event.workshop_name || 'Plataforma'}}<span v-if="event.reason"> · {{event.reason}}</span></p><small>{{event.actor_email || 'Administrador'}}</small></div><time>{{date(event.created_at)}}</time></li><li v-if="!audit.length" class="muted">No hay cambios administrativos registrados.</li></ul><button v-if="audit.length>5" class="ops-button audit-more" @click="expandedAudit=!expandedAudit">{{expandedAudit?'Ver menos':'Ver todo el historial'}}</button></section>
    </div>
    <UiDialog v-if="selected && !action" :title="selected.name" :busy="busy" @close="selected=null"><p class="muted">{{selected.owner_email}}</p><div class="workshop-info"><span>Estado <b>{{statusLabels[selected.status]}}</b></span><span>Equipo <b>{{selected.members}} {{selected.members===1?'miembro':'miembros'}}</b></span><span>Pedidos activos <b>{{selected.active_orders}}</b></span></div><label class="demo-control"><div><strong>Taller de demostración</strong><p>Se excluye de las métricas habituales de adopción.</p></div><button role="switch" :aria-checked="selected.is_demo" :aria-label="`Taller de demostración: ${selected.name}`" :disabled="busy" @click="toggleDemo(selected)"><i/></button></label><p v-if="selected.deletion_scheduled_for" class="deletion-note">Eliminación programada: {{date(selected.deletion_scheduled_for)}}</p><div class="management-actions"><button v-if="selected.status !== 'ACTIVE'" class="ops-button" @click="changeStatus(selected,'ACTIVE')">{{selected.status==='PENDING_DELETION'?'Cancelar eliminación':'Reactivar taller'}}</button><button v-if="selected.status !== 'SUSPENDED'" class="ops-button" @click="changeStatus(selected,'SUSPENDED')">Suspender taller</button><button v-if="selected.status !== 'PENDING_DELETION'" class="ops-button danger-outline" @click="changeStatus(selected,'PENDING_DELETION')">Programar eliminación</button><button v-else class="ops-button danger-outline" @click="purge(selected)">Eliminar definitivamente</button></div></UiDialog>
    <UiDialog v-if="action" :title="actionTitle" :busy="busy" @close="action=null"><p><strong>{{action.workshop.name}}</strong></p><template v-if="action.purge"><p>Esta acción elimina el taller y sus datos de forma definitiva.</p><label class="action-field">Nombre exacto del taller<input v-model="confirmation" autocomplete="off"/></label><label class="action-field">Código de 6 dígitos<input v-model="verificationCode" inputmode="numeric" maxlength="6" autocomplete="one-time-code"/></label></template><template v-else><p class="muted">{{action.status==='SUSPENDED'?'El equipo perderá el acceso hasta que reactives el taller.':action.status==='PENDING_DELETION'?'La eliminación quedará programada. Podrás cancelarla durante el período de retención.':'El equipo podrá volver a acceder al taller.'}}</p><label class="action-field">Motivo<textarea v-model="reason" rows="3" maxlength="500" placeholder="Describe el motivo de este cambio"/></label></template><template #actions><button class="ops-button" :disabled="busy" @click="action=null">Cancelar</button><button class="ops-button ops-button--primary" :disabled="busy" @click="confirmAction">{{busy?'Guardando…':'Confirmar cambio'}}</button></template></UiDialog>
  </main>
</template>
<style scoped>
.console-page{min-height:100vh;background:#f4f6fb;color:var(--tf-ink);font-family:var(--tf-font-body)}.console-bar{display:flex;align-items:center;justify-content:space-between;padding:1rem clamp(1rem,4vw,3rem);background:var(--tf-shell);color:white}.console-brand{display:flex;gap:1.5rem;align-items:center}.console-brand>span{padding-left:1.5rem;border-left:1px solid #ffffff33;font-size:.8rem;color:#c3d4ea}.console-bar>button{border:1px solid #ffffff40;border-radius:10px;padding:.5rem .8rem;color:white;background:transparent;font:inherit;min-height:44px;cursor:pointer}.console-layout{max-width:1400px;padding:2rem clamp(1rem,3vw,2.5rem);margin:auto;display:grid;gap:1.5rem}.console-heading{display:flex;justify-content:space-between;align-items:end;gap:1rem}.console-heading p{margin:0 0 .25rem;color:var(--tf-accent-text);font-size:.8rem;font-weight:600}.console-heading h1{font-size:2rem;margin:0 0 .4rem}.console-heading span{color:var(--tf-text-subtle);font-size:.9rem}.demo-filter{display:flex;gap:.5rem;align-items:center;font-size:.8rem;color:var(--tf-text-subtle)}.demo-filter input{width:18px;height:18px;accent-color:var(--tf-primary)}.summary-grid{display:grid;grid-template-columns:repeat(3,minmax(0,1fr));gap:1rem}.summary-grid article{position:relative;display:grid;gap:.35rem;padding:1.5rem;border:1px solid #e0e7f1;border-radius:18px;background:white}.summary-grid article>span:not(.stat-icon){font-size:.85rem;color:var(--tf-text-subtle)}.summary-grid strong{font-size:2.5rem;line-height:1.2;font-weight:700;letter-spacing:-.05em}.summary-grid small{font-size:.78rem;color:var(--tf-text-subtle)}.stat-icon{position:absolute;right:1.25rem;top:1.25rem;display:grid;place-items:center;width:44px;height:44px;border-radius:13px;background:#eaf0fc;color:#315c9d}.stat-icon--green{background:#e4f5f0;color:#187a63}.stat-icon--violet{background:#f0eafa;color:#7756a2}.platform-health{padding:1.25rem 1.5rem;border-radius:16px;background:white;border:1px solid var(--tf-border-soft);display:grid;gap:.75rem}.platform-health>span{font-size:.9rem;font-weight:600}.health-track{display:flex;gap:4px;height:9px;overflow:hidden;border-radius:99px;background:#edf1f7}.health-track i{background:#25856d;min-width:0}.health-track .health-suspended{background:#e6aa46}.health-track .health-deletion{background:#c76376}.health-legend{display:flex;flex-wrap:wrap;gap:.5rem 1.5rem;font-size:.75rem;color:var(--tf-text-subtle)}.content,.card{padding:1.5rem;border:1px solid #e0e7f1;border-radius:18px;background:white;min-width:0}.content h2{font-size:1.2rem;margin:0}.heading{display:flex;align-items:center;justify-content:space-between;gap:1rem}.heading p,.muted{font-size:.85rem;color:var(--tf-text-subtle);margin:.4rem 0}.heading form{display:flex;gap:.5rem}.console-search{display:flex;align-items:center;gap:.5rem;border:1px solid var(--tf-border);border-radius:10px;padding:0 .75rem;color:var(--tf-text-subtle)}.console-search input{border:0;outline:0;background:transparent;min-height:44px;min-width:0;width:230px;font:inherit;font-size:.85rem}.heading form>button,.login button{border:0;border-radius:10px;padding:.6rem 1rem;background:var(--tf-primary);color:white;font:inherit;font-weight:600;cursor:pointer;min-height:44px}.workshop-filters{display:flex;flex-wrap:wrap;gap:.4rem;margin:1.25rem 0}.workshop-filters button{border:0;border-radius:999px;padding:.5rem .8rem;background:#f0f3f8;color:var(--tf-text-subtle);font:inherit;font-size:.8rem;min-height:40px;cursor:pointer}.workshop-filters button[aria-pressed="true"]{background:var(--tf-primary);color:white}.table-wrap{overflow:auto}table{border-collapse:collapse;width:100%;min-width:960px;font-size:.85rem}th,td{text-align:left;vertical-align:middle;padding:1rem .75rem;border-bottom:1px solid #edf0f5}thead th{font-size:.75rem;color:var(--tf-text-subtle);font-weight:600;background:#fafbfd}td small{display:block;color:var(--tf-text-subtle);font-size:.72rem;margin-top:.2rem}.workshop-name{font:inherit;font-weight:600;background:transparent;border:0;padding:0;color:var(--tf-ink);cursor:pointer;text-align:left;min-height:32px}.workshop-name:hover{text-decoration:underline}.manage-button{display:flex;align-items:center;gap:.5rem;padding:.5rem .65rem;border:1px solid var(--tf-border-soft);background:white;border-radius:9px;min-height:40px;font:inherit;font-size:.8rem;color:var(--tf-primary);cursor:pointer}.status,.demo-badge{display:inline-flex;border-radius:999px;padding:.25rem .6rem;font-size:.72rem;font-weight:600;white-space:nowrap}.status--active{background:#e6f5ef;color:#196c51}.status--suspended{background:#fff3d9;color:#8b6219}.status--pending_deletion{background:#fbe9ed;color:#a12d49}.demo-badge{margin-top:.5rem;background:#eaf0fc;color:#315c9d}.audit{list-style:none;padding:0;margin:1.25rem 0 0}.audit li{display:flex;gap:.8rem;align-items:start;padding:1rem 0;border-bottom:1px solid #edf0f5}.audit-dot{width:9px;height:9px;background:var(--tf-accent-text);border-radius:50%;flex:none;margin-top:.4rem}.audit strong{font-size:.85rem;font-weight:600}.audit p{font-size:.8rem;color:var(--tf-text-subtle);margin:.25rem 0}.audit small,.audit time{font-size:.75rem;color:var(--tf-text-subtle)}.audit time{margin-left:auto;white-space:nowrap}.login{max-width:470px;margin:3rem auto}.login h1{font-size:1.6rem}.login form,.login label{display:grid;gap:.5rem}.login form{gap:1rem;margin-top:1.5rem}.login input,.action-field input,.action-field textarea{width:100%;min-height:44px;border:1px solid var(--tf-border);border-radius:10px;padding:.65rem;font:inherit;background:white}.action-field{display:grid;gap:.5rem;margin-top:1rem;font-size:.85rem;font-weight:600}.workshop-info{display:grid;grid-template-columns:repeat(3,minmax(0,1fr));gap:.6rem;padding:1rem 0}.workshop-info span{display:grid;gap:.35rem;font-size:.75rem;color:var(--tf-text-subtle)}.workshop-info b{font-size:.9rem;color:var(--tf-ink)}.demo-control{display:flex;align-items:center;justify-content:space-between;gap:1rem;padding:1.25rem 0;border-block:1px solid var(--tf-border-soft)}.demo-control strong{font-size:.9rem}.demo-control p{font-size:.8rem;color:var(--tf-text-subtle);margin:.25rem 0}.demo-control button{width:48px;height:28px;min-width:48px;min-height:28px;border:0;border-radius:999px;background:#a7b4c8;cursor:pointer;padding:3px}.demo-control button i{display:block;width:22px;height:22px;border-radius:50%;background:white;transition:transform .18s}.demo-control button[aria-checked="true"]{background:var(--tf-accent-text)}.demo-control button[aria-checked="true"] i{transform:translateX(20px)}.management-actions{display:flex;gap:.65rem;flex-wrap:wrap;margin-top:1.25rem}.danger-outline{color:#a12d49;border-color:#edc9d1}.deletion-note{font-size:.85rem;color:#a12d49}.empty-workshops{text-align:center;padding:2rem;color:var(--tf-text-subtle)}button:disabled{opacity:.6;cursor:wait}button:focus-visible,a:focus-visible{outline:3px solid var(--tf-focus);outline-offset:3px}
@media(max-width:760px){.summary-grid{grid-template-columns:1fr}.console-heading,.heading{align-items:start;flex-direction:column}.heading form,.console-search{width:100%;min-width:0}.console-search input{width:100%}.content{padding:1rem}.console-heading h1{font-size:1.6rem}.console-brand>span{display:none}.console-layout{padding:1rem;gap:1rem}.audit li{flex-wrap:wrap}.audit time{margin-left:1.35rem}.workshop-info{grid-template-columns:1fr 1fr}}

.demo-control button{height:44px;min-height:44px;position:relative;background:transparent!important}.demo-control button::before{content:"";position:absolute;left:0;right:0;top:8px;bottom:8px;border-radius:999px;background:#a7b4c8}.demo-control button i{position:absolute;left:3px;top:11px}.demo-control button[aria-checked="true"]::before{background:var(--tf-accent-text)}

.console-layout{min-width:0;grid-template-columns:minmax(0,1fr);width:100%}.console-layout>*{min-width:0}.audit-more{margin-top:1rem}
@media(max-width:760px){.summary-grid{grid-template-columns:repeat(3,minmax(0,1fr));gap:.5rem}.summary-grid article{padding:1rem .7rem;gap:.5rem;border-radius:14px}.summary-grid .stat-icon{display:none}.summary-grid strong{font-size:2rem}.summary-grid article>span:not(.stat-icon){font-size:.75rem}.summary-grid small{font-size:.65rem}}

/* Keep absolutely positioned screen-reader labels inside the horizontal scroller. */
.table-wrap{position:relative}

@media(max-width:760px){.table-wrap table,.table-wrap tbody{display:block;min-width:0;width:100%}.table-wrap thead{position:absolute;width:1px;height:1px;overflow:hidden;clip-path:inset(50%)}.table-wrap tbody{display:grid;gap:.8rem}.table-wrap tbody tr{display:grid;grid-template-columns:minmax(0,1fr) minmax(0,1fr);gap:.8rem;padding:1rem;border:1px solid var(--tf-border-soft);border-radius:14px;background:white}.table-wrap td{padding:0;border:0;min-width:0;overflow-wrap:anywhere}.table-wrap td:first-child,.table-wrap td:last-child{grid-column:1/-1}.table-wrap td[data-label]::before{content:attr(data-label);display:block;color:var(--tf-text-subtle);font-size:.7rem;margin-bottom:.3rem}.table-wrap .manage-button{justify-content:center;width:100%;min-height:44px}.workshop-name{font-size:.9rem}}
</style>
