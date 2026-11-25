<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { Pause, Play, UserRound } from '@lucide/vue'
import UiNotice from '../../components/UiNotice.vue'
import UiAlert from '../../components/UiAlert.vue'
import UiDialog from '../../components/UiDialog.vue'
import type { Order } from './api'
import { productionApi } from './production-api'
import { teamApi, type TeamMember } from '../team/api'
import { useSessionStore } from '../session/session.store'
const props = defineProps<{ order: Order }>()
const emit = defineEmits<{ updated: [order: Order] }>()
const session = useSessionStore()
const manager = computed(() => ['OWNER', 'ADMIN'].includes(session.principal?.role ?? ''))
const editable = computed(() => manager.value && props.order.status === 'ACTIVE')
const members = ref<TeamMember[]>([])
const responsible = ref('')
const reason = ref('')
const busy = ref(false)
const message = ref('')
const error = ref('')
const pauseDialog = ref<'pause' | 'resume' | null>(null)
const memberName = (id: string) => members.value.find(m => m.user_id === id)?.display_name || (id === props.order.responsible_user_id ? props.order.responsible_name : '') || 'Miembro del taller'
const fieldLabels: Record<string,string> = { specification:'Diseño / especificaciones',product:'Producto',quantity:'Cantidad',promised_at:'Fecha de entrega',client_id:'Cliente',responsible_user_id:'Responsable',block_reason:'Pausa de producción' }
const fieldName = (key:string) => fieldLabels[key] ?? key
const displayValue = (value:unknown,key:string) => value === null || value === '' ? 'Sin definir' : key === 'responsible_user_id' ? memberName(String(value)) : key === 'client_id' ? 'Cliente del taller' : key === 'promised_at' ? new Date(String(value)).toLocaleString('es-PE',{timeZone:'America/Lima'}) : String(value)
watch(() => props.order.responsible_user_id,id => {responsible.value = id ?? ''},{immediate:true})
onMounted(async () => {
  if (!manager.value) return
  try { members.value = (await teamApi.list()).data.members.filter(m => m.status === 'ACTIVE') }
  catch { error.value = 'No se pudo cargar el equipo. Recarga la página para asignar un responsable.' }
})
function openPause() { reason.value = ''; error.value = ''; pauseDialog.value = props.order.block_reason ? 'resume' : 'pause' }
async function save(action:'assign'|'pause'|'resume') {
  if (busy.value || !session.csrfToken || !editable.value) return
  error.value = ''; message.value = ''
  if (action === 'pause' && (reason.value.trim().length < 5 || reason.value.trim().length > 300)) {
    error.value = 'Describe el motivo de la pausa con entre 5 y 300 caracteres.'
    return
  }
  busy.value = true
  try {
    const payload = {version:props.order.version,responsible_user_id:action === 'assign' ? responsible.value || null : props.order.responsible_user_id || null,block_reason:action === 'assign' ? props.order.block_reason || '' : action === 'pause' ? reason.value.trim() : ''}
    emit('updated',(await productionApi.operation(props.order.id,payload,session.csrfToken)).data)
    message.value = action === 'assign' ? 'Responsable actualizado.' : action === 'pause' ? 'Pedido en pausa. No se podrán completar etapas hasta reanudarlo.' : 'Producción reanudada. Puedes continuar con la etapa actual.'
    pauseDialog.value = null
  } catch (err:any) { error.value = err?.status === 409 ? 'La orden cambió. Actualiza el detalle antes de guardar.' : 'No se pudo guardar el cambio. Inténtalo nuevamente.' }
  finally { busy.value = false }
}
</script>
<template>
  <section class="operation-panel" aria-label="Organización del pedido">
    <div class="operation-grid">
      <section class="operation-card">
        <div class="operation-heading"><span class="operation-icon"><UserRound :size="21" /></span><h2>Responsable del pedido</h2></div>
        <p>Responsable: {{ order.responsible_user_id ? memberName(order.responsible_user_id) : 'Sin asignar' }}</p>
        <p class="operation-help">Coordina el pedido. El resto del equipo puede seguir registrando avances según su rol.</p>
        <form v-if="editable" @submit.prevent="save('assign')">
          <label for="order-responsible">Asignar a un miembro del equipo</label>
          <div class="assignment-row"><select id="order-responsible" v-model="responsible" :disabled="busy">
            <option value="">Sin asignar</option>
            <option v-if="responsible && !members.some(m => m.user_id === responsible)" :value="responsible">{{ memberName(responsible) }} (actual)</option>
            <option v-for="member in members" :key="member.user_id" :value="member.user_id">{{ member.display_name }}</option>
          </select><button type="submit" class="ops-button" :disabled="busy || responsible === (order.responsible_user_id || '')">Guardar responsable</button></div>
        </form>
      </section>
      <section :class="['operation-card',{'operation-card--paused':order.block_reason}]">
        <div class="operation-heading"><span class="operation-icon"><Pause v-if="order.block_reason" :size="21"/><Play v-else :size="21"/></span><h2>Estado de producción</h2><span class="production-state">{{ order.status !== 'ACTIVE' ? (order.status === 'COMPLETED' ? 'Completada' : 'Cancelada') : order.block_reason ? 'En pausa' : 'En curso' }}</span></div>
        <p v-if="order.block_reason"><strong>Motivo:</strong> {{ order.block_reason }}</p>
        <p class="operation-help">{{ order.block_reason ? 'Las etapas están detenidas. Reanuda el pedido cuando se resuelva el motivo.' : 'Pausa este pedido solo si algo impide continuar, como falta de tela o aprobación del diseño.' }}</p>
        <button v-if="editable" type="button" :class="['ops-button',{'ops-button--primary':order.block_reason}]" :disabled="busy" @click="openPause"><Play v-if="order.block_reason" :size="17"/><Pause v-else :size="17"/>{{ order.block_reason ? 'Reanudar producción' : 'Pausar pedido' }}</button>
      </section>
    </div>
    <UiNotice :message="message"/><UiAlert :message="error"/>
    <details>
      <summary>Historial de cambios de la orden</summary>
      <p v-if="!order.changes?.length">Sin cambios registrados.</p>
      <ol v-else>
        <li v-for="change in order.changes" :key="change.id">
          <strong>{{ new Date(change.created_at).toLocaleString('es-PE', { timeZone: 'America/Lima' }) }}</strong> · {{ change.actor_name || memberName(change.actor_user_id) }}
          <p v-for="(value, key) in change.changes" :key="key">{{ fieldName(String(key)) }}: {{ displayValue(value.from, String(key)) }} → {{ displayValue(value.to, String(key)) }}<span v-if="value.reason">. Motivo: {{ value.reason }}</span></p>
        </li>
      </ol>
    </details>
    <UiDialog v-if="pauseDialog" :title="pauseDialog === 'pause' ? 'Pausar pedido' : 'Reanudar producción'" :busy="busy" @close="pauseDialog = null">
      <template v-if="pauseDialog === 'pause'"><p>El pedido seguirá registrado, pero no se podrán completar etapas hasta reanudarlo.</p><label for="order-block">Motivo de la pausa</label><textarea id="order-block" v-model="reason" rows="3" maxlength="300" placeholder="Ej. Falta confirmar el nuevo diseño con el cliente"/><small>Entre 5 y 300 caracteres.</small></template>
      <p v-else>Se quitará la pausa y el equipo podrá continuar registrando avances en la etapa actual.</p>
      <template #actions><button class="ops-button" :disabled="busy" @click="pauseDialog = null">Cancelar</button><button class="ops-button ops-button--primary" :disabled="busy" @click="save(pauseDialog!)">{{ busy ? 'Guardando…' : pauseDialog === 'pause' ? 'Confirmar pausa' : 'Reanudar producción' }}</button></template>
    </UiDialog>
  </section>
