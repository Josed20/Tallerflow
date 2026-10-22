<script setup lang="ts">
import { computed, ref } from 'vue'
import { RouterLink, useRoute, useRouter } from 'vue-router'
import AuthLayout from '../../layouts/AuthLayout.vue'
import UiAlert from '../../components/UiAlert.vue'
import UiButton from '../../components/UiButton.vue'
import UiField from '../../components/UiField.vue'
import { useSessionStore } from '../session/session.store'
import { ApiError } from '../session/api'
import { useResendVerification } from '../signup/useResendVerification'

const router = useRouter()
const route = useRoute()
const session = useSessionStore()
const email = ref(String(route.query.email ?? ''))
const password = ref('')
const attempted = ref(false)
const submitting = ref(false)
const serverError = ref('')
// Sprint 4 - José: an open-signup owner who has not opened the emailed link yet.
const unverifiedEmail = ref('')
const resend = useResendVerification()
const sessionNotice = computed(() => route.query.session === 'suspended'
  ? 'El taller está suspendido. Consulta con su propietario antes de volver a entrar.'
  : route.query.session === 'expired'
    ? 'Tu sesión terminó. Inicia sesión nuevamente.'
    : '')

const emailError = computed(() => attempted.value && !/^\S+@\S+\.\S+$/.test(email.value) ? 'Ingresa un correo válido.' : '')
const passwordError = computed(() => attempted.value && !password.value ? 'La contraseña es obligatoria.' : '')

async function submit() {
  if (submitting.value) return
  attempted.value = true
  serverError.value = ''
  unverifiedEmail.value = ''
  if (emailError.value || passwordError.value) return

  submitting.value = true
  try {
    await session.login(email.value, password.value)
    await router.replace(session.requiresPasswordChange ? '/change-password' : (route.query.redirect?.toString() ?? '/app'))
  } catch (error) {
    if (error instanceof ApiError && error.code === 'EMAIL_NOT_VERIFIED') {
      unverifiedEmail.value = email.value.trim()
      return
    }
    serverError.value = error instanceof ApiError && error.code === 'WORKSHOP_SUSPENDED'
      ? 'Este taller está suspendido. Consulta con su propietario.'
      : 'No pudimos iniciar sesión. Revisa tus datos e inténtalo de nuevo.'
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <AuthLayout>
    <h1 id="auth-title">Controla tus pedidos sin preguntar todo el día</h1>
    <p class="auth-subtitle">Plataforma de trazabilidad para talleres de confección textil</p>
    <form class="auth-form" novalidate @submit.prevent="submit">
      <p v-if="sessionNotice" class="auth-session-notice" role="status">{{ sessionNotice }}</p>
      <UiAlert v-if="serverError" :message="serverError" />
      <div v-if="unverifiedEmail" class="auth-session-notice auth-unverified" role="status">
        <p><strong>Activa tu taller primero.</strong> Abre el enlace que enviamos a {{ unverifiedEmail }} para verificar tu correo.</p>
        <p v-if="resend.failed.value" class="auth-unverified__error">{{ resend.failed.value }}</p>
        <p v-else-if="resend.sent.value">Listo, te enviamos un enlace nuevo. Revisa también spam.</p>
        <UiButton type="button" variant="secondary" :loading="resend.sending.value" :disabled="resend.remaining.value > 0" @click="resend.resend(unverifiedEmail)">
          {{ resend.remaining.value > 0 ? `Reenviar en ${resend.remaining.value} s` : 'Reenviar el correo' }}
        </UiButton>
      </div>
      <UiField id="email" v-model="email" label="Correo electrónico" type="email" autocomplete="email" placeholder="ejemplo@taller.pe" :error="emailError" />
      <UiField id="password" v-model="password" label="Contraseña" type="password" autocomplete="current-password" placeholder="••••••••" :error="passwordError" />
      <div class="auth-recovery-link">
        <RouterLink to="/forgot-password" class="auth-link">
          ¿Olvidaste tu contraseña?
        </RouterLink>
      </div>
      <UiButton type="submit" :loading="submitting">Iniciar sesión</UiButton>
      <p class="auth-signup-link">¿Tu taller aún no tiene cuenta? <RouterLink to="/registro" class="auth-link">Créala aquí</RouterLink></p>
    </form>
  </AuthLayout>
</template>

<style scoped>
.auth-session-notice {
  margin: 0;
  padding: 0.75rem 1rem;
  border: 1px solid var(--tf-border);
  border-radius: 0.5rem;
  background: var(--tf-surface);
  color: var(--tf-ink);
}

.auth-unverified {
  display: grid;
  gap: 0.75rem;
}

.auth-unverified p {
  margin: 0;
}

.auth-unverified__error {
  color: var(--tf-danger);
}

.auth-signup-link {
  margin: 0;
  color: var(--tf-text-muted);
  font-size: 0.875rem;
  text-align: center;
}

.auth-recovery-link {
  display: flex;
  justify-content: flex-end;
  margin-top: -0.25rem;
}

.auth-link {
  display: inline-flex;
  min-height: 44px;
  align-items: center;
  color: var(--tf-primary);
  font-size: 0.8125rem;
  font-weight: 600;
  text-decoration: none;
  border-radius: 0.25rem;
}

.auth-link:hover {
  text-decoration: underline;
}

.auth-link:focus-visible {
  outline: 3px solid var(--tf-focus);
  outline-offset: 2px;
}
</style>
