<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ApiError } from '../session/api'
import UiAlert from '../../components/UiAlert.vue'
import UiButton from '../../components/UiButton.vue'
import UiField from '../../components/UiField.vue'
import { teamApi } from '../team/api'

const route = useRoute()
const router = useRouter()
const token = computed(() => String(route.query.token ?? ''))
const name = ref('')
const password = ref('')
const error = ref('')
const loading = ref(false)
const done = ref(false)
const invitedEmail = ref('')
const showRecovery = ref(false)

function goToLogin() {
  return router.replace({
    path: '/login',
    query: invitedEmail.value ? { email: invitedEmail.value } : undefined,
  })
}

async function join() {
  loading.value = true
  error.value = ''
  showRecovery.value = false
  if (name.value.trim() === '') {
    error.value = 'Ingresa tu nombre para aceptar la invitación.'
    loading.value = false
    return
  }
  if (password.value.length < 8) {
    error.value = 'La clave debe tener al menos 8 caracteres. Mejor si mezclas palabras, numeros o simbolos.'
    loading.value = false
    return
  }
  try {
    const result = await teamApi.consume(token.value, name.value, password.value)
    invitedEmail.value = result.data.email
    done.value = true
  } catch (err) {
    password.value = ''
    if (err instanceof ApiError && err.code === 'TEAM_INVALID') {
      error.value = 'Revisa tu nombre y clave. La clave debe tener al menos 8 caracteres.'
      return
    }
    showRecovery.value = true
    error.value = 'La invitación no está disponible o ya fue usada.'
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <main class="auth-layout">
    <section class="auth-card">
      <div class="auth-card__content">
        <div class="brand-mark">TF</div>
        <h1>Unirte al taller</h1>
        <p class="auth-subtitle">Crea tu acceso para entrar al equipo de TallerFlow.</p>

        <form v-if="!done" class="auth-form" @submit.prevent="join">
          <UiField id="join-name" v-model="name" label="Nombre" autocomplete="name" />
          <UiField id="join-password" v-model="password" label="Clave" type="password" autocomplete="new-password" />
          <UiAlert v-if="error" :message="error" />
          <UiButton type="submit" :loading="loading">Aceptar invitación</UiButton>
        </form>

        <div v-else class="join-success" role="status" aria-live="polite">
          <div class="join-success__icon" aria-hidden="true">&#10003;</div>
          <div>
            <h2>Cuenta creada correctamente</h2>
            <p>Ya formas parte del taller. Inicia sesión con:</p>
            <strong>{{ invitedEmail }}</strong>
            <p>y la contraseña que acabas de crear.</p>
          </div>
          <UiButton @click="goToLogin">Iniciar sesión</UiButton>
        </div>

        <div v-if="showRecovery" class="join-recovery">
          <p>Si ya completaste el registro, tu cuenta está lista. Intenta iniciar sesión con el correo que recibió la invitación.</p>
          <UiButton @click="goToLogin">Ir a iniciar sesión</UiButton>
        </div>
      </div>
    </section>
  </main>
</template>
