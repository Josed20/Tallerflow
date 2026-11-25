<script setup lang="ts">
import { onUnmounted, ref, watch } from 'vue'
import { CircleAlert, Info, X } from '@lucide/vue'
const props = defineProps<{ message: string; kind: 'error' | 'notice' }>()
const closed = ref(false)
let timer: ReturnType<typeof setTimeout> | undefined
function stopTimer() {
  clearTimeout(timer)
}
function startTimer() {
  stopTimer()
  if (props.kind === 'notice' && props.message) timer = setTimeout(() => { closed.value = true }, 6500)
}
watch(() => props.message, () => { closed.value = false; startTimer() }, { immediate: true })
onUnmounted(stopTimer)
</script>

<template>
  <Teleport to="body">
    <div v-if="message && !closed" class="toast" :class="kind" :aria-live="kind === 'error' ? 'assertive' : 'polite'" :role="kind === 'error' ? 'alert' : 'status'"
      @mouseenter="stopTimer" @mouseleave="startTimer" @focusin="stopTimer" @focusout="startTimer">
      <CircleAlert v-if="kind === 'error'" :size="21" aria-hidden="true" />
      <Info v-else :size="21" aria-hidden="true" />
      <p>{{ message }}</p>
      <button type="button" aria-label="Cerrar aviso" @click="closed = true"><X :size="18" /></button>
    </div>
  </Teleport>
</template>

<style scoped>
.toast { position:fixed; z-index:10000; inset-block-start:max(1rem,env(safe-area-inset-top)); inset-inline-end:max(1rem,env(safe-area-inset-right)); display:flex; align-items:center; gap:.75rem; width:fit-content; max-width:min(28rem,calc(100vw - 2rem)); padding:.45rem .45rem .45rem 1rem; box-sizing:border-box; color:white; background:#163c4e; border-radius:10px; box-shadow:0 6px 24px #10253830; font-family:var(--tf-font-body) }
.toast.error { background:#84243a }
.toast > svg { flex:none }
p { margin:0; font-size:.95rem; line-height:1.45; overflow-wrap:anywhere }
button { flex:none; display:grid; place-items:center; width:44px; height:44px; border:0; border-radius:6px; background:transparent; color:inherit; cursor:pointer }
button:hover,button:focus-visible { background:#ffffff24 }
button:focus-visible { outline:2px solid white; outline-offset:-3px }
</style>
