<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
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

async function join() {
  loading.value = true
  error.value = ''
  try {
    await teamApi.consume(token.value, name.value, password.value)
    done.value = true
  } catch {
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

        <div v-else class="auth-form">
          <UiAlert message="Tu usuario fue creado. Ya puedes iniciar sesión." />
          <UiButton @click="router.replace('/login')">Ir al login</UiButton>
        </div>
      </div>
    </section>
  </main>
</template>
