<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import type { Order } from './api'
import { productionApi } from './production-api'
import { teamApi, type TeamMember } from '../team/api'
import { useSessionStore } from '../session/session.store'
const props = defineProps<{ order: Order }>()
const emit = defineEmits<{ updated: [order: Order] }>()
const session = useSessionStore()
const manager = computed(() => ['OWNER', 'ADMIN'].includes(session.principal?.role ?? ''))
const members = ref<TeamMember[]>([])
const responsible = ref('')
const reason = ref('')
const busy = ref(false)
const message = ref('')
const memberName = (id: string) => members.value.find(m => m.user_id === id)?.display_name || (id === props.order.responsible_user_id ? props.order.responsible_name : '') || 'Miembro del taller'
const fieldLabels: Record<string, string> = { specification: 'Diseño / especificaciones', product: 'Producto', quantity: 'Cantidad', promised_at: 'Fecha de entrega', client_id: 'Cliente', responsible_user_id: 'Responsable', block_reason: 'Bloqueo' }
const fieldName = (key: string) => fieldLabels[key] ?? key
const displayValue = (value: unknown, key: string) => value === null || value === '' ? 'Sin definir' : key === 'responsible_user_id' ? memberName(String(value)) : key === 'client_id' ? 'Cliente del taller' : key === 'promised_at' ? new Date(String(value)).toLocaleString('es-PE', { timeZone: 'America/Lima' }) : String(value)
watch(() => props.order, order => { responsible.value = order.responsible_user_id ?? ''; reason.value = order.block_reason ?? '' }, { immediate: true })
onMounted(async () => { if (manager.value) { try { members.value = (await teamApi.list()).data.members.filter(m => m.status === 'ACTIVE') } catch { message.value = 'No se pudo cargar el equipo. Actualiza para asignar un responsable.' } } })
async function save() {
  if (busy.value || !session.csrfToken) return
  busy.value = true; message.value = ''
  try {
    emit('updated', (await productionApi.operation(props.order.id, { responsible_user_id: responsible.value || null, block_reason: reason.value.trim(), version: props.order.version }, session.csrfToken)).data)
    message.value = 'Responsable y bloqueo actualizados.'
  } catch (err: any) { message.value = err?.status === 409 ? 'La orden cambió. Actualiza el detalle antes de guardar.' : 'No se pudo guardar. Revisa el responsable y el motivo (5 a 300 caracteres).' }
  finally { busy.value = false }
}
</script>
<template>
  <section class="operation-panel" aria-labelledby="operation-title">
    <h2 id="operation-title">Responsable y bloqueos</h2>
    <p>Responsable: {{ order.responsible_user_id ? memberName(order.responsible_user_id) : 'Sin asignar' }}</p>
    <p v-if="order.block_reason" role="status"><strong>Producción bloqueada:</strong> {{ order.block_reason }}. Resuelve el motivo antes de completar una etapa.</p>
    <form v-if="manager && order.status === 'ACTIVE'" @submit.prevent="save">
      <label for="order-responsible">Responsable de la orden</label>
      <select id="order-responsible" v-model="responsible" :disabled="busy">
        <option value="">Sin asignar</option>
        <option v-if="responsible && !members.some(m => m.user_id === responsible)" :value="responsible">{{ memberName(responsible) }} (asignación actual)</option>
        <option v-for="member in members" :key="member.user_id" :value="member.user_id">{{ member.display_name }}</option>
      </select>
      <label for="order-block">Motivo del bloqueo (vacío para desbloquear)</label>
      <textarea id="order-block" v-model="reason" maxlength="300" :minlength="reason ? 5 : undefined" rows="2" :disabled="busy" />
      <button type="submit" :disabled="busy">{{ busy ? 'Guardando…' : 'Guardar responsable y bloqueo' }}</button>
    </form>
    <p role="status">{{ message }}</p>
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
  </section>
</template>
<style scoped>
.operation-panel { margin-block: 1rem; padding: 1rem; background: var(--tf-surface); border: 1px solid var(--tf-border); border-radius: .5rem; overflow-wrap: anywhere; }
form { display: grid; gap: .5rem; } select, textarea, button { min-height: 44px; padding: .65rem; font: inherit; border: 1px solid var(--tf-border); border-radius: .35rem; } summary { min-height: 44px; cursor: pointer; } li { margin-block: 1rem; } h2 { font-size: 1.1rem; }
</style>
