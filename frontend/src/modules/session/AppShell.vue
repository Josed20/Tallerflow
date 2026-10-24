<script setup lang="ts">
import { computed } from 'vue'
import { useRouter } from 'vue-router'
import { Columns3, LayoutDashboard, LogOut, Plus, Users, UsersRound } from '@lucide/vue'
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
      <RouterLink class="app-shell__brand" :to="canManage ? '/app' : '/app/orders'" aria-label="TallerFlow, ir al inicio"><BrandLogo :size="26" tone="dark" decorative /></RouterLink>
      <!-- Etiqueta tejida del taller (DESIGN.md, Components). -->
      <p class="app-shell__workshop"><strong>{{ session.principal?.workshop.name }}</strong><small>Taller</small></p>
      <nav class="app-shell__nav" aria-label="Secciones internas">
        <RouterLink v-if="canManage" to="/app"><LayoutDashboard :size="18" aria-hidden="true" /> Panel</RouterLink>
        <RouterLink to="/app/orders"><Columns3 :size="18" aria-hidden="true" /> Órdenes</RouterLink>
        <RouterLink v-if="canManage" to="/app/clients"><Users :size="18" aria-hidden="true" /> Clientes</RouterLink>
        <RouterLink v-if="canManage" class="app-shell__nav-action" to="/app/orders/new"><Plus :size="18" aria-hidden="true" /> Nueva orden</RouterLink>
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
.app-shell__sidebar { position: relative; display: flex; flex-direction: column; gap: 1rem; padding: 1.2rem; color: var(--tf-on-shell-muted); background: var(--tf-shell); border-bottom: 3px solid var(--tf-shell-thread); }
.app-shell__brand { display: inline-flex; width: fit-content; min-height: 44px; align-items: center; color: var(--tf-on-shell); font-size: 1.05rem; text-decoration: none; }
.app-shell__workshop { position: relative; display: grid; gap: .15rem; margin: 0; padding: .6rem 1.1rem .55rem; border: 1px solid var(--tf-shell-line); border-bottom: 3px solid var(--tf-shell-thread); border-radius: var(--tf-radius-sm); color: var(--tf-on-shell); background: var(--tf-primary); }
/* Stitch lines at both ends, like a sewn-in garment label. */
.app-shell__workshop::before, .app-shell__workshop::after { content: ""; position: absolute; top: .4rem; bottom: .45rem; border-left: 2px dashed rgb(255 255 255 / 45%); }
.app-shell__workshop::before { left: .4rem; }
.app-shell__workshop::after { right: .4rem; }
.app-shell__workshop strong { font-family: var(--tf-font-display); font-size: .72rem; font-weight: 700; letter-spacing: .07em; line-height: 1.3; text-transform: uppercase; overflow-wrap: anywhere; }
.app-shell__workshop small { color: var(--tf-on-shell-muted); font-size: .72rem; }
.app-shell__nav { display: flex; flex-wrap: wrap; gap: .3rem; }
.app-shell__nav a, .app-shell__logout { min-height: 44px; display: inline-flex; align-items: center; gap: .7rem; border: 0; border-radius: var(--tf-radius); padding: .6rem .75rem; color: var(--tf-on-shell-muted); background: transparent; font: inherit; font-size: .85rem; font-weight: 700; text-decoration: none; cursor: pointer; }
.app-shell__nav a:hover, .app-shell__logout:hover { color: var(--tf-on-shell); background: var(--tf-shell-hover); }
.app-shell__nav a { position: relative; }
.app-shell__nav a.router-link-exact-active { color: var(--tf-on-shell); background: var(--tf-shell-raised); }
.app-shell__nav a.router-link-exact-active::before { content: ""; position: absolute; left: 0; top: .6rem; bottom: .6rem; width: 3px; border-radius: 2px; background: var(--tf-shell-thread); }
.app-shell__nav a:focus-visible, .app-shell__logout:focus-visible, .app-shell__brand:focus-visible { outline: 3px solid var(--tf-shell-accent); outline-offset: 2px; }
.app-shell__footer { display: flex; align-items: center; gap: 1rem; margin-left: auto; }
.app-shell__user { display: flex; align-items: center; gap: .6rem; min-width: 0; }
.app-shell__user > span:last-child { display: grid; min-width: 0; }
.app-shell__user strong { max-width: 12rem; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; color: var(--tf-on-shell); font-size: .79rem; }
.app-shell__user small { color: var(--tf-on-shell-muted); font-size: .75rem; }
.app-shell__avatar { display: grid; width: 2.1rem; height: 2.1rem; flex: 0 0 auto; place-items: center; border-radius: 50%; color: var(--tf-on-shell); background: var(--tf-shell-raised); font-size: .8rem; font-weight: 800; }
.app-shell__main, .app-shell__content { min-width: 0; }
.app-shell__header { border-bottom: 1px solid var(--tf-border-soft); background: var(--tf-surface); }
@media (min-width: 900px) { .app-shell { display: grid; grid-template-columns: 14.5rem minmax(0, 1fr); } .app-shell__sidebar { position: sticky; top: 0; height: 100vh; border-bottom: 0; border-right: 3px solid var(--tf-shell-thread); } .app-shell__nav { flex-direction: column; } .app-shell__footer { width: 100%; flex-direction: column; align-items: stretch; gap: .6rem; margin: auto 0 0; padding-top: 1rem; border-top: 1px solid var(--tf-shell-line); } .app-shell__logout { width: 100%; } }
@media (max-width: 899px) { .app-shell__sidebar { gap: .75rem; } .app-shell__workshop { order: 2; padding: .4rem 1rem .35rem; } .app-shell__workshop small { display: none; } .app-shell__nav a.router-link-exact-active::before { left: 25%; right: 25%; top: auto; bottom: 0; width: auto; height: 3px; } .app-shell__footer { margin-left: 0; } .app-shell__user { display: none; } .app-shell__nav { order: 3; display: flex; flex-wrap: nowrap; gap: .35rem; margin: 0 -1.2rem; padding: 0 1.2rem .1rem; overflow-x: auto; scrollbar-width: none; } .app-shell__nav::-webkit-scrollbar { display: none; } .app-shell__nav a { flex: 1 1 0; flex-direction: column; justify-content: center; min-width: 0; padding: .4rem .25rem; font-size: .75rem; gap: .15rem; white-space: nowrap; } .app-shell__nav a svg { flex: 0 0 auto; } .app-shell__nav .app-shell__nav-action { display: none; } .app-shell__logout { position: absolute; top: .7rem; right: 1rem; font-size: 0; } .app-shell__logout svg { width: 20px; height: 20px; } }
</style>
