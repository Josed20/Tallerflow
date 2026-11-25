<script setup lang="ts">
import UiAlert from '../../components/UiAlert.vue'
import UiNotice from '../../components/UiNotice.vue'
import MetricTrend from '../../components/MetricTrend.vue'
import { computed, ref, watch } from 'vue'
import { api } from '../session/api'

const props = defineProps<{ csrf: string; includeDemo: boolean; revision?: number }>()
const emit = defineEmits<{ imported: [] }>()
type Day = { day: string; first_opens: number | null; registrations: number | null; logins: number | null; activities: number | null; active_users: number | null; active_workshops: number | null; play_device_installs: number | null; play_user_installs: number | null; play_uninstalls: number | null; play_imported_at: string | null; coverage_started_at: string }
type Totals = { active_users: number | null; active_workshops: number | null; coverage_complete: boolean }
const today = new Date().toISOString().slice(0, 10)
const periods = [7, 30, 90]
function periodStart(days: number) {
  const date = new Date(today)
  date.setUTCDate(date.getUTCDate() - days + 1)
  return date.toISOString().slice(0, 10)
}
const start = ref(periodStart(30))
function selectPeriod(days: number) {
  start.value = periodStart(days)
  end.value = today
  void load()
}
const end = ref(today)
const channel = ref('')
const version = ref('')
const rows = ref<Day[]>([])
const totals = ref<Totals | null>(null)
const busy = ref(false)
const message = ref('')
const error = ref('')
const selected = ref<File | null>(null)
const showDaily = ref(false)
const showImport = ref(false)
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
    <header class="metrics-heading">
      <h2 id="metrics-title">Actividad y adquisición</h2>
      <div class="periods" role="group" aria-label="Últimos días">
        <button v-for="days in periods" :key="days" type="button"
          :aria-pressed="start === periodStart(days) && end === today"
          :disabled="busy" @click="selectPeriod(days)">{{ days }} días</button>
      </div>
    </header>

    <form class="filters" @submit.prevent="load">
      <label>Desde<input v-model="start" type="date" :max="end" required /></label>
      <label>Hasta<input v-model="end" type="date" :min="start" :max="today" required /></label>
      <label>Canal<select v-model="channel"><option value="">Todos</option><option value="android">Android</option><option value="web">Web</option></select></label>
      <label>Versión<input v-model="version" maxlength="64" placeholder="Todas" /></label>
      <button :disabled="busy">{{ busy ? 'Cargando…' : 'Aplicar filtros' }}</button>
    </form>
    <UiAlert :message="error" /><UiNotice :message="message" />
    <div v-if="totals" class="totals"><article><strong>{{ value(eventTotals.first_opens) }}</strong><span>Primeras aperturas</span></article><article><strong>{{ value(eventTotals.registrations) }}</strong><span>Registros verificados</span></article><article><strong>{{ value(totals.active_users) }}</strong><span>Usuarios activos únicos</span></article><article><strong>{{ value(totals.active_workshops) }}</strong><span>Talleres activos únicos</span></article></div>
    <p v-if="totals && !totals.coverage_complete">El período incluye días sin cobertura completa. El gráfico muestra los días con registros disponibles; los totales incompletos se indican como «Sin datos».</p>
    <MetricTrend :rows="rows" />
    <details class="metrics-detail" @toggle="showDaily = ($event.target as HTMLDetailsElement).open"><summary>Ver detalle diario</summary><div v-if="showDaily" class="scroll"><table><caption>Eventos de uso y reporte Play por día (UTC)</caption><thead><tr><th>Fecha</th><th>Primeras aperturas</th><th>Registros verificados</th><th>Logins</th><th>Actividad</th><th>Usuarios activos diarios</th><th>Play: instalaciones dispositivos</th><th>Play: instalaciones usuarios</th><th>Play: desinstalaciones dispositivos</th><th>Play: importado UTC</th></tr></thead>
      <tbody><tr v-for="row in rows" :key="row.day"><th>{{ row.day.slice(0, 10) }}</th><td>{{ value(row.first_opens) }}</td><td>{{ value(row.registrations) }}</td><td>{{ value(row.logins) }}</td><td>{{ value(row.activities) }}</td><td>{{ value(row.active_users) }}</td><td>{{ value(row.play_device_installs) }}</td><td>{{ value(row.play_user_installs) }}</td><td>{{ value(row.play_uninstalls) }}</td><td>{{ row.play_imported_at ? row.play_imported_at.replace('T', ' ').slice(0, 19) : 'Sin reporte' }}</td></tr></tbody></table></div></details>
    <details class="metrics-detail"><summary>Cómo se calculan estas métricas</summary><div class="metric-help"><p>Las fechas se agrupan en UTC. Las primeras aperturas no equivalen a descargas de Google Play. Las instalaciones proceden de los reportes CSV oficiales.</p><p>Los usuarios y talleres activos se cuentan una sola vez en el período; no se suman los usuarios de cada día. Las aperturas anónimas pueden incluir demostraciones que aún no se pueden identificar.</p><p>Para filtrar por versión: Android y Play usan el código (p. ej. 3); la web usa la versión publicada (p. ej. 1.1.0). Los registros cuentan cuentas verificadas y los accesos cuentan inicios de sesión completados.</p><p>Los reportes oficiales de Play no permiten excluir talleres demo. Los días sin cobertura no se convierten en cero. Los eventos de uso se conservan 90 días.</p></div></details>
    <details class="metrics-detail" @toggle="showImport = ($event.target as HTMLDetailsElement).open"><summary>Importar reporte de Google Play</summary><form v-if="showImport" class="import" @submit.prevent="importCSV"><h3>Importar instalaciones Google Play</h3><p>Solo administradores de plataforma. Reporte total o por App Version Code; paquete pe.tallerflow.app. Máximo 4 MiB / 10 000 filas. No mezclar dimensiones país/dispositivo.</p><label>Reporte CSV<input type="file" accept=".csv,text/csv" @change="selected = ($event.target as HTMLInputElement).files?.[0] || null" /></label><button :disabled="busy || !selected">Importar CSV</button><p>La importación usa el archivo descargado de Google Play Console.</p></form></details>
  </section>
