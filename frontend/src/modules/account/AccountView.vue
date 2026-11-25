<script setup lang="ts">
import { computed, ref } from 'vue'
import { getAnalyticsDisabled, setAnalyticsDisabled } from '../../native/analytics'
import { useSessionStore } from '../session/session.store'
const session = useSessionStore()
const roleLabels: Record<string, string> = { OWNER: 'Propietario', ADMIN: 'Administrador', SUPERVISOR: 'Supervisor', OPERATOR: 'Operador' }
const role = computed(() => roleLabels[session.principal?.role ?? ''] ?? 'Miembro')
const analyticsEnabled = ref(false)
const preferenceError = ref('')
const preferenceNotice = ref('')
const browserOptOut = navigator.doNotTrack === '1' || Boolean((navigator as Navigator & { globalPrivacyControl?: boolean }).globalPrivacyControl)
try { analyticsEnabled.value = !getAnalyticsDisabled() && !browserOptOut }
catch { preferenceError.value = 'No se pudo leer la preferencia de este dispositivo.' }
function changeAnalytics(event: Event) {
  const input = event.target as HTMLInputElement
  preferenceError.value = ''
  preferenceNotice.value = ''
  try {
    setAnalyticsDisabled(!input.checked)
    analyticsEnabled.value = input.checked
    preferenceNotice.value = input.checked ? 'Estadísticas de uso activadas en este dispositivo.' : 'Estadísticas de uso desactivadas. Se borró la cola pendiente y se detuvo el envío.'
  } catch {
    input.checked = analyticsEnabled.value
    preferenceError.value = 'No se pudo guardar la preferencia. Revisa el almacenamiento del navegador e inténtalo nuevamente.'
  }
}
</script>

<template>
  <main class="account-page">
    <header><h1>Cuenta y privacidad</h1><p>Tu perfil, tu taller y el control de tus datos.</p></header>
    <section aria-labelledby="profile-title">
      <h2 id="profile-title">Mi cuenta</h2>
      <dl><div><dt>Nombre</dt><dd>{{ session.principal?.displayName || 'Sin nombre' }}</dd></div><div><dt>Correo</dt><dd>{{ session.principal?.email }}</dd></div><div><dt>Taller</dt><dd>{{ session.principal?.workshop.name }}</dd></div><div><dt>Rol</dt><dd>{{ role }}</dd></div></dl>
    </section>
    <section aria-labelledby="analytics-title">
      <h2 id="analytics-title">Estadísticas de uso</h2>
      <label class="analytics-choice"><input type="checkbox" :checked="analyticsEnabled" :disabled="browserOptOut" aria-describedby="analytics-help" @change="changeAnalytics" /><span>Compartir estadísticas de uso en este dispositivo</span></label>
      <p id="analytics-help">Nos ayudan a conocer el uso de TallerFlow. Al desactivarlas se borra la cola pendiente y se detiene el envío sin recargar la página. Incluyen primera apertura e interacciones con un identificador aleatorio. No se borran eventos ya recibidos; al reactivar se genera un identificador nuevo. Esta preferencia se aplica a este navegador o instalación.</p>
      <p v-if="browserOptOut">Tu navegador solicita no compartir estadísticas (DNT o GPC). Respetamos esa preferencia.</p>
      <p v-if="preferenceError" role="alert">{{ preferenceError }}</p>
      <p v-if="preferenceNotice" role="status">{{ preferenceNotice }}</p>
    </section>
    <section aria-labelledby="privacy-title">
      <h2 id="privacy-title">Privacidad y datos</h2>
      <p>Consulta cómo se usan tus datos o los pasos para solicitar la eliminación de tu cuenta.</p>
      <div class="account-links"><RouterLink to="/privacidad">Política de privacidad</RouterLink><RouterLink to="/eliminar-cuenta">Solicitar eliminación de cuenta</RouterLink></div>
      <p class="account-note">La solicitud se tramita por correo. Abrir las instrucciones no elimina tu cuenta ni el taller.</p>
    </section>
  </main>
</template>

<style scoped>
.account-page { display: grid; gap: 1.5rem; max-width: 64rem; margin: 0 auto; padding: clamp(1rem, 2.5vw, 2rem); color: var(--tf-ink); }
h1 { margin: 0; font-size: var(--tf-page-title-size); }
h2 { margin: 0 0 1rem; font-size: 1.125rem; }
p { color: var(--tf-text-subtle); max-width: 65ch; }
section { padding: 1.25rem; border: 1px solid var(--tf-border); border-radius: var(--tf-radius-sm); background: var(--tf-surface); }
dl { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 1rem; margin: 0; }
dt { color: var(--tf-text-subtle); font-size: .85rem; }
dd { margin: .25rem 0 0; overflow-wrap: anywhere; }
.account-links { display: flex; flex-wrap: wrap; gap: .5rem 1.5rem; }
a { display: inline-flex; align-items: center; min-height: 48px; color: var(--tf-primary); font-weight: 700; text-decoration-color: var(--tf-accent); text-underline-offset: 3px; }
a:focus-visible { outline: 3px solid var(--tf-focus); outline-offset: 2px; }
.analytics-choice { display: flex; align-items: center; gap: .75rem; min-height: 48px; cursor: pointer; }
.analytics-choice input { width: 1.25rem; height: 1.25rem; flex: 0 0 auto; accent-color: var(--tf-primary); }
.analytics-choice input:focus-visible { outline: 3px solid var(--tf-focus); outline-offset: 3px; }
[role="alert"] { color: var(--tf-error); }
.account-note { font-size: .85rem; }
@media (max-width: 560px) { dl { grid-template-columns: minmax(0, 1fr); } }
</style>
