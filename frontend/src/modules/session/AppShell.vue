<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { Columns3, Settings, LayoutDashboard, LogOut, Plus, Users, UsersRound, PanelLeftClose, PanelLeftOpen, UserRound } from '@lucide/vue'
import { isNativeApp } from '../../native/platform'
import BrandLogo from '../../components/BrandLogo.vue'
import { useSessionStore } from './session.store'
import { canManageOperations } from './role-policy'

const session = useSessionStore()
const router = useRouter()
const route = useRoute()
const native = isNativeApp()
const collapsed = ref(false)
const canManage = computed(() => canManageOperations(session.principal?.role))

async function signOut() {
  await session.logout()
  await router.replace('/login')
}
</script>

<template>
  <div class="app-shell" :class="{ 'app-shell--collapsed': collapsed, 'app-shell--android': native }">
    <header v-if="native" class="android-app-bar">
      <strong>{{ session.principal?.workshop.name }}</strong>
    </header>
    <nav v-if="native" class="android-app-nav" aria-label="Navegación de Android">
      <RouterLink v-if="canManage" to="/app"><LayoutDashboard :size="22" />Panel</RouterLink>
      <RouterLink to="/app/orders" :class="{ 'android-active': route.path.startsWith('/app/orders/') && route.path !== '/app/orders/new' }"><Columns3 :size="22" />Órdenes</RouterLink>
      <RouterLink v-if="canManage" to="/app/orders/new"><Plus :size="22" />Nueva</RouterLink>
      <RouterLink v-if="canManage" to="/app/clients"><Users :size="22" />Clientes</RouterLink>
      <RouterLink to="/app/account" :class="{ 'android-active': route.path.startsWith('/app/account/') || route.name === 'team' || route.name === 'settings' }"><UserRound :size="22" />Cuenta</RouterLink>
    </nav>
    <aside class="app-shell__sidebar" aria-label="Navegación principal">
      <RouterLink class="app-shell__brand" :to="canManage ? '/app' : '/app/orders'" aria-label="TallerFlow, ir al inicio"><BrandLogo :size="26" tone="dark" decorative /></RouterLink>
      <!-- Etiqueta tejida del taller (DESIGN.md, Components). -->
      <button class="sidebar-toggle" type="button" :aria-label="collapsed ? 'Expandir barra lateral' : 'Plegar barra lateral'" :aria-expanded="!collapsed" aria-controls="workshop-nav" @click="collapsed = !collapsed">
        <PanelLeftOpen v-if="collapsed" :size="20" />
        <PanelLeftClose v-else :size="20" />
      </button>
      <p class="app-shell__workshop"><strong>{{ session.principal?.workshop.name }}</strong><small>Taller</small></p>
      <nav id="workshop-nav" class="app-shell__nav" aria-label="Secciones internas">
        <RouterLink v-if="canManage" to="/app"><LayoutDashboard :size="18" aria-hidden="true" /> <span class="sidebar-label">Panel</span></RouterLink>
        <RouterLink to="/app/orders"><Columns3 :size="18" aria-hidden="true" /> <span class="sidebar-label">Órdenes</span></RouterLink>
        <RouterLink v-if="canManage" to="/app/clients"><Users :size="18" aria-hidden="true" /> <span class="sidebar-label">Clientes</span></RouterLink>
        <RouterLink v-if="canManage" class="app-shell__nav-action" to="/app/orders/new"><Plus :size="18" aria-hidden="true" /> <span class="sidebar-label">Nueva orden</span></RouterLink>
        <RouterLink v-if="canManage" to="/app/team"><UsersRound :size="18" aria-hidden="true" /> <span class="sidebar-label">Equipo</span></RouterLink>
        <RouterLink v-if="canManage" to="/app/settings"><Settings :size="18" aria-hidden="true" /> <span class="sidebar-label">Ajustes</span></RouterLink>
      </nav>
      <div class="app-shell__footer">
<RouterLink to="/app/account" class="app-shell__user" aria-label="Mi cuenta" title="Mi cuenta"><span class="app-shell__avatar" aria-hidden="true">{{ (session.principal?.displayName || session.principal?.email || 'U').charAt(0).toUpperCase() }}</span><span><strong>{{ session.principal?.displayName || session.principal?.email }}</strong><small>{{ session.principal?.role === 'OWNER' ? 'Propietario' : session.principal?.role === 'ADMIN' ? 'Administrador' : 'Operador' }}</small></span></RouterLink>
        <button class="app-shell__logout" type="button" title="Cerrar sesión" @click="signOut"><LogOut :size="17" aria-hidden="true" /> <span class="sidebar-label">Cerrar sesión</span></button>
      </div>
    </aside>
    <div class="app-shell__main"><header v-if="$slots.header" class="app-shell__header"><slot name="header" /></header><div class="app-shell__content"><slot /></div></div>
  </div>
</template>

