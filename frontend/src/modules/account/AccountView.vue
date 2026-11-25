<script setup lang="ts">
import UiAlert from '../../components/UiAlert.vue'
import UiNotice from '../../components/UiNotice.vue'
import { isNativeApp } from '../../native/platform'
import { computed, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ChevronRight, LogOut, Settings, UsersRound } from '@lucide/vue'
import { canManageOperations } from '../session/role-policy'
import { getAnalyticsDisabled, setAnalyticsDisabled } from '../../native/analytics'
import { useSessionStore } from '../session/session.store'
const session = useSessionStore()
const router = useRouter()
const native = isNativeApp()
const canManage = computed(() => canManageOperations(session.principal?.role))
const signingOut = ref(false)
const signOutError = ref('')
async function signOut() {
  signingOut.value = true
  signOutError.value = ''
  try {
    await session.logout()
    await router.replace('/login')
  } catch {
    if (!session.isAuthenticated) await router.replace('/login')
    else signOutError.value = 'No se pudo cerrar la sesión. Inténtalo de nuevo.'
  } finally {
    signingOut.value = false
  }
}
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
    <header><h1>Mi cuenta</h1><p>Tu perfil, tu taller y el control de tus datos.</p></header>
    <section aria-labelledby="profile-title">
      <h2 id="profile-title">Perfil</h2>
      <dl><div><dt>Nombre</dt><dd>{{ session.principal?.displayName || 'Sin nombre' }}</dd></div><div><dt>Correo</dt><dd>{{ session.principal?.email }}</dd></div><div><dt>Taller</dt><dd>{{ session.principal?.workshop.name }}</dd></div><div><dt>Rol</dt><dd>{{ role }}</dd></div></dl>
    </section>
    <section v-if="native && canManage" aria-labelledby="workshop-settings-title">
      <h2 id="workshop-settings-title">Administración del taller</h2>
      <div class="account-links">
        <RouterLink to="/app/team"><span><UsersRound :size="20" aria-hidden="true" /> Equipo</span><ChevronRight :size="18" aria-hidden="true" /></RouterLink>
        <RouterLink to="/app/settings"><span><Settings :size="20" aria-hidden="true" /> Ajustes del taller</span><ChevronRight :size="18" aria-hidden="true" /></RouterLink>
      </div>
    </section>
    <section v-if="!native" aria-labelledby="app-title">
      <h2 id="app-title">TallerFlow para Android</h2>
      <a href="https://play.google.com/store/apps/details?id=pe.tallerflow.app" target="_blank" rel="noopener noreferrer">Abrir en Google Play ↗</a>
    </section>
    <section aria-labelledby="analytics-title">
      <h2 id="analytics-title">Estadísticas de uso</h2>
      <label class="analytics-choice"><input type="checkbox" :checked="analyticsEnabled" :disabled="browserOptOut" aria-describedby="analytics-help" @change="changeAnalytics" /><span>Compartir estadísticas de uso en este dispositivo</span></label>
      <p id="analytics-help">Nos ayudan a conocer el uso de TallerFlow. Al desactivarlas se borra la cola pendiente y se detiene el envío sin recargar la página. Incluyen primera apertura e interacciones con un identificador aleatorio. No se borran eventos ya recibidos; al reactivar se genera un identificador nuevo. Esta preferencia se aplica a este navegador o instalación.</p>
      <p v-if="browserOptOut">Tu navegador solicita no compartir estadísticas (DNT o GPC). Respetamos esa preferencia.</p>
      <UiAlert :message="preferenceError" />
      <UiNotice :message="preferenceNotice" />
    </section>
    <section aria-labelledby="privacy-title">
      <h2 id="privacy-title">Privacidad y datos</h2>
      <p>Consulta cómo se usan tus datos o los pasos para solicitar la eliminación de tu cuenta.</p>
      <div class="account-links"><RouterLink to="/app/account/privacy">Política de privacidad <span aria-hidden="true">›</span></RouterLink><RouterLink to="/app/account/delete">Solicitar eliminación de cuenta <span aria-hidden="true">›</span></RouterLink></div>

    </section>
    <UiAlert :message="signOutError" />
    <button v-if="native" type="button" class="account-logout" :disabled="signingOut" @click="signOut">
      <LogOut :size="20" aria-hidden="true" /> {{ signingOut ? 'Cerrando sesión…' : 'Cerrar sesión' }}
    </button>
  </main>
</template>

<style scoped>
.account-document :deep(.legal-header),.account-document :deep(.legal-footer){display:none}.account-document :deep(.legal-page){min-height:0;background:transparent}.account-links button,.account-document>button{min-height:44px;padding:.6rem 1rem;border:1px solid var(--tf-border);border-radius:8px;background:var(--tf-surface);color:var(--tf-primary);font:inherit;cursor:pointer}
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

.account-document { margin-top: 1rem; padding-top: 1rem; border-top: 1px solid var(--tf-border-soft); }
.account-document :deep(.legal-container) { width: 100%; max-width: none; padding: 0; }
.account-document :deep(.legal-content) { padding-top: 1rem; }
.account-document :deep(.legal-article) { padding: 0; border: 0; background: transparent; box-shadow: none; }
.account-document :deep(.legal-article__header h1) { font-size: 1.5rem; }

.account-page{max-width:900px;gap:1.25rem}.account-page>section{border:1px solid var(--tf-border-soft);border-radius:16px;padding:1.5rem} .account-links{display:grid;gap:0}.account-links a{display:flex;justify-content:space-between;gap:1rem;min-height:58px;padding:.75rem 0;border-bottom:1px solid var(--tf-border-soft);color:var(--tf-primary);text-decoration:none;font-weight:600}.account-links a:last-child{border-bottom:0}.account-links a:hover{color:var(--tf-accent-text)}
.account-links a > span, .account-logout { display:flex; align-items:center; gap:.65rem; }
.account-logout { justify-content:center; min-height:48px; padding:.6rem 1rem; border:1px solid #d6aab3; border-radius:10px; background:white; color:#84243a; font:inherit; font-weight:700; cursor:pointer; }
.account-logout:hover { background:#fff3f5; }
.account-logout:focus-visible { outline:3px solid var(--tf-focus); outline-offset:2px; }
</style>
