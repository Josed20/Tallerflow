<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { api } from '../session/api'

const props = defineProps<{ csrf: string; includeDemo: boolean; revision?: number }>()
const emit = defineEmits<{ imported: [] }>()
type Day = { day: string; first_opens: number | null; registrations: number | null; logins: number | null; activities: number | null; active_users: number | null; active_workshops: number | null; play_device_installs: number | null; play_user_installs: number | null; play_uninstalls: number | null; play_imported_at: string | null; coverage_started_at: string }
type Totals = { active_users: number | null; active_workshops: number | null; coverage_complete: boolean }
const today = new Date().toISOString().slice(0, 10)
const start = ref(new Date(Date.now() - 29 * 86400000).toISOString().slice(0, 10))
const end = ref(today)
const channel = ref('')
const version = ref('')
const rows = ref<Day[]>([])
const totals = ref<Totals | null>(null)
const busy = ref(false)
const message = ref('')
const error = ref('')
const selected = ref<File | null>(null)
let request = 0
function value(n: number | null) { return n === null ? 'Sin datos' : n.toLocaleString('es-PE') }
async function load() {
  const current = ++request
  busy.value = true; error.value = ''
  try {
    const query = new URLSearchParams({ start: start.value, end: end.value, channel: channel.value, version: version.value, include_demo: String(props.includeDemo) })
    const result = await api.get<{ data: Day[]; totals: Totals }>(`/api/v1/console/analytics?${query}`)
    if (current === request) { rows.value = result.data; totals.value = result.totals }
  } catch { if (current === request) { error.value = 'No se pudieron cargar las métricas. Revisa las fechas e intenta nuevamente.'; rows.value = []; totals.value = null } }
  finally { if (current === request) busy.value = false }
}
async function importCSV() {
  const file = selected.value
  if (!file) return
  if (file.size > 4 * 1024 * 1024) { error.value = 'El límite es 4 MiB.'; return }
  busy.value = true; error.value = ''; message.value = ''
  try {
    const data = new Uint8Array(await file.arrayBuffer())
    const encoding = data[0] === 0xff && data[1] === 0xfe ? 'utf-16le' : data[0] === 0xfe && data[1] === 0xff ? 'utf-16be' : 'utf-8'
    const csv = new TextDecoder(encoding, { fatal: true }).decode(data)
    const result = await api.post<{ data: { rows: number } }>('/api/v1/console/analytics/play/import', { csv, source_name: file.name }, props.csrf)
    message.value = `${result.data.rows} filas importadas de Google Play. Reimportar actualiza las mismas fechas y versiones.`
    emit('imported'); await load()
  } catch { error.value = 'No se importó el reporte. Usa el CSV de instalaciones de pe.tallerflow.app, total o por App Version Code, UTF-8/UTF-16 y fechas de los últimos 25 meses.' }
  finally { busy.value = false }
}
watch(() => [props.includeDemo, props.revision], load, { immediate: true })
const eventTotals = computed(() => {
  const sum = (key: 'first_opens' | 'registrations') => {
    if (!totals.value?.coverage_complete || !rows.value.length || rows.value.some(row => row[key] === null)) return null
    return rows.value.reduce((total, row) => total + row[key]!, 0)
  }
  return { first_opens: sum('first_opens'), registrations: sum('registrations') }
})
</script>

