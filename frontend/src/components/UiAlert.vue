<script setup lang="ts">
import { ref, watch } from 'vue'
import { AlertCircle, X } from '@lucide/vue'
const props = defineProps<{ message: string }>()
const closed = ref(false)
watch(() => props.message, () => { closed.value = false })
</script>
<template><Teleport to="body"><div v-if="message && !closed" class="feedback-toast" role="alert" aria-live="assertive"><AlertCircle :size="20" aria-hidden="true"/><p>{{ message }}</p><button type="button" aria-label="Cerrar aviso" @click="closed = true"><X :size="18" /></button></div></Teleport></template>
<style scoped>.feedback-toast{position:fixed;z-index:500;top:1.25rem;right:1.25rem;display:flex;align-items:start;gap:.75rem;width:min(420px,calc(100vw - 2.5rem));padding:1rem;background:#fff;border:1px solid #e5b9b9;border-left:4px solid #b42332;border-radius:12px;color:#70202b;box-shadow:0 8px 30px rgb(15 35 60 / 16%);font-family:var(--tf-font-body);box-sizing:border-box}.feedback-toast p{margin:0;flex:1;line-height:1.5;font-size:.9rem}.feedback-toast>svg{flex-shrink:0;margin-top:3px}button{display:grid;place-items:center;border:0;background:transparent;color:inherit;cursor:pointer;min-width:32px;min-height:32px}button:focus-visible{outline:2px solid currentColor;outline-offset:2px}</style>
