<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { ordersApi } from './api'
import { clientsApi, type Client } from '../clients/api'
import { useSessionStore } from '../session/session.store'
import UiButton from '../../components/UiButton.vue'
import UiAlert from '../../components/UiAlert.vue'
import { ArrowLeft, ArrowRight, Plus, TriangleAlert } from '@lucide/vue'

const router = useRouter()
const session = useSessionStore()

const clients = ref<Client[]>([])
const loadingClients = ref(true)
const submitting = ref(false)
const error = ref('')

// Form state
const clientId = ref('')
const product = ref('')
const specification = ref('')
const quantity = ref<number | ''>('')
const unit = ref('UNITS')
const promisedDate = ref('')
const promisedTime = ref('18:00')

// Idempotency key generated per form lifecycle
const idempotencyKey = ref(crypto.randomUUID())
const productionStages = ['Corte', 'Costura', 'Acabado', 'Control de calidad', 'Planchado', 'Empaque', 'Entrega']

async function loadClients() {
  loadingClients.value = true
  try {
    const res = await clientsApi.list()
    clients.value = res.data || []
    if (clients.value.length > 0) {
      clientId.value = clients.value[0].id
    }
  } catch (err: any) {
    error.value = 'No se pudieron cargar los clientes. Asegúrate de tener al menos uno registrado.'
  } finally {
    loadingClients.value = false
  }
}

// Set default delivery date 5 days ahead at 18:00
function setDefaultDate() {
  const d = new Date()
  d.setDate(d.getDate() + 5)
  const yyyy = d.getFullYear()
  const mm = String(d.getMonth() + 1).padStart(2, '0')
  const dd = String(d.getDate()).padStart(2, '0')
  promisedDate.value = `${yyyy}-${mm}-${dd}`
}

async function handleSubmit() {
  if (!clientId.value) {
    error.value = 'Debes seleccionar un cliente.'
    return
  }
  if (!product.value.trim()) {
    error.value = 'El nombre del producto es obligatorio.'
    return
  }
  if (!quantity.value || Number(quantity.value) <= 0) {
    error.value = 'La cantidad debe ser mayor a 0.'
    return
  }
  if (!promisedDate.value) {
    error.value = 'Debes especificar una fecha de entrega.'
    return
  }
  if (!session.csrfToken) {
    error.value = 'Tu sesión ya no permite crear órdenes. Actualiza la página e inicia sesión nuevamente.'
    return
  }

  const isoDateTime = new Date(`${promisedDate.value}T${promisedTime.value || '18:00'}:00`).toISOString()

  submitting.value = true
  error.value = ''
  try {
    const res = await ordersApi.create(
      {
        client_id: clientId.value,
        product: product.value.trim(),
        specification: specification.value.trim(),
        quantity: Number(quantity.value),
        unit: unit.value,
        promised_at: isoDateTime,
      },
      idempotencyKey.value,
      session.csrfToken,
    )
    // Navigate straight to the created order detail
    router.push(`/app/orders/${res.data.id}`)
  } catch (err: any) {
    error.value = err?.message || 'Error al crear la orden de trabajo.'
    submitting.value = false
  }
}

onMounted(() => {
  setDefaultDate()
  loadClients()
})
</script>

