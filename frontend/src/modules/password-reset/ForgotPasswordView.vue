<script setup lang="ts">
import { computed, nextTick, ref } from 'vue'
import { RouterLink } from 'vue-router'
import AuthLayout from '../../layouts/AuthLayout.vue'
import UiAlert from '../../components/UiAlert.vue'
import UiButton from '../../components/UiButton.vue'
import UiField from '../../components/UiField.vue'
import { api } from '../session/api'

const email = ref('')
const attempted = ref(false)
const submitting = ref(false)
const serverError = ref('')
const isSent = ref(false)

const emailError = computed(() => {
  if (!attempted.value) return ''
  const trimmed = email.value.trim()
  if (!trimmed) return 'El correo electrónico es obligatorio.'
  if (!/^\S+@\S+\.\S+$/.test(trimmed)) return 'Ingresa un correo válido.'
  return ''
})

async function submit() {
  if (submitting.value) return
  attempted.value = true
  serverError.value = ''

  if (emailError.value) {
    await nextTick()
    document.getElementById('recovery-email')?.focus()
    return
  }

  submitting.value = true
  try {
    await api.post('/api/v1/auth/password-resets', { email: email.value.trim() })
    isSent.value = true
  } catch {
    serverError.value = 'No pudimos procesar tu solicitud. Por favor intenta de nuevo.'
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <AuthLayout>
    <template v-if="!isSent">
      <h1 id="auth-title">Recupera tu contraseña</h1>
      <p class="auth-subtitle">Ingresa tu correo registrado para enviarte un enlace de recuperación.</p>

      <form class="auth-form" novalidate @submit.prevent="submit">
        <UiAlert v-if="serverError" :message="serverError" />
        <UiField
          id="recovery-email"
          v-model="email"
          label="Correo electrónico"
          type="email"
          autocomplete="email"
          placeholder="ejemplo@taller.pe"
          :error="emailError"
        />
        <UiButton type="submit" :loading="submitting">
          Enviar enlace de recuperación
        </UiButton>
        <div class="auth-back-link">
          <RouterLink to="/login" class="secondary-link">
            ← Volver a Iniciar sesión
          </RouterLink>
        </div>
      </form>
    </template>

    <template v-else>
      <div class="recovery-sent" aria-live="polite">
        <div class="recovery-sent__icon" aria-hidden="true">✉</div>
        <h1 id="auth-title">Revisa tu correo</h1>
        <p class="auth-subtitle">
          Si la dirección ingresada está registrada en TallerFlow, recibirás un enlace seguro para restablecer tu contraseña.
        </p>
        <p class="recovery-sent__notice">
          El enlace expirará en 1 hora. Revisa tu bandeja de entrada y la carpeta de spam.
        </p>
        <div class="auth-back-link">
          <RouterLink to="/login" class="secondary-link secondary-link--button">
            Volver a Iniciar sesión
          </RouterLink>
        </div>
      </div>
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

.secondary-link--button {
  width: 100%;
  background: var(--tf-surface-soft, #eff4ff);
  border: 1px solid var(--tf-border, #bcc9c6);
  border-radius: 0.5rem;
}

.recovery-sent {
  display: grid;
  gap: 0.75rem;
  text-align: center;
}

.recovery-sent__icon {
  display: grid;
  width: 3.5rem;
  height: 3.5rem;
  margin: 0 auto 0.5rem;
  place-items: center;
  border-radius: 50%;
  background: var(--tf-surface-soft, #eff4ff);
  color: var(--tf-primary, #00685f);
  font-size: 1.5rem;
}

.recovery-sent__notice {
  font-size: 0.8125rem;
  color: var(--tf-text-muted, #3d4947);
  margin: 0 0 1rem;
}
</style>
