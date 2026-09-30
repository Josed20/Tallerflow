<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import UiAlert from '../../components/UiAlert.vue'
import UiButton from '../../components/UiButton.vue'
import UiField from '../../components/UiField.vue'
import { ApiError } from '../session/api'
import { useSessionStore } from '../session/session.store'
import { teamApi, type TeamInvitation, type TeamMember } from './api'

type DirectoryView = 'members' | 'invitations'
type InvitationAction = 'regenerate' | 'cancel'

interface PendingInvitationAction {
  kind: InvitationAction
  invitation: TeamInvitation
}

interface RevealedInvitationLink {
  id: string
  email: string
  url: string
}

const session = useSessionStore()
const route = useRoute()
const router = useRouter()
const members = ref<TeamMember[]>([])
const invitations = ref<TeamInvitation[]>([])
const email = ref('')
const role = ref<'ADMIN' | 'OPERATOR'>('OPERATOR')
const joinUrl = ref('')
const loading = ref(false)
const error = ref('')
const toast = ref('')
const busyInvitationId = ref('')
const teamLoading = ref(false)
const search = ref('')
const membersLimit = ref(12)
const invitationsLimit = ref(12)
const pendingInvitationAction = ref<PendingInvitationAction | null>(null)
const revealedInvitationLink = ref<RevealedInvitationLink | null>(null)
let toastTimeout: ReturnType<typeof setTimeout> | undefined

const canInviteAdmin = computed(() => session.principal?.role === 'OWNER')
const directoryView = computed<DirectoryView>(() => route.query.view === 'invitations' ? 'invitations' : 'members')
const normalizedSearch = computed(() => search.value.trim().toLocaleLowerCase())
const filteredMembers = computed(() => members.value.filter((member) => {
  const query = normalizedSearch.value
  return !query || [member.display_name, member.email, roleLabel(member.role), member.status === 'ACTIVE' ? 'activo' : 'desactivado']
    .some((value) => value.toLocaleLowerCase().includes(query))
}))
const filteredInvitations = computed(() => invitations.value.filter((invitation) => {
  const query = normalizedSearch.value
  return !query || [invitation.email, roleLabel(invitation.role), invitationExpiry(invitation.expires_at)]
    .some((value) => value.toLocaleLowerCase().includes(query))
}))
const visibleMembers = computed(() => filteredMembers.value.slice(0, membersLimit.value))
const visibleInvitations = computed(() => filteredInvitations.value.slice(0, invitationsLimit.value))

function showToast(message: string) {
  toast.value = message
  if (toastTimeout) clearTimeout(toastTimeout)
  toastTimeout = setTimeout(() => { toast.value = '' }, 4500)
}

function dismissToast() {
  if (toastTimeout) clearTimeout(toastTimeout)
  toast.value = ''
}

function setDirectoryView(view: DirectoryView) {
  search.value = ''
  membersLimit.value = 12
  invitationsLimit.value = 12
  router.replace({ query: { ...route.query, view } })
}

function resetResultLimit() {
  if (directoryView.value === 'members') membersLimit.value = 12
  else invitationsLimit.value = 12
}

async function loadTeam() {
  teamLoading.value = true
  try {
    const response = await teamApi.list()
    members.value = response.data.members
    invitations.value = response.data.invitations
  } catch {
    error.value = 'No se pudo cargar el equipo. Actualiza la página e inténtalo otra vez.'
  } finally {
    teamLoading.value = false
  }
}

async function invite() {
  if (!session.csrfToken) return
  loading.value = true
  error.value = ''
  joinUrl.value = ''
  try {
    const response = await teamApi.invite(email.value, role.value, session.csrfToken)
    joinUrl.value = response.data.join_url
    invitations.value = [response.data.invitation, ...invitations.value]
    email.value = ''
    role.value = 'OPERATOR'
    showToast(`Invitación creada para ${response.data.invitation.email}.`)
  } catch (err) {
    error.value = err instanceof ApiError && err.code === 'TEAM_INVITATION_DUPLICATE'
      ? 'Ya hay una invitación pendiente para ese correo. Puedes abrirla en Invitaciones y generar un enlace nuevo.'
      : 'No se pudo crear la invitación. Revisa el correo e inténtalo otra vez.'
  } finally {
    loading.value = false
  }
}

