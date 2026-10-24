<script setup lang="ts">
import { computed, nextTick, ref } from 'vue'
import { RouterLink } from 'vue-router'
import { Mail } from '@lucide/vue'
import AuthLayout from '../../layouts/AuthLayout.vue'
import UiAlert from '../../components/UiAlert.vue'
import UiButton from '../../components/UiButton.vue'
import UiField from '../../components/UiField.vue'
import { ApiError } from '../session/api'
import { signupApi } from './api'
import { useResendVerification } from './useResendVerification'

const workshopName = ref('')
const ownerName = ref('')
const email = ref('')
const password = ref('')
const passwordConfirmation = ref('')
const acceptTerms = ref(false)
const attempted = ref(false)
const submitting = ref(false)
const serverError = ref('')
const sentTo = ref('')
const resend = useResendVerification()

const workshopNameError = computed(() => attempted.value && !workshopName.value.trim() ? 'Ingresa el nombre de tu taller.' : '')
const ownerNameError = computed(() => attempted.value && !ownerName.value.trim() ? 'Ingresa tu nombre.' : '')
const emailError = computed(() => attempted.value && !/^\S+@\S+\.\S+$/.test(email.value.trim()) ? 'Ingresa un correo válido.' : '')
const passwordError = computed(() => attempted.value && password.value.length < 12 ? 'Usa al menos 12 caracteres.' : '')
const confirmationError = computed(() => {
  if (!attempted.value) return ''
  if (!passwordConfirmation.value) return 'Repite tu contraseña.'
  return passwordConfirmation.value !== password.value ? 'Las contraseñas no coinciden.' : ''
})
const termsError = computed(() => attempted.value && !acceptTerms.value ? 'Acepta los términos y la política de privacidad para continuar.' : '')

const errors = computed(() => [
  { id: 'signup-workshop', message: workshopNameError.value },
  { id: 'signup-owner', message: ownerNameError.value },
  { id: 'signup-email', message: emailError.value },
  { id: 'signup-password', message: passwordError.value },
  { id: 'signup-password-confirmation', message: confirmationError.value },
  { id: 'signup-terms', message: termsError.value },
])

function messageFor(error: unknown) {
  if (!(error instanceof ApiError)) return 'No pudimos crear tu taller. Revisa tu conexión e inténtalo de nuevo.'
  switch (error.code) {
    case 'SIGNUP_CLOSED':
      return 'El registro de talleres está cerrado por ahora. Vuelve a intentarlo en unos días.'
    case 'SIGNUP_RATE_LIMITED':
      return 'Hiciste muchos intentos seguidos. Espera unos minutos e inténtalo de nuevo.'
    case 'SIGNUP_VALIDATION_FAILED':
      return 'Revisa los datos del formulario e inténtalo de nuevo.'
    default:
      return 'No pudimos crear tu taller. Inténtalo de nuevo en unos minutos.'
  }
}

async function submit() {
  if (submitting.value) return
  attempted.value = true
  serverError.value = ''
  const firstError = errors.value.find((entry) => entry.message)
  if (firstError) {
    await nextTick()
    document.getElementById(firstError.id)?.focus()
    return
  }

  submitting.value = true
  try {
    await signupApi.register({
      workshop_name: workshopName.value.trim(),
      owner_name: ownerName.value.trim(),
      email: email.value.trim(),
      password: password.value,
      accept_terms: true,
    })
    sentTo.value = email.value.trim()
    password.value = ''
    passwordConfirmation.value = ''
  } catch (error) {
    serverError.value = messageFor(error)
  } finally {
    submitting.value = false
  }
}

function editEmail() {
  sentTo.value = ''
  attempted.value = false
  nextTick(() => document.getElementById('signup-email')?.focus())
}
</script>

