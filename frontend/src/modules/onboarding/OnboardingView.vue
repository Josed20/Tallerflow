<script setup lang="ts">
import { computed, nextTick, ref } from 'vue'
import { useRouter } from 'vue-router'
import AuthLayout from '../../layouts/AuthLayout.vue'
import UiAlert from '../../components/UiAlert.vue'
import UiButton from '../../components/UiButton.vue'
import UiField from '../../components/UiField.vue'
import { useOnboardingStore } from './onboarding.store'

const router = useRouter()
const onboarding = useOnboardingStore()
const workshopName = ref('')
const ownerName = ref('')
const email = ref('')
const password = ref('')
const passwordConfirmation = ref('')
const attempted = ref(false)
const submitting = ref(false)
const serverError = ref('')

const workshopNameError = computed(() => attempted.value && !workshopName.value.trim() ? 'Ingresa el nombre del taller.' : '')
const ownerNameError = computed(() => attempted.value && !ownerName.value.trim() ? 'Ingresa tu nombre.' : '')
const emailError = computed(() => attempted.value && !/^\S+@\S+\.\S+$/.test(email.value) ? 'Ingresa un correo válido.' : '')
const passwordError = computed(() => attempted.value && password.value.length < 12 ? 'Usa al menos 12 caracteres.' : '')
const confirmationError = computed(() => {
  if (!attempted.value) return ''
  if (!passwordConfirmation.value) return 'Confirma tu contraseña.'
  return passwordConfirmation.value !== password.value ? 'Las contraseñas no coinciden.' : ''
})

const errors = computed(() => [
  { id: 'workshop-name', message: workshopNameError.value },
  { id: 'owner-name', message: ownerNameError.value },
  { id: 'onboarding-email', message: emailError.value },
  { id: 'onboarding-password', message: passwordError.value },
  { id: 'password-confirmation', message: confirmationError.value },
])

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
    await onboarding.create({
      workshop_name: workshopName.value.trim(),
      owner_name: ownerName.value.trim(),
      email: email.value.trim(),
      password: password.value,
      password_confirmation: passwordConfirmation.value,
    })
    await router.replace('/app')
  } catch {
    if (onboarding.availability === 'claimed') {
      await router.replace('/login')
      return
    }
    serverError.value = 'No pudimos crear el taller. Inténtalo nuevamente en unos minutos.'
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <AuthLayout>
    <h1 id="auth-title">Crea tu taller en TallerFlow</h1>
    <p class="auth-subtitle">Registra el primer taller y su cuenta propietaria para empezar.</p>
    <form class="auth-form" novalidate @submit.prevent="submit">
      <UiAlert v-if="serverError" :message="serverError" />
      <UiField id="workshop-name" v-model="workshopName" label="Nombre del taller" autocomplete="organization" :error="workshopNameError" />
      <UiField id="owner-name" v-model="ownerName" label="Tu nombre" autocomplete="name" :error="ownerNameError" />
      <UiField id="onboarding-email" v-model="email" label="Correo electrónico" type="email" autocomplete="email" :error="emailError" />
      <UiField id="onboarding-password" v-model="password" label="Contraseña" type="password" autocomplete="new-password" description="Mínimo 12 caracteres." :error="passwordError" />
      <UiField id="password-confirmation" v-model="passwordConfirmation" label="Confirmar contraseña" type="password" autocomplete="new-password" :error="confirmationError" />
      <UiButton type="submit" :loading="submitting">Crear mi taller</UiButton>
    </form>
  </AuthLayout>
</template>
