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

type VerifyState = 'password' | 'invalid' | 'failed'

const route = useRoute()
const router = useRouter()
const session = useSessionStore()
const state = ref<VerifyState>('password')
const token = ref('')
const password = ref('')
const passwordError = ref('')
const verifying = ref(false)
const email = ref('')
const attempted = ref(false)
const resend = useResendVerification()
const emailError = computed(() => attempted.value && !/^\S+@\S+\.\S+$/.test(email.value.trim()) ? 'Ingresa un correo válido.' : '')

// The link alone does not activate the workshop: whoever opens it also has
// to know the password chosen at signup, so nobody can activate an account
// someone else registered with their address.
async function verify() {
  passwordError.value = ''
  if (!password.value) {
    passwordError.value = 'Escribe la contraseña que elegiste al registrarte.'
    return
  }
  verifying.value = true
  try {
    const result = await signupApi.verify(token.value, password.value)
    await session.acceptSession(result)
    await router.replace('/app')
  } catch (error) {
    if (error instanceof ApiError && error.code === 'VERIFICATION_PASSWORD_INVALID') {
      passwordError.value = 'Esa no es la contraseña con la que se registró este taller.'
    } else {
      state.value = error instanceof ApiError && (error.code === 'VERIFICATION_TOKEN_INVALID' || error.status === 400) ? 'invalid' : 'failed'
    }
  } finally {
    verifying.value = false
  }
}

function requestNewLink() {
  attempted.value = true
  if (emailError.value) return
  resend.resend(email.value)
}

onMounted(() => {
  token.value = typeof route.query.token === 'string' ? route.query.token : ''
  // The token works once; keep it out of the address bar, the history and
  // the Referer of any link on the next screen.
  window.history.replaceState(window.history.state, '', route.path)
  if (!token.value) state.value = 'invalid'
})
</script>

<template>
  <AuthLayout>
    <template v-if="state === 'password'">
      <h1 id="auth-title">Activa tu taller</h1>
      <p class="auth-subtitle">Por seguridad, escribe la contraseña que elegiste al registrarte.</p>
      <form class="auth-form" novalidate @submit.prevent="verify">
        <UiField id="verify-password" v-model="password" label="Tu contraseña" type="password" autocomplete="current-password" :error="passwordError" />
        <UiButton type="submit" :loading="verifying">Activar mi taller</UiButton>
        <p class="verify-hint">¿No la recuerdas o no fuiste tú quien se registró? <RouterLink to="/registro" class="verify-inline-link">Regístrate de nuevo con este correo</RouterLink> y te llegará un enlace nuevo.</p>
      </form>
    </template>

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

.verify-hint {
  margin: 0;
  color: var(--tf-text-subtle);
  font-size: 0.875rem;
  line-height: 1.5;
}

.verify-inline-link {
  color: var(--tf-primary);
  font-weight: 600;
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
