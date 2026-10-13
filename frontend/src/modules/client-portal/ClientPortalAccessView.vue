<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { clientPortalApi } from './api'
import BrandLogo from '../../components/BrandLogo.vue'

const route = useRoute()
const router = useRouter()
const email = ref('')
const submitting = ref(false)
const confirmation = ref(false)
const error = ref('')

async function requestLink() {
  if (!email.value.trim()) {
    error.value = 'Ingresa un correo electrónico para continuar.'
    return
  }
  submitting.value = true
  error.value = ''
  try {
    await clientPortalApi.requestAccess(email.value.trim())
    confirmation.value = true
    email.value = ''
  } catch {
    error.value = 'El portal no está disponible todavía. Inténtalo nuevamente más tarde.'
  } finally {
    submitting.value = false
  }
}

async function consumeToken(token: string) {
  submitting.value = true
  error.value = ''
  try {
    await clientPortalApi.consumeAccessLink(token)
    await router.replace('/portal/orders')
  } catch {
    error.value = 'Este enlace no es válido o ya venció. Solicita uno nuevo.'
  } finally {
    submitting.value = false
  }
}

onMounted(() => {
  const token = route.query.token
  if (typeof token === 'string' && token) void consumeToken(token)
})
</script>

<template>
  <main class="portal-access">
    <section class="portal-access__card" aria-labelledby="portal-access-title">
      <BrandLogo class="portal-access__brand" />
      <h1 id="portal-access-title">Consulta el avance de tus pedidos</h1>
      <p>Recibirás un enlace privado para ver únicamente los pedidos asociados a tu correo.</p>
      <p v-if="error" class="portal-access__error" role="alert">{{ error }}</p>
      <p v-if="confirmation" class="portal-access__confirmation" role="status">Si el correo está asociado a un pedido, recibirás un enlace privado en unos minutos.</p>
      <form v-else @submit.prevent="requestLink">
        <label for="portal-email">Correo electrónico</label>
        <input id="portal-email" v-model="email" type="email" autocomplete="email" required :disabled="submitting" />
        <button type="submit" :disabled="submitting">{{ submitting ? 'Enviando…' : 'Enviar enlace' }}</button>
      </form>
    </section>
  </main>
</template>

<style scoped>
.portal-access { display: grid; min-height: 100vh; place-items: center; padding: 1rem; background: var(--tf-page); color: var(--tf-ink); }
.portal-access__card { width: min(100%, 29rem); padding: 1.5rem; border: 1px solid var(--tf-border-soft); border-radius: .5rem; background: #fff; box-shadow: var(--tf-shadow-md); }
.portal-access__brand { width: 11.5rem; margin: 0 0 1rem; }
.portal-access h1 { margin: 0; font-size: 1.75rem; }
.portal-access form { display: grid; gap: .6rem; margin-top: 1rem; font-weight: 700; }
.portal-access input, .portal-access button { min-height: 44px; border-radius: .5rem; padding: .625rem .75rem; font: inherit; }
.portal-access input { border: 1px solid var(--tf-border); }
.portal-access button { border: 0; background: var(--tf-primary); color: #fff; font-weight: 800; cursor: pointer; }
.portal-access button:hover { background: var(--tf-primary-hover); }
.portal-access input:focus-visible, .portal-access button:focus-visible { outline: 3px solid var(--tf-focus); outline-offset: 2px; }
.portal-access__error { color: var(--tf-danger); }
.portal-access__confirmation { border-radius: .5rem; padding: .75rem; background: var(--tf-success-soft); color: var(--tf-success); font-weight: 700; }
</style>
