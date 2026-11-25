<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { productionApi, type ProductionSettings } from './production-api'
import { useSessionStore } from '../session/session.store'
const session = useSessionStore()
const settings = ref<ProductionSettings | null>(null)
const busy = ref(false)
const message = ref('')
async function load() {
  try { settings.value = (await productionApi.get()).data; message.value = '' }
  catch { message.value = 'No se pudo cargar la plantilla. Vuelve a abrir esta sección.' }
}
async function save() {
  if (!settings.value || busy.value || !session.csrfToken) return
  busy.value = true; message.value = ''
  try {
    settings.value = (await productionApi.save(settings.value, session.csrfToken)).data
    message.value = 'Plantilla guardada. Solo se aplicará a nuevas órdenes.'
  } catch (err: any) { message.value = err?.status === 409 ? 'Otra persona cambió la plantilla. Recarga antes de editar.' : 'No se pudo guardar. Revisa nombres y descripciones.' }
  finally { busy.value = false }
}
function useConfectionExample() {
  if (!settings.value) return
  settings.value.stages = [
    { name: 'MUESTRA', client_description: 'Preparamos una prenda de muestra para confirmar el diseño.' },
    { name: 'CORTE', client_description: 'Cortamos las piezas de tela que se usarán para confeccionar las prendas.' },
    { name: 'CONFECCIÓN', client_description: 'Unimos y cosemos las piezas de cada prenda.' },
    { name: 'PLANCHADO', client_description: 'Planchamos las prendas confeccionadas.' },
    { name: 'EMBOLSADO', client_description: 'Embolsamos y organizamos las prendas para su entrega.' },
    { name: 'ENTREGA', client_description: 'El pedido está listo para coordinar la entrega con el taller.' },
  ]
}
onMounted(load)
</script>
<template>
  <details class="production-settings">
    <summary>Configurar etapas de producción</summary>
    <p>La plantilla se copia al crear una orden. Las órdenes anteriores conservan sus etapas. Puedes definir de una a veinte etapas.</p>
    <p role="status">{{ message }}</p>
    <button v-if="!settings" class="ops-button" type="button" @click="load">Volver a cargar la plantilla</button>
    <form v-if="settings" @submit.prevent="save">
      <fieldset :disabled="busy">
        <legend>Plantilla para nuevas órdenes</legend>
        <button class="ops-button" type="button" @click="useConfectionExample">Cargar ejemplo de confección</button>
        <ol>
          <li v-for="(stage, index) in settings.stages" :key="index">
            <label :for="`stage-name-${index}`">Etapa {{ index + 1 }}</label>
            <input :id="`stage-name-${index}`" v-model="stage.name" required maxlength="80" />
            <label :for="`stage-description-${index}`">Descripción para el cliente</label>
            <textarea :id="`stage-description-${index}`" v-model="stage.client_description" required maxlength="500" rows="2" />
            <div>
              <button type="button" :disabled="index === 0" @click="[settings.stages[index - 1], settings.stages[index]] = [stage, settings.stages[index - 1]]">Subir</button>
              <button type="button" :disabled="index === settings.stages.length - 1" @click="[settings.stages[index + 1], settings.stages[index]] = [stage, settings.stages[index + 1]]">Bajar</button>
              <button type="button" :disabled="settings.stages.length === 1" @click="settings.stages.splice(index, 1)">Quitar</button>
            </div>
          </li>
        </ol>
        <button type="button" :disabled="settings.stages.length >= 20" @click="settings.stages.push({ name: '', client_description: '' })">Añadir etapa</button>
        <button class="ops-button ops-button--primary" type="submit">{{ busy ? 'Guardando…' : 'Guardar plantilla' }}</button>
        <button type="button" @click="load">Recargar plantilla</button>
      </fieldset>
    </form>
  </details>
</template>
<style scoped>
.production-settings { margin-block: 1.5rem; padding: 1rem; background: var(--tf-surface); border: 1px solid var(--tf-border); border-radius: .5rem; }
summary { min-height: 44px; cursor: pointer; font-weight: 700; }
fieldset { border: 0; padding: 0; min-width: 0; } legend { font-weight: 700; margin-bottom: .75rem; } ol { margin: 1rem 0; padding: 0; list-style: none; display: grid; gap: 1rem; } li { padding: 1rem; border: 1px solid var(--tf-border-soft); border-radius: .5rem; background: var(--tf-surface-soft); } label, input, textarea { display: block; width: 100%; } label { margin: .5rem 0 .35rem; font-size: .875rem; font-weight: 700; } input, textarea { padding: .65rem; border: 1px solid var(--tf-border); border-radius: .35rem; background: var(--tf-surface); font: inherit; } input:focus-visible, textarea:focus-visible, summary:focus-visible, button:focus-visible { outline: 3px solid var(--tf-focus); outline-offset: 2px; } button { min-height: 44px; padding: .5rem .75rem; margin: .25rem; cursor: pointer; border: 1px solid var(--tf-border); border-radius: .4rem; color: var(--tf-primary); background: var(--tf-surface); font: inherit; } button.ops-button--primary { color: var(--tf-on-primary); background: var(--tf-primary); border-color: var(--tf-primary); } button:disabled { opacity: .5; cursor: default; } p { color: var(--tf-text-subtle); }
</style>