<template>
  <AuthLayout>
    <template v-if="!sentTo">
      <h1 id="auth-title">Crea la cuenta de tu taller</h1>
      <p class="auth-subtitle">Registra tu taller y empieza a seguir cada orden de Corte a Entrega.</p>
      <form class="auth-form" novalidate @submit.prevent="submit">
        <UiAlert v-if="serverError" :message="serverError" />
        <UiField id="signup-workshop" v-model="workshopName" label="Nombre del taller" autocomplete="organization" placeholder="Confecciones Lucho" :error="workshopNameError" />
        <UiField id="signup-owner" v-model="ownerName" label="Tu nombre" autocomplete="name" :error="ownerNameError" />
        <UiField id="signup-email" v-model="email" label="Correo electrónico" type="email" autocomplete="email" placeholder="ejemplo@taller.pe" description="Te enviaremos un enlace para activar tu taller." :error="emailError" />
        <UiField id="signup-password" v-model="password" label="Contraseña" type="password" autocomplete="new-password" description="Mínimo 12 caracteres. Una frase corta es fácil de recordar." :error="passwordError" />
        <UiField id="signup-password-confirmation" v-model="passwordConfirmation" label="Repite tu contraseña" type="password" autocomplete="new-password" :error="confirmationError" />
        <div class="signup-terms">
          <label class="signup-terms__label" for="signup-terms">
            <input id="signup-terms" v-model="acceptTerms" type="checkbox" :aria-invalid="Boolean(termsError)" :aria-describedby="termsError ? 'signup-terms-error' : undefined" />
            <span>Acepto los <RouterLink to="/terminos" target="_blank" class="signup-link">términos de servicio</RouterLink> y la <RouterLink to="/privacidad" target="_blank" class="signup-link">política de privacidad</RouterLink>.</span>
          </label>
          <p v-if="termsError" id="signup-terms-error" class="ui-field__error" role="alert">{{ termsError }}</p>
        </div>
        <UiButton type="submit" :loading="submitting">Crear mi taller</UiButton>
        <p class="signup-switch">¿Ya tienes cuenta? <RouterLink to="/login" class="signup-link">Inicia sesión</RouterLink></p>
      </form>
    </template>

    <div v-else class="signup-sent" aria-live="polite">
      <div class="signup-sent__icon" aria-hidden="true"><Mail :size="24" /></div>
      <h1 id="auth-title">Revisa tu correo</h1>
      <p class="auth-subtitle">Te enviamos un enlace a <strong>{{ sentTo }}</strong> para activar tu taller. Vence en 24 horas.</p>
      <p class="signup-sent__notice">Si no lo ves en unos minutos, revisa la carpeta de spam o promociones.</p>
      <UiAlert v-if="resend.failed.value" :message="resend.failed.value" />
      <p v-if="resend.sent.value && !resend.failed.value" class="signup-sent__confirmation" role="status">Listo, te enviamos un enlace nuevo.</p>
      <UiButton type="button" variant="secondary" :loading="resend.sending.value" :disabled="resend.remaining.value > 0" @click="resend.resend(sentTo)">
        {{ resend.remaining.value > 0 ? `Reenviar en ${resend.remaining.value} s` : 'Reenviar el correo' }}
      </UiButton>
      <button type="button" class="signup-link signup-link--button" @click="editEmail">¿Escribiste mal tu correo? Corrígelo</button>
      <p class="signup-switch">¿Ya tenías cuenta con este correo? <RouterLink to="/login" class="signup-link">Inicia sesión</RouterLink></p>
    </div>
  </AuthLayout>
</template>

<style scoped>
.signup-terms {
  display: grid;
  gap: 0.25rem;
}

.signup-terms__label {
  display: flex;
  gap: 0.625rem;
  align-items: flex-start;
  min-height: 44px;
  color: var(--tf-ink);
  font-size: 0.875rem;
  line-height: 1.45;
  cursor: pointer;
}

.signup-terms__label input {
  flex: none;
  width: 1.25rem;
  height: 1.25rem;
  margin: 0.1rem 0 0;
  accent-color: var(--tf-primary);
}

.signup-terms__label input:focus-visible {
  outline: 3px solid var(--tf-focus);
  outline-offset: 2px;
}

.signup-link {
  color: var(--tf-primary);
  font-weight: 600;
  text-decoration: underline;
  text-underline-offset: 2px;
  border-radius: 0.25rem;
}

.signup-link:focus-visible {
  outline: 3px solid var(--tf-focus);
  outline-offset: 2px;
}

.signup-link--button {
  min-height: 44px;
  padding: 0 0.5rem;
  border: 0;
  background: none;
  font: inherit;
  font-size: 0.875rem;
  cursor: pointer;
}

.signup-switch {
  margin: 0;
  color: var(--tf-text-muted);
  font-size: 0.875rem;
  text-align: center;
}

.signup-sent {
  display: grid;
  gap: 0.75rem;
  text-align: center;
}

.signup-sent__icon {
  display: grid;
  width: 3.5rem;
  height: 3.5rem;
  margin: 0 auto;
  place-items: center;
  border-radius: 999px;
  background: var(--tf-primary-soft);
  color: var(--tf-primary);
}

.signup-sent__notice,
.signup-sent__confirmation {
  margin: 0;
  color: var(--tf-text-muted);
  font-size: 0.875rem;
}

.signup-sent__confirmation {
  color: var(--tf-ink);
  font-weight: 600;
}
</style>
