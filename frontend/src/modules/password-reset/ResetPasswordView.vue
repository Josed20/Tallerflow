<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { RouterLink, useRoute, useRouter } from 'vue-router'
import AuthLayout from '../../layouts/AuthLayout.vue'
import UiAlert from '../../components/UiAlert.vue'
import UiButton from '../../components/UiButton.vue'
import UiField from '../../components/UiField.vue'
import { api, ApiError } from '../session/api'

type ViewState = 'ready' | 'invalid' | 'expired' | 'success'

const route = useRoute()
const router = useRouter()

const rawToken = computed(() => {
  const q = route.query.token
  return (typeof q === 'string' ? q.trim() : '')
})

const state = ref<ViewState>(rawToken.value ? 'ready' : 'invalid')
const newPassword = ref('')
const confirmPassword = ref('')
const attempted = ref(false)
const submitting = ref(false)
const serverError = ref('')

onMounted(() => {
  if (!rawToken.value) {
    state.value = 'invalid'
  }
})

const newPasswordError = computed(() => {
  if (!attempted.value) return ''
  if (!newPassword.value) return 'La nueva contraseña es obligatoria.'
  if (newPassword.value.length < 12) return 'Usa al menos 12 caracteres.'
  return ''
})

const confirmPasswordError = computed(() => {
  if (!attempted.value) return ''
  if (!confirmPassword.value) return 'Confirma tu nueva contraseña.'
  if (newPassword.value !== confirmPassword.value) return 'Las contraseñas no coinciden.'
  return ''
})

async function submit() {
  if (submitting.value) return
  attempted.value = true
  serverError.value = ''

  if (newPasswordError.value || confirmPasswordError.value) return
  if (!rawToken.value) {
    state.value = 'invalid'
    return
  }

  submitting.value = true
  try {
    await api.post('/api/v1/auth/password-resets/consume', {
      token: rawToken.value,
      new_password: newPassword.value,
    })
    state.value = 'success'
  } catch (err: unknown) {
    if (err instanceof ApiError) {
      if (err.status === 400) {
        // Can be expired or invalid
        serverError.value = 'El enlace de recuperación es inválido o ha expirado.'
      } else if (err.status === 429) {
        serverError.value = 'Demasiados intentos. Por favor espera unos minutos antes de reintentar.'
      } else {
        serverError.value = 'No pudimos actualizar la contraseña. Inténtalo nuevamente.'
      }
    } else {
      serverError.value = 'No pudimos conectar con el servidor. Inténtalo más tarde.'
    }
  } finally {
    submitting.value = false
  }
}

function goToLogin() {
  router.push('/login')
}
</script>

<template>
  <AuthLayout>
    <!-- SUCCESS STATE -->
    <template v-if="state === 'success'">
      <div class="reset-result" aria-live="polite">
        <div class="reset-result__icon reset-result__icon--success" aria-hidden="true">✓</div>
        <h1 id="auth-title">Contraseña restablecida</h1>
        <p class="auth-subtitle">
          Tu contraseña ha sido actualizada exitosamente. Por seguridad, tus sesiones anteriores han sido cerradas.
        </p>
        <UiButton type="button" @click="goToLogin">
          Iniciar sesión
        </UiButton>
      </div>
    </template>

    <!-- INVALID TOKEN STATE -->
    <template v-else-if="state === 'invalid'">
      <div class="reset-result" aria-live="assertive">
        <div class="reset-result__icon reset-result__icon--error" aria-hidden="true">!</div>
        <h1 id="auth-title">Enlace inválido</h1>
        <p class="auth-subtitle">
          El enlace de recuperación es inválido, incompleto o ya ha sido utilizado.
        </p>
        <RouterLink to="/forgot-password" class="secondary-button">
          Solicitar un nuevo enlace
        </RouterLink>
      </div>
    </template>

    <!-- EXPIRED TOKEN STATE -->
    <template v-else-if="state === 'expired'">
      <div class="reset-result" aria-live="assertive">
        <div class="reset-result__icon reset-result__icon--error" aria-hidden="true">⏱</div>
        <h1 id="auth-title">Enlace expirado</h1>
        <p class="auth-subtitle">
          El enlace de recuperación ha superado el tiempo máximo de validez (1 hora).
        </p>
        <RouterLink to="/forgot-password" class="secondary-button">
          Solicitar un nuevo enlace
        </RouterLink>
      </div>
    </template>

    <!-- FORM STATE (READY) -->
    <template v-else>
      <h1 id="auth-title">Crea tu nueva contraseña</h1>
      <p class="auth-subtitle">Crea una contraseña segura de al menos 12 caracteres para tu cuenta.</p>

      <form class="auth-form" novalidate @submit.prevent="submit">
        <UiAlert v-if="serverError" :message="serverError" />

        <UiField
          id="new-password"
          v-model="newPassword"
          label="Nueva contraseña"
          type="password"
          autocomplete="new-password"
          placeholder="Mínimo 12 caracteres"
          :error="newPasswordError"
        />

        <UiField
          id="confirm-password"
          v-model="confirmPassword"
          label="Confirmar nueva contraseña"
          type="password"
          autocomplete="new-password"
          placeholder="Repite tu nueva contraseña"
          :error="confirmPasswordError"
        />

        <UiButton type="submit" :loading="submitting">
          Restablecer contraseña
        </UiButton>

        <div class="auth-back-link">
          <RouterLink to="/login" class="secondary-link">
            ← Volver a Iniciar sesión
          </RouterLink>
        </div>
      </form>
    </template>
  </AuthLayout>
</template>

<style scoped>
.auth-back-link {
  margin-top: 1rem;
  text-align: center;
}

.secondary-link {
  display: inline-flex;
  min-height: 44px;
  align-items: center;
  justify-content: center;
  color: var(--tf-primary, #00685f);
  font-size: 0.875rem;
  font-weight: 600;
  text-decoration: none;
  border-radius: 0.375rem;
  padding: 0.25rem 0.5rem;
}

.secondary-link:hover {
  text-decoration: underline;
}

.secondary-link:focus-visible {
  outline: 3px solid #89f5e7;
  outline-offset: 2px;
}

.secondary-button {
  display: inline-flex;
  min-height: 44px;
  width: 100%;
  align-items: center;
  justify-content: center;
  color: #fff;
  background: var(--tf-primary, #00685f);
  font-size: 0.9375rem;
  font-weight: 700;
  text-decoration: none;
  border-radius: 0.5rem;
  padding: 0.625rem 1rem;
  box-shadow: 0 1px 2px rgb(11 28 48 / 12%);
}

.secondary-button:hover {
  background: var(--tf-primary-hover, #008378);
}

.secondary-button:focus-visible {
  outline: 3px solid #89f5e7;
  outline-offset: 2px;
}

.reset-result {
  display: grid;
  gap: 0.75rem;
  text-align: center;
}

.reset-result__icon {
  display: grid;
  width: 3.5rem;
  height: 3.5rem;
  margin: 0 auto 0.5rem;
  place-items: center;
  border-radius: 50%;
  font-size: 1.5rem;
}

.reset-result__icon--success {
  background: #d1f4e0;
  color: #0b7a42;
}

.reset-result__icon--error {
  background: var(--tf-error-bg, #ffdad6);
  color: var(--tf-error, #ba1a1a);
}
</style>