async function copyInvitationLink(url: string) {
  try {
    await navigator.clipboard.writeText(url)
    showToast('Enlace copiado.')
  } catch {
    error.value = 'No se pudo copiar el enlace. Selecciónalo y cópialo manualmente.'
  }
}

function requestInvitationAction(kind: InvitationAction, invitation: TeamInvitation) {
  pendingInvitationAction.value = { kind, invitation }
}

function closeInvitationAction() {
  if (!busyInvitationId.value) pendingInvitationAction.value = null
}

async function confirmInvitationAction() {
  const action = pendingInvitationAction.value
  if (!action || !session.csrfToken) return
  busyInvitationId.value = action.invitation.id
  error.value = ''
  try {
    if (action.kind === 'regenerate') {
      const response = await teamApi.regenerateInvitation(action.invitation.id, session.csrfToken)
      invitations.value = invitations.value.map((invitation) => invitation.id === action.invitation.id ? response.data.invitation : invitation)
      revealedInvitationLink.value = { id: action.invitation.id, email: response.data.invitation.email, url: response.data.join_url }
      joinUrl.value = ''
      showToast(`Enlace nuevo listo para ${response.data.invitation.email}.`)
    } else {
      await teamApi.cancelInvitation(action.invitation.id, session.csrfToken)
      invitations.value = invitations.value.filter((invitation) => invitation.id !== action.invitation.id)
      if (revealedInvitationLink.value?.id === action.invitation.id) revealedInvitationLink.value = null
      showToast(`Invitación cancelada para ${action.invitation.email}.`)
    }
    pendingInvitationAction.value = null
  } catch {
    error.value = action.kind === 'regenerate'
      ? 'No se pudo generar un enlace nuevo. Inténtalo otra vez.'
      : 'No se pudo cancelar la invitación. Inténtalo otra vez.'
  } finally {
    busyInvitationId.value = ''
  }
}

async function deactivate(member: TeamMember) {
  if (!session.csrfToken) return
  error.value = ''
  try {
    const response = await teamApi.updateMember(member.id, { status: member.status === 'ACTIVE' ? 'INACTIVE' : 'ACTIVE' }, session.csrfToken)
    members.value = members.value.map((current) => current.id === member.id ? response.data : current)
    showToast(member.status === 'ACTIVE' ? `${member.display_name} fue desactivado.` : `${member.display_name} fue activado.`)
  } catch {
    error.value = 'No se pudo actualizar el miembro. Inténtalo otra vez.'
  }
}

function roleLabel(value: TeamInvitation['role'] | TeamMember['role']) {
  if (value === 'OWNER') return 'Propietario'
  return value === 'ADMIN' ? 'Administrador' : 'Operador'
}

function invitationExpiry(expiresAt: string) {
  const diffMs = new Date(expiresAt).getTime() - Date.now()
  if (diffMs <= 0) return 'Vencida'
  const hours = Math.ceil(diffMs / (60 * 60 * 1000))
  if (hours < 24) return `Vence en ${hours} h`
  const days = Math.ceil(hours / 24)
  return `Vence en ${days} ${days === 1 ? 'día' : 'días'}`
}

onMounted(loadTeam)
onBeforeUnmount(() => {
  if (toastTimeout) clearTimeout(toastTimeout)
})
</script>

