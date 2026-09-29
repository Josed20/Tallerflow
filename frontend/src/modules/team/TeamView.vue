<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import UiAlert from '../../components/UiAlert.vue'
import UiButton from '../../components/UiButton.vue'
import UiField from '../../components/UiField.vue'
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

const canInviteAdmin = computed(() => session.principal?.role === 'OWNER')

async function loadTeam() {
  error.value = ''
  const response = await teamApi.list()
  members.value = response.data.members
  invitations.value = response.data.invitations
}

async function invite() {
  if (!session.csrfToken) return
  loading.value = true
  error.value = ''
  joinUrl.value = ''
  try {
    const response = await teamApi.invite(email.value, role.value, session.csrfToken)
    joinUrl.value = response.data.join_url
    email.value = ''
    role.value = 'OPERATOR'
    await loadTeam()
  } catch {
    error.value = 'No se pudo crear la invitación.'
  } finally {
    loading.value = false
  }
}

async function deactivate(member: TeamMember) {
  if (!session.csrfToken) return
  error.value = ''
  try {
    await teamApi.updateMember(member.id, { status: member.status === 'ACTIVE' ? 'INACTIVE' : 'ACTIVE' }, session.csrfToken)
    await loadTeam()
  } catch {
    error.value = 'No se pudo actualizar el miembro.'
  }
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

    <section class="team-panel">
      <form class="team-form" @submit.prevent="invite">
        <UiField id="invite-email" v-model="email" label="Correo" type="email" autocomplete="email" />
        <label class="team-select">
          Rol
          <select v-model="role">
            <option value="OPERATOR">Operador</option>
            <option v-if="canInviteAdmin" value="ADMIN">Administrador</option>
          </select>
        </label>
        <UiButton type="submit" :loading="loading">Invitar</UiButton>
      </form>
      <UiAlert v-if="error" :message="error" />
      <p v-if="joinUrl" class="join-url">{{ joinUrl }}</p>
    </section>

    <section class="team-grid" aria-label="Miembros activos e invitaciones">
      <article v-for="member in members" :key="member.id" class="team-card">
        <div>
          <h2>{{ member.display_name }}</h2>
          <p>{{ member.email }}</p>
        </div>
        <strong>{{ member.role }}</strong>
        <span>{{ member.status }}</span>
        <UiButton v-if="session.principal?.role === 'OWNER' && member.user_id !== session.principal.id" @click="deactivate(member)">
          {{ member.status === 'ACTIVE' ? 'Desactivar' : 'Activar' }}
        </UiButton>
      </article>
    </section>

    <section v-if="invitations.length" class="team-panel">
      <h2>Invitaciones pendientes</h2>
      <ul class="team-list">
        <li v-for="invitation in invitations" :key="invitation.id">
          <span>{{ invitation.email }}</span>
          <strong>{{ invitation.role }}</strong>
        </li>
      </ul>
    </section>
  </main>
</template>
