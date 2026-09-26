<script setup lang="ts">
import { useRouter } from 'vue-router'
import UiButton from '../../components/UiButton.vue'
import { useSessionStore } from './session.store'

const session = useSessionStore()
const router = useRouter()

async function signOut() {
  await session.logout()
  await router.replace('/login')
}
</script>

<template>
  <main class="session-home">
    <section class="session-home__card">
      <p class="eyebrow">Taller activo</p>
      <h1>{{ session.principal?.workshop.name }}</h1>
      <p>{{ session.principal?.email }} · {{ session.principal?.role }}</p>
      <UiButton @click="signOut">Cerrar sesión</UiButton>
    </section>
  </main>
</template>
