<script setup lang="ts">
import { nextTick, onMounted, onUnmounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import type { Order } from './api'
import { orderNumberFromQuery } from './order-search'
import { canScanNatively, scanOrderCode } from '../../native/scanner'
import { X } from '@lucide/vue'

const props = defineProps<{ orders: Order[] }>()
const emit = defineEmits<{ close: [] }>()

const router = useRouter()
const scannerError = ref('')
const manualCodeInput = ref('')
const isScanning = ref(false)
const nativeScanner = canScanNatively()
const videoRef = ref<HTMLVideoElement | null>(null)
let mediaStream: MediaStream | null = null
let scanInterval: ReturnType<typeof setInterval> | null = null

function resolveScannedText(rawText: string): string | null {
  const text = rawText.trim()
  if (!text) return null

  // Case 1: Full URL like http://.../app/orders/:id or /app/orders/:id
  const urlMatch = text.match(/\/app\/orders\/([a-zA-Z0-9_-]+)/)
  if (urlMatch && urlMatch[1]) {
    return urlMatch[1]
  }

  // Case 2: Direct UUID (e.g. 019623e8-...)
  if (/^[0-9a-fA-F-]{16,}$/.test(text)) {
    return text
  }

  // A printed code or its short numeric form both identify the same order.
  const orderNumber = orderNumberFromQuery(text)
  const found = props.orders.find((o) => o.code.toUpperCase() === text.toUpperCase() || (orderNumber !== null && o.order_number === orderNumber))
  if (found) {
    return found.id
  }

  return null
}

async function openScanner() {
  scannerError.value = ''
  manualCodeInput.value = ''
  isScanning.value = true
  if (nativeScanner) {
    try {
      const code = await scanOrderCode()
      if (code) await handleCodeScanned(code)
      else scannerError.value = 'No se leyó ningún código. Puedes intentarlo de nuevo o escribirlo.'
    } catch {
      scannerError.value = 'No se pudo abrir el escáner. Escribe el código de la orden.'
    } finally {
      isScanning.value = false
    }
    return
  }
  await nextTick()
  if (!('BarcodeDetector' in window)) {
    scannerError.value = 'Este navegador no admite lectura automática. Ingresa el código de la orden manualmente.'
    isScanning.value = false
    return
  }
  // Start video stream if camera available
  if (navigator?.mediaDevices?.getUserMedia) {
    try {
      mediaStream = await navigator.mediaDevices.getUserMedia({
        video: { facingMode: 'environment', width: { ideal: 1280 }, height: { ideal: 720 } }
      })
      if (videoRef.value) {
        videoRef.value.srcObject = mediaStream
        await videoRef.value.play()
        startBarcodeDetection()
      } else {
        scannerError.value = 'No se pudo iniciar la vista de cámara. Ingresa el código manualmente.'
        isScanning.value = false
      }
    } catch (err: any) {
      scannerError.value = 'No se pudo acceder a la cámara. Puedes escribir el código de la orden manualmente.'
      isScanning.value = false
    }
  } else {
    scannerError.value = 'Cámara no soportada en este navegador. Ingresa el código manualmente.'
    isScanning.value = false
  }
}

function startBarcodeDetection() {
  // Check if standard BarcodeDetector is available
  if ('BarcodeDetector' in window) {
    try {
      const barcodeDetector = new (window as any).BarcodeDetector({ formats: ['qr_code', 'code_128', 'code_39'] })
      scanInterval = setInterval(async () => {
        if (!videoRef.value || videoRef.value.readyState < 2) return
        try {
          const barcodes = await barcodeDetector.detect(videoRef.value)
          if (barcodes.length > 0) {
            const rawValue = barcodes[0].rawValue
            if (rawValue) {
              handleCodeScanned(rawValue)
            }
          }
        } catch (_) {
          // ignore detection frame errors
        }
      }, 300)
      return
    } catch (_) {
      // Fallback
    }
  }
  scannerError.value = 'Este navegador no admite lectura automática. Ingresa el código de la orden manualmente.'
  isScanning.value = false
}

async function handleCodeScanned(text: string) {
  const orderId = resolveScannedText(text)
  if (orderId) {
    closeScanner()
    await router.push(`/app/orders/${orderId}`)
  } else {
    scannerError.value = `No se encontró la orden con código o enlace: "${text}"`
  }
}

async function handleManualSubmit() {
  if (!manualCodeInput.value.trim()) return
  await handleCodeScanned(manualCodeInput.value.trim())
}

function stopScanner() {
  if (scanInterval) {
    clearInterval(scanInterval)
    scanInterval = null
  }
  if (mediaStream) {
    mediaStream.getTracks().forEach((track) => track.stop())
    mediaStream = null
  }
  isScanning.value = false
}

function closeScanner() {
  stopScanner()
  emit('close')
}

onMounted(openScanner)
onUnmounted(stopScanner)
</script>

<template>
  <div
    class="scanner-modal-backdrop"
    role="dialog"
    aria-modal="true"
    aria-labelledby="scanner-title"
  >
    <div class="scanner-modal">
      <header class="scanner-modal__header">
        <div>
          <h2 id="scanner-title">Escanear Ficha Viajera</h2>
          <p class="scanner-modal__subtitle">Apunta al código impreso o a un QR estándar del lote</p>
        </div>
        <button class="scanner-modal__close" type="button" @click="closeScanner" aria-label="Cerrar escáner">
          <X :size="20" aria-hidden="true" />
        </button>
      </header>
  
      <div class="scanner-modal__body">
        <div v-if="!nativeScanner" class="video-container">
          <video
            ref="videoRef"
            class="video-preview"
            autoplay
            playsinline
            muted
          ></video>
          <div class="scanner-reticle">
            <div class="reticle-corner reticle-corner--tl"></div>
            <div class="reticle-corner reticle-corner--tr"></div>
            <div class="reticle-corner reticle-corner--bl"></div>
            <div class="reticle-corner reticle-corner--br"></div>
            <div class="scanner-laser"></div>
          </div>
        </div>
  
        <p v-if="scannerError" class="scanner-error-text">{{ scannerError }}</p>
        <button v-if="nativeScanner" class="manual-submit-btn" type="button" :disabled="isScanning" @click="openScanner">
          {{ isScanning ? 'Abriendo cámara...' : 'Escanear de nuevo' }}
        </button>
  
        <form class="manual-input-form" @submit.prevent="handleManualSubmit">
          <label for="manual-code-input">O ingresa el código o ID de la orden:</label>
          <div class="manual-input-row">
            <input
              id="manual-code-input"
              v-model="manualCodeInput"
              type="text"
              placeholder="Ej. OT-000003, 3 o pega el enlace"
              autocomplete="off"
            />
            <button class="manual-submit-btn" type="submit" @click.prevent="handleManualSubmit">
              Abrir orden
            </button>
          </div>
        </form>
      </div>
    </div>
  </div>
</template>

<style scoped>
.scanner-modal-backdrop {
  position: fixed;
  inset: 0;
  background: var(--tf-overlay);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 999;
  padding: 1rem;
}

.scanner-modal {
  background: var(--tf-surface);
  border-radius: 1rem;
  max-width: 32rem;
  width: 100%;
  overflow: hidden;
  box-shadow: 0 20px 25px -5px rgba(0, 0, 0, 0.2);
}

.scanner-modal__header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  padding: 1.25rem 1.5rem;
  background: var(--tf-ink);
  color: var(--tf-on-primary);
}

