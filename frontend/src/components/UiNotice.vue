<script setup lang="ts">
import { onUnmounted, ref, watch } from 'vue'
import { Info, X } from '@lucide/vue'
const props = defineProps<{ message: string }>()
const closed = ref(false)
let timeout:ReturnType<typeof setTimeout>|undefined
watch(() => props.message, message => {
  closed.value=false
  if(timeout)clearTimeout(timeout)
  if(message)timeout=setTimeout(()=>{closed.value=true},6000)
},{immediate:true})
onUnmounted(()=>{if(timeout)clearTimeout(timeout)})
</script>
<template><Teleport to="body"><div v-if="message && !closed" class="notice-toast" role="status" aria-live="polite"><Info :size="20" aria-hidden="true"/><p>{{ message }}</p><button type="button" aria-label="Cerrar aviso" @click="closed = true"><X :size="18"/></button></div></Teleport></template>
<style scoped>.notice-toast{position:fixed;top:1.25rem;right:1.25rem;z-index:400;display:flex;gap:.75rem;align-items:start;width:min(420px,calc(100vw - 2.5rem));padding:1rem;box-sizing:border-box;background:#fff;border:1px solid #cddbea;border-left:4px solid #0d3870;border-radius:12px;box-shadow:0 8px 30px rgb(15 35 60 / 16%);color:#183451;font-family:var(--tf-font-body)}p{margin:0;flex:1;line-height:1.5;font-size:.9rem}svg{flex-shrink:0;margin-top:3px}button{border:0;background:transparent;color:inherit;cursor:pointer;display:grid;place-items:center;min-width:32px;min-height:32px}button:focus-visible{outline:2px solid currentColor;outline-offset:2px}</style>