<template>
  <main class="create-order-page">
    <header class="page-header">
      <button class="btn-back" type="button" @click="router.push('/app/orders')"><ArrowLeft :size="17" aria-hidden="true" /> Volver a órdenes</button>
      <h1>Nueva orden de trabajo</h1>
    </header>

    <UiAlert v-if="error" :message="error" />

    <section v-if="loadingClients" class="loading-state">
      <p>Cargando lista de clientes...</p>
    </section>

    <section v-else-if="clients.length === 0" class="no-clients-alert">
      <div class="empty-box">
        <h3><TriangleAlert :size="20" aria-hidden="true" /> No tienes clientes registrados</h3>
        <p>Para crear una orden de confección, primero debes registrar a tu cliente.</p>
        <UiButton @click="router.push('/app/clients')">
          <Plus :size="17" aria-hidden="true" /> Registrar cliente
        </UiButton>
      </div>
    </section>

    <form v-else @submit.prevent="handleSubmit" class="order-form">
      <!-- 1. Cliente -->
      <div class="field-group">
        <label for="order-client">Cliente *</label>
        <select id="order-client" v-model="clientId" required class="input-select">
          <option v-for="c in clients" :key="c.id" :value="c.id">
            {{ c.name }} {{ c.phone ? `(${c.phone})` : '' }}
          </option>
        </select>
      </div>

      <!-- 2. Producto -->
      <div class="field-group">
        <label for="order-product">Producto / Prenda *</label>
        <input
          id="order-product"
          v-model="product"
          type="text"
          required
          maxlength="160"
          placeholder="Ej. Polos cuello redondo manga corta 30/1"
          class="input-text"
        />
      </div>

      <!-- 3. Especificación -->
      <div class="field-group">
        <label for="order-spec">Detalles / Especificaciones técnicas</label>
        <textarea
          id="order-spec"
          v-model="specification"
          rows="3"
          placeholder="Ej. Tela reactiva negro y blanco, rib 1x1, estampado en pecho 20x15cm, tallas S-M-L-XL repartidas."
          class="input-textarea"
        ></textarea>
      </div>

      <!-- 4. Cantidad y Unidad -->
      <div class="form-row-2">
        <div class="field-group">
          <label for="order-qty">Cantidad *</label>
          <input
            id="order-qty"
            v-model="quantity"
            type="number"
            min="1"
            step="any"
            required
            class="input-text"
          />
        </div>

        <div class="field-group">
          <label for="order-unit">Unidad *</label>
          <select id="order-unit" v-model="unit" class="input-select">
            <option value="UNITS">Unidades (Prendas)</option>
            <option value="DOZENS">Docenas</option>
            <option value="KILOGRAMS">Kilos</option>
            <option value="METERS">Metros</option>
          </select>
        </div>
      </div>

      <!-- 5. Fecha y Hora de Entrega -->
      <div class="form-row-2">
        <div class="field-group">
          <label for="order-date">Fecha de Entrega Prometida *</label>
          <input
            id="order-date"
            v-model="promisedDate"
            type="date"
            required
            class="input-text"
          />
        </div>

        <div class="field-group">
          <label for="order-time">Hora límite</label>
          <input
            id="order-time"
            v-model="promisedTime"
            type="time"
            class="input-text"
          />
        </div>
      </div>

      <div class="stages-preview">
        <p class="stages-title">Recorrido de producción</p>
        <p class="stages-caption">La orden comenzará en Corte. Cada estación activa la siguiente al completarse.</p>
        <ol class="stages-list">
          <li v-for="(stage, index) in productionStages" :key="stage"><span class="stages-list__number">{{ index + 1 }}</span><span>{{ stage }}</span><ArrowRight v-if="index < productionStages.length - 1" :size="14" aria-hidden="true" /></li>
        </ol>
      </div>

      <!-- Botones de Acción -->
      <div class="form-actions">
        <button
          type="button"
          class="btn-secondary"
          @click="router.push('/app/orders')"
        >
          Cancelar
        </button>
        <UiButton type="submit" :disabled="submitting">
          {{ submitting ? 'Creando orden...' : 'Crear orden' }}
        </UiButton>
      </div>
    </form>
  </main>
</template>

<style scoped>
.create-order-page {
  max-width: 600px;
  margin: 0 auto;
  padding: 1rem;
}

.page-header {
  margin-bottom: 1.25rem;
}

.btn-back {
  background: none;
  border: none;
  color: var(--tf-primary);
  font-size: 0.95rem;
  font-weight: 600;
  cursor: pointer;
  padding: 0.25rem 0;
  margin-bottom: 0.25rem;
}

h1 {
  font-size: 1.35rem;
  color: var(--tf-ink);
  margin: 0;
}

.order-form {
  background: #fff;
  border: 1px solid var(--tf-border-soft);
  border-radius: 12px;
  padding: 1.25rem;
  display: flex;
  flex-direction: column;
  gap: 1rem;
  box-shadow: 0 2px 6px rgba(0, 0, 0, 0.05);
}

.field-group {
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
}

.field-group label {
  font-size: 0.85rem;
  font-weight: 600;
  color: #334155;
}