<template>
  <section class="metrics" aria-labelledby="metrics-title">
    <h2 id="metrics-title">Actividad y adquisición</h2>
    <p>Fechas y límites diarios en UTC. Primera apertura observada ≠ descarga. Las aperturas anónimas incluyen instalaciones sin clasificación demo.</p>
    <form class="filters" @submit.prevent="load">
      <label>Desde<input v-model="start" type="date" :max="end" required /></label>
      <label>Hasta<input v-model="end" type="date" :min="start" :max="today" required /></label>
      <label>Canal<select v-model="channel"><option value="">Todos</option><option value="android">Android</option><option value="web">Web</option></select></label>
      <label>Versión<input v-model="version" maxlength="64" placeholder="Todas; Android: código 3" /></label>
      <button :disabled="busy">{{ busy ? 'Cargando…' : 'Aplicar filtros' }}</button>
    </form>
    <p>Android y Play usan el código de versión (p. ej. 3), distinto del nombre 1.1.0. Web usa el identificador de compilación o, por defecto, la versión del paquete (1.1.0). Los registros y logins requieren evidencia del servidor.</p>
    <p v-if="error" role="alert">{{ error }}</p><p v-if="message" role="status">{{ message }}</p>
    <div v-if="totals" class="totals"><article><strong>{{ value(eventTotals.first_opens) }}</strong><span>Primeras aperturas observadas en el período</span></article><article><strong>{{ value(eventTotals.registrations) }}</strong><span>Registros verificados en el período</span></article><article><strong>{{ value(totals.active_users) }}</strong><span>Usuarios únicos activos en todo el período</span></article><article><strong>{{ value(totals.active_workshops) }}</strong><span>Talleres únicos activos en todo el período</span></article></div>
    <p v-if="totals && !totals.coverage_complete">El rango incluye días sin cobertura completa de telemetría; los totales se muestran como «Sin datos». Los eventos crudos se conservan 90 días.</p>
    <div class="scroll"><table><caption>Eventos de uso y reporte Play por día (UTC)</caption><thead><tr><th>Fecha</th><th>Primeras aperturas</th><th>Registros verificados</th><th>Logins</th><th>Actividad</th><th>Usuarios activos diarios</th><th>Play: instalaciones dispositivos</th><th>Play: instalaciones usuarios</th><th>Play: desinstalaciones dispositivos</th><th>Play: importado UTC</th></tr></thead>
      <tbody><tr v-for="row in rows" :key="row.day"><th>{{ row.day.slice(0, 10) }}</th><td>{{ value(row.first_opens) }}</td><td>{{ value(row.registrations) }}</td><td>{{ value(row.logins) }}</td><td>{{ value(row.activities) }}</td><td>{{ value(row.active_users) }}</td><td>{{ value(row.play_device_installs) }}</td><td>{{ value(row.play_user_installs) }}</td><td>{{ value(row.play_uninstalls) }}</td><td>{{ row.play_imported_at ? row.play_imported_at.replace('T', ' ').slice(0, 19) : 'Sin reporte' }}</td></tr></tbody></table></div>
    <p>Fuente de uso: eventos del cliente; actividad exige interacción con sesión. Fuente de instalaciones: Google Play CSV, independiente de aperturas. Las cifras Play no permiten excluir talleres demo. Los datos faltantes no se convierten en cero.</p>
    <form class="import" @submit.prevent="importCSV"><h3>Importar instalaciones Google Play</h3><p>Solo administradores de plataforma. Reporte total o por App Version Code; paquete pe.tallerflow.app. Máximo 4 MiB / 10 000 filas. No mezclar dimensiones país/dispositivo.</p><label>Reporte CSV<input type="file" accept=".csv,text/csv" @change="selected = ($event.target as HTMLInputElement).files?.[0] || null" /></label><button :disabled="busy || !selected">Importar CSV</button><p>Importación automática: pendiente de acceso al bucket de reportes y credenciales autorizadas. No se solicitan credenciales aquí.</p></form>
  </section>
</template>

<style scoped>
.metrics{display:grid;gap:1rem}.metrics h2,.metrics h3,.metrics p{margin:0}.metrics p{font-size:.9rem;color:#535e6e}.filters{display:flex;flex-wrap:wrap;gap:1rem;align-items:end}label{display:grid;gap:.35rem;font-weight:600}input,select,button{min-height:44px;padding:.5rem;border:1px solid #b9c0ca;border-radius:6px;font:inherit}button{background:#087f6d;color:white;cursor:pointer}button:disabled{opacity:.6;cursor:wait}.scroll{overflow:auto}table{width:100%;border-collapse:collapse;font-size:.85rem}td,th{padding:.6rem;text-align:left;border-bottom:1px solid #dde1e7}caption{text-align:left;padding:.5rem 0}.totals{display:flex;flex-wrap:wrap;gap:1rem}.totals article{display:grid;gap:.4rem;background:#edf7f5;padding:1rem;border-radius:8px}.totals strong{font-size:1.5rem}.import{display:grid;gap:.8rem;border-top:1px solid #dde1e7;padding-top:1rem}.import button{justify-self:start}
</style>