</template>

<style scoped>
.metrics { display: grid; gap: 1.25rem; color: var(--tf-ink); }
.metrics, .metrics > *, .filters label { min-width: 0; }
.metrics-heading, .periods { display: flex; align-items: center; flex-wrap: wrap; }
.metrics-heading { justify-content: space-between; gap: .75rem; }
.metrics h2, .metrics h3, .metrics p { margin: 0; }
.metrics h2 { font: 700 1.35rem var(--tf-font-body); letter-spacing: -.035em; }
.metrics p { color: var(--tf-text-subtle); font-size: .88rem; line-height: 1.55; }
.periods { gap: 2px; padding: 3px; background: #edf3f5; border-radius: 8px; }
button, input, select { min-height: 44px; box-sizing: border-box; border-radius: 6px; font: inherit; }
button { padding: .5rem .9rem; border: 0; color: white; background: #16594e; cursor: pointer; }
.periods button { background: transparent; color: #244753; }
.periods button[aria-pressed="true"] { background: #15394a; color: white; }
button:hover:enabled { text-decoration: underline; text-underline-offset: 3px; }
button:disabled { opacity: .65; }
:where(button, summary, input, select):focus-visible { outline: 3px solid var(--tf-focus); outline-offset: 2px; }
.filters { display: grid; grid-template-columns: repeat(4, minmax(0, 1fr)) auto; align-items: end; gap: .75rem; }
label { display: grid; gap: .35rem; font-size: .84rem; font-weight: 600; }
input, select { width: 100%; min-width: 0; padding: .5rem; border: 1px solid var(--tf-border); background: white; color: inherit; }
.totals { display: grid; grid-template-columns: repeat(4,minmax(0,1fr)); border-block: 1px solid #ccdedb; background: #eff7f5; }
.totals article { display: flex; flex-direction: column-reverse; justify-content: end; gap: .5rem; padding: 1rem; }
.totals strong { font-size: clamp(1.4rem, 2.6vw, 2.2rem); line-height: 1.1; color: #155447; font-variant-numeric: tabular-nums; }
.totals span { font-size: .85rem; color: #365953; }
.metrics-detail { border-top: 1px solid var(--tf-border-soft); }
summary { padding-block: .75rem; min-height: 44px; color: var(--tf-primary); font-weight: 600; cursor: pointer; }
.scroll { overflow: auto; max-height: 28rem; }
table { width: 100%; min-width: 1100px; border-collapse: collapse; font-size: .85rem; font-variant-numeric: tabular-nums; }
thead { position: sticky; top: 0; background: #eef3f5; }
td, th, caption { padding: .65rem; text-align: left; border-bottom: 1px solid var(--tf-border-soft); }
tbody th { white-space: nowrap; }
.import, .metric-help { display: grid; gap: .75rem; padding-block: .75rem; }
.import button { justify-self: start; }
@media (max-width: 800px) {
  .filters, .totals { grid-template-columns: repeat(2,minmax(0,1fr)); }
  .filters button { grid-column: 1/-1; }
}
</style>
