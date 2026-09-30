<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import UiAlert from '../../components/UiAlert.vue'
import UiButton from '../../components/UiButton.vue'
import UiField from '../../components/UiField.vue'
import { ApiError } from '../session/api'
import { useSessionStore } from '../session/session.store'
import { teamApi, type TeamInvitation, type TeamMember } from './api'

const session = useSessionStore()
const members = ref<TeamMember[]>([])
const invitations = ref<TeamInvitation[]>([])
const email = ref('')
const role = ref<'ADMIN' | 'OPERATOR'>('OPERATOR')
const joinUrl = ref('')
const loading = ref(false)
const error = ref('')
const notice = ref('')
const busyInvitationId = ref('')
const teamLoading = ref(false)

const canInviteAdmin = computed(() => session.principal?.role === 'OWNER')

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
    notice.value = `La invitación para ${response.data.invitation.email} está lista. Copia el enlace y compártelo con esa persona.`
    email.value = ''
    role.value = 'OPERATOR'
    await loadTeam()
  } catch (err) {
    error.value = err instanceof ApiError && err.code === 'TEAM_INVITATION_DUPLICATE'
      ? 'Ya hay una invitación pendiente para ese correo. Puedes generar un enlace nuevo desde la lista.'
      : 'No se pudo crear la invitación. Revisa el correo e inténtalo otra vez.'
  } finally {
    loading.value = false
  }
}

async function copyInvitationLink() {
  if (!joinUrl.value) return
  try {
    await navigator.clipboard.writeText(joinUrl.value)
    notice.value = 'Enlace copiado. Ya puedes enviarlo por el medio que prefieras.'
  } catch {
    error.value = 'No se pudo copiar el enlace. Selecciónalo y cópialo manualmente.'
  }
}

async function regenerateInvitation(invitation: TeamInvitation) {
  if (!session.csrfToken || !window.confirm(`Se invalidará el enlace anterior para ${invitation.email}. ¿Generar uno nuevo?`)) return
  busyInvitationId.value = invitation.id
  error.value = ''
  try {
    const response = await teamApi.regenerateInvitation(invitation.id, session.csrfToken)
    joinUrl.value = response.data.join_url
    notice.value = `Se generó un enlace nuevo para ${response.data.invitation.email}. El anterior ya no funciona.`
    await loadTeam()
  } catch {
    error.value = 'No se pudo generar un enlace nuevo. Inténtalo otra vez.'
  } finally {
    busyInvitationId.value = ''
  }
}

async function cancelInvitation(invitation: TeamInvitation) {
  if (!session.csrfToken || !window.confirm(`¿Cancelar la invitación pendiente para ${invitation.email}? El enlace dejará de funcionar.`)) return
  busyInvitationId.value = invitation.id
  error.value = ''
  try {
    await teamApi.cancelInvitation(invitation.id, session.csrfToken)
    joinUrl.value = ''
    notice.value = `La invitación para ${invitation.email} fue cancelada.`
    await loadTeam()
  } catch {
    error.value = 'No se pudo cancelar la invitación. Inténtalo otra vez.'
  } finally {
    busyInvitationId.value = ''
  }
}

async function deactivate(member: TeamMember) {
  if (!session.csrfToken) return
  error.value = ''
  try {
    await teamApi.updateMember(member.id, { status: member.status === 'ACTIVE' ? 'INACTIVE' : 'ACTIVE' }, session.csrfToken)
    notice.value = member.status === 'ACTIVE' ? `${member.display_name} fue desactivado.` : `${member.display_name} fue activado.`
    await loadTeam()
  } catch {
    error.value = 'No se pudo actualizar el miembro. Inténtalo otra vez.'
  }
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
      <div v-if="notice" class="team-notice" role="status" aria-live="polite">{{ notice }}</div>
      <div v-if="joinUrl" class="team-link-result">
        <label for="join-link">Enlace de invitación</label>
        <div class="team-link-result__row">
          <input id="join-link" :value="joinUrl" readonly aria-describedby="join-link-help" />
          <UiButton type="button" @click="copyInvitationLink">Copiar enlace</UiButton>
        </div>
        <p id="join-link-help">Por seguridad, este enlace se muestra ahora. Si lo pierdes, genera uno nuevo desde las invitaciones pendientes.</p>
      </div>
    </section>

    <section class="team-section" aria-labelledby="members-title">
      <div class="team-section__heading">
        <div>
          <h2 id="members-title">Miembros</h2>
          <p>Las cuentas se conservan para mantener el historial del taller. Puedes desactivarlas y reactivarlas.</p>
        </div>
        <span class="team-count">{{ members.length }}</span>
      </div>
      <p v-if="teamLoading" class="team-empty" role="status">Cargando equipo…</p>
      <div v-else-if="members.length" class="team-grid">
        <article v-for="member in members" :key="member.id" class="team-card">
          <div>
            <h3>{{ member.display_name }}</h3>
            <p>{{ member.email }}</p>
          </div>
          <div class="team-card__meta">
            <strong>{{ member.role === 'OWNER' ? 'Propietario' : member.role === 'ADMIN' ? 'Administrador' : 'Operador' }}</strong>
            <span :class="['team-status', member.status === 'ACTIVE' ? 'team-status--active' : 'team-status--inactive']">{{ member.status === 'ACTIVE' ? 'Activo' : 'Desactivado' }}</span>
          </div>
          <UiButton v-if="session.principal?.role === 'OWNER' && member.user_id !== session.principal.id" @click="deactivate(member)">
            {{ member.status === 'ACTIVE' ? 'Desactivar' : 'Activar' }}
          </UiButton>
        </article>
      </div>
      <p v-else class="team-empty">Todavía no hay miembros para mostrar.</p>
    </section>

    <section v-if="invitations.length" class="team-section" aria-labelledby="pending-title">
      <div class="team-section__heading">
        <div>
          <h2 id="pending-title">Invitaciones pendientes</h2>
          <p>Un enlace pendiente puede regenerarse o cancelarse. Al regenerarlo, el anterior deja de funcionar.</p>
        </div>
        <span class="team-count">{{ invitations.length }}</span>
      </div>
      <ul class="team-list">
        <li v-for="invitation in invitations" :key="invitation.id">
          <div class="team-list__identity">
            <strong>{{ invitation.email }}</strong>
            <span>{{ invitation.role === 'ADMIN' ? 'Administrador' : 'Operador' }} · {{ invitationExpiry(invitation.expires_at) }}</span>
          </div>
          <div class="team-list__actions">
            <UiButton type="button" :loading="busyInvitationId === invitation.id" @click="regenerateInvitation(invitation)">Generar enlace nuevo</UiButton>
            <button class="team-text-button team-text-button--danger" type="button" :disabled="Boolean(busyInvitationId)" @click="cancelInvitation(invitation)">Cancelar</button>
          </div>
        </li>
      </ul>
    </section>
  </main>
</template>
