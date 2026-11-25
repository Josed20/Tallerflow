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
    <h2 id="metrics-title">Actividad y adquisición</h2>
    <p>Uso del sistema e instalaciones de Google Play en el período seleccionado.</p>
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
.metrics h2 {font-family:var(--tf-font-body);font-variation-settings:normal;font-size:1.15rem;letter-spacing:-.025em;}
.metrics,.metrics>*{min-width:0}.metrics input,.metrics select{min-width:0;max-width:100%;box-sizing:border-box}.metrics .filters label{flex:1 1 140px;min-width:0}
.scroll{max-height:28rem;border:1px solid #dde1e7;border-radius:8px}.scroll table{min-width:1100px}.scroll thead{position:sticky;top:0;background:white;box-shadow:0 1px 0 #dde1e7}.scroll tbody th{white-space:nowrap;font-variant-numeric:tabular-nums}.metrics .totals{display:grid;grid-template-columns:repeat(auto-fit,minmax(min(100%,220px),1fr))}.totals article{min-width:0}
.metrics{display:grid;gap:1rem}.metrics h2,.metrics h3,.metrics p{margin:0}.metrics p{font-size:.9rem;color:#535e6e}.filters{display:flex;flex-wrap:wrap;gap:1rem;align-items:end}label{display:grid;gap:.35rem;font-weight:600}input,select,button{min-height:44px;padding:.5rem;border:1px solid #b9c0ca;border-radius:6px;font:inherit}button{background:#087f6d;color:white;cursor:pointer}button:disabled{opacity:.6;cursor:wait}.scroll{overflow:auto}table{width:100%;border-collapse:collapse;font-size:.85rem}td,th{padding:.6rem;text-align:left;border-bottom:1px solid #dde1e7}caption{text-align:left;padding:.5rem 0}.totals{display:flex;flex-wrap:wrap;gap:1rem}.totals article{display:grid;gap:.4rem;background:#edf7f5;padding:1rem;border-radius:8px}.totals strong{font-size:1.5rem}.import{display:grid;gap:.8rem;border-top:1px solid #dde1e7;padding-top:1rem}.import button{justify-self:start}

.metrics .totals{gap:.75rem}.metrics .totals article{padding:1.1rem;border-radius:14px;background:#f0f5fd;color:var(--tf-primary);border:1px solid #dce6f6}.metrics .totals article:nth-child(2){background:#eff8f4;color:#23735c;border-color:#d8eadd}.metrics .totals article:nth-child(3){background:#f5f0fa;color:#715496;border-color:#e6ddf0}.metrics .totals article:nth-child(4){background:#fff8eb;color:#8c6524;border-color:#eee1c5}.metrics .totals strong{font-size:1.7rem;line-height:1.2}.metrics .totals span{font-size:.8rem;line-height:1.5}.metrics-detail{border-top:1px solid var(--tf-border-soft);padding-top:1rem}.metrics-detail summary{min-height:44px;cursor:pointer;color:var(--tf-primary);font-size:.9rem;font-weight:600}.filters input,.filters select{border-radius:10px;background:white}.filters button{border-radius:10px;background:var(--tf-primary)}.metrics h3{font:700 1rem var(--tf-font-body)}

.metrics-detail{min-width:0;max-width:100%;overflow:hidden}.metrics-detail .scroll{min-width:0;max-width:100%;overflow:auto}.import{min-width:0}.import input[type="file"]{width:100%;max-width:100%}

.metric-help{display:grid;gap:.75rem;padding:.5rem 0}.metric-help p{font-size:.8rem;line-height:1.6}
</style>
