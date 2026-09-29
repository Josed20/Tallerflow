<script setup lang="ts">
import { useRouter } from 'vue-router'
import UiButton from '../../components/UiButton.vue'
import { useSessionStore } from '../session/session.store'

const router = useRouter()
const session = useSessionStore()

function navigateHome() {
  if (session.isAuthenticated) {
    router.push(session.requiresPasswordChange ? '/change-password' : '/app')
  } else {
    router.push('/login')
  }
}
</script>

<template>
  <main class="not-found-layout">
    <div class="not-found-card">
      <div class="brand-mark" aria-hidden="true">TF</div>
      <p class="error-code" aria-hidden="true">404</p>
      <h1>Página no encontrada</h1>
      <p class="not-found-subtitle">
        La página que buscas no existe, ha sido movida o la dirección ingresada no es correcta.
      </p>
      <div class="not-found-actions">
        <UiButton type="button" @click="navigateHome">
          Volver al inicio
        </UiButton>
      </div>
    </div>
  </main>
</template>

<style scoped>
.not-found-layout {
  display: grid;
  min-height: 100vh;
  place-items: center;
  padding: 1.5rem;
  background: #f8f9ff;
}

.not-found-card {
  position: relative;
  width: min(100%, 28rem);
  padding: 2.5rem 1.5rem;
  text-align: center;
  background: var(--tf-surface, #ffffff);
  border: 1px solid #dce9ff;
  border-radius: 1rem;
  box-shadow: 0 8px 30px rgb(11 28 48 / 8%);
}

.error-code {
  margin: 0.25rem 0;
  color: var(--tf-primary, #00685f);
  font-size: 3.5rem;
  font-weight: 800;
  line-height: 1;
  letter-spacing: -0.04em;
}

h1 {
  margin: 0.5rem 0;
  font-size: 1.5rem;
  font-weight: 700;
  color: #0b1c30;
}

.not-found-subtitle {
  max-width: 22rem;
  margin: 0.5rem auto 1.75rem;
  color: var(--tf-text-muted, #3d4947);
  font-size: 0.875rem;
  line-height: 1.5;
}

.not-found-actions {
  display: flex;
  justify-content: center;
}

@media (max-width: 360px) {
  .not-found-layout {
    padding: 0.75rem;
  }
  .not-found-card {
    padding: 1.75rem 1rem;
  }
  .error-code {
    font-size: 2.75rem;
  }
  h1 {
    font-size: 1.25rem;
  }
}
</style>