.scanner-modal__header h2 {
  margin: 0 0 .25rem;
  font-size: 1.2rem;
  color: var(--tf-on-primary);
}

.scanner-modal__subtitle {
  margin: 0;
  font-size: .85rem;
  color: var(--tf-border);
}

.scanner-modal__close {
  background: transparent;
  border: none;
  color: var(--tf-border-soft);
  font-size: 1.3rem;
  cursor: pointer;
  padding: .25rem;
}

.scanner-modal__body {
  padding: 1.5rem;
  display: grid;
  gap: 1.25rem;
}

.video-container {
  position: relative;
  width: 100%;
  height: 220px;
  background: var(--tf-ink);
  border-radius: .75rem;
  overflow: hidden;
  display: flex;
  align-items: center;
  justify-content: center;
}

.video-preview {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.scanner-reticle {
  position: absolute;
  width: 150px;
  height: 150px;
  pointer-events: none;
}

.reticle-corner {
  position: absolute;
  width: 20px;
  height: 20px;
  border-color: var(--tf-shell-accent);
}

.reticle-corner--tl { top: 0; left: 0; border-top: 3px solid; border-left: 3px solid; }
.reticle-corner--tr { top: 0; right: 0; border-top: 3px solid; border-right: 3px solid; }
.reticle-corner--bl { bottom: 0; left: 0; border-bottom: 3px solid; border-left: 3px solid; }
.reticle-corner--br { bottom: 0; right: 0; border-bottom: 3px solid; border-right: 3px solid; }

.scanner-laser {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  height: 2px;
  background: var(--tf-shell-accent);
  animation: laser-sweep 2s infinite ease-in-out;
}

@keyframes laser-sweep {
  0% { top: 0; }
  50% { top: 100%; }
  100% { top: 0; }
}

.scanner-error-text {
  margin: 0;
  padding: .75rem;
  background: var(--tf-danger-soft);
  border-radius: .375rem;
  color: var(--tf-danger);
  font-size: .85rem;
}

.manual-input-form {
  display: grid;
  gap: .5rem;
}

.manual-input-form label {
  font-size: .85rem;
  font-weight: 700;
  color: var(--tf-ink-soft);
}

.manual-input-row {
  display: flex;
  gap: .5rem;
}

.manual-input-row input {
  flex: 1;
  min-height: 42px;
  border: 1px solid var(--tf-border-soft);
  border-radius: .5rem;
  padding: .5rem .75rem;
  font: inherit;
}

.manual-submit-btn {
  background: var(--tf-primary);
  color: var(--tf-on-primary);
  border: none;
  border-radius: .5rem;
  padding: .5rem 1rem;
  font: inherit;
  font-weight: 700;
  cursor: pointer;
  white-space: nowrap;
}

.manual-submit-btn:hover {
  background: var(--tf-primary-hover);
}

@media (max-width: 640px) {
  .manual-input-row { flex-direction: column; }
  .manual-submit-btn { min-height: 44px; }
}

@media (prefers-reduced-motion: reduce) {
  .scanner-laser { animation: none; }
}
</style>
