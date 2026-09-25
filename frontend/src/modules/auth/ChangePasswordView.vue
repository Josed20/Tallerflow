<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRouter } from 'vue-router'
import AuthLayout from '../../layouts/AuthLayout.vue'
import UiAlert from '../../components/UiAlert.vue'
import UiButton from '../../components/UiButton.vue'
import UiField from '../../components/UiField.vue'
import { useSessionStore } from '../session/session.store'

const router = useRouter()
const session = useSessionStore()
const currentPassword = ref('')
const newPassword = ref('')
const attempted = ref(false)
const submitting = ref(false)
const serverError = ref('')
const newPasswordError = computed(() => attempted.value && newPassword.value.length < 12 ? 'Usa al menos 12 caracteres.' : '')

async function submit() {
  attempted.value = true
  serverError.value = ''
  if (!currentPassword.value || newPasswordError.value) return
  submitting.value = true
  try {
    await session.changePassword(currentPassword.value, newPassword.value)
    await router.replace('/app')
  } catch {
    serverError.value = 'No pudimos actualizar la contraseña. Inténtalo nuevamente.'
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <AuthLayout>
    <h1 id="auth-title">Actualiza tu contraseña</h1>
    <p class="auth-subtitle">Por seguridad, necesitas crear una contraseña nueva antes de continuar.</p>
    <form class="auth-form" novalidate @submit.prevent="submit">
      <UiAlert v-if="serverError" :message="serverError" />
      <UiField id="current-password" v-model="currentPassword" label="Contraseña actual" type="password" autocomplete="current-password" :error="attempted && !currentPassword ? 'La contraseña actual es obligatoria.' : ''" />
      <UiField id="new-password" v-model="newPassword" label="Nueva contraseña" type="password" autocomplete="new-password" :error="newPasswordError" />
      <UiButton type="submit" :loading="submitting">Actualizar contraseña</UiButton>
    </form>
  </AuthLayout>
</template>
