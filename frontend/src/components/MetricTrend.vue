<script setup lang="ts">
import { computed, ref } from 'vue'
const props = defineProps<{ rows: { day: string; active_users: number | null; first_opens: number | null; registrations: number | null; play_device_installs: number | null }[] }>()
type Metric = 'active_users' | 'first_opens' | 'registrations' | 'play_device_installs'
const metric = ref<Metric>('active_users')
const options: { key: Metric; label: string }[] = [{key:'active_users',label:'Usuarios activos'},{key:'first_opens',label:'Primeras aperturas'},{key:'registrations',label:'Registros'},{key:'play_device_installs',label:'Instalaciones de Play'}]
const current = computed(() => options.find(option => option.key === metric.value)!)
const known = computed(() => props.rows.filter(row => row[metric.value] !== null))
const latest = computed(() => known.value[known.value.length-1])
const observedMax = computed(() => Math.max(0,...known.value.map(row => row[metric.value]!)))
const max = computed(() => Math.max(1, observedMax.value))
const segments = computed(() => {
  const paths: string[] = []; let path = ''
  props.rows.forEach((row,index) => {
    const value = row[metric.value]
    if(value === null){if(path)paths.push(path);path='';return}
    const x = 30 + (index / Math.max(1,props.rows.length - 1)) * 700
    const y = 170 - (value/max.value)*130
    path += `${path ? ' L' : 'M'}${x},${y}`
  })
  if(path)paths.push(path)
  return paths
})
</script>
<template>
  <section class="metric-trend" aria-label="Gráfico de actividad diaria">
    <div class="trend-heading"><div><h3>Actividad diaria</h3><p>{{ known.length }} de {{ rows.length }} días con datos de {{ current.label.toLocaleLowerCase('es-PE') }}.</p></div><label><span class="sr-only">Métrica del gráfico</span><select v-model="metric"><option v-for="option in options" :key="option.key" :value="option.key">{{ option.label }}</option></select></label></div>
    <div v-if="!known.length" class="trend-empty">Todavía no hay datos de esta métrica en el período seleccionado.</div>
    <template v-else>
      <p v-if="latest" class="latest-value"><strong>{{latest[metric]}}</strong><span>{{current.label}} el {{latest.day.slice(0,10)}}</span></p>
      <svg viewBox="0 0 760 185" role="img" :aria-label="`Tendencia de ${current.label}. Máximo diario: ${observedMax}. Los días sin datos no se unen.`">
        <g class="grid"><path d="M30 40H730M30 105H730M30 170H730"/></g>

        <path v-for="path in segments" :key="path" :d="path" class="trend-line"/>
        <template v-for="(row,index) in rows" :key="row.day"><circle v-if="row[metric] !== null" :cx="30+index/Math.max(1,rows.length-1)*700" :cy="170-row[metric]!/max*130" r="4"><title>{{ row.day.slice(0,10) }}: {{ row[metric] }} {{ current.label }}</title></circle></template>

      </svg>
      <div class="chart-dates"><span>{{rows[0]?.day.slice(0,10)}}</span><span>{{rows[rows.length-1]?.day.slice(0,10)}}</span></div>
      <p class="trend-note">Los días sin cobertura se mantienen sin datos. Puedes consultar las cifras exactas en el detalle diario.</p>
    </template>
  </section>
</template>
<style scoped>
.metric-trend{padding:1.25rem;border:1px solid var(--tf-border-soft);border-radius:16px;background:#fbfcfe}.trend-heading{display:flex;justify-content:space-between;align-items:center;gap:1rem;flex-wrap:wrap}h3{margin:0;font-size:1rem}.trend-heading p,.trend-note{margin:.35rem 0 0;color:var(--tf-text-subtle);font-size:.8rem}select{min-height:44px;max-width:100%;border:1px solid var(--tf-border);border-radius:10px;background:white;padding:.5rem .75rem;font:inherit;font-size:.85rem}svg{display:block;width:100%;max-height:300px;margin-top:1rem;overflow:visible}.grid{stroke:#e2e9f2;stroke-width:1;stroke-dasharray:3 4}text{font:11px var(--tf-font-body);fill:var(--tf-text-subtle)}.trend-line{fill:none;stroke:var(--tf-accent-text);stroke-width:3;stroke-linecap:round;stroke-linejoin:round}circle{fill:white;stroke:var(--tf-accent-text);stroke-width:2}.trend-empty{display:grid;place-items:center;text-align:center;min-height:150px;color:var(--tf-text-subtle);font-size:.9rem}

.latest-value{display:flex;align-items:baseline;gap:.65rem;margin:1rem 0 0}.latest-value strong{font-size:1.65rem;color:var(--tf-accent-text);line-height:1.2}.latest-value span{color:var(--tf-text-subtle);font-size:.75rem}.chart-dates{display:flex;justify-content:space-between;font-size:.7rem;color:var(--tf-text-subtle);margin:0 .5rem .75rem}
</style>