<template>
  <main class="team-page">
    <header class="team-header">
      <div>
        <p class="eyebrow">Equipo</p>
        <h1>Miembros del taller</h1>
      </div>
      <RouterLink class="text-link" to="/app">Volver</RouterLink>
    </header>

    <section class="team-panel" aria-labelledby="invite-title">
      <div class="team-panel__heading">
        <div>
          <h2 id="invite-title">Invitar al equipo</h2>
          <p>Comparte el enlace con la persona. Ella creará su nombre y clave al abrirlo.</p>
        </div>
        <span class="team-help">Los enlaces vencen en 7 días.</span>
      </div>
      <form class="team-form" @submit.prevent="invite">
        <UiField id="invite-email" v-model="email" name="invite-email" label="Correo de la persona" type="email" autocomplete="email" :required="true" :spellcheck="false" />
        <label class="team-select">
          Rol
          <select v-model="role" name="invite-role">
            <option value="OPERATOR">Operador</option>
            <option v-if="canInviteAdmin" value="ADMIN">Administrador</option>
          </select>
        </label>
        <UiButton type="submit" :loading="loading">Crear invitación</UiButton>
      </form>
      <UiAlert v-if="error" :message="error" />
      <div v-if="joinUrl" class="team-link-result">
        <label for="join-link">Enlace de invitación listo</label>
        <div class="team-link-result__row">
          <input id="join-link" :value="joinUrl" readonly aria-describedby="join-link-help" />
          <UiButton type="button" @click="copyInvitationLink(joinUrl)">Copiar enlace</UiButton>
        </div>
        <p id="join-link-help">Este enlace funciona hasta vencer, ser usado, cancelado o reemplazado por uno nuevo.</p>
      </div>
    </section>

    <section class="team-directory" aria-label="Directorio del equipo">
      <div class="team-directory__toolbar">
        <div class="team-tabs" role="tablist" aria-label="Directorio del equipo">
          <button class="team-tab" :class="{ 'team-tab--active': directoryView === 'members' }" type="button" role="tab" :aria-selected="directoryView === 'members'" @click="setDirectoryView('members')">
            Miembros <span>{{ members.length }}</span>
          </button>
          <button class="team-tab" :class="{ 'team-tab--active': directoryView === 'invitations' }" type="button" role="tab" :aria-selected="directoryView === 'invitations'" @click="setDirectoryView('invitations')">
            Invitaciones <span>{{ invitations.length }}</span>
          </button>
        </div>
        <label class="team-search">
          <span>Buscar {{ directoryView === 'members' ? 'miembro' : 'invitación' }}</span>
          <input v-model="search" name="team-search" type="search" autocomplete="off" :placeholder="directoryView === 'members' ? 'Nombre, correo, rol o estado' : 'Correo o rol'" @input="resetResultLimit" />
        </label>
      </div>

      <section v-if="directoryView === 'members'" class="team-directory__panel" aria-labelledby="members-title">
        <div class="team-section__heading">
          <div>
            <h2 id="members-title">Miembros</h2>
            <p>Activa o desactiva el acceso sin perder el historial del taller.</p>
          </div>
          <span class="team-count">{{ filteredMembers.length }}</span>
        </div>
        <p v-if="teamLoading" class="team-empty" role="status">Cargando equipo…</p>
        <div v-else-if="filteredMembers.length" class="team-grid">
          <article v-for="member in visibleMembers" :key="member.id" class="team-card">
            <div>
              <h3>{{ member.display_name }}</h3>
              <p>{{ member.email }}</p>
            </div>
            <div class="team-card__meta">
              <strong>{{ roleLabel(member.role) }}</strong>
              <span :class="['team-status', member.status === 'ACTIVE' ? 'team-status--active' : 'team-status--inactive']">{{ member.status === 'ACTIVE' ? 'Activo' : 'Desactivado' }}</span>
            </div>
            <UiButton v-if="session.principal?.role === 'OWNER' && member.user_id !== session.principal.id" @click="deactivate(member)">
              {{ member.status === 'ACTIVE' ? 'Desactivar' : 'Activar' }}
            </UiButton>
          </article>
        </div>
        <p v-else class="team-empty">No se encontraron miembros con esa búsqueda.</p>
        <button v-if="!teamLoading && visibleMembers.length < filteredMembers.length" class="team-show-more" type="button" @click="membersLimit += 12">Mostrar más miembros</button>
      </section>

      <section v-else class="team-directory__panel" aria-labelledby="pending-title">
        <div class="team-section__heading">
          <div>
            <h2 id="pending-title">Invitaciones pendientes</h2>
            <p>Generar un enlace nuevo invalida el anterior. Cancelar bloquea el acceso por completo.</p>
          </div>
          <span class="team-count">{{ filteredInvitations.length }}</span>
        </div>
        <ul v-if="filteredInvitations.length" class="team-list">
          <li v-for="invitation in visibleInvitations" :key="invitation.id" class="team-invitation-item">
            <div class="team-invitation-item__row">
              <div class="team-list__identity">
                <strong>{{ invitation.email }}</strong>
                <span>{{ roleLabel(invitation.role) }} · {{ invitationExpiry(invitation.expires_at) }}</span>
              </div>
              <div class="team-list__actions">
                <UiButton type="button" :loading="busyInvitationId === invitation.id" @click="requestInvitationAction('regenerate', invitation)">Generar enlace nuevo</UiButton>
                <button class="team-text-button team-text-button--danger" type="button" :disabled="Boolean(busyInvitationId)" @click="requestInvitationAction('cancel', invitation)">Cancelar</button>
              </div>
            </div>
            <div v-if="revealedInvitationLink?.id === invitation.id" class="team-inline-link">
              <label :for="`invitation-link-${invitation.id}`">Enlace nuevo listo para copiar</label>
              <div class="team-link-result__row">
                <input :id="`invitation-link-${invitation.id}`" :value="revealedInvitationLink.url" readonly />
                <UiButton type="button" @click="copyInvitationLink(revealedInvitationLink.url)">Copiar enlace</UiButton>
              </div>
            </div>
          </li>
        </ul>
        <p v-else class="team-empty">{{ invitations.length ? 'No se encontraron invitaciones con esa búsqueda.' : 'No hay invitaciones pendientes.' }}</p>
        <button v-if="visibleInvitations.length < filteredInvitations.length" class="team-show-more" type="button" @click="invitationsLimit += 12">Mostrar más invitaciones</button>
      </section>
    </section>

    <Teleport to="body">
      <div v-if="pendingInvitationAction" class="team-modal-backdrop" @click.self="closeInvitationAction">
        <section class="team-modal" role="dialog" aria-modal="true" aria-labelledby="invitation-action-title" @keydown.esc="closeInvitationAction">
          <p class="eyebrow">Confirmar acción</p>
          <h2 id="invitation-action-title">{{ pendingInvitationAction.kind === 'regenerate' ? 'Generar enlace nuevo' : 'Cancelar invitación' }}</h2>
          <p>{{ pendingInvitationAction.kind === 'regenerate'
            ? `El enlace anterior para ${pendingInvitationAction.invitation.email} dejará de funcionar.`
            : `La persona con correo ${pendingInvitationAction.invitation.email} ya no podrá usar este enlace.` }}</p>
          <div class="team-modal__actions">
            <button class="team-text-button" type="button" :disabled="Boolean(busyInvitationId)" @click="closeInvitationAction">Volver</button>
            <UiButton type="button" :loading="Boolean(busyInvitationId)" @click="confirmInvitationAction">{{ pendingInvitationAction.kind === 'regenerate' ? 'Generar enlace' : 'Cancelar invitación' }}</UiButton>
          </div>
        </section>
      </div>
    </Teleport>

    <Teleport to="body">
      <div v-if="toast" class="team-toast" role="status" aria-live="polite">
        <span>{{ toast }}</span>
        <button type="button" aria-label="Cerrar aviso" @click="dismissToast">×</button>
      </div>
    </Teleport>
  </main>
</template>
