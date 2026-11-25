<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { ArrowRight, Check, ExternalLink } from '@lucide/vue'
import { clientsApi } from '../clients/api'
import { teamApi } from '../team/api'
import { isNativeApp } from '../../native/platform'

const props = defineProps<{ workshopId: string; hasOrders: boolean }>()
const native = isNativeApp()
const ready = ref(false)
const hasClients = ref(false)
const hasTeam = ref(false)
const hidden = ref(false)
const storeOpened = ref(false)
const steps = computed(() => [
  { label: 'Registrar un cliente', to: '/app/clients', done: hasClients.value },
  { label: 'Crear una orden', to: '/app/orders/new', done: props.hasOrders },
  { label: 'Invitar al equipo', to: '/app/team', done: hasTeam.value },
])
const total = native ? 3 : 4
const completed = computed(() => steps.value.filter(step => step.done).length + Number(!native && storeOpened.value))

function readFlag(name: string) {
  try { return localStorage.getItem(`tallerflow_${name}_${props.workshopId}`) === 'true' } catch { return false }
}
function saveFlag(name: string) {
  try { localStorage.setItem(`tallerflow_${name}_${props.workshopId}`, 'true') } catch { /* Storage may be disabled. */ }
}
function dismiss() {
  hidden.value = true
  saveFlag('first_steps_dismissed')
}
function visitStore() {
  storeOpened.value = true
  saveFlag('play_store_opened')
}
onMounted(async () => {
  hidden.value = readFlag('first_steps_dismissed')
  if (hidden.value) return
  storeOpened.value = readFlag('play_store_opened')
  const [clients, team] = await Promise.allSettled([clientsApi.list(), teamApi.list()])
  if (clients.status !== 'fulfilled' || team.status !== 'fulfilled') return
  hasClients.value = Boolean(clients.value.data?.length)
  const roster = team.value.data
  hasTeam.value = (roster?.members?.length ?? 0) + (roster?.invitations?.length ?? 0) > 1
  ready.value = true
})
</script>

<template>
  <section v-if="ready && !hidden && completed < total" class="setup-guide" aria-labelledby="setup-title">
    <header>
      <div><h2 id="setup-title">Primeros pasos en tu taller</h2><span>{{ completed }} de {{ total }} completados</span></div>
      <button type="button" @click="dismiss">Ocultar</button>
    </header>
    <ul>
      <li v-for="(step, index) in steps" :key="step.to">
        <RouterLink :to="step.to">
          <span class="marker" :class="{ done: step.done }"><Check v-if="step.done" :size="16" aria-label="Completado"/><template v-else>{{ index + 1 }}</template></span>
          {{ step.label }} <ArrowRight :size="16" aria-hidden="true"/>
        </RouterLink>
      </li>
      <li v-if="!native"><a href="https://play.google.com/store/apps/details?id=pe.tallerflow.app" target="_blank" rel="noopener noreferrer" @click="visitStore">
        <span class="marker" :class="{ done: storeOpened }"><Check v-if="storeOpened" :size="16" aria-label="Completado"/><template v-else>4</template></span>
        Ver app en Google Play <ExternalLink :size="16" aria-label="Abre otra pestaña"/>
      </a></li>
    </ul>
  </section>
</template>

<style scoped>
.setup-guide { background:white; border-radius:12px; padding:clamp(1rem,2vw,1.5rem); box-shadow:inset 4px 0 #087e76; color:#172f4c }
header, header div, a { display:flex; align-items:center; gap:.75rem }
header { justify-content:space-between }
header div { flex-wrap:wrap; gap:.45rem 1rem }
h2 { font-size:1.1rem; margin:0 }
header span { font-size:.85rem; color:#53657a }
button { background:none; border:0; color:#315374; min-height:44px; cursor:pointer; font:inherit }
ul { display:grid; grid-template-columns:repeat(auto-fit,minmax(min(100%,200px),1fr)); list-style:none; padding:0; margin:.6rem 0 0; gap:.25rem .8rem }
a { padding:.6rem .3rem; min-height:44px; font-weight:600; font-size:.9rem; color:inherit; text-decoration:none; border-top:1px solid #d6e1e8 }
a:hover,button:hover { color:#006a64; text-decoration:underline }
a>svg { margin-left:auto }
.marker { flex:none; display:grid; place-items:center; width:27px; height:27px; border-radius:50%; background:#e8edf2; font-size:.8rem }
.done { background:#087064; color:white }
</style>