.form-row-2 {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 0.75rem;
}

.input-text,
.input-select,
.input-textarea {
  width: 100%;
  padding: 0.65rem 0.85rem;
  border: 1px solid var(--tf-border);
  border-radius: 8px;
  font-size: 0.95rem;
  background: #fff;
}

.input-textarea {
  resize: vertical;
}

.stages-preview {
  background: var(--tf-primary-soft);
  border: 1px dashed var(--tf-primary-border);
  border-radius: 8px;
  padding: 0.75rem 1rem;
  font-size: 0.85rem;
}

.stages-title {
  font-weight: 700;
  margin: 0 0 0.4rem 0;
  color: var(--tf-primary-strong);
}

.stages-list {
  margin: 0;
  padding-left: 1.2rem;
  color: #334155;
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 0.2rem;
}

.form-actions {
  display: flex;
  justify-content: flex-end;
  gap: 0.75rem;
  margin-top: 0.5rem;
}

.btn-secondary {
  background: none;
  border: 1px solid #cbd9df;
  padding: 0.6rem 1.2rem;
  border-radius: 8px;
  font-weight: 600;
  color: #475569;
  cursor: pointer;
}

.no-clients-alert {
  text-align: center;
  padding: 2rem 1rem;
  background: #fff;
  border: 1px solid #cbd9df;
  border-radius: 12px;
}

.create-order-page { max-width: 54rem; padding: clamp(1rem, 2.5vw, 2rem); }
.page-header { margin-bottom: 1.5rem; }
.page-header h1 { margin-top: .25rem; font-size: var(--tf-page-title-size); color: var(--tf-ink); }
.btn-back { min-height: 44px; color: var(--tf-primary); }
.order-form { gap: 1.25rem; padding: clamp(1rem, 2.5vw, 1.75rem); border-color: var(--tf-border-soft); border-radius: .5rem; box-shadow: none; }
.field-group label { color: var(--tf-ink); font-weight: 700; }
.input-text, .input-select, .input-textarea { min-height: 46px; border-color: var(--tf-border); border-radius: .4rem; background: #fff; }
.input-text:focus-visible, .input-select:focus-visible, .input-textarea:focus-visible { outline: 2px solid var(--tf-focus); border-color: var(--tf-primary); }
.stages-preview { border: 1px solid var(--tf-primary-border); border-radius: .4rem; background: var(--tf-primary-soft); }
.stages-title { color: var(--tf-primary-strong); }
.stages-list { color: var(--tf-ink-soft); }
.stages-preview { padding: 1rem 0 0; border: 0; border-top: 1px solid var(--tf-border-soft); border-radius: 0; background: transparent; }
.stages-title { margin: 0 0 .15rem; color: var(--tf-ink); font-size: .9rem; }
.stages-caption { margin: 0 0 .85rem; color: var(--tf-text-subtle); font-size: .82rem; }
.stages-list { display: flex; flex-wrap: wrap; align-items: center; gap: .5rem .6rem; padding: 0; list-style: none; }
.stages-list li { display: inline-flex; align-items: center; gap: .4rem; color: var(--tf-ink-soft); font-size: .8rem; }
.stages-list li:first-child { color: var(--tf-primary-strong); font-weight: 800; }
.stages-list__number { display: inline-grid; width: 1.35rem; height: 1.35rem; place-items: center; border: 1px solid var(--tf-border); border-radius: 50%; color: var(--tf-text-subtle); font-size: .7rem; font-weight: 800; }
.stages-list li:first-child .stages-list__number { border-color: var(--tf-primary); color: #fff; background: var(--tf-primary); }
.btn-back { display: inline-flex; align-items: center; gap: .35rem; }
.no-clients-alert h3 { display: flex; align-items: center; justify-content: center; gap: .4rem; }
.btn-secondary { min-height: 44px; color: var(--tf-primary); border-color: var(--tf-primary-border); background: #fff; }
.btn-secondary:hover { background: var(--tf-primary-soft); }
@media (max-width: 560px) { .form-row-2 { grid-template-columns: 1fr; } .form-actions { flex-direction: column-reverse; } .form-actions > * { width: 100%; } }
</style>
