<script setup lang="ts">
import { onMounted, ref } from 'vue'

defineProps<{ code: string; stage: string; nextStage: string | null; busy: boolean }>()
const emit = defineEmits<{ confirm: []; cancel: [] }>()
const cancelButton = ref<HTMLButtonElement | null>(null)
onMounted(() => cancelButton.value?.focus())
</script>

<template>
  <div class="advance-overlay" @click.self="!busy && emit('cancel')" @keydown.esc="!busy && emit('cancel')">
    <section class="advance-dialog" role="dialog" aria-modal="true" aria-labelledby="advance-title" aria-describedby="advance-description">
      <p class="advance-dialog__eyebrow">Confirmar avance · {{ code }}</p>
      <h2 id="advance-title">¿Terminó {{ stage.toLocaleLowerCase('es-PE') }}?</h2>
      <p id="advance-description">{{ nextStage ? `La orden pasará a ${nextStage.toLocaleLowerCase('es-PE')}.` : 'La orden quedará completada.' }} Revisa que el trabajo de esta estación esté listo antes de continuar.</p>
      <div class="advance-dialog__steps" aria-hidden="true"><span>{{ stage }}</span><span>→</span><strong>{{ nextStage || 'Completada' }}</strong></div>
      <div class="advance-dialog__actions">
        <button ref="cancelButton" type="button" class="advance-dialog__cancel" :disabled="busy" @click="emit('cancel')">Seguir en esta etapa</button>
        <button type="button" class="advance-dialog__confirm" :disabled="busy" @click="emit('confirm')">{{ busy ? 'Guardando...' : 'Confirmar avance' }}</button>
      </div>
    </section>
  </div>
</template>

<style scoped>
.advance-overlay { position: fixed; inset: 0; z-index: 110; display: grid; place-items: center; padding: 1rem; background: rgb(22 35 61 / 55%); }
.advance-dialog { width: min(100%, 30rem); padding: 1.5rem; border: 1px solid var(--tf-border-soft); border-radius: .5rem; background: var(--tf-surface); box-shadow: var(--tf-shadow-md); }
.advance-dialog__eyebrow { margin: 0 0 .5rem; color: var(--tf-primary); font-size: .76rem; font-weight: 800; text-transform: uppercase; }
.advance-dialog h2 { margin: 0 0 .55rem; color: var(--tf-ink); font-size: 1.3rem; line-height: 1.3; }
.advance-dialog p:not(.advance-dialog__eyebrow) { margin: 0; color: var(--tf-text-subtle); font-size: .9rem; line-height: 1.55; }
.advance-dialog__steps { display: flex; align-items: center; flex-wrap: wrap; gap: .55rem; margin: 1.1rem 0; padding: .7rem 0; border-top: 1px solid var(--tf-border-soft); border-bottom: 1px solid var(--tf-border-soft); color: var(--tf-ink-soft); font-size: .83rem; }
.advance-dialog__steps strong { color: var(--tf-primary-strong); }
.advance-dialog__actions { display: flex; justify-content: flex-end; flex-wrap: wrap; gap: .6rem; }
.advance-dialog__actions button { min-height: 44px; padding: .55rem .85rem; border-radius: .4rem; font: inherit; font-size: .85rem; font-weight: 750; cursor: pointer; }
.advance-dialog__cancel { border: 1px solid var(--tf-border); color: var(--tf-ink); background: var(--tf-surface); }
.advance-dialog__confirm { border: 1px solid var(--tf-primary); color: var(--tf-on-primary); background: var(--tf-primary); }
.advance-dialog__confirm:hover:not(:disabled) { background: var(--tf-primary-hover); }
.advance-dialog__actions button:disabled { opacity: .65; cursor: wait; }
.advance-dialog__actions button:focus-visible { outline: 3px solid var(--tf-focus); outline-offset: 2px; }
@media (max-width: 480px) { .advance-dialog__actions { flex-direction: column-reverse; } .advance-dialog__actions button { width: 100%; } }
</style>
