<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue'
import { X } from '@lucide/vue'
defineProps<{ title: string; busy?: boolean; wide?: boolean }>()
const emit = defineEmits<{ close: [] }>()
const dialog = ref<HTMLDialogElement | null>(null)
const opener = typeof document === 'undefined' ? null : document.activeElement as HTMLElement | null
onMounted(() => {
  if (dialog.value?.showModal) dialog.value.showModal()
  else dialog.value?.setAttribute('open', '')
})
onUnmounted(() => opener?.focus())
</script>
<template>
  <Teleport to="body">
    <dialog ref="dialog" class="tf-dialog" :class="{ 'tf-dialog--wide': wide }" aria-labelledby="tf-dialog-title" @cancel.prevent="!busy && emit('close')" @click="($event.target === dialog && !busy) && emit('close')">
      <header><h2 id="tf-dialog-title">{{ title }}</h2><button type="button" :disabled="busy" aria-label="Cerrar ventana" @click="emit('close')"><X :size="20" /></button></header>
      <div class="tf-dialog__body"><slot /></div>
      <footer v-if="$slots.actions"><slot name="actions" /></footer>
    </dialog>
  </Teleport>
</template>
<style scoped>
.tf-dialog{width:min(520px,calc(100vw - 2rem));max-height:calc(100dvh - 2rem);margin:auto;padding:0;border:1px solid var(--tf-border-soft);border-radius:20px;color:var(--tf-ink);background:white;box-shadow:0 24px 80px #09234c33;font-family:var(--tf-font-body);overflow:auto}
.tf-dialog--wide{width:min(1100px,calc(100vw - 2rem))}
.tf-dialog::backdrop{background:#08274e80;backdrop-filter:blur(3px)}
header{display:flex;align-items:center;justify-content:space-between;gap:1rem;padding:1.25rem 1.5rem;border-bottom:1px solid var(--tf-border-soft)}
h2{margin:0;font:700 1.2rem/1.3 var(--tf-font-body);letter-spacing:-.025em}
header button{display:grid;place-items:center;width:44px;height:44px;flex:none;border:0;border-radius:50%;background:var(--tf-surface-soft);color:var(--tf-ink);cursor:pointer}
.tf-dialog__body{padding:1.5rem}footer{display:flex;justify-content:flex-end;gap:.75rem;padding:1rem 1.5rem;border-top:1px solid var(--tf-border-soft)}
button:focus-visible{outline:3px solid var(--tf-focus);outline-offset:2px}
</style>