<style scoped>
.app-shell__content :deep(h1),.app-shell__content :deep(h2),.app-shell__content :deep(h3){font-family:var(--tf-font-body);font-variation-settings:normal;letter-spacing:-.025em}
.app-shell__content :deep(h1){font-weight:700}.app-shell__content :deep(h2){font-weight:650}
.app-shell .app-shell__workshop{background:transparent;border:0;border-bottom:1px solid var(--tf-shell-line);border-radius:0;padding:.75rem .25rem}.app-shell .app-shell__workshop::before,.app-shell .app-shell__workshop::after{display:none}.app-shell .app-shell__workshop strong{text-transform:none;letter-spacing:normal;font-family:var(--tf-font-body);font-size:.9rem}
.app-shell__content :deep(.clients-notice),.app-shell__content :deep(.detail-notice){position:fixed;top:1.25rem;right:1.25rem;width:min(420px,calc(100vw - 2.5rem));z-index:300;box-shadow:0 8px 30px rgb(15 35 60 / 16%);border-radius:12px;padding:1rem}

.app-shell .app-shell__user{display:flex;text-decoration:none;min-height:52px;padding:.5rem;border-radius:8px}.app-shell__user:hover{background:var(--tf-shell-hover)}
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

.app-shell__user:focus-visible { outline: 3px solid var(--tf-shell-accent); outline-offset: 2px; }
.app-shell__user.router-link-active { background: var(--tf-shell-raised); }
@media(max-width:899px) {
  .app-shell__sidebar { display: grid; grid-template-columns: minmax(0, 1fr) auto; gap: .5rem; padding: .75rem 1rem; }
  .app-shell__brand { max-width: 100%; }
  .app-shell__footer { grid-column: 2; grid-row: 1; gap: .25rem; }
  .app-shell .app-shell__user { padding: .25rem; min-height: 44px; }
  .app-shell__user > span:last-child { display: none; }
  .app-shell__logout { position: static; padding: .5rem; }
  .app-shell__workshop, .app-shell__nav { grid-column: 1 / -1; }
  .app-shell .app-shell__workshop { padding: .35rem 0 .6rem; }
  .app-shell__nav { margin: 0; padding: 0; }
}

.sidebar-toggle, .android-app-bar, .android-app-nav { display: none; }
@media (min-width: 900px) {
  .sidebar-toggle {
    display: grid; place-items: center; position: absolute; top: 1.3rem; right: .45rem;
    width: 34px; height: 40px; color: var(--tf-on-shell);
    border: 0; border-radius: 6px; background: var(--tf-shell-hover); cursor: pointer;
  }
  .app-shell--collapsed { grid-template-columns: 4.5rem minmax(0, 1fr); }
  .app-shell--collapsed .app-shell__sidebar { padding-inline: .55rem; }
  .app-shell--collapsed .sidebar-toggle { position: static; width: 100%; order: -1; }
  .app-shell--collapsed .app-shell__brand, .app-shell--collapsed .app-shell__workshop,
  .app-shell--collapsed .app-shell__user > span:last-child { display: none; }
  .app-shell--collapsed .sidebar-label {
    position: absolute; width: 1px; height: 1px; overflow: hidden; clip-path: inset(50%); white-space: nowrap;
  }
  .app-shell--collapsed .app-shell__nav a, .app-shell--collapsed .app-shell__logout,
  .app-shell--collapsed .app-shell__user { justify-content: center; }
}
@media (max-width: 899px) {
  .app-shell__logout .sidebar-label { font-size: 0; }
}
.app-shell--android { display: block; padding-bottom: calc(72px + env(safe-area-inset-bottom)); }
.app-shell--android .app-shell__sidebar { display: none; }
.android-app-bar {
  display: flex; position: sticky; top: 0; z-index: 35;
  align-items: center; justify-content: space-between; gap: 1rem;
  padding: max(.45rem, env(safe-area-inset-top)) 1rem .45rem;
  color: #fff; background: #0d3341; min-height: 60px;
}
.android-app-bar strong { min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.android-app-nav {
  position: fixed; bottom: 0; inset-inline: 0; z-index: 35; display: flex; justify-content: center;
  padding: .35rem .35rem max(.35rem,env(safe-area-inset-bottom));
  background: #fff; box-shadow: 0 -1px 0 #b9cdd2; color: #244852;
}
.android-app-nav a {
  flex: 1; max-width: 130px; display: grid; justify-items: center; gap: .25rem;
  padding: .35rem .15rem; min-height: 56px; font-size: .72rem; font-weight: 650;
  text-decoration: none; color: inherit;
}
.android-app-nav a:is(.router-link-exact-active,.android-active) { color: #086550; }
.android-app-nav a:is(.router-link-exact-active,.android-active) svg {
  background: #ceeadd; width: 46px; height: 26px; padding: 2px 11px; border-radius: 14px;
}
</style>
