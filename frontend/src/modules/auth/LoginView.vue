<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import AuthLayout from '../../layouts/AuthLayout.vue'
import UiAlert from '../../components/UiAlert.vue'
import UiButton from '../../components/UiButton.vue'
import UiField from '../../components/UiField.vue'
import { useSessionStore } from '../session/session.store'

const router = useRouter()
const route = useRoute()
const session = useSessionStore()
const email = ref(String(route.query.email ?? ''))
const password = ref('')
const attempted = ref(false)
const submitting = ref(false)
const serverError = ref('')

const emailError = computed(() => attempted.value && !/^\S+@\S+\.\S+$/.test(email.value) ? 'Ingresa un correo válido.' : '')
const passwordError = computed(() => attempted.value && !password.value ? 'La contraseña es obligatoria.' : '')

async function submit() {
  if (submitting.value) return
	attempted.value = true
  serverError.value = ''
  if (emailError.value || passwordError.value) return

  submitting.value = true
  try {
    await session.login(email.value, password.value)
    await router.replace(session.requiresPasswordChange ? '/change-password' : (route.query.redirect?.toString() ?? '/app'))
  } catch {
    serverError.value = 'No pudimos iniciar sesión. Revisa tus datos e inténtalo de nuevo.'
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
      <UiAlert v-if="serverError" :message="serverError" />
      <UiField id="email" v-model="email" label="Correo electrónico" type="email" autocomplete="email" placeholder="ejemplo@taller.pe" :error="emailError" />
      <UiField id="password" v-model="password" label="Contraseña" type="password" autocomplete="current-password" placeholder="••••••••" :error="passwordError" />
      <UiButton type="submit" :loading="submitting">Iniciar sesión</UiButton>
    </form>
  </AuthLayout>
</template>
