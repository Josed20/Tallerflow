<script setup lang="ts">
import { computed } from 'vue'
import { useRouter } from 'vue-router'
import { Columns3, Factory, LayoutDashboard, LogOut, Plus, Users, UsersRound } from '@lucide/vue'
import BrandLogo from '../../components/BrandLogo.vue'
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
      <RouterLink class="app-shell__brand" :to="canManage ? '/app' : '/app/orders'" aria-label="TallerFlow, ir al inicio"><BrandLogo :size="30" /></RouterLink>
      <div class="app-shell__workshop"><Factory :size="16" aria-hidden="true" /><span>{{ session.principal?.workshop.name }}</span></div>
      <nav class="app-shell__nav" aria-label="Secciones internas">
        <RouterLink v-if="canManage" to="/app"><LayoutDashboard :size="18" aria-hidden="true" /> Panel</RouterLink>
        <RouterLink to="/app/orders"><Columns3 :size="18" aria-hidden="true" /> Órdenes</RouterLink>
        <RouterLink v-if="canManage" to="/app/clients"><Users :size="18" aria-hidden="true" /> Clientes</RouterLink>
        <RouterLink v-if="canManage" to="/app/orders/new"><Plus :size="18" aria-hidden="true" /> Nueva orden</RouterLink>
        <RouterLink v-if="canManage" to="/app/team"><UsersRound :size="18" aria-hidden="true" /> Equipo</RouterLink>
      </nav>
      <div class="app-shell__footer">
        <div class="app-shell__user"><span class="app-shell__avatar" aria-hidden="true">{{ (session.principal?.displayName || session.principal?.email || 'U').charAt(0).toUpperCase() }}</span><span><strong>{{ session.principal?.displayName || session.principal?.email }}</strong><small>{{ session.principal?.role === 'OWNER' ? 'Propietario' : session.principal?.role === 'ADMIN' ? 'Administrador' : 'Operador' }}</small></span></div>
        <button class="app-shell__logout" type="button" title="Cerrar sesión" @click="signOut"><LogOut :size="17" aria-hidden="true" /> Cerrar sesión</button>
      </div>
    </aside>
    <div class="app-shell__main"><header v-if="$slots.header" class="app-shell__header"><slot name="header" /></header><div class="app-shell__content"><slot /></div></div>
  </div>
</template>

<style scoped>
.app-shell { min-height: 100vh; background: var(--tf-page); color: var(--tf-ink); }
.app-shell__sidebar { display: flex; flex-direction: column; gap: 1rem; padding: 1.2rem; color: var(--tf-on-shell-muted); background: var(--tf-shell); }
.app-shell__brand { display: inline-flex; width: fit-content; align-items: center; color: var(--tf-on-shell); font-size: 1.05rem; text-decoration: none; }
.app-shell__workshop { display: flex; align-items: center; gap: .5rem; padding: .8rem; border: 1px solid var(--tf-shell-line); border-radius: var(--tf-radius); color: var(--tf-on-shell); font-size: .8rem; font-weight: 700; overflow-wrap: anywhere; }
.app-shell__nav { display: flex; flex-wrap: wrap; gap: .3rem; }
.app-shell__nav a, .app-shell__logout { min-height: 44px; display: inline-flex; align-items: center; gap: .7rem; border: 0; border-radius: var(--tf-radius); padding: .6rem .75rem; color: var(--tf-on-shell-muted); background: transparent; font: inherit; font-size: .85rem; font-weight: 700; text-decoration: none; cursor: pointer; }
.app-shell__nav a:hover, .app-shell__logout:hover { color: var(--tf-on-shell); background: var(--tf-shell-hover); }
.app-shell__nav a.router-link-exact-active { color: var(--tf-on-shell); background: var(--tf-shell-raised); }
.app-shell__nav a:focus-visible, .app-shell__logout:focus-visible, .app-shell__brand:focus-visible { outline: 3px solid var(--tf-shell-accent); outline-offset: 2px; }
.app-shell__footer { display: flex; align-items: center; gap: 1rem; margin-left: auto; }
.app-shell__user { display: flex; align-items: center; gap: .6rem; min-width: 0; }
.app-shell__user > span:last-child { display: grid; min-width: 0; }
.app-shell__user strong { max-width: 12rem; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; color: var(--tf-on-shell); font-size: .79rem; }
.app-shell__user small { color: var(--tf-on-shell-muted); font-size: .75rem; }
.app-shell__avatar { display: grid; width: 2.1rem; height: 2.1rem; flex: 0 0 auto; place-items: center; border-radius: 50%; color: var(--tf-on-shell); background: var(--tf-shell-raised); font-size: .8rem; font-weight: 800; }
.app-shell__main, .app-shell__content { min-width: 0; }
.app-shell__header { border-bottom: 1px solid var(--tf-border-soft); background: var(--tf-surface); }
@media (min-width: 900px) { .app-shell { display: grid; grid-template-columns: 14.5rem minmax(0, 1fr); } .app-shell__sidebar { position: sticky; top: 0; height: 100vh; } .app-shell__nav { flex-direction: column; } .app-shell__footer { width: 100%; flex-direction: column; align-items: stretch; gap: .6rem; margin: auto 0 0; padding-top: 1rem; border-top: 1px solid var(--tf-shell-line); } .app-shell__logout { width: 100%; } }
@media (max-width: 899px) { .app-shell__sidebar { gap: .75rem; } .app-shell__workshop { display: none; } .app-shell__footer { margin-left: 0; } .app-shell__user { display: none; } .app-shell__nav { order: 3; display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); } .app-shell__nav a { min-width: 0; padding: .55rem .45rem; font-size: .78rem; gap: .4rem; } .app-shell__nav a svg { flex: 0 0 auto; } .app-shell__logout { position: absolute; top: .7rem; right: 1rem; font-size: 0; } .app-shell__logout svg { width: 20px; height: 20px; } }
@media (max-width: 480px) { .app-shell__nav { grid-template-columns: repeat(2, minmax(0, 1fr)); } }
</style>