</template>
<style scoped>
.operation-panel{margin:1.5rem 0}.operation-grid{display:grid;grid-template-columns:1fr 1fr;gap:1rem}.operation-card{padding:1.4rem;border:1px solid var(--tf-border-soft);border-radius:16px;background:white;min-width:0}.operation-card--paused{background:#fff9ec;border-color:#ead6a7}.operation-heading{display:flex;align-items:center;flex-wrap:wrap;gap:.65rem}.operation-heading h2{margin:0;font-size:1.05rem}.operation-icon{display:grid;place-items:center;width:38px;height:38px;border-radius:10px;background:var(--tf-primary-soft);color:var(--tf-primary)}.production-state{font-size:.75rem;font-weight:650;border-radius:999px;padding:.3rem .65rem;background:var(--tf-success-soft);color:var(--tf-success)}.operation-card--paused .production-state{color:var(--tf-warning);background:var(--tf-warning-soft)}.operation-help{color:var(--tf-text-subtle);font-size:.875rem;line-height:1.6;max-width:60ch}form{display:grid;gap:.5rem}label{display:block;font-size:.85rem;font-weight:600}.assignment-row{display:flex;flex-wrap:wrap;gap:.5rem}select{flex:1;min-width:140px;max-width:100%;min-height:44px;border:1px solid var(--tf-border);border-radius:10px;padding:.6rem;font:inherit;background:white}textarea{display:block;width:100%;margin:.5rem 0;min-height:100px;padding:.75rem;border:1px solid var(--tf-border);border-radius:10px;font:inherit;resize:vertical}small{color:var(--tf-text-subtle)}details{margin-top:1rem;padding:1rem 1.4rem;border:1px solid var(--tf-border-soft);border-radius:12px;background:white}summary{min-height:32px;font-size:.9rem;font-weight:600;cursor:pointer}li{margin-block:1rem;font-size:.85rem}@media(max-width:1100px){.operation-grid{grid-template-columns:1fr}}
</style>
