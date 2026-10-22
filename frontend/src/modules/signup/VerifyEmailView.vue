<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { RouterLink, useRoute, useRouter } from 'vue-router'
import AuthLayout from '../../layouts/AuthLayout.vue'
import UiAlert from '../../components/UiAlert.vue'
import UiButton from '../../components/UiButton.vue'
import UiField from '../../components/UiField.vue'
import { ApiError } from '../session/api'
import { useSessionStore } from '../session/session.store'
import { signupApi } from './api'
import { useResendVerification } from './useResendVerification'

type VerifyState = 'verifying' | 'invalid' | 'failed'

const route = useRoute()
const router = useRouter()
const session = useSessionStore()
const state = ref<VerifyState>('verifying')
const email = ref('')
const attempted = ref(false)
const resend = useResendVerification()
const emailError = computed(() => attempted.value && !/^\S+@\S+\.\S+$/.test(email.value.trim()) ? 'Ingresa un correo válido.' : '')

async function verify(token: string) {
  state.value = 'verifying'
  try {
    const result = await signupApi.verify(token)
    await session.acceptSession(result)
    await router.replace('/app')
  } catch (error) {
    state.value = error instanceof ApiError && (error.code === 'VERIFICATION_TOKEN_INVALID' || error.status === 400) ? 'invalid' : 'failed'
  }
}

function requestNewLink() {
  attempted.value = true
  if (emailError.value) return
  resend.resend(email.value)
}

onMounted(() => {
  const token = typeof route.query.token === 'string' ? route.query.token : ''
  // The token works once; keep it out of the address bar, the history and
  // the Referer of any link on the next screen.
  window.history.replaceState(window.history.state, '', route.path)
  if (!token) {
    state.value = 'invalid'
    return
  }
  verify(token)
})
</script>

<template>
  <AuthLayout>
    <div v-if="state === 'verifying'" class="verify-state" role="status" aria-live="polite">
      <span class="ui-button__spinner verify-state__spinner" aria-hidden="true" />
      <h1 id="auth-title">Activando tu taller…</h1>
      <p class="auth-subtitle">Un momento, estamos verificando tu correo.</p>
    </div>

    <div v-else-if="state === 'failed'" class="verify-state">
      <h1 id="auth-title">No pudimos activar tu taller</h1>
      <p class="auth-subtitle">Parece un problema de conexión. Tu enlace sigue sirviendo: vuelve a abrirlo desde el correo en un momento.</p>
      <RouterLink to="/login" class="verify-link">Ir a iniciar sesión</RouterLink>
    </div>

    <template v-else>
      <h1 id="auth-title">Este enlace ya no sirve</h1>
      <p class="auth-subtitle">Venció, ya se usó o se copió incompleto. Si ya activaste tu taller, solo inicia sesión. Si no, pide un enlace nuevo.</p>
      <form class="auth-form" novalidate @submit.prevent="requestNewLink">
        <UiAlert v-if="resend.failed.value" :message="resend.failed.value" />
        <p v-if="resend.sent.value && !resend.failed.value" class="verify-confirmation" role="status">Si ese correo tiene un taller por activar, te llegará un enlace nuevo en unos minutos.</p>
        <UiField id="verify-email" v-model="email" label="Correo con el que te registraste" type="email" autocomplete="email" :error="emailError" />
        <UiButton type="submit" :loading="resend.sending.value" :disabled="resend.remaining.value > 0">
          {{ resend.remaining.value > 0 ? `Pedir otro en ${resend.remaining.value} s` : 'Enviarme un enlace nuevo' }}
        </UiButton>
        <RouterLink to="/login" class="verify-link">Ir a iniciar sesión</RouterLink>
      </form>
    </template>
  </AuthLayout>
</template>

<style scoped>
.verify-state {
  display: grid;
  gap: 0.75rem;
  justify-items: center;
  text-align: center;
}

.verify-state__spinner {
  width: 1.75rem;
  height: 1.75rem;
  border-color: var(--tf-primary-soft);
  border-top-color: var(--tf-primary);
}

.verify-confirmation {
  margin: 0;
  color: var(--tf-ink);
  font-size: 0.875rem;
  font-weight: 600;
}

.verify-link {
  display: inline-flex;
  min-height: 44px;
  align-items: center;
  justify-content: center;
  color: var(--tf-primary);
  font-size: 0.875rem;
  font-weight: 600;
  border-radius: 0.375rem;
}

.verify-link:focus-visible {
  outline: 3px solid var(--tf-focus);
  outline-offset: 2px;
}
</style>
