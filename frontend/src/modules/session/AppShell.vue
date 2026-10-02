<script setup lang="ts">
import { computed } from 'vue'
import { useRouter } from 'vue-router'
import { useSessionStore } from './session.store'
import { canManageOperations } from './role-policy'

const session = useSessionStore()
const router = useRouter()
const canManage = computed(() => canManageOperations(session.principal?.role))

async function signOut() {
  await session.logout()
  await router.replace('/login')
}
</script>

<template>
  <div class="app-shell">
    <aside class="app-shell__sidebar" aria-label="Navegación principal">
      <RouterLink class="app-shell__brand" to="/app">TallerFlow</RouterLink>
      <nav class="app-shell__nav" aria-label="Secciones internas">
        <RouterLink to="/app">Panel</RouterLink>
        <RouterLink to="/app/orders">Órdenes</RouterLink>
        <RouterLink v-if="canManage" to="/app/clients">Clientes</RouterLink>
        <RouterLink v-if="canManage" to="/app/orders/new">Nueva orden</RouterLink>
        <RouterLink v-if="canManage" to="/app/team">Equipo</RouterLink>
      </nav>
      <button class="app-shell__logout" type="button" @click="signOut">Cerrar sesión</button>
    </aside>

    <div class="app-shell__main">
      <header v-if="$slots.header" class="app-shell__header"><slot name="header" /></header>
      <main class="app-shell__content"><slot /></main>
    </div>
  </div>
</template>

<style scoped>
.app-shell { min-height: 100vh; background: var(--tf-page, #f7faf9); color: #0f172a; }
.app-shell__sidebar { display: flex; flex-direction: column; gap: 1.25rem; padding: 1.25rem; color: #e2e8f0; background: #0f172a; }
.app-shell__brand { color: #fff; font-size: 1.125rem; font-weight: 800; text-decoration: none; }
.app-shell__nav { display: flex; flex-wrap: wrap; gap: .5rem; }
.app-shell__nav a, .app-shell__logout { min-height: 44px; display: inline-flex; align-items: center; border: 0; border-radius: .5rem; padding: .625rem .75rem; color: #cbd5e1; background: transparent; font: inherit; font-weight: 700; text-decoration: none; cursor: pointer; }
.app-shell__nav a:hover, .app-shell__nav a.router-link-exact-active, .app-shell__logout:hover { color: #fff; background: rgb(13 148 136 / 30%); }
.app-shell__nav a:focus-visible, .app-shell__logout:focus-visible { outline: 3px solid #5eead4; outline-offset: 2px; }
.app-shell__logout { margin-left: auto; }
.app-shell__main { min-width: 0; }
.app-shell__header { border-bottom: 1px solid #e2e8f0; background: #fff; }
.app-shell__content { min-width: 0; }
@media (min-width: 860px) {
  .app-shell { display: grid; grid-template-columns: 15rem minmax(0, 1fr); }
  .app-shell__sidebar { position: sticky; top: 0; min-height: 100vh; align-items: stretch; }
  .app-shell__nav { flex-direction: column; }
  .app-shell__logout { margin-top: auto; margin-left: 0; }
}
</style>
